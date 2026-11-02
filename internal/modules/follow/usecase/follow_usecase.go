package usecase

import (
	"context"
	"time"

	"github.com/anjal-niraula/engage-engine/internal/modules/follow/domain"
	"github.com/anjal-niraula/engage-engine/internal/shared/pagination"
)

// FollowUseCase defines business rules for social graph relationships.
type FollowUseCase interface {
	Follow(ctx context.Context, followerID, followeeID string) (*FollowToggleOutput, error)
	Unfollow(ctx context.Context, followerID, followeeID string) (*FollowToggleOutput, error)
	ListFollowers(ctx context.Context, userID string, limit int, cursorStr string) ([]*FollowUserOutput, *string, bool, error)
	ListFollowing(ctx context.Context, userID string, limit int, cursorStr string) ([]*FollowUserOutput, *string, bool, error)
}

type followUseCase struct {
	followRepo domain.FollowRepository
}

// NewFollowUseCase constructs an instance of FollowUseCase.
func NewFollowUseCase(followRepo domain.FollowRepository) FollowUseCase {
	return &followUseCase{followRepo: followRepo}
}

func (uc *followUseCase) Follow(ctx context.Context, followerID, followeeID string) (*FollowToggleOutput, error) {
	if followerID == followeeID {
		return nil, domain.ErrCannotFollowSelf
	}

	_, err := uc.followRepo.Follow(ctx, followerID, followeeID)
	if err != nil {
		return nil, err
	}

	return &FollowToggleOutput{
		UserID:      followeeID,
		IsFollowing: true,
	}, nil
}

func (uc *followUseCase) Unfollow(ctx context.Context, followerID, followeeID string) (*FollowToggleOutput, error) {
	if followerID == followeeID {
		return nil, domain.ErrCannotFollowSelf
	}

	_, err := uc.followRepo.Unfollow(ctx, followerID, followeeID)
	if err != nil {
		return nil, err
	}

	return &FollowToggleOutput{
		UserID:      followeeID,
		IsFollowing: false,
	}, nil
}

func (uc *followUseCase) ListFollowers(ctx context.Context, userID string, limit int, cursorStr string) ([]*FollowUserOutput, *string, bool, error) {
	var cursorID string
	var cursorTime *time.Time
	if cursorStr != "" {
		decoded, err := pagination.DecodeCursor(cursorStr)
		if err == nil && decoded != nil {
			cursorID = decoded.ID
			cursorTime = &decoded.CreatedAt
		}
	}

	users, err := uc.followRepo.ListFollowers(ctx, userID, limit+1, cursorID, cursorTime)
	if err != nil {
		return nil, nil, false, err
	}

	hasMore := len(users) > limit
	if hasMore {
		users = users[:limit]
	}

	var nextCursor *string
	if len(users) > 0 && hasMore {
		last := users[len(users)-1]
		token := pagination.EncodeCursor(last.ID, last.FollowedAt)
		nextCursor = &token
	}

	output := make([]*FollowUserOutput, 0, len(users))
	for _, u := range users {
		output = append(output, &FollowUserOutput{
			ID:          u.ID,
			Username:    u.Username,
			DisplayName: u.DisplayName,
			AvatarURL:   u.AvatarURL,
			FollowedAt:  u.FollowedAt,
		})
	}

	return output, nextCursor, hasMore, nil
}

func (uc *followUseCase) ListFollowing(ctx context.Context, userID string, limit int, cursorStr string) ([]*FollowUserOutput, *string, bool, error) {
	var cursorID string
	var cursorTime *time.Time
	if cursorStr != "" {
		decoded, err := pagination.DecodeCursor(cursorStr)
		if err == nil && decoded != nil {
			cursorID = decoded.ID
			cursorTime = &decoded.CreatedAt
		}
	}

	users, err := uc.followRepo.ListFollowing(ctx, userID, limit+1, cursorID, cursorTime)
	if err != nil {
		return nil, nil, false, err
	}

	hasMore := len(users) > limit
	if hasMore {
		users = users[:limit]
	}

	var nextCursor *string
	if len(users) > 0 && hasMore {
		last := users[len(users)-1]
		token := pagination.EncodeCursor(last.ID, last.FollowedAt)
		nextCursor = &token
	}

	output := make([]*FollowUserOutput, 0, len(users))
	for _, u := range users {
		output = append(output, &FollowUserOutput{
			ID:          u.ID,
			Username:    u.Username,
			DisplayName: u.DisplayName,
			AvatarURL:   u.AvatarURL,
			FollowedAt:  u.FollowedAt,
		})
	}

	return output, nextCursor, hasMore, nil
}
