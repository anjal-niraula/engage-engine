package domain

import (
	"context"
	"time"
)

// FollowRepository defines social graph operations between users.
type FollowRepository interface {
	Follow(ctx context.Context, followerID, followeeID string) (bool, error)
	Unfollow(ctx context.Context, followerID, followeeID string) (bool, error)
	ListFollowers(ctx context.Context, userID string, limit int, cursorID string, cursorTime *time.Time) ([]*FollowUser, error)
	ListFollowing(ctx context.Context, userID string, limit int, cursorID string, cursorTime *time.Time) ([]*FollowUser, error)
}
