package domain

import (
	"context"
	"time"
)

// LikeRepository defines data operations for likes and counters.
type LikeRepository interface {
	LikePost(ctx context.Context, userID, postID string) (bool, int64, error)
	UnlikePost(ctx context.Context, userID, postID string) (bool, int64, error)
	ListPostLikes(ctx context.Context, postID string, limit int, cursorID string, cursorTime *time.Time) ([]*LikerUser, error)

	LikeComment(ctx context.Context, userID, commentID string) (bool, int64, error)
	UnlikeComment(ctx context.Context, userID, commentID string) (bool, int64, error)
}
