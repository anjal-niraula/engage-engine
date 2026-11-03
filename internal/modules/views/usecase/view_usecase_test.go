package usecase_test

import (
	"context"
	"testing"

	"github.com/anjal-niraula/engage-engine/internal/modules/views/domain"
	"github.com/anjal-niraula/engage-engine/internal/modules/views/usecase"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type mockAnalyticsRepo struct {
	mock.Mock
}

func (m *mockAnalyticsRepo) RecordViewInCache(ctx context.Context, postID uuid.UUID, sessionID string) error {
	args := m.Called(ctx, postID, sessionID)
	return args.Error(0)
}

func (m *mockAnalyticsRepo) GetPostStats(ctx context.Context, postID uuid.UUID) (*domain.PostStats, error) {
	args := m.Called(ctx, postID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.PostStats), args.Error(1)
}

func (m *mockAnalyticsRepo) GetPostOwnerID(ctx context.Context, postID uuid.UUID) (uuid.UUID, error) {
	args := m.Called(ctx, postID)
	return args.Get(0).(uuid.UUID), args.Error(1)
}

func (m *mockAnalyticsRepo) GetPostAnalytics(ctx context.Context, postID uuid.UUID) (*domain.PostAnalytics, error) {
	args := m.Called(ctx, postID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.PostAnalytics), args.Error(1)
}

func (m *mockAnalyticsRepo) GetCreatorAnalytics(ctx context.Context, userID uuid.UUID, period string) (*domain.CreatorAnalytics, error) {
	args := m.Called(ctx, userID, period)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.CreatorAnalytics), args.Error(1)
}

type mockProducer struct {
	mock.Mock
}

func (m *mockProducer) PublishViewEvent(ctx context.Context, event domain.ViewEvent) error {
	args := m.Called(ctx, event)
	return args.Error(0)
}

func TestAnalyticsUseCase_RecordViewAndStats(t *testing.T) {
	repo := new(mockAnalyticsRepo)
	prod := new(mockProducer)
	uc := usecase.NewAnalyticsUseCase(repo, prod)

	postID := uuid.New()
	sessionID := "sess_12345"

	repo.On("RecordViewInCache", mock.Anything, postID, sessionID).Return(nil)
	prod.On("PublishViewEvent", mock.Anything, mock.Anything).Return(nil)

	err := uc.RecordView(context.Background(), usecase.RecordViewInput{
		PostID:     postID,
		SessionID:  sessionID,
		DurationMS: 5000,
	})
	assert.NoError(t, err)

	expectedStats := &domain.PostStats{
		PostID:         postID,
		LikesCount:     10,
		CommentsCount:  2,
		ViewsCount:     100,
		EngagementRate: "12.00%",
	}
	repo.On("GetPostStats", mock.Anything, postID).Return(expectedStats, nil)

	stats, err := uc.GetPostStats(context.Background(), postID)
	assert.NoError(t, err)
	assert.Equal(t, int64(100), stats.ViewsCount)
	assert.Equal(t, "12.00%", stats.EngagementRate)
}

func TestAnalyticsUseCase_GetPostAnalytics_Authorization(t *testing.T) {
	repo := new(mockAnalyticsRepo)
	uc := usecase.NewAnalyticsUseCase(repo, nil)

	postID := uuid.New()
	ownerID := uuid.New()
	otherUserID := uuid.New()

	repo.On("GetPostOwnerID", mock.Anything, postID).Return(ownerID, nil)

	// Non-owner gets forbidden
	_, err := uc.GetPostAnalytics(context.Background(), postID, otherUserID)
	assert.ErrorIs(t, err, domain.ErrForbidden)

	// Owner succeeds
	expectedAnalytics := &domain.PostAnalytics{
		PostID:           postID,
		TotalImpressions: 500,
	}
	repo.On("GetPostAnalytics", mock.Anything, postID).Return(expectedAnalytics, nil)

	res, err := uc.GetPostAnalytics(context.Background(), postID, ownerID)
	assert.NoError(t, err)
	assert.Equal(t, int64(500), res.TotalImpressions)
}
