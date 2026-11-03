package usecase

import (
	"context"
	"time"

	"github.com/anjal-niraula/engage-engine/internal/modules/like/domain"
	"github.com/anjal-niraula/engage-engine/internal/shared/pagination"
)

// LikeUseCase coordinates business rules for liking and unliking content.
type LikeUseCase interface {
	LikePost(ctx context.Context, userID, postID string) (*LikeToggleOutput, error)
	UnlikePost(ctx context.Context, userID, postID string) (*LikeToggleOutput, error)
	ListPostLikes(ctx context.Context, postID string, limit int, cursorStr string) ([]*LikerUserOutput, *string, bool, error)

	LikeComment(ctx context.Context, userID, commentID string) (*LikeToggleOutput, error)
	UnlikeComment(ctx context.Context, userID, commentID string) (*LikeToggleOutput, error)
}

// LikeFilter defines the interface for recording/evicting likes in a probabilistic filter.
type LikeFilter interface {
	RecordLike(userID, postID string) error
	RecordUnlike(userID, postID string) bool
}

type likeUseCase struct {
	likeRepo domain.LikeRepository
	filter   LikeFilter
}

// NewLikeUseCase creates a new LikeUseCase instance with optional Cuckoo like filter.
func NewLikeUseCase(likeRepo domain.LikeRepository, filter ...LikeFilter) LikeUseCase {
	var f LikeFilter
	if len(filter) > 0 {
		f = filter[0]
	}
	return &likeUseCase{
		likeRepo: likeRepo,
		filter:   f,
	}
}

func (uc *likeUseCase) LikePost(ctx context.Context, userID, postID string) (*LikeToggleOutput, error) {
	isNewLike, count, err := uc.likeRepo.LikePost(ctx, userID, postID)
	if err != nil {
		return nil, err
	}
	if uc.filter != nil && isNewLike {
		_ = uc.filter.RecordLike(userID, postID)
	}
	return &LikeToggleOutput{
		TargetID:   postID,
		Liked:      true,
		LikesCount: count,
	}, nil
}

func (uc *likeUseCase) UnlikePost(ctx context.Context, userID, postID string) (*LikeToggleOutput, error) {
	wasDeleted, count, err := uc.likeRepo.UnlikePost(ctx, userID, postID)
	if err != nil {
		return nil, err
	}
	if uc.filter != nil && wasDeleted {
		_ = uc.filter.RecordUnlike(userID, postID)
	}
	return &LikeToggleOutput{
		TargetID:   postID,
		Liked:      false,
		LikesCount: count,
	}, nil
}

func (uc *likeUseCase) ListPostLikes(ctx context.Context, postID string, limit int, cursorStr string) ([]*LikerUserOutput, *string, bool, error) {
	var cursorID string
	var cursorTime *time.Time
	if cursorStr != "" {
		decoded, err := pagination.DecodeCursor(cursorStr)
		if err == nil && decoded != nil {
			cursorID = decoded.ID
			cursorTime = &decoded.CreatedAt
		}
	}

	likers, err := uc.likeRepo.ListPostLikes(ctx, postID, limit+1, cursorID, cursorTime)
	if err != nil {
		return nil, nil, false, err
	}

	hasMore := len(likers) > limit
	if hasMore {
		likers = likers[:limit]
	}

	var nextCursor *string
	if len(likers) > 0 && hasMore {
		last := likers[len(likers)-1]
		token := pagination.EncodeCursor(last.UserID, last.LikedAt)
		nextCursor = &token
	}

	output := make([]*LikerUserOutput, 0, len(likers))
	for _, l := range likers {
		output = append(output, &LikerUserOutput{
			UserID:      l.UserID,
			Username:    l.Username,
			DisplayName: l.DisplayName,
			AvatarURL:   l.AvatarURL,
			LikedAt:     l.LikedAt,
		})
	}

	return output, nextCursor, hasMore, nil
}

func (uc *likeUseCase) LikeComment(ctx context.Context, userID, commentID string) (*LikeToggleOutput, error) {
	_, count, err := uc.likeRepo.LikeComment(ctx, userID, commentID)
	if err != nil {
		return nil, err
	}
	return &LikeToggleOutput{
		TargetID:   commentID,
		Liked:      true,
		LikesCount: count,
	}, nil
}

func (uc *likeUseCase) UnlikeComment(ctx context.Context, userID, commentID string) (*LikeToggleOutput, error) {
	_, count, err := uc.likeRepo.UnlikeComment(ctx, userID, commentID)
	if err != nil {
		return nil, err
	}
	return &LikeToggleOutput{
		TargetID:   commentID,
		Liked:      false,
		LikesCount: count,
	}, nil
}
