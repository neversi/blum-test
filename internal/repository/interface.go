package repository

import (
	"blum-test/common/models"
	"context"
	"time"
)

type ICurrencyRepository interface {
	ListEnabledCurrencies(ctx context.Context) ([]models.Currency, error)
	ListCurrenciesByUpdatedAtGt(ctx context.Context, updatedAt time.Time) ([]models.Currency, error)

	GetLastUpdatedCurrency(ctx context.Context) (*models.Currency, error)
}
