package usecase

import (
	"context"
	"time"

	"github.com/anjal-niraula/engage-engine/internal/modules/notification/domain"
	"github.com/anjal-niraula/engage-engine/internal/shared/pagination"
	"github.com/google/uuid"
)

type NotificationUseCase interface {
	ListNotifications(ctx context.Context, userID uuid.UUID, limit int, cursorStr string) ([]*NotificationOutput, *string, bool, error)
	MarkAsRead(ctx context.Context, id uuid.UUID, userID uuid.UUID) error
	CreateNotification(ctx context.Context, n *domain.Notification) error
}

type notificationUseCase struct {
	repo domain.NotificationRepository
}

func NewNotificationUseCase(repo domain.NotificationRepository) NotificationUseCase {
	return &notificationUseCase{repo: repo}
}

func (uc *notificationUseCase) ListNotifications(ctx context.Context, userID uuid.UUID, limit int, cursorStr string) ([]*NotificationOutput, *string, bool, error) {
	if limit <= 0 || limit > 50 {
		limit = 20
	}

	var cursorID string
	var cursorTime *time.Time
	if cursorStr != "" {
		decoded, err := pagination.DecodeCursor(cursorStr)
		if err == nil && decoded != nil {
			cursorID = decoded.ID
			cursorTime = &decoded.CreatedAt
		}
	}

	notifications, err := uc.repo.ListByUser(ctx, userID, cursorID, cursorTime, limit+1)
	if err != nil {
		return nil, nil, false, err
	}

	hasMore := len(notifications) > limit
	if hasMore {
		notifications = notifications[:limit]
	}

	outputs := make([]*NotificationOutput, 0, len(notifications))
	for _, n := range notifications {
		outputs = append(outputs, &NotificationOutput{
			ID:         n.ID,
			Type:       n.Type,
			TargetID:   n.TargetID,
			TargetType: n.TargetType,
			IsRead:     n.IsRead,
			CreatedAt:  n.CreatedAt,
			Actor:      n.Actor,
		})
	}

	var nextCursor *string
	if hasMore && len(notifications) > 0 {
		last := notifications[len(notifications)-1]
		c := pagination.EncodeCursor(last.ID.String(), last.CreatedAt)
		nextCursor = &c
	}

	return outputs, nextCursor, hasMore, nil
}

func (uc *notificationUseCase) MarkAsRead(ctx context.Context, id uuid.UUID, userID uuid.UUID) error {
	return uc.repo.MarkAsRead(ctx, id, userID)
}

func (uc *notificationUseCase) CreateNotification(ctx context.Context, n *domain.Notification) error {
	if n.ID == uuid.Nil {
		n.ID = uuid.New()
	}
	if n.CreatedAt.IsZero() {
		n.CreatedAt = time.Now().UTC()
	}
	return uc.repo.Create(ctx, n)
}
