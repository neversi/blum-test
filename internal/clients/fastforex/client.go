package fastforex

import (
	"blum-test/common/config"
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

	if _, err := c.GetRatesByUSD(context.Background(), []string{"EUR"}); err != nil && errors.Is(err, ErrInvalidAPIKey) {
		return nil, ErrInvalidAPIKey
	}

	return c, nil
}

func (c *Client) GetRatesByUSD(ctx context.Context, codes []string) (map[string]json.Number, error) {
	toParam := strings.Builder{}
	for i, code := range codes {
		if i == len(codes)-1 {
			toParam.WriteString(code)
		} else {
			toParam.WriteString(code + ",")
		}
	}
	fmt.Println(toParam.String())

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
	defer resp.RawBody().Close()

	if resp.StatusCode() == http.StatusTooManyRequests {
		return nil, ErrRateLimit
	}

	if resp.StatusCode() == http.StatusUnauthorized {
		return nil, ErrInvalidAPIKey
	}

	if resp.StatusCode() != http.StatusOK {
		return nil, fmt.Errorf("error while fetching rates: %v", resp.String())
	}

	var payload struct {
		Results map[string]json.Number `json:"results"`
	}

	if err := jsoniter.ConfigCompatibleWithStandardLibrary.NewDecoder(
		resp.RawBody(),
	).Decode(&payload); err != nil {
		return nil, fmt.Errorf("error while decoding fetching rates: %w", err)
	}

	return payload.Results, nil
}
