package models

import (
	"encoding/json"
	"time"
)

type (
	CurrencyType string
	CurrencyCode string
)

const (
	Fiat   CurrencyType = "FIAT"
	Crypto CurrencyType = "CRYPTO"

	// Gold CurrencyType = "GOLD"
)

const (
	USD  CurrencyCode = "USD"
	USDT CurrencyCode = "USDT"
)

type Currency struct {
	Name      string
	Code      CurrencyCode
	Type      CurrencyType
	IsEnabled bool
	UpdatedAt time.Time
}

func (c *CurrencyCode) UnmarshalJSON(data []byte) error {
	var res string
	if err := json.Unmarshal(data, &res); err != nil {
		return err
	}

	*c = CurrencyCode(res)

	return nil
}
