package domain

import (
	"time"

	"github.com/google/uuid"
)

type Comment struct {
	ID           uuid.UUID  `json:"id"`
	PostID       uuid.UUID  `json:"post_id"`
	AuthorID     uuid.UUID  `json:"author_id"`
	ParentID     *uuid.UUID `json:"parent_id,omitempty"`
	Content      string     `json:"content"`
	LikesCount   int64      `json:"likes_count"`
	RepliesCount int64      `json:"replies_count"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
	DeletedAt    *time.Time `json:"deleted_at,omitempty"`
}
