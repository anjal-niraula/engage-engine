package usecase_test

import (
	"context"
	"testing"
	"time"

	"github.com/anjal-niraula/engage-engine/internal/modules/notification/domain"
	"github.com/anjal-niraula/engage-engine/internal/modules/notification/usecase"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type mockNotificationRepo struct {
	mock.Mock
}

func (m *mockNotificationRepo) Create(ctx context.Context, n *domain.Notification) error {
	args := m.Called(ctx, n)
	return args.Error(0)
}

func (m *mockNotificationRepo) ListByUser(ctx context.Context, userID uuid.UUID, cursorID string, cursorTime *time.Time, limit int) ([]*domain.Notification, error) {
	args := m.Called(ctx, userID, cursorID, cursorTime, limit)
	return args.Get(0).([]*domain.Notification), args.Error(1)
}

func (m *mockNotificationRepo) MarkAsRead(ctx context.Context, id uuid.UUID, userID uuid.UUID) error {
	args := m.Called(ctx, id, userID)
	return args.Error(0)
}

func TestNotificationUseCase_ListAndMarkRead(t *testing.T) {
	repo := new(mockNotificationRepo)
	uc := usecase.NewNotificationUseCase(repo)

	userID := uuid.New()
	notifID := uuid.New()
	now := time.Now().UTC()

	notifications := []*domain.Notification{
		{
			ID:         notifID,
			UserID:     userID,
			ActorID:    uuid.New(),
			Type:       domain.NotificationTypeLikePost,
			TargetID:   uuid.New().String(),
			TargetType: "post",
			IsRead:     false,
			CreatedAt:  now,
		},
	}

	repo.On("ListByUser", mock.Anything, userID, "", (*time.Time)(nil), 21).Return(notifications, nil)
	repo.On("MarkAsRead", mock.Anything, notifID, userID).Return(nil)

	outs, cursor, hasMore, err := uc.ListNotifications(context.Background(), userID, 20, "")
	assert.NoError(t, err)
	assert.Len(t, outs, 1)
	assert.Nil(t, cursor)
	assert.False(t, hasMore)

	err = uc.MarkAsRead(context.Background(), notifID, userID)
	assert.NoError(t, err)

	repo.AssertExpectations(t)
}
