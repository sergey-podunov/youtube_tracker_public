package redisclient

import (
	"context"
	"errors"
	"log/slog"

	"github.com/redis/go-redis/v9"
)

// Config holds the Redis connection settings.
type Config struct {
	Addr     string
	Password string
	DB       int
}

// NewClient connects to Redis using cfg and verifies the connection with a
// ping. The caller owns the returned client and must Close it on shutdown.
func NewClient(ctx context.Context, logger *slog.Logger, cfg Config) (*redis.Client, error) {
	if cfg.Addr == "" {
		return nil, errors.New("redis address is required")
	}

	logger.Info("Redis connection config",
		slog.Group("config",
			"addr", cfg.Addr,
			"db", cfg.DB,
		))

	client := redis.NewClient(&redis.Options{
		Addr:     cfg.Addr,
		Password: cfg.Password,
		DB:       cfg.DB,
	})

	if err := client.Ping(ctx).Err(); err != nil {
		_ = client.Close()
		logger.Error("Unable to connect to Redis", "error", err)
		return nil, err
	}

	logger.Info("Connected to Redis")
	return client, nil
}
