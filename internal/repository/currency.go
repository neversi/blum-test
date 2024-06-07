package repository

import (
	"blum-test/common/logger"
	"blum-test/common/models"
	"blum-test/internal/db"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/jackc/pgx/v4/pgxpool"
)

type repo struct {
	client *pgxpool.Pool

	currencyTypes map[int]models.CurrencyType
}

func NewCurrencyPostgresRepository(client *pgxpool.Pool) ICurrencyRepository {
	return &repo{
		client:        client,
		currencyTypes: make(map[int]models.CurrencyType),
	}
}

func (r *repo) ListEnabledCurrencies(ctx context.Context) ([]models.Currency, error) {
	query := `
		SELECT c.name, c.code, t.name, c.is_enabled, c.updated_at
		FROM currencies c LEFT JOIN currency_types t ON c.type_id = t.id
		WHERE c.is_enabled = true;
	`

	rows, err := r.client.Query(ctx, query)
	if err != nil && !db.CheckErrNoRows(err) {
		return nil, fmt.Errorf("error while quering db: %w", err)
	}
	defer rows.Close()

	res := []models.Currency{}
	for rows.Next() {
		currency := models.Currency{}
		if err := rows.Scan(
			&currency.Name,
			&currency.Code,
			&currency.Type,
			&currency.IsEnabled,
			&currency.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("error while scanning values: %w", err)
		}

		res = append(res, currency)
	}

	return res, nil
}

func (r *repo) updateCurrencyTypeMap(ctx context.Context) error {
	query := `
		SELECT id, name FROM currency_types;
	`

	rows, err := r.client.Query(ctx, query)
	if err != nil && !db.CheckErrNoRows(err) {
		return fmt.Errorf("error while quering db: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var typeId int
		var currencyType string
		if err := rows.Scan(
			&typeId,
			&currencyType,
		); err != nil {
			return fmt.Errorf("error while scanning values: %w", err)
		}

		r.currencyTypes[typeId] = models.CurrencyType(currencyType)
	}

	return nil
}

func (r *repo) SubscribeToCurrencyUpdates(ctx context.Context) (<-chan CurrencyNotification, error) {
	if err := r.updateCurrencyTypeMap(ctx); err != nil {
		return nil, fmt.Errorf("could not update currency types: %w", err)
	}

	conn, err := r.client.Acquire(ctx)
	if err != nil {
		return nil, fmt.Errorf("could not acquire conn: %w", err)
	}

	res := make(chan CurrencyNotification, 10)

	_, err = conn.Exec(ctx, "LISTEN currency_events")
	if err != nil {
		return nil, fmt.Errorf("could not LISTEN to currency events: %w", err)
	}

	go func() {
		defer conn.Release()
		for {
			notification, err := conn.Conn().WaitForNotification(ctx)
			if err != nil {
				logger.JSONLogger.Error(
					"error waiting notification: %w",
					err,
				)
				return
			}

			if err := r.updateCurrencyTypeMap(ctx); err != nil {
				logger.JSONLogger.Error(
					"could not update currency types: %w",
					err,
				)
				return
			}

			payload := CurrencyNotification{}

			if err := json.Unmarshal([]byte(notification.Payload), &payload); err != nil {
				logger.JSONLogger.Error(
					"unexpected event in the channel",
					slog.Any("payload", notification.Payload),
					slog.Any("error", err),
				)
			}

			payload.Currency.CurrencyType = r.currencyTypes[payload.Currency.TypeId]

			res <- payload
		}
	}()

	return res, nil
}
