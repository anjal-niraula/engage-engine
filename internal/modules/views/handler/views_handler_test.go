package handler_test

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/anjal-niraula/engage-engine/internal/modules/views/domain"
	"github.com/anjal-niraula/engage-engine/internal/modules/views/handler"
	"github.com/anjal-niraula/engage-engine/internal/modules/views/usecase"
	"github.com/anjal-niraula/engage-engine/internal/shared/middleware"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type mockAnalyticsUseCase struct {
	mock.Mock
}

func (m *mockAnalyticsUseCase) RecordView(ctx context.Context, in usecase.RecordViewInput) error {
	args := m.Called(ctx, in)
	return args.Error(0)
}

func (m *mockAnalyticsUseCase) GetPostStats(ctx context.Context, postID uuid.UUID) (*domain.PostStats, error) {
	args := m.Called(ctx, postID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.PostStats), args.Error(1)
}

func (m *mockAnalyticsUseCase) GetPostAnalytics(ctx context.Context, postID uuid.UUID, requestingUserID uuid.UUID) (*domain.PostAnalytics, error) {
	args := m.Called(ctx, postID, requestingUserID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.PostAnalytics), args.Error(1)
}

func (m *mockAnalyticsUseCase) GetUserAnalytics(ctx context.Context, userID uuid.UUID, period string) (*domain.CreatorAnalytics, error) {
	args := m.Called(ctx, userID, period)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.CreatorAnalytics), args.Error(1)
}

func TestAnalyticsHandler_Endpoints(t *testing.T) {
	logger := zerolog.Nop()
	uc := new(mockAnalyticsUseCase)
	h := handler.NewAnalyticsHandler(uc, &logger)

	tokenSecret := "test-secret-at-least-32-chars-long!"
	userID := uuid.New()
	token, _ := middleware.GenerateToken(userID.String(), tokenSecret, time.Hour)

	mux := http.NewServeMux()
	handler.RegisterRoutes(mux, h, middleware.RequireAuth(tokenSecret), middleware.OptionalAuth(tokenSecret))

	postID := uuid.New()

	t.Run("TrackView", func(t *testing.T) {
		reqBody := `{"post_id":"` + postID.String() + `","session_id":"sess_12345","duration_ms":3000}`
		req := httptest.NewRequest(http.MethodPost, "/api/v1/views", bytes.NewBufferString(reqBody))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		uc.On("RecordView", mock.Anything, mock.MatchedBy(func(in usecase.RecordViewInput) bool {
			return in.PostID == postID && in.SessionID == "sess_12345" && in.DurationMS == 3000
		})).Return(nil).Once()

		mux.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusAccepted, rec.Code)
	})

	t.Run("TrackView_AuthenticatedUserDeduplicationIdentity", func(t *testing.T) {
		reqBody := `{"post_id":"` + postID.String() + `","session_id":"random_client_string","duration_ms":3000}`
		req := httptest.NewRequest(http.MethodPost, "/api/v1/views", bytes.NewBufferString(reqBody))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)
		rec := httptest.NewRecorder()

		uc.On("RecordView", mock.Anything, mock.MatchedBy(func(in usecase.RecordViewInput) bool {
			return in.PostID == postID && in.SessionID == "user:"+userID.String()
		})).Return(nil).Once()

		mux.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusAccepted, rec.Code)
	})

	t.Run("GetPostStats", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/posts/"+postID.String()+"/stats", nil)
		rec := httptest.NewRecorder()

		uc.On("GetPostStats", mock.Anything, postID).Return(&domain.PostStats{
			PostID:         postID,
			LikesCount:     10,
			EngagementRate: "5.00%",
		}, nil).Once()

		mux.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusOK, rec.Code)
	})

	t.Run("GetPostAnalytics", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/posts/"+postID.String()+"/analytics", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		rec := httptest.NewRecorder()

		uc.On("GetPostAnalytics", mock.Anything, postID, userID).Return(&domain.PostAnalytics{
			PostID:           postID,
			TotalImpressions: 100,
		}, nil).Once()

		mux.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusOK, rec.Code)
	})

	t.Run("GetUserAnalytics", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/users/me/analytics?range=30d", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		rec := httptest.NewRecorder()

		uc.On("GetUserAnalytics", mock.Anything, userID, "30d").Return(&domain.CreatorAnalytics{
			Period:     "30d",
			TotalReach: 500,
		}, nil).Once()

		mux.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusOK, rec.Code)
	})
}
