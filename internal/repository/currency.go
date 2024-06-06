package repository

import (
	"blum-test/common/models"
	"blum-test/internal/db"
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v4/pgxpool"
)

type repo struct {
	client *pgxpool.Pool
}

func NewCurrencyPostgresRepository(client *pgxpool.Pool) ICurrencyRepository {
	return &repo{
		client: client,
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

func (r *repo) ListCurrenciesByUpdatedAtGt(ctx context.Context, updatedAt time.Time) ([]models.Currency, error) {
	query := `
		SELECT c.name, c.code, t.name, c.is_enabled, c.updated_at
		FROM currencies c LEFT JOIN currency_types t ON c.type_id = t.id
		WHERE c.updated_at > $1;
	`

	rows, err := r.client.Query(ctx, query, updatedAt)
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

func (r *repo) GetLastUpdatedCurrency(ctx context.Context) (*models.Currency, error) {
	query := `
		SELECT c.name, c.code, t.name, c.is_enabled, c.updated_at
		FROM currencies c LEFT JOIN currency_types t ON c.type_id = t.id
		ORDER BY c.updated_at DESC LIMIT 1;
	`

	currency := models.Currency{}
	if err := r.client.QueryRow(ctx, query).Scan(
		&currency.Name,
		&currency.Code,
		&currency.Type,
		&currency.IsEnabled,
		&currency.UpdatedAt,
	); err != nil {
		if db.CheckErrNoRows(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("error while fetching value: %w", err)
	}

	return &currency, nil
}
