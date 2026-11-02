package usecase_test

import (
	"context"
	"testing"
	"time"

	"github.com/anjal-niraula/engage-engine/internal/modules/like/domain"
	"github.com/anjal-niraula/engage-engine/internal/modules/like/usecase"
)

type mockLikeRepo struct {
	postLikes map[string]int64
	userLikes map[string]bool
}

func newMockLikeRepo() *mockLikeRepo {
	return &mockLikeRepo{
		postLikes: make(map[string]int64),
		userLikes: make(map[string]bool),
	}
}

func (m *mockLikeRepo) LikePost(ctx context.Context, userID, postID string) (bool, int64, error) {
	key := userID + ":" + postID
	if m.userLikes[key] {
		return false, m.postLikes[postID], nil
	}
	m.userLikes[key] = true
	m.postLikes[postID]++
	return true, m.postLikes[postID], nil
}

func (m *mockLikeRepo) UnlikePost(ctx context.Context, userID, postID string) (bool, int64, error) {
	key := userID + ":" + postID
	if !m.userLikes[key] {
		return false, m.postLikes[postID], nil
	}
	delete(m.userLikes, key)
	if m.postLikes[postID] > 0 {
		m.postLikes[postID]--
	}
	return true, m.postLikes[postID], nil
}

func (m *mockLikeRepo) ListPostLikes(ctx context.Context, postID string, limit int, cursorID string, cursorTime *time.Time) ([]*domain.LikerUser, error) {
	return []*domain.LikerUser{
		{UserID: "u1", Username: "alice", DisplayName: "Alice", LikedAt: time.Now()},
	}, nil
}

func (m *mockLikeRepo) LikeComment(ctx context.Context, userID, commentID string) (bool, int64, error) {
	return true, 1, nil
}

func (m *mockLikeRepo) UnlikeComment(ctx context.Context, userID, commentID string) (bool, int64, error) {
	return true, 0, nil
}

func TestLikeUseCase_PostLikeUnlike(t *testing.T) {
	repo := newMockLikeRepo()
	uc := usecase.NewLikeUseCase(repo)
	ctx := context.Background()

	// Like Post
	out, err := uc.LikePost(ctx, "user-1", "post-100")
	if err != nil {
		t.Fatalf("unexpected error liking post: %v", err)
	}
	if !out.Liked || out.LikesCount != 1 {
		t.Errorf("expected liked=true and count=1, got %+v", out)
	}

	// Idempotent Like
	out2, err := uc.LikePost(ctx, "user-1", "post-100")
	if err != nil {
		t.Fatalf("unexpected error re-liking: %v", err)
	}
	if out2.LikesCount != 1 {
		t.Errorf("expected count to stay 1, got %d", out2.LikesCount)
	}

	// Unlike Post
	out3, err := uc.UnlikePost(ctx, "user-1", "post-100")
	if err != nil {
		t.Fatalf("unexpected error unliking: %v", err)
	}
	if out3.Liked || out3.LikesCount != 0 {
		t.Errorf("expected liked=false and count=0, got %+v", out3)
	}
}
