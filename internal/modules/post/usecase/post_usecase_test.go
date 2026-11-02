package usecase_test

import (
	"context"
	"testing"
	"time"

	"github.com/anjal-niraula/engage-engine/internal/modules/post/domain"
	"github.com/anjal-niraula/engage-engine/internal/modules/post/usecase"
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
	p.ID = "post-uuid-123"
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
	result := make([]*domain.Post, 0)
	for _, p := range m.posts {
		if p.AuthorID == authorID && p.DeletedAt == nil {
			result = append(result, p)
		}
	}
	return result, nil
}

func (m *mockPostRepo) ListFeed(ctx context.Context, userID string, limit int, cursorID string, cursorTime *time.Time) ([]*domain.Post, error) {
	result := make([]*domain.Post, 0)
	for _, p := range m.posts {
		if p.DeletedAt == nil {
			result = append(result, p)
		}
	}
	return result, nil
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
	res := make(map[string]bool)
	for _, pid := range postIDs {
		res[pid] = m.userLikes[userID+":"+pid]
	}
	return res, nil
}

func TestPostUseCase_CreateGetDelete(t *testing.T) {
	repo := newMockPostRepo()
	repo.authors["user-1"] = &domain.Author{
		ID:          "user-1",
		Username:    "johndoe",
		DisplayName: "John Doe",
	}

	uc := usecase.NewPostUseCase(repo)
	ctx := context.Background()

	caption := "Sunset in Santorini"
	out, err := uc.CreatePost(ctx, "user-1", usecase.CreatePostInput{
		Caption:   &caption,
		MediaURLs: []string{"https://cdn.example.com/img1.jpg"},
		MediaType: "image",
	})
	if err != nil {
		t.Fatalf("unexpected error creating post: %v", err)
	}

	if out.ID == "" || *out.Caption != caption {
		t.Errorf("unexpected output: %+v", out)
	}

	// Get Post
	repo.userLikes["user-1:"+out.ID] = true
	getOut, err := uc.GetPost(ctx, out.ID, "user-1")
	if err != nil {
		t.Fatalf("unexpected error getting post: %v", err)
	}
	if !getOut.HasLiked {
		t.Errorf("expected has_liked to be true")
	}

	// Delete Unauthorized
	err = uc.DeletePost(ctx, out.ID, "intruder-user")
	if err != domain.ErrForbiddenPostAction {
		t.Errorf("expected ErrForbiddenPostAction, got %v", err)
	}

	// Delete Success
	err = uc.DeletePost(ctx, out.ID, "user-1")
	if err != nil {
		t.Fatalf("unexpected error deleting post: %v", err)
	}

	// Verify post is gone
	_, err = uc.GetPost(ctx, out.ID, "user-1")
	if err != domain.ErrPostNotFound {
		t.Errorf("expected ErrPostNotFound, got %v", err)
	}
}
