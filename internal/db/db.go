package db

import (
	"context"
	"edutalks/internal/config"
	"edutalks/internal/logger"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

func NewPostgresConnection(cfg *config.Config) (*pgxpool.Pool, error) {
	dsn := cfg.GetDSN()

	logger.Log.Info("Подключение к Postgres...", zap.String("dsn", cfg.GetDSNSafe()))

	pool, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		logger.Log.Error("Не удалось создать пул подключений к Postgres",
			zap.String("dsn", cfg.GetDSNSafe()), zap.Error(err))
		return nil, err
	}

	if err := pool.Ping(context.Background()); err != nil {
		logger.Log.Error("Не удалось подключиться к Postgres (ping failed)",
			zap.String("dsn", cfg.GetDSNSafe()), zap.Error(err))
		return nil, err
	}

	logger.Log.Info("Соединение с Postgres успешно установлено", zap.String("dsn", cfg.GetDSNSafe()))
	return pool, nil
}
