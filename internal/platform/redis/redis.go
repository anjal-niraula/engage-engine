package redis

import (
	"context"
	"fmt"
	"time"

	"github.com/anjal-niraula/engage-engine/internal/platform/config"
	loggerPkg "github.com/anjal-niraula/engage-engine/internal/platform/logger"
	"github.com/newrelic/go-agent/v3/integrations/nrredis-v9"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"
)

// Client wraps the go-redis client with logging and lifecycle methods.
type Client struct {
	*redis.Client
	log *zerolog.Logger
}

// New creates a new Redis client with production timeouts, connection pooling, and APM hooks.
func New(cfg config.RedisConfig, logger *zerolog.Logger, loggerService *loggerPkg.LoggerService) (*Client, error) {
	opts := &redis.Options{
		Addr:         cfg.Address,
		DialTimeout:  5 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
		PoolSize:     25,
		MinIdleConns: 5,
		MaxRetries:   3,
	}

	rdb := redis.NewClient(opts)

	if loggerService != nil && loggerService.GetApplication() != nil {
		rdb.AddHook(nrredis.NewHook(opts))
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := rdb.Ping(ctx).Err(); err != nil {
		return &Client{Client: rdb, log: logger}, fmt.Errorf("failed to ping redis: %w", err)
	}

	logger.Info().Str("address", cfg.Address).Msg("connected to redis")
	return &Client{Client: rdb, log: logger}, nil
}

// Ping verifies Redis connection health.
func (c *Client) Ping(ctx context.Context) error {
	if c == nil || c.Client == nil {
		return fmt.Errorf("redis client is not initialized")
	}
	return c.Client.Ping(ctx).Err()
}

// Close gracefully closes the Redis client connection.
func (c *Client) Close() error {
	if c != nil && c.Client != nil {
		c.log.Info().Msg("closing redis client connection")
		return c.Client.Close()
	}
	return nil
}
