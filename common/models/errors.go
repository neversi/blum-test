package models

import "fmt"

type ErrInvalidCurrencyPair struct {
	Base  *Currency
	Quote *Currency
}

func (e *ErrInvalidCurrencyPair) Error() string {
	return fmt.Sprintf(
		"currency pair \"%s/%s\" types is not compatible (%s/%s)",
		e.Base.Code, e.Quote.Code, e.Base.Type, e.Quote.Type,
	)
}

type ErrCurrencyNotAvailable struct {
	Code CurrencyCode
}

func (e *ErrCurrencyNotAvailable) Error() string {
	return fmt.Sprintf("currency with code \"%s\" is not available for convertion", e.Code)
}
