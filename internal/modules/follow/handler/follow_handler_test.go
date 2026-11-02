package handler_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/anjal-niraula/engage-engine/internal/modules/follow/domain"
	"github.com/anjal-niraula/engage-engine/internal/modules/follow/handler"
	"github.com/anjal-niraula/engage-engine/internal/modules/follow/usecase"
	"github.com/anjal-niraula/engage-engine/internal/shared/middleware"
)

type mockFollowRepo struct{}

func (m *mockFollowRepo) Follow(ctx context.Context, followerID, followeeID string) (bool, error) {
	return true, nil
}

func (m *mockFollowRepo) Unfollow(ctx context.Context, followerID, followeeID string) (bool, error) {
	return true, nil
}

func (m *mockFollowRepo) ListFollowers(ctx context.Context, userID string, limit int, cursorID string, cursorTime *time.Time) ([]*domain.FollowUser, error) {
	return []*domain.FollowUser{}, nil
}

func (m *mockFollowRepo) ListFollowing(ctx context.Context, userID string, limit int, cursorID string, cursorTime *time.Time) ([]*domain.FollowUser, error) {
	return []*domain.FollowUser{}, nil
}

const testSecret = "jwt-secret-for-follow-tests-32b!"

func TestFollowHandler_Endpoints(t *testing.T) {
	repo := &mockFollowRepo{}
	uc := usecase.NewFollowUseCase(repo)
	h := handler.NewFollowHandler(uc)

	mux := http.NewServeMux()
	handler.RegisterRoutes(mux, h, middleware.RequireAuth(testSecret), middleware.OptionalAuth(testSecret))

	token, _ := middleware.GenerateToken("a3b8c2d1-0000-4000-8000-111122223333", testSecret, 1*time.Hour)
	targetID := "b4c9d3e2-0000-4000-8000-222233334444"

	// 1. Follow User
	req := httptest.NewRequest("POST", "/api/v1/users/"+targetID+"/follow", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK on follow user, got %d: %s", rec.Code, rec.Body.String())
	}

	// 2. Unfollow User
	reqDel := httptest.NewRequest("DELETE", "/api/v1/users/"+targetID+"/follow", nil)
	reqDel.Header.Set("Authorization", "Bearer "+token)
	recDel := httptest.NewRecorder()
	mux.ServeHTTP(recDel, reqDel)

	if recDel.Code != http.StatusOK {
		t.Fatalf("expected 200 OK on unfollow user, got %d", recDel.Code)
	}

	// 3. List Followers
	reqFollowers := httptest.NewRequest("GET", "/api/v1/users/"+targetID+"/followers", nil)
	recFollowers := httptest.NewRecorder()
	mux.ServeHTTP(recFollowers, reqFollowers)

	if recFollowers.Code != http.StatusOK {
		t.Fatalf("expected 200 OK on list followers, got %d", recFollowers.Code)
	}
}
