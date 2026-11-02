package usecase_test

import (
	"context"
	"testing"
	"time"

	"github.com/anjal-niraula/engage-engine/internal/modules/follow/domain"
	"github.com/anjal-niraula/engage-engine/internal/modules/follow/usecase"
)

type mockFollowRepo struct {
	follows map[string]bool
}

func newMockFollowRepo() *mockFollowRepo {
	return &mockFollowRepo{follows: make(map[string]bool)}
}

func (m *mockFollowRepo) Follow(ctx context.Context, followerID, followeeID string) (bool, error) {
	key := followerID + ":" + followeeID
	if m.follows[key] {
		return false, nil
	}
	m.follows[key] = true
	return true, nil
}

func (m *mockFollowRepo) Unfollow(ctx context.Context, followerID, followeeID string) (bool, error) {
	key := followerID + ":" + followeeID
	if !m.follows[key] {
		return false, nil
	}
	delete(m.follows, key)
	return true, nil
}

func (m *mockFollowRepo) ListFollowers(ctx context.Context, userID string, limit int, cursorID string, cursorTime *time.Time) ([]*domain.FollowUser, error) {
	return []*domain.FollowUser{
		{ID: "f1", Username: "follower1", DisplayName: "Follower 1", FollowedAt: time.Now()},
	}, nil
}

func (m *mockFollowRepo) ListFollowing(ctx context.Context, userID string, limit int, cursorID string, cursorTime *time.Time) ([]*domain.FollowUser, error) {
	return []*domain.FollowUser{}, nil
}

func TestFollowUseCase_SelfFollow(t *testing.T) {
	repo := newMockFollowRepo()
	uc := usecase.NewFollowUseCase(repo)

	_, err := uc.Follow(context.Background(), "user-1", "user-1")
	if err != domain.ErrCannotFollowSelf {
		t.Errorf("expected ErrCannotFollowSelf, got %v", err)
	}
}

func TestFollowUseCase_FollowAndUnfollow(t *testing.T) {
	repo := newMockFollowRepo()
	uc := usecase.NewFollowUseCase(repo)
	ctx := context.Background()

	out, err := uc.Follow(ctx, "user-1", "user-2")
	if err != nil {
		t.Fatalf("unexpected error following user: %v", err)
	}
	if !out.IsFollowing {
		t.Errorf("expected is_following true")
	}

	outUnfollow, err := uc.Unfollow(ctx, "user-1", "user-2")
	if err != nil {
		t.Fatalf("unexpected error unfollowing user: %v", err)
	}
	if outUnfollow.IsFollowing {
		t.Errorf("expected is_following false")
	}
}
