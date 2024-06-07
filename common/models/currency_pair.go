package models

import (
	"fmt"
)

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
