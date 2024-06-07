package service

import (
	"blum-test/common/models"
	"context"
	"fmt"
	"time"
)

func (c *RateCalculator) pollRates(ctx context.Context) error {
	ticker := time.NewTicker(c.RatePollingInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Info("finishing rate polling")
			return nil
		case <-ticker.C:
			currencies := make(map[models.CurrencyCode]models.Currency)
			c.currencies.Range(func(key models.CurrencyCode, value models.Currency) bool {
				currencies[key] = value
				return true
			})
			if err := c.fetchRates(ctx, currencies); err != nil {
				retErr := fmt.Errorf("error while fetching rates: %w", err)
				log.Error("error polling rates", retErr)
				return retErr
			}
		}
	}
}
