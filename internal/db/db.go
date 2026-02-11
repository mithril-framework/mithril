package db

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

// New creates a new pgx connection pool from connString.
// Uses ParseConfig and NewWithConfig for optional tuning (e.g. MaxConns).
func New(ctx context.Context, connString string) (*pgxpool.Pool, error) {
	cfg, err := pgxpool.ParseConfig(connString)
	if err != nil {
		return nil, err
	}
	cfg.MaxConns = 10
	return pgxpool.NewWithConfig(ctx, cfg)
}
