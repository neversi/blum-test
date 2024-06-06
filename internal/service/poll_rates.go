package service

import (
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
			if err := c.fetchAndUpdateRates(ctx); err != nil {
				retErr := fmt.Errorf("error while fetching rates: %w", err)
				log.Error("pollRates", retErr)
				return retErr
			}
		}
	}
}
