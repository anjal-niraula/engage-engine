package usecase

import (
	"time"

	"github.com/anjal-niraula/engage-engine/internal/modules/comment/domain"
	"github.com/google/uuid"
)

type CreateCommentInput struct {
	PostID   uuid.UUID
	AuthorID uuid.UUID
	ParentID *uuid.UUID
	Content  string
}

type CommentOutput struct {
	ID           uuid.UUID            `json:"id"`
	PostID       uuid.UUID            `json:"post_id"`
	ParentID     *uuid.UUID           `json:"parent_id,omitempty"`
	Content      string               `json:"content"`
	LikesCount   int64                `json:"likes_count"`
	RepliesCount int64                `json:"replies_count"`
	CreatedAt    time.Time            `json:"created_at"`
	Author       domain.CommentAuthor `json:"author"`
}
