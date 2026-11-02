package domain

import (
	"context"
	"time"
)

// PostRepository defines persistence operations for posts.
type PostRepository interface {
	Create(ctx context.Context, p *Post) error
	GetByID(ctx context.Context, id string) (*Post, error)
	Delete(ctx context.Context, id string, authorID string) error
	ListByAuthorID(ctx context.Context, authorID string, limit int, cursorID string, cursorTime *time.Time) ([]*Post, error)
	ListFeed(ctx context.Context, userID string, limit int, cursorID string, cursorTime *time.Time) ([]*Post, error)
	GetAuthorByUsername(ctx context.Context, username string) (*Author, error)
	BatchGetAuthors(ctx context.Context, authorIDs []string) (map[string]*Author, error)
	BatchCheckLiked(ctx context.Context, userID string, postIDs []string) (map[string]bool, error)
}
