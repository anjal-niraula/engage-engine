package domain

import (
	"context"

	"github.com/google/uuid"
)

type ViewRepository interface {
	RecordViewInCache(ctx context.Context, postID uuid.UUID, sessionID string) error
	GetPostStats(ctx context.Context, postID uuid.UUID) (*PostStats, error)
	GetPostOwnerID(ctx context.Context, postID uuid.UUID) (uuid.UUID, error)
	GetPostAnalytics(ctx context.Context, postID uuid.UUID) (*PostAnalytics, error)
	GetCreatorAnalytics(ctx context.Context, userID uuid.UUID, period string) (*CreatorAnalytics, error)
}

type AnalyticsRepository = ViewRepository

type EventProducer interface {
	PublishViewEvent(ctx context.Context, event ViewEvent) error
}
