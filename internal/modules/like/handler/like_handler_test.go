package handler_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/anjal-niraula/engage-engine/internal/modules/like/domain"
	"github.com/anjal-niraula/engage-engine/internal/modules/like/handler"
	"github.com/anjal-niraula/engage-engine/internal/modules/like/usecase"
	"github.com/anjal-niraula/engage-engine/internal/shared/middleware"
)

type mockLikeRepo struct {
	count int64
}

func (m *mockLikeRepo) LikePost(ctx context.Context, userID, postID string) (bool, int64, error) {
	m.count++
	return true, m.count, nil
}

func (m *mockLikeRepo) UnlikePost(ctx context.Context, userID, postID string) (bool, int64, error) {
	if m.count > 0 {
		m.count--
	}
	return true, m.count, nil
}

func (m *mockLikeRepo) ListPostLikes(ctx context.Context, postID string, limit int, cursorID string, cursorTime *time.Time) ([]*domain.LikerUser, error) {
	return []*domain.LikerUser{}, nil
}

func (m *mockLikeRepo) LikeComment(ctx context.Context, userID, commentID string) (bool, int64, error) {
	return true, 1, nil
}

func (m *mockLikeRepo) UnlikeComment(ctx context.Context, userID, commentID string) (bool, int64, error) {
	return true, 0, nil
}

const testSecret = "jwt-secret-for-like-tests-32bytes!"

func TestLikeHandler_Endpoints(t *testing.T) {
	repo := &mockLikeRepo{}
	uc := usecase.NewLikeUseCase(repo)
	h := handler.NewLikeHandler(uc)

	mux := http.NewServeMux()
	handler.RegisterRoutes(mux, h, middleware.RequireAuth(testSecret), middleware.OptionalAuth(testSecret))

	token, _ := middleware.GenerateToken("a3b8c2d1-0000-4000-8000-111122223333", testSecret, 1*time.Hour)
	validPostID := "c5d0e4f3-0000-4000-8000-333344445555"

	// 1. Like Post
	req := httptest.NewRequest("POST", "/api/v1/posts/"+validPostID+"/likes", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK on like post, got %d: %s", rec.Code, rec.Body.String())
	}

	// 2. Unlike Post
	reqDel := httptest.NewRequest("DELETE", "/api/v1/posts/"+validPostID+"/likes", nil)
	reqDel.Header.Set("Authorization", "Bearer "+token)
	recDel := httptest.NewRecorder()
	mux.ServeHTTP(recDel, reqDel)

	if recDel.Code != http.StatusOK {
		t.Fatalf("expected 200 OK on unlike post, got %d", recDel.Code)
	}

	// 3. List Likes
	reqList := httptest.NewRequest("GET", "/api/v1/posts/"+validPostID+"/likes", nil)
	recList := httptest.NewRecorder()
	mux.ServeHTTP(recList, reqList)

	if recList.Code != http.StatusOK {
		t.Fatalf("expected 200 OK on list likes, got %d", recList.Code)
	}
}
