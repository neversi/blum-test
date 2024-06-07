package service

import (
	"blum-test/common/models"
	"context"
	"fmt"
)

func (c *RateCalculator) listenCurrencyUpdates(ctx context.Context) error {
	notificationChan, err := c.repo.SubscribeToCurrencyUpdates(ctx)
	if err != nil {
		return fmt.Errorf("error subscribing to currency updates: %w", err)
	}

	for {
		select {
		case <-ctx.Done():
			log.Info("finishing polling currency updates")
			return nil

		case notification := <-notificationChan:
			currency := models.Currency{
				Name:      notification.Currency.Name,
				Code:      models.CurrencyCode(notification.Currency.Code),
				Type:      notification.Currency.CurrencyType,
				IsEnabled: notification.Currency.IsEnabled,
			}

			if notification.Operation == "DELETE" {
				currency.IsEnabled = false
			}

			updatedCurrencies := make(map[models.CurrencyCode]models.Currency)
			c.currencies.Range(func(key models.CurrencyCode, value models.Currency) bool {
				updatedCurrencies[key] = value
				return true
			})
			c.fetchRates(ctx, updatedCurrencies)
			c.updateCurrencies([]models.Currency{currency})
		}
	}
}
