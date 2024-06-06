package db

import (
	"blum-test/common/config"
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/jackc/pgx"
	"github.com/jackc/pgx/v4/pgxpool"
)

func NewPostgresClient(ctx context.Context, cfg *config.Postgres) (*pgxpool.Pool, error) {
	ctx, cancel := context.WithTimeout(ctx, time.Second*5)
	defer cancel()

	pool, err := pgxpool.Connect(ctx, cfg.DSN())
	if err != nil {
		return nil, err
	}

	return pool, nil
}

func CheckErrNoRows(err error) bool {
	return errors.Is(err, sql.ErrNoRows) || errors.Is(err, pgx.ErrNoRows)
}
