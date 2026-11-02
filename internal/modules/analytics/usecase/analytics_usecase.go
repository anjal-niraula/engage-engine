package usecase

import (
	"context"
	"time"

	"github.com/anjal-niraula/engage-engine/internal/modules/analytics/domain"
	"github.com/google/uuid"
)

type AnalyticsUseCase interface {
	RecordView(ctx context.Context, in RecordViewInput) error
	GetPostStats(ctx context.Context, postID uuid.UUID) (*domain.PostStats, error)
	GetPostAnalytics(ctx context.Context, postID uuid.UUID, requestingUserID uuid.UUID) (*domain.PostAnalytics, error)
	GetUserAnalytics(ctx context.Context, userID uuid.UUID, period string) (*domain.CreatorAnalytics, error)
}

type analyticsUseCase struct {
	repo     domain.AnalyticsRepository
	producer domain.EventProducer
}

func NewAnalyticsUseCase(repo domain.AnalyticsRepository, producer domain.EventProducer) AnalyticsUseCase {
	return &analyticsUseCase{
		repo:     repo,
		producer: producer,
	}
}

func (uc *analyticsUseCase) RecordView(ctx context.Context, in RecordViewInput) error {
	// 1. Fast cache update (Redis HLL and raw counter)
	if err := uc.repo.RecordViewInCache(ctx, in.PostID, in.SessionID); err != nil {
		return err
	}

	// 2. Publish view event asynchronously to Kafka if producer is configured
	if uc.producer != nil {
		event := domain.ViewEvent{
			PostID:     in.PostID,
			SessionID:  in.SessionID,
			DurationMS: in.DurationMS,
			ViewedAt:   time.Now().UTC(),
		}
		// Kafka publish is decoupled from HTTP latency
		go func() {
			bgCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			_ = uc.producer.PublishViewEvent(bgCtx, event)
		}()
	}

	return nil
}

func (uc *analyticsUseCase) GetPostStats(ctx context.Context, postID uuid.UUID) (*domain.PostStats, error) {
	return uc.repo.GetPostStats(ctx, postID)
}

func (uc *analyticsUseCase) GetPostAnalytics(ctx context.Context, postID uuid.UUID, requestingUserID uuid.UUID) (*domain.PostAnalytics, error) {
	ownerID, err := uc.repo.GetPostOwnerID(ctx, postID)
	if err != nil {
		return nil, err
	}
	if ownerID != requestingUserID {
		return nil, domain.ErrForbidden
	}

	return uc.repo.GetPostAnalytics(ctx, postID)
}

func (uc *analyticsUseCase) GetUserAnalytics(ctx context.Context, userID uuid.UUID, period string) (*domain.CreatorAnalytics, error) {
	if period != "7d" && period != "30d" && period != "90d" {
		period = "30d"
	}
	return uc.repo.GetCreatorAnalytics(ctx, userID, period)
}
