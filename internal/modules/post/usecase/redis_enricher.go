package usecase

import (
	"context"
	"fmt"
	"strconv"

	platformRedis "github.com/anjal-niraula/engage-engine/internal/platform/redis"
	"github.com/redis/go-redis/v9"
)

// RedisStatsEnricher enriches posts with realtime view counters and unique reach (HyperLogLog) from Redis.
type RedisStatsEnricher struct {
	redis *platformRedis.Client
}

// NewRedisStatsEnricher constructs an enricher backed by Redis.
func NewRedisStatsEnricher(redis *platformRedis.Client) *RedisStatsEnricher {
	return &RedisStatsEnricher{redis: redis}
}

// EnrichPosts pipelines queries to Redis to enrich post models with realtime views and unique reach.
func (e *RedisStatsEnricher) EnrichPosts(ctx context.Context, posts ...*PostOutput) {
	if e == nil || e.redis == nil || e.redis.Client == nil || len(posts) == 0 {
		return
	}

	pipe := e.redis.Pipeline()

	type cmdPair struct {
		post      *PostOutput
		viewCmd   *redis.StringCmd
		uniqueCmd *redis.IntCmd
	}

	cmds := make([]cmdPair, 0, len(posts))
	for _, p := range posts {
		if p == nil || p.ID == "" {
			continue
		}
		counterKey := fmt.Sprintf("analytics:counter:views:%s", p.ID)
		hllKey := fmt.Sprintf("analytics:hll:views:%s", p.ID)

		cmds = append(cmds, cmdPair{
			post:      p,
			viewCmd:   pipe.Get(ctx, counterKey),
			uniqueCmd: pipe.PFCount(ctx, hllKey),
		})
	}

	if len(cmds) == 0 {
		return
	}

	// Single pipelined round-trip for ALL posts in the batch
	_, _ = pipe.Exec(ctx)

	for _, c := range cmds {
		var redisViews int64
		if val, err := c.viewCmd.Result(); err == nil {
			if parsed, err := strconv.ParseInt(val, 10, 64); err == nil {
				redisViews = parsed
			}
		}

		var uniqueViews int64
		if count, err := c.uniqueCmd.Result(); err == nil && count > 0 {
			uniqueViews = count
		}

		if redisViews > c.post.ViewsCount {
			c.post.ViewsCount = redisViews
		}

		if uniqueViews == 0 && c.post.ViewsCount > 0 {
			uniqueViews = 1
		}
		if uniqueViews > c.post.ViewsCount {
			c.post.ViewsCount = uniqueViews
		}
		c.post.UniqueViews = uniqueViews
	}
}

