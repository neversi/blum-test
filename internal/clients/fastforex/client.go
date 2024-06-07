package fastforex

import (
	"blum-test/common/config"
	"blum-test/common/models"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/go-resty/resty/v2"
	jsoniter "github.com/json-iterator/go"
)

const (
	fastForexBaseURL     = "https://api.fastforex.io"
	fastForexApiKeyParam = "api_key"
)

type Client struct {
	apiKey string
	cli    *resty.Client
}

func NewClient(cfg *config.FastForex) (*Client, error) {
	if cfg.ApiKey == "" {
		return nil, ErrInvalidAPIKey
	}

	cli := resty.New().
		SetTimeout(cfg.RequestTimeout).
		SetBaseURL(fastForexBaseURL).
		SetRetryCount(cfg.RetriesCount).
		SetQueryParam(fastForexApiKeyParam, cfg.ApiKey)

	c := &Client{
		apiKey: cfg.ApiKey,
		cli:    cli,
	}

	if _, err := c.GetFiatRates(context.Background(), "USD", []models.CurrencyCode{"EUR"}); err != nil && errors.Is(err, ErrInvalidAPIKey) {
		return nil, ErrInvalidAPIKey
	}

	return c, nil
}

const cryptoPairsLimitPerRequest = 10
const workersCount = 3

type CryptoRatesResponse struct {
	Rates map[models.CurrencyCode]json.Number
}

func (c *Client) GetCryptoRates(
	ctx context.Context,
	bases []models.CurrencyCode,
	quote models.CurrencyCode,
) (*CryptoRatesResponse, error) {
	if len(bases) == 0 {
		return &CryptoRatesResponse{
			Rates: make(map[models.CurrencyCode]json.Number),
		}, nil
	}
	requestsParam := []string{}

	pairs := strings.Builder{}
	for i := 0; i < len(bases); i += cryptoPairsLimitPerRequest {
		pairs.Reset()
		for j := i; j < i+cryptoPairsLimitPerRequest && j < len(bases); j++ {
			pairs.WriteString(string(bases[j]))
			pairs.WriteString(string("/" + quote))

			if j < i+cryptoPairsLimitPerRequest-1 && j < len(bases)-1 {
				pairs.WriteRune(',')
			}
		}
		requestsParam = append(requestsParam, pairs.String())
	}

	type result struct {
		rates map[string]json.Number
		err   error
	}

	requests := make(chan string, len(requestsParam))
	results := make(chan result, len(requestsParam))

	makeRequest := func(ctx context.Context, request string) result {
		var resp *resty.Response
		var err error
		if err := resty.Backoff(
			func() (*resty.Response, error) {
				resp, err = c.cli.R().SetQueryParams(
					map[string]string{
						"pairs": request,
					},
				).
					SetContext(ctx).
					Get("/crypto/fetch-prices")

				return resp, err
			},
		); err != nil {
			return result{nil, fmt.Errorf("could not make a request: %w", err)}
		}
		defer resp.RawBody().Close()

		if resp.StatusCode() == http.StatusTooManyRequests {
			return result{nil, ErrRateLimit}
		}

		if resp.StatusCode() == http.StatusUnauthorized {
			return result{nil, ErrInvalidAPIKey}
		}

		if resp.StatusCode() != http.StatusOK {
			return result{nil, fmt.Errorf("error response /crypto/fetch-prices: %v", resp.String())}
		}

		var payload struct {
			Prices map[string]json.Number `json:"prices"`
		}

		if err := jsoniter.ConfigCompatibleWithStandardLibrary.Unmarshal(resp.Body(), &payload); err != nil {
			return result{nil, fmt.Errorf("error while decoding crypto rates: %w", err)}
		}

		return result{payload.Prices, nil}
	}

	worker := func(ctx context.Context, requests <-chan string, results chan<- result) {
		for request := range requests {
			results <- makeRequest(ctx, request)
		}
	}

	for i := 0; i < workersCount; i++ {
		go worker(ctx, requests, results)
	}

	for i := 0; i < len(requestsParam); i++ {
		requests <- requestsParam[i]
	}

	close(requests)

	res := CryptoRatesResponse{
		Rates: make(map[models.CurrencyCode]json.Number),
	}

	for i := 0; i < len(requestsParam); i++ {
		result := <-results
		if result.err != nil {
			return nil, fmt.Errorf("error while requesting crypto rates: %w", result.err)
		}

		for pair, rate := range result.rates {
			base := strings.ToUpper(strings.Split(pair, "/")[0])
			res.Rates[models.CurrencyCode(base)] = rate
		}
	}

	return &res, nil
}

type FiatRatesResponse struct {
	Rates map[models.CurrencyCode]json.Number `json:"results"`
}

func (c *Client) GetFiatRates(
	ctx context.Context,
	base models.CurrencyCode,
	quotes []models.CurrencyCode,
) (*FiatRatesResponse, error) {
	if len(quotes) == 0 {
		return &FiatRatesResponse{
			Rates: make(map[models.CurrencyCode]json.Number),
		}, nil
	}
	toParam := strings.Builder{}
	for i, code := range quotes {
		toParam.WriteString(string(code))
		if i < len(quotes)-1 {
			toParam.WriteRune(',')
		}
	}

	var resp *resty.Response
	var err error

	if err := resty.Backoff(
		func() (*resty.Response, error) {
			resp, err = c.cli.R().SetQueryParams(
				map[string]string{
					"from": "USD",
					"to":   toParam.String(),
				},
			).
				SetContext(ctx).
				Get("/fetch-multi")

			return resp, err
		},
	); err != nil {
		return nil, fmt.Errorf("could not make a request: %w", err)
	}

	if resp.StatusCode() == http.StatusTooManyRequests {
		return nil, ErrRateLimit
	}

	if resp.StatusCode() == http.StatusUnauthorized {
		return nil, ErrInvalidAPIKey
	}

	if resp.StatusCode() != http.StatusOK {
		return nil, fmt.Errorf("error response /fetch-multi: %v", resp.String())
	}

	var payload FiatRatesResponse

	if err := jsoniter.ConfigCompatibleWithStandardLibrary.Unmarshal(resp.Body(), &payload); err != nil {
		return nil, fmt.Errorf("error while decoding fiat rates: %w", err)
	}

	return &payload, nil
}
