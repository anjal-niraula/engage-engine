package domain

import (
	"time"

	"github.com/google/uuid"
)

type ViewEvent struct {
	PostID     uuid.UUID `json:"post_id"`
	SessionID  string    `json:"session_id"`
	DurationMS int       `json:"duration_ms"`
	ViewedAt   time.Time `json:"viewed_at"`
}

type PostStats struct {
	PostID         uuid.UUID `json:"post_id"`
	LikesCount     int64     `json:"likes_count"`
	CommentsCount  int64     `json:"comments_count"`
	ViewsCount     int64     `json:"views_count"`
	EngagementRate string    `json:"engagement_rate"`
}

type DailyAnalytics struct {
	Date        string `json:"date"`
	Views       int64  `json:"views"`
	UniqueViews int64  `json:"unique_views"`
	Likes       int64  `json:"likes"`
}

type PostAnalytics struct {
	PostID             uuid.UUID        `json:"post_id"`
	TotalImpressions   int64            `json:"total_impressions"`
	UniqueReach        int64            `json:"unique_reach"`
	TotalLikes         int64            `json:"total_likes"`
	TotalComments      int64            `json:"total_comments"`
	AverageWatchTimeMS int64            `json:"average_watch_time_ms"`
	EngagementRate     float64          `json:"engagement_rate"`
	DailyBreakdown     []DailyAnalytics `json:"daily_breakdown"`
}

type TopPostSummary struct {
	PostID uuid.UUID `json:"post_id"`
	Reach  int64     `json:"reach"`
	Likes  int64     `json:"likes"`
}

type CreatorAnalytics struct {
	Period               string           `json:"period"`
	TotalReach           int64            `json:"total_reach"`
	TotalImpressions     int64            `json:"total_impressions"`
	NetFollowersGained   int64            `json:"net_followers_gained"`
	TotalPostEngagements int64            `json:"total_post_engagements"`
	TopPerformingPosts   []TopPostSummary `json:"top_performing_posts"`
}
