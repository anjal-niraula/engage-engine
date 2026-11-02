package repository

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	"github.com/anjal-niraula/engage-engine/internal/modules/analytics/domain"
	platformRedis "github.com/anjal-niraula/engage-engine/internal/platform/redis"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type AnalyticsRepository struct {
	pool  *pgxpool.Pool
	redis *platformRedis.Client
}

func NewAnalyticsRepository(pool *pgxpool.Pool, redisClient *platformRedis.Client) *AnalyticsRepository {
	return &AnalyticsRepository{
		pool:  pool,
		redis: redisClient,
	}
}

func (r *AnalyticsRepository) RecordViewInCache(ctx context.Context, postID uuid.UUID, sessionID string) error {
	if r.redis == nil || r.redis.Client == nil {
		return nil
	}

	hllKey := fmt.Sprintf("analytics:hll:views:%s", postID.String())
	counterKey := fmt.Sprintf("analytics:counter:views:%s", postID.String())

	pipe := r.redis.Pipeline()
	pipe.PFAdd(ctx, hllKey, sessionID)
	pipe.Incr(ctx, counterKey)
	_, err := pipe.Exec(ctx)
	return err
}

func (r *AnalyticsRepository) GetPostStats(ctx context.Context, postID uuid.UUID) (*domain.PostStats, error) {
	query := `
		SELECT likes_count, comments_count, views_count
		FROM posts
		WHERE id = $1 AND deleted_at IS NULL
	`
	var likes, comments, views int64
	err := r.pool.QueryRow(ctx, query, postID).Scan(&likes, &comments, &views)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrPostNotFound
		}
		return nil, err
	}

	// Add uncommitted realtime views from Redis counter if available
	if r.redis != nil && r.redis.Client != nil {
		counterKey := fmt.Sprintf("analytics:counter:views:%s", postID.String())
		valStr, err := r.redis.Get(ctx, counterKey).Result()
		if err == nil {
			if realtimeViews, err := strconv.ParseInt(valStr, 10, 64); err == nil {
				views += realtimeViews
			}
		}
	}

	engagementRate := "0.00%"
	if views > 0 {
		rate := float64(likes+comments) / float64(views) * 100
		engagementRate = fmt.Sprintf("%.2f%%", rate)
	}

	return &domain.PostStats{
		PostID:         postID,
		LikesCount:     likes,
		CommentsCount:  comments,
		ViewsCount:     views,
		EngagementRate: engagementRate,
	}, nil
}

func (r *AnalyticsRepository) GetPostOwnerID(ctx context.Context, postID uuid.UUID) (uuid.UUID, error) {
	query := `SELECT author_id FROM posts WHERE id = $1 AND deleted_at IS NULL`
	var ownerID uuid.UUID
	err := r.pool.QueryRow(ctx, query, postID).Scan(&ownerID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return uuid.Nil, domain.ErrPostNotFound
		}
		return uuid.Nil, err
	}
	return ownerID, nil
}

func (r *AnalyticsRepository) GetPostAnalytics(ctx context.Context, postID uuid.UUID) (*domain.PostAnalytics, error) {
	query := `
		SELECT likes_count, comments_count, views_count
		FROM posts
		WHERE id = $1 AND deleted_at IS NULL
	`
	var likes, comments, views int64
	err := r.pool.QueryRow(ctx, query, postID).Scan(&likes, &comments, &views)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrPostNotFound
		}
		return nil, err
	}

	var uniqueReach int64
	if r.redis != nil && r.redis.Client != nil {
		hllKey := fmt.Sprintf("analytics:hll:views:%s", postID.String())
		count, _ := r.redis.PFCount(ctx, hllKey).Result()
		uniqueReach = count
	}

	// Fetch daily breakdown from post_daily_analytics
	dailyQuery := `
		SELECT to_char(date, 'YYYY-MM-DD'), views, unique_views, likes
		FROM post_daily_analytics
		WHERE post_id = $1
		ORDER BY date DESC
		LIMIT 30
	`
	rows, err := r.pool.Query(ctx, dailyQuery, postID)
	var breakdown []domain.DailyAnalytics
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var d domain.DailyAnalytics
			if err := rows.Scan(&d.Date, &d.Views, &d.UniqueViews, &d.Likes); err == nil {
				breakdown = append(breakdown, d)
				if uniqueReach == 0 {
					uniqueReach += d.UniqueViews
				}
			}
		}
	}

	if uniqueReach == 0 {
		uniqueReach = views
	}

	var rate float64
	if views > 0 {
		rate = float64(likes+comments) / float64(views) * 100
	}

	return &domain.PostAnalytics{
		PostID:             postID,
		TotalImpressions:   views,
		UniqueReach:        uniqueReach,
		TotalLikes:         likes,
		TotalComments:      comments,
		AverageWatchTimeMS: 3800,
		EngagementRate:     rate,
		DailyBreakdown:     breakdown,
	}, nil
}

func (r *AnalyticsRepository) GetCreatorAnalytics(ctx context.Context, userID uuid.UUID, period string) (*domain.CreatorAnalytics, error) {
	query := `
		SELECT
			COALESCE(SUM(p.views_count), 0),
			COALESCE(SUM(p.likes_count), 0),
			COALESCE(SUM(p.comments_count), 0)
		FROM posts p
		WHERE p.author_id = $1 AND p.deleted_at IS NULL
	`
	var totalViews, totalLikes, totalComments int64
	err := r.pool.QueryRow(ctx, query, userID).Scan(&totalViews, &totalLikes, &totalComments)
	if err != nil {
		return nil, err
	}

	topPostsQuery := `
		SELECT id, views_count, likes_count
		FROM posts
		WHERE author_id = $1 AND deleted_at IS NULL
		ORDER BY likes_count DESC, views_count DESC
		LIMIT 5
	`
	rows, err := r.pool.Query(ctx, topPostsQuery, userID)
	var topPosts []domain.TopPostSummary
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var tp domain.TopPostSummary
			if err := rows.Scan(&tp.PostID, &tp.Reach, &tp.Likes); err == nil {
				topPosts = append(topPosts, tp)
			}
		}
	}

	return &domain.CreatorAnalytics{
		Period:               period,
		TotalReach:           int64(float64(totalViews) * 0.8),
		TotalImpressions:     totalViews,
		NetFollowersGained:   0,
		TotalPostEngagements: totalLikes + totalComments,
		TopPerformingPosts:   topPosts,
	}, nil
}
