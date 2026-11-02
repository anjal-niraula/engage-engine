package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/anjal-niraula/engage-engine/internal/modules/post/domain"
	"github.com/anjal-niraula/engage-engine/internal/modules/post/handler"
	"github.com/anjal-niraula/engage-engine/internal/modules/post/usecase"
	"github.com/anjal-niraula/engage-engine/internal/shared/middleware"
)

type mockPostRepo struct {
	posts     map[string]*domain.Post
	authors   map[string]*domain.Author
	userLikes map[string]bool
}

func newMockPostRepo() *mockPostRepo {
	return &mockPostRepo{
		posts:     make(map[string]*domain.Post),
		authors:   make(map[string]*domain.Author),
		userLikes: make(map[string]bool),
	}
}

func (m *mockPostRepo) Create(ctx context.Context, p *domain.Post) error {
	p.ID = "c5d0e4f3-0000-4000-8000-333344445555"
	m.posts[p.ID] = p
	return nil
}

func (m *mockPostRepo) GetByID(ctx context.Context, id string) (*domain.Post, error) {
	p, ok := m.posts[id]
	if !ok || p.DeletedAt != nil {
		return nil, domain.ErrPostNotFound
	}
	return p, nil
}

func (m *mockPostRepo) Delete(ctx context.Context, id string, authorID string) error {
	p, ok := m.posts[id]
	if !ok || p.DeletedAt != nil {
		return domain.ErrPostNotFound
	}
	if p.AuthorID != authorID {
		return domain.ErrForbiddenPostAction
	}
	now := time.Now()
	p.DeletedAt = &now
	return nil
}

func (m *mockPostRepo) ListByAuthorID(ctx context.Context, authorID string, limit int, cursorID string, cursorTime *time.Time) ([]*domain.Post, error) {
	res := make([]*domain.Post, 0)
	for _, p := range m.posts {
		if p.AuthorID == authorID && p.DeletedAt == nil {
			res = append(res, p)
		}
	}
	return res, nil
}

func (m *mockPostRepo) ListFeed(ctx context.Context, userID string, limit int, cursorID string, cursorTime *time.Time) ([]*domain.Post, error) {
	res := make([]*domain.Post, 0)
	for _, p := range m.posts {
		if p.DeletedAt == nil {
			res = append(res, p)
		}
	}
	return res, nil
}

func (m *mockPostRepo) GetAuthorByUsername(ctx context.Context, username string) (*domain.Author, error) {
	for _, a := range m.authors {
		if a.Username == username {
			return a, nil
		}
	}
	return nil, domain.ErrPostNotFound
}

func (m *mockPostRepo) BatchGetAuthors(ctx context.Context, authorIDs []string) (map[string]*domain.Author, error) {
	res := make(map[string]*domain.Author)
	for _, id := range authorIDs {
		if a, ok := m.authors[id]; ok {
			res[id] = a
		}
	}
	return res, nil
}

func (m *mockPostRepo) BatchCheckLiked(ctx context.Context, userID string, postIDs []string) (map[string]bool, error) {
	return make(map[string]bool), nil
}

const testSecret = "jwt-secret-for-post-handler-test-32b"

func setupPostRouter() (*http.ServeMux, *mockPostRepo) {
	repo := newMockPostRepo()
	uc := usecase.NewPostUseCase(repo)
	h := handler.NewPostHandler(uc)

	mux := http.NewServeMux()
	handler.RegisterRoutes(mux, h, middleware.RequireAuth(testSecret), middleware.OptionalAuth(testSecret))
	return mux, repo
}

func TestPostHandler_CreateAndGet(t *testing.T) {
	mux, _ := setupPostRouter()
	token, _ := middleware.GenerateToken("a3b8c2d1-0000-4000-8000-111122223333", testSecret, 1*time.Hour)

	// Create Post
	body := `{"caption":"First post!","media_urls":["https://cdn.example.com/p1.jpg"],"media_type":"image","location":"Berlin"}`
	req := httptest.NewRequest("POST", "/api/v1/posts", bytes.NewBufferString(body))
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected status 201 Created, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp struct {
		Success bool               `json:"success"`
		Data    usecase.PostOutput `json:"data"`
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &resp)

	postID := resp.Data.ID

	// Get Post
	reqGet := httptest.NewRequest("GET", "/api/v1/posts/"+postID, nil)
	recGet := httptest.NewRecorder()
	mux.ServeHTTP(recGet, reqGet)

	if recGet.Code != http.StatusOK {
		t.Fatalf("expected status 200 OK on get post, got %d: %s", recGet.Code, recGet.Body.String())
	}

	// Delete Post
	reqDel := httptest.NewRequest("DELETE", "/api/v1/posts/"+postID, nil)
	reqDel.Header.Set("Authorization", "Bearer "+token)
	recDel := httptest.NewRecorder()
	mux.ServeHTTP(recDel, reqDel)

	if recDel.Code != http.StatusNoContent {
		t.Fatalf("expected 204 No Content on delete post, got %d", recDel.Code)
	}
}

func TestPostHandler_ValidationFailure(t *testing.T) {
	mux, _ := setupPostRouter()
	token, _ := middleware.GenerateToken("a3b8c2d1-0000-4000-8000-111122223333", testSecret, 1*time.Hour)

	// Invalid media_type
	body := `{"caption":"Test","media_urls":["https://cdn.example.com/img.jpg"],"media_type":"unsupported"}`
	req := httptest.NewRequest("POST", "/api/v1/posts", bytes.NewBufferString(body))
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400 Bad Request, got %d", rec.Code)
	}
}
