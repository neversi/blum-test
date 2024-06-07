package service

import (
	"blum-test/common/config"
	"blum-test/common/logger"
	. "blum-test/common/models"
	"blum-test/common/utils"
	"blum-test/internal/clients/fastforex"
	"blum-test/internal/repository"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"sync/atomic"

	"github.com/shopspring/decimal"

	"golang.org/x/sync/errgroup"
)

// RateCalculator stores and updates data about currency pairs' rates
// in order to maintain data relevance. Also can make convertations
// between two currencies through USD cross-rates.
type RateCalculator struct {
	ctx    context.Context
	cancel context.CancelFunc

	config.Service
	isRunning int32

	// currencies is the map which under the hoods use sync.Map
	// the reason why sync.Map is better (imho) than other
	// primitive synchronization methods is that in highly
	// concurrent (huge DAU/high RPS) environment
	// mechanisms like mutexescould delay data relevance because
	// of high amount of read operations while sync.Map by
	// distributing values by shards provides more sofisticated
	// writes operation which is not blocked by highly read operations
	//
	// also mutex synchronization could be used too, though it
	// could lead to some delayed rates updates, due to its nature
	currencies utils.MapThSf[CurrencyCode, Currency]
	// same logic applied here
	ratesInUSD utils.MapThSf[CurrencyCode, decimal.Decimal]

	repo   repository.ICurrencyRepository
	client *fastforex.Client
}

var log = logger.JSONLogger.With(slog.String("service", "rate_calculator"))

func NewRateCalculator(
	cfg *config.Service,
	repo repository.ICurrencyRepository,
	client *fastforex.Client,
) *RateCalculator {
	return &RateCalculator{
		Service: *cfg,

		repo:   repo,
		client: client,
	}
}

func (c *RateCalculator) getIsRunning() bool {
	val := atomic.LoadInt32(&c.isRunning)
	if val == 0 {
		return false
	}
	return true
}

func (c *RateCalculator) setIsRunning(to bool) {
	if to {
		atomic.StoreInt32(&c.isRunning, 1)
	}
	atomic.StoreInt32(&c.isRunning, 0)
}

func (c *RateCalculator) Start() error {
	if c.getIsRunning() {
		return ErrServiceStarted
	}
	c.setIsRunning(true)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	c.ctx = ctx
	c.cancel = cancel

	defer c.setIsRunning(false)

	log.Info("starting service...")

	workerGroup, ctxEG := errgroup.WithContext(ctx)

	log.Debug("starting polling currencies...")
	workerGroup.Go(func() error {
		return c.listenCurrencyUpdates(ctxEG)
	})

	log.Debug("getting initial currencies and rates")
	if err := c.fetchEnabledCurrencies(ctx); err != nil {
		return err
	}

	currencies := make(map[CurrencyCode]Currency)

	c.currencies.Range(func(key CurrencyCode, value Currency) bool {
		currencies[key] = value
		return true
	})

	if err := c.fetchRates(ctx, currencies); err != nil {
		return err
	}

	log.Debug("starting polling rates...")
	workerGroup.Go(func() error {
		return c.pollRates(ctxEG)
	})

	if err := workerGroup.Wait(); err != nil && !errors.Is(err, context.Canceled) {
		return fmt.Errorf("error while executing group: %w", err)
	}

	return nil
}

func (c *RateCalculator) Stop() {
	if c.cancel != nil {
		c.cancel()
		c.cancel = nil
	}
}

func (c *RateCalculator) Convert(
	ctx context.Context,
	base, quote string,
	amount float64,
	decimals int64,
) (res float64, err error) {
	base = strings.ToUpper(base)
	quote = strings.ToUpper(quote)
	defer func() {
		if r := recover(); r != nil {
			log.Error("panic while converting", r)
			err = ErrServiceInternal
		}
	}()
	baseCurrency, err := c.getCurrency(base)
	if err != nil {
		return 0, err
	}
	quoteCurrency, err := c.getCurrency(quote)
	if err != nil {
		return 0, err
	}

	pair := CurrencyPair{
		Base:  *baseCurrency,
		Quote: *quoteCurrency,
	}

	if err := pair.Validate(); err != nil {
		return 0, err
	}

	baseUsdRate, ok := c.ratesInUSD.Load(CurrencyCode(base))
	if !ok || baseUsdRate.Cmp(decimal.Zero) == 0 {
		log.Error("zero denominator", slog.Any("currency_code", baseCurrency.Code))
		return 0, ErrInvalidInternalRate
	}

	quoteUsdRate, ok := c.ratesInUSD.Load(CurrencyCode(quote))
	if !ok {
		return 0, ErrInvalidInternalRate
	}

	fmt.Println(baseUsdRate, quoteUsdRate)

	crossRate := quoteUsdRate.Div(baseUsdRate)

	convertAmount := crossRate.Mul(decimal.NewFromFloat(amount))

	res, _ = convertAmount.Round(int32(decimals)).Float64()

	return res, nil
}

func (c *RateCalculator) updateCurrencies(currencies []Currency) {
	for _, currency := range currencies {
		if currency.IsEnabled {
			c.currencies.Store(currency.Code, currency)
		} else {
			c.currencies.Delete(currency.Code)
		}
	}
}

func (c *RateCalculator) getCurrency(code string) (*Currency, error) {
	code = strings.ToUpper(code)

	currency, ok := c.currencies.Load(CurrencyCode(code))
	if !ok {
		return nil, &ErrCurrencyNotAvailable{
			Code: CurrencyCode(code),
		}
	}

	return &currency, nil
}

func (c *RateCalculator) fetchUsdtUsdRate(ctx context.Context) (float64, error) {
	usdtUsdRates, err := c.client.GetCryptoRates(
		ctx,
		[]CurrencyCode{USDT},
		USD,
	)
	if err != nil {
		return 0, fmt.Errorf("GetCryptoRates(): %w", err)
	}

	usdtUsdRate, err := usdtUsdRates.Rates[USDT].Float64()
	if err != nil {
		log.Error(
			"invalid USDT/USD rate",
			slog.String("actual_value", usdtUsdRates.Rates[USDT].String()),
			slog.Any("error", err),
		)
		return 0, fmt.Errorf("invalid USDT/USD rate: %w", err)
	}

	return usdtUsdRate, nil
}

func (c *RateCalculator) fetchRates(
	ctx context.Context,
	currencies map[CurrencyCode]Currency,
) error {
	fiatCodes := []CurrencyCode{}
	cryptoCodes := []CurrencyCode{}

	for code, val := range currencies {
		switch val.Type {
		case Crypto:
			cryptoCodes = append(cryptoCodes, code)
		case Fiat:
			fiatCodes = append(fiatCodes, code)
		}
	}

	usdtUsdRate, err := c.fetchUsdtUsdRate(ctx)
	if err != nil {
		return fmt.Errorf("getUsdtUsdRate(): %w", err)
	}
	_ = usdtUsdRate

	fiatRates, err := c.client.GetFiatRates(ctx, USD, fiatCodes)
	if err != nil {
		return fmt.Errorf("GetFiatRates(): %w", err)
	}

	cryptoRates, err := c.client.GetCryptoRates(ctx, cryptoCodes, USDT)
	if err != nil {
		return fmt.Errorf("GetCryptoRates(): %w", err)
	}

	ratesFloat := map[CurrencyCode]float64{}
	for code, rateUSD := range fiatRates.Rates {
		rateFloat, err := rateUSD.Float64()
		if err != nil {
			log.Error("invalid rate", slog.String("actual_value", rateUSD.String()), err)
			return fmt.Errorf("invalid rate from response: %w", err)
		}

		ratesFloat[CurrencyCode(code)] = rateFloat
	}

	for code, rateUSD := range cryptoRates.Rates {
		rateFloat, err := rateUSD.Float64()
		if err != nil {
			log.Error("invalid rate", slog.String("actual_value", rateUSD.String()), err)
			return fmt.Errorf("invalid rate from response: %w", err)
		}

		ratesFloat[code] = 1 / (rateFloat) * usdtUsdRate
	}

	ratesFloat[USDT] = usdtUsdRate
	fmt.Println(ratesFloat)

	for code, rateUSD := range ratesFloat {
		c.ratesInUSD.Store(code, decimal.NewFromFloat(rateUSD))
	}

	return nil
}

func (c *RateCalculator) fetchEnabledCurrencies(ctx context.Context) error {
	currencies, err := c.repo.ListEnabledCurrencies(ctx)
	if err != nil {
		return fmt.Errorf("ListEnabledCurrencies(): %w", err)
	}

	c.updateCurrencies(currencies)
	return nil
}
