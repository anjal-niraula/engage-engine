package domain

import (
	"time"

	"github.com/google/uuid"
)

type NotificationType string

const (
	NotificationTypeLikePost    NotificationType = "like_post"
	NotificationTypeLikeComment NotificationType = "like_comment"
	NotificationTypeComment     NotificationType = "comment"
	NotificationTypeFollow      NotificationType = "follow"
)

type NotificationActor struct {
	ID          uuid.UUID `json:"id"`
	Username    string    `json:"username"`
	DisplayName string    `json:"display_name"`
	AvatarURL   *string   `json:"avatar_url"`
}

type Notification struct {
	ID         uuid.UUID          `json:"id"`
	UserID     uuid.UUID          `json:"user_id"`
	ActorID    uuid.UUID          `json:"actor_id"`
	Actor      *NotificationActor `json:"actor,omitempty"`
	Type       NotificationType   `json:"type"`
	TargetID   string             `json:"target_id"`
	TargetType string             `json:"target_type"`
	IsRead     bool               `json:"is_read"`
	CreatedAt  time.Time          `json:"created_at"`
}
