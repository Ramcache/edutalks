package db

import (
	"context"
	"edutalks/internal/config"

	"github.com/jackc/pgx/v5/pgxpool"
)

func NewPostgresConnection(cfg *config.Config) (*pgxpool.Pool, error) {
	dsn := cfg.GetDSN()
	pool, err := pgxpool.New(context.Background(), dsn)

	if err != nil {
		return nil, err
	}

	err = pool.Ping(context.Background())
	if err != nil {
		return nil, err
	}

	return pool, nil
}
