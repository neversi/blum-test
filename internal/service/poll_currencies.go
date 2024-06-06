package service

import (
	"context"
	"fmt"
	"time"
)

func (c *RateCalculator) pollCurrencyUpdates(ctx context.Context) error {
	var lastUpdatedAt time.Time
	lastUpdatedAtCurrency, err := c.repo.GetLastUpdatedCurrency(ctx)
	if err != nil {
		return fmt.Errorf("error while fetching last updated at currency: %w", err)
	}

	lastUpdatedAt = lastUpdatedAtCurrency.UpdatedAt

	ticker := time.NewTicker(c.CurrencyPollingInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Info("finishing polling currency updates")
			return nil
		case <-ticker.C:
			currencies, err := c.repo.ListCurrenciesByUpdatedAtGt(ctx, lastUpdatedAt)
			if err != nil {
				retErr := fmt.Errorf("error while fetching from db: %w", err)
				log.Error("pollCurrencyUpdates", retErr)
				return retErr
			}

			c.updateCurrencies(currencies)
		}
	}
}
