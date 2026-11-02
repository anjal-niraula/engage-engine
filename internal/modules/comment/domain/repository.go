package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type CommentAuthor struct {
	ID          uuid.UUID `json:"id"`
	Username    string    `json:"username"`
	DisplayName string    `json:"display_name"`
	AvatarURL   *string   `json:"avatar_url"`
}

type CommentWithAuthor struct {
	Comment Comment       `json:"comment"`
	Author  CommentAuthor `json:"author"`
}

type CommentRepository interface {
	Create(ctx context.Context, comment *Comment) error
	GetByID(ctx context.Context, id uuid.UUID) (*Comment, error)
	GetPostOwnerID(ctx context.Context, postID uuid.UUID) (uuid.UUID, error)
	GetAuthor(ctx context.Context, userID uuid.UUID) (*CommentAuthor, error)
	ListByPost(ctx context.Context, postID uuid.UUID, cursorTime time.Time, cursorID uuid.UUID, limit int) ([]CommentWithAuthor, bool, error)
	ListReplies(ctx context.Context, parentID uuid.UUID, cursorTime time.Time, cursorID uuid.UUID, limit int) ([]CommentWithAuthor, bool, error)
	SoftDelete(ctx context.Context, comment *Comment) error
}
