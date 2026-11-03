package worker

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/anjal-niraula/engage-engine/internal/modules/views/domain"
	"github.com/anjal-niraula/engage-engine/internal/platform/server"
	"github.com/google/uuid"
	"github.com/newrelic/go-agent/v3/newrelic"
	"github.com/rs/zerolog"
	segmentio "github.com/segmentio/kafka-go"
)

// StartAllConsumers starts Kafka micro-batch consumers in background goroutines.
func StartAllConsumers(ctx context.Context, srv *server.Server, log *zerolog.Logger) {
	if srv.Kafka == nil {
		log.Warn().Msg("Kafka is not connected, consumers not started")
		return
	}

	go RunViewsMicroBatchConsumer(ctx, srv, log)
	go RunFollowsMicroBatchConsumer(ctx, srv, log)
	go RunLikesMicroBatchConsumer(ctx, srv, log)

	log.Info().Msg("🚀 High-throughput Kafka micro-batch consumers active (automatic write-behind sync to PostgreSQL)")
}

// RunViewsMicroBatchConsumer consumes engage.views and flushes batches to PostgreSQL.
func RunViewsMicroBatchConsumer(ctx context.Context, srv *server.Server, log *zerolog.Logger) {
	topic := "engage.views"
	groupID := "engage-views-aggregator"
	consumer := srv.Kafka.NewConsumer(topic, groupID)
	defer consumer.Close()

	const maxBatch = 500
	const maxWait = 200 * time.Millisecond

	msgs := make([]segmentio.Message, 0, maxBatch)
	ticker := time.NewTicker(maxWait)
	defer ticker.Stop()

	flush := func() {
		if len(msgs) == 0 {
			return
		}
		start := time.Now()

		viewDeltas := make(map[uuid.UUID]int64)
		durationSums := make(map[uuid.UUID]int64)

		for _, m := range msgs {
			var event domain.ViewEvent
			if err := json.Unmarshal(m.Value, &event); err == nil && event.PostID != uuid.Nil {
				viewDeltas[event.PostID]++
				durationSums[event.PostID] += int64(event.DurationMS)
			}
		}

		if srv.DB != nil && srv.DB.Pool != nil && len(viewDeltas) > 0 {
			flushCtx := ctx
			if srv.LoggerService != nil && srv.LoggerService.GetApplication() != nil {
				txn := srv.LoggerService.GetApplication().StartTransaction("KafkaWorker.ViewsFlush")
				defer txn.End()
				flushCtx = newrelic.NewContext(ctx, txn)
			}

			tx, err := srv.DB.Pool.Begin(flushCtx)
			if err == nil {
				for postID, delta := range viewDeltas {
					_, _ = tx.Exec(flushCtx, `
						UPDATE posts 
						SET views_count = views_count + $1 
						WHERE id = $2 AND deleted_at IS NULL
					`, delta, postID)

					_, _ = tx.Exec(flushCtx, `
						INSERT INTO post_daily_analytics (post_id, date, views)
						VALUES ($1, CURRENT_DATE, $2)
						ON CONFLICT (post_id, date)
						DO UPDATE SET 
							views = post_daily_analytics.views + $2
					`, postID, delta)
				}
				_ = tx.Commit(flushCtx)
			}
		}

		_ = consumer.CommitMessages(ctx, msgs...)

		log.Info().
			Int("batch_size", len(msgs)).
			Int("unique_posts", len(viewDeltas)).
			Str("latency", time.Since(start).String()).
			Msg("👁️  [Kafka Views Worker] Flushed micro-batch -> PostgreSQL")

		msgs = msgs[:0]
	}

	for {
		select {
		case <-ctx.Done():
			flush()
			return
		case <-ticker.C:
			flush()
		default:
			fetchCtx, cancel := context.WithTimeout(ctx, 100*time.Millisecond)
			msg, err := consumer.FetchMessage(fetchCtx)
			cancel()

			if err != nil {
				if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
					if len(msgs) > 0 {
						flush()
					}
					continue
				}
				time.Sleep(50 * time.Millisecond)
				continue
			}

			msgs = append(msgs, msg)
			if len(msgs) >= maxBatch {
				flush()
			}
		}
	}
}

// RunFollowsMicroBatchConsumer consumes engage.follows and flushes batches to PostgreSQL.
func RunFollowsMicroBatchConsumer(ctx context.Context, srv *server.Server, log *zerolog.Logger) {
	topic := "engage.follows"
	groupID := "engage-follows-aggregator"
	consumer := srv.Kafka.NewConsumer(topic, groupID)
	defer consumer.Close()

	const maxBatch = 500
	const maxWait = 200 * time.Millisecond

	msgs := make([]segmentio.Message, 0, maxBatch)
	ticker := time.NewTicker(maxWait)
	defer ticker.Stop()

	flush := func() {
		if len(msgs) == 0 {
			return
		}
		start := time.Now()

		followDeltas := make(map[string]int64)
		for _, m := range msgs {
			var event domain.FollowEvent
			if err := json.Unmarshal(m.Value, &event); err == nil && event.FolloweeID != "" {
				delta := int64(1)
				if event.Action == "unfollow" {
					delta = -1
				}
				followDeltas[event.FolloweeID] += delta
			}
		}

		if srv.DB != nil && srv.DB.Pool != nil && len(followDeltas) > 0 {
			for target, delta := range followDeltas {
				var newCount int64
				var username string
				err := srv.DB.Pool.QueryRow(ctx, `
					UPDATE users 
					SET followers_count = GREATEST(0, followers_count + $1) 
					WHERE (id::text = $2 OR username = $2)
					RETURNING followers_count, username
				`, delta, target).Scan(&newCount, &username)

				if err == nil {
					log.Info().
						Str("username", username).
						Int64("batch_delta", delta).
						Int64("new_followers_count", newCount).
						Str("latency", time.Since(start).String()).
						Msg("👥 [Kafka Follows Worker] Flushed micro-batch -> PostgreSQL")
				}
			}
		}

		_ = consumer.CommitMessages(ctx, msgs...)
		msgs = msgs[:0]
	}

	for {
		select {
		case <-ctx.Done():
			flush()
			return
		case <-ticker.C:
			flush()
		default:
			fetchCtx, cancel := context.WithTimeout(ctx, 100*time.Millisecond)
			msg, err := consumer.FetchMessage(fetchCtx)
			cancel()

			if err != nil {
				if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
					if len(msgs) > 0 {
						flush()
					}
					continue
				}
				time.Sleep(50 * time.Millisecond)
				continue
			}

			msgs = append(msgs, msg)
			if len(msgs) >= maxBatch {
				flush()
			}
		}
	}
}

// RunLikesMicroBatchConsumer consumes engage.likes and flushes batches to PostgreSQL.
func RunLikesMicroBatchConsumer(ctx context.Context, srv *server.Server, log *zerolog.Logger) {
	topic := "engage.likes"
	groupID := "engage-likes-aggregator"
	consumer := srv.Kafka.NewConsumer(topic, groupID)
	defer consumer.Close()

	const maxBatch = 500
	const maxWait = 200 * time.Millisecond

	msgs := make([]segmentio.Message, 0, maxBatch)
	ticker := time.NewTicker(maxWait)
	defer ticker.Stop()

	flush := func() {
		if len(msgs) == 0 {
			return
		}
		start := time.Now()

		likeDeltas := make(map[string]int64)
		for _, m := range msgs {
			var event domain.LikeEvent
			if err := json.Unmarshal(m.Value, &event); err == nil && event.PostID != "" {
				delta := int64(1)
				if event.Action == "unlike" {
					delta = -1
				}
				likeDeltas[event.PostID] += delta
			}
		}

		if srv.DB != nil && srv.DB.Pool != nil && len(likeDeltas) > 0 {
			for postID, delta := range likeDeltas {
				var newLikes int64
				err := srv.DB.Pool.QueryRow(ctx, `
					UPDATE posts 
					SET likes_count = GREATEST(0, likes_count + $1) 
					WHERE id::text = $2 AND deleted_at IS NULL
					RETURNING likes_count
				`, delta, postID).Scan(&newLikes)

				if err == nil {
					log.Info().
						Str("post_id", postID).
						Int64("batch_delta", delta).
						Int64("new_likes_count", newLikes).
						Str("latency", time.Since(start).String()).
						Msg("❤️  [Kafka Likes Worker] Flushed micro-batch -> PostgreSQL")
				}
			}
		}

		_ = consumer.CommitMessages(ctx, msgs...)
		msgs = msgs[:0]
	}

	for {
		select {
		case <-ctx.Done():
			flush()
			return
		case <-ticker.C:
			flush()
		default:
			fetchCtx, cancel := context.WithTimeout(ctx, 100*time.Millisecond)
			msg, err := consumer.FetchMessage(fetchCtx)
			cancel()

			if err != nil {
				if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
					if len(msgs) > 0 {
						flush()
					}
					continue
				}
				time.Sleep(50 * time.Millisecond)
				continue
			}

			msgs = append(msgs, msg)
			if len(msgs) >= maxBatch {
				flush()
			}
		}
	}
}

