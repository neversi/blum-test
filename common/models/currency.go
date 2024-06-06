package models

import (
	"fmt"
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

type Currency struct {
	Name      string
	Code      CurrencyCode
	Type      CurrencyType
	IsEnabled bool
	UpdatedAt time.Time
}

type CurrencyPair struct {
	Base  Currency
	Quote Currency
}

func (i CurrencyPair) Validate() error {
	if !i.Base.IsEnabled {
		return &ErrCurrencyNotAvailable{
			Code: i.Base.Code,
		}
	}

	if !i.Quote.IsEnabled {
		return &ErrCurrencyNotAvailable{
			Code: i.Quote.Code,
		}
	}

	if i.Base.Type == i.Quote.Type {
		return &ErrInvalidCurrencyPair{
			Base:  &i.Base,
			Quote: &i.Quote,
		}
	}

	return nil
}

func (i CurrencyPair) String() string {
	return fmt.Sprintf("%s/%s", i.Base.Code, i.Quote.Code)
}
