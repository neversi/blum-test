package service

import (
	"blum-test/common/config"
	"blum-test/common/logger"
	"blum-test/common/models"
	"blum-test/internal/clients/fastforex"
	"blum-test/internal/repository"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"math/big"
	"strings"
	"sync"
	"sync/atomic"

	"golang.org/x/sync/errgroup"
)

type RateCalculator struct {
	config.Service
	isStarted int32

	currenciesMu sync.RWMutex
	currencies   map[models.CurrencyCode]models.Currency

	ratesMu    sync.RWMutex
	ratesToUSD map[models.CurrencyCode]*big.Float

	repo   repository.ICurrencyRepository
	client *fastforex.Client
}

var log = logger.JSONLogger.With(slog.String("service", "rate_calculator"))

func NewRateCalculator(cfg *config.Service, repo repository.ICurrencyRepository, client *fastforex.Client) *RateCalculator {
	return &RateCalculator{
		Service:    *cfg,
		currencies: make(map[models.CurrencyCode]models.Currency),
		ratesToUSD: make(map[models.CurrencyCode]*big.Float),

		repo:   repo,
		client: client,
	}
}

func (c *RateCalculator) getIsStarted() bool {
	val := atomic.LoadInt32(&c.isStarted)
	if val == 0 {
		return false
	}
	return true
}

func (c *RateCalculator) setIsStarted(to bool) {
	if to {
		atomic.StoreInt32(&c.isStarted, 1)
	}
	atomic.StoreInt32(&c.isStarted, 0)
}

func (c *RateCalculator) Start(ctx context.Context) error {
	if c.getIsStarted() {
		return ErrServiceStarted
	}
	c.setIsStarted(true)
	defer c.setIsStarted(false)

	log.Info("starting service...")

	workerGroup, ctxEG := errgroup.WithContext(ctx)

	log.Debug("starting polling currencies...")
	workerGroup.Go(func() error {
		return c.pollCurrencyUpdates(ctxEG)
	})

	log.Debug("getting initial currencies and rates")
	if err := c.fetchAndUpdateCurrencies(ctx); err != nil {
		return err
	}

	if err := c.fetchAndUpdateRates(ctx); err != nil {
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

func (c *RateCalculator) Convert(
	ctx context.Context,
	base, quote string,
	amount float64,
) (res float64, err error) {
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

	pair := models.CurrencyPair{
		Base:  *baseCurrency,
		Quote: *quoteCurrency,
	}

	if err := pair.Validate(); err != nil {
		return 0, err
	}

	baseUsdRate, quoteUsdRate := c.getRate(base, quote)
	if baseUsdRate.Cmp(big.NewFloat(0)) == 0 {
		log.Error("zero denominator", slog.Any("currency_code", baseCurrency.Code))
		return 0, ErrInvalidInternalRate
	}

	crossRate := new(big.Float).Quo(quoteUsdRate, baseUsdRate)

	res, _ = crossRate.Float64()

	return res, nil
}

func (c *RateCalculator) updateCurrencies(currencies []models.Currency) {
	c.currenciesMu.Lock()
	defer c.currenciesMu.Unlock()

	for _, currency := range currencies {
		c.currencies[currency.Code] = currency
	}
}

func (c *RateCalculator) getCurrency(code string) (*models.Currency, error) {
	code = strings.ToUpper(code)

	c.currenciesMu.RLock()
	defer c.currenciesMu.RUnlock()

	currency, ok := c.currencies[models.CurrencyCode(code)]
	if !ok {
		return nil, &models.ErrCurrencyNotAvailable{
			Code: models.CurrencyCode(code),
		}
	}

	return &currency, nil
}

func (c *RateCalculator) getRate(base, quote string) (*big.Float, *big.Float) {
	base = strings.ToUpper(base)
	quote = strings.ToUpper(quote)

	c.ratesMu.RLock()
	defer c.ratesMu.RUnlock()

	return c.ratesToUSD[models.CurrencyCode(base)], c.ratesToUSD[models.CurrencyCode(quote)]
}

func (c *RateCalculator) updateRates(ratesUSD map[string]float64) {
	c.ratesMu.Lock()
	defer c.ratesMu.Unlock()

	for code, rateUSD := range ratesUSD {
		c.ratesToUSD[models.CurrencyCode(code)] = big.NewFloat(rateUSD)
	}
}

func (c *RateCalculator) fetchAndUpdateRates(ctx context.Context) error {
	quotes := []string{}

	c.currenciesMu.RLock()
	for code := range c.currencies {
		quotes = append(quotes, string(code))
	}
	c.currenciesMu.RUnlock()

	ratesUSD, err := c.client.GetRatesByUSD(ctx, quotes)
	if err != nil {
		return fmt.Errorf("error while fetching rates: %w", err)
	}

	ratesFloat := map[string]float64{}
	for code, rateUSD := range ratesUSD {
		rateFloat, err := rateUSD.Float64()
		if err != nil {
			log.Error("invalid rate", slog.String("actual_value", rateUSD.String()), err)
			return fmt.Errorf("invalid rate from response: %w", err)
		}
		ratesFloat[code] = rateFloat
	}

	c.updateRates(ratesFloat)

	return nil
}

func (c *RateCalculator) fetchAndUpdateCurrencies(ctx context.Context) error {
	currencies, err := c.repo.ListEnabledCurrencies(ctx)
	if err != nil {
		return fmt.Errorf("error while fetching from db: %w", err)
	}

	c.updateCurrencies(currencies)
	return nil
}
