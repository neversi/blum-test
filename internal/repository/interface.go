package repository

import (
	"blum-test/common/models"
	"context"
)

type ICurrencyRepository interface {
	ListEnabledCurrencies(ctx context.Context) ([]models.Currency, error)
	SubscribeToCurrencyUpdates(ctx context.Context) (<-chan CurrencyNotification, error)
}

type CurrencyNotification struct {
	Operation string `json:"operation"`
	Currency  struct {
		Name         string              `json:"name"`
		Code         string              `json:"code"`
		IsEnabled    bool                `json:"is_enabled"`
		TypeId       int                 `json:"type_id"`
		CurrencyType models.CurrencyType `json:"-"`
	} `json:"currency"`
}
