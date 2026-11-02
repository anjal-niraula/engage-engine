package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type NotificationRepository interface {
	Create(ctx context.Context, notification *Notification) error
	ListByUser(ctx context.Context, userID uuid.UUID, cursorID string, cursorTime *time.Time, limit int) ([]*Notification, error)
	MarkAsRead(ctx context.Context, id uuid.UUID, userID uuid.UUID) error
}
