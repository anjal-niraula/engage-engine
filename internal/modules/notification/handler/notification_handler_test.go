package handler_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/anjal-niraula/engage-engine/internal/modules/notification/domain"
	"github.com/anjal-niraula/engage-engine/internal/modules/notification/handler"
	"github.com/anjal-niraula/engage-engine/internal/modules/notification/usecase"
	"github.com/anjal-niraula/engage-engine/internal/shared/middleware"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type mockNotificationUseCase struct {
	mock.Mock
}

func (m *mockNotificationUseCase) ListNotifications(ctx context.Context, userID uuid.UUID, limit int, cursorStr string) ([]*usecase.NotificationOutput, *string, bool, error) {
	args := m.Called(ctx, userID, limit, cursorStr)
	if args.Get(0) == nil {
		return nil, nil, args.Bool(2), args.Error(3)
	}
	return args.Get(0).([]*usecase.NotificationOutput), args.Get(1).(*string), args.Bool(2), args.Error(3)
}

func (m *mockNotificationUseCase) MarkAsRead(ctx context.Context, id uuid.UUID, userID uuid.UUID) error {
	args := m.Called(ctx, id, userID)
	return args.Error(0)
}

func (m *mockNotificationUseCase) CreateNotification(ctx context.Context, n *domain.Notification) error {
	args := m.Called(ctx, n)
	return args.Error(0)
}

func TestNotificationHandler_Endpoints(t *testing.T) {
	logger := zerolog.Nop()
	uc := new(mockNotificationUseCase)
	h := handler.NewNotificationHandler(uc, &logger)

	tokenSecret := "test-secret-at-least-32-chars-long!"
	userID := uuid.New()
	token, _ := middleware.GenerateToken(userID.String(), tokenSecret, time.Hour)

	mux := http.NewServeMux()
	handler.RegisterRoutes(mux, h, middleware.RequireAuth(tokenSecret), middleware.OptionalAuth(tokenSecret))

	notifID := uuid.New()

	t.Run("ListNotifications", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/notifications", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		rec := httptest.NewRecorder()

		uc.On("ListNotifications", mock.Anything, userID, 20, "").Return([]*usecase.NotificationOutput{}, (*string)(nil), false, nil).Once()

		mux.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusOK, rec.Code)
	})

	t.Run("MarkAsRead", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPatch, "/api/v1/notifications/"+notifID.String()+"/read", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		rec := httptest.NewRecorder()

		uc.On("MarkAsRead", mock.Anything, notifID, userID).Return(nil).Once()

		mux.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusOK, rec.Code)
	})
}
