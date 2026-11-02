package usecase

import (
	"time"

	"github.com/anjal-niraula/engage-engine/internal/modules/notification/domain"
	"github.com/google/uuid"
)

type NotificationOutput struct {
	ID         uuid.UUID                 `json:"id"`
	Type       domain.NotificationType   `json:"type"`
	TargetID   string                    `json:"target_id"`
	TargetType string                    `json:"target_type"`
	IsRead     bool                      `json:"is_read"`
	CreatedAt  time.Time                 `json:"created_at"`
	Actor      *domain.NotificationActor `json:"actor,omitempty"`
}
