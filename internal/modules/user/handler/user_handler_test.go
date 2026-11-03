package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/anjal-niraula/engage-engine/internal/modules/user/domain"
	"github.com/anjal-niraula/engage-engine/internal/modules/user/handler"
	"github.com/anjal-niraula/engage-engine/internal/modules/user/usecase"
	"github.com/anjal-niraula/engage-engine/internal/shared/middleware"
	"github.com/anjal-niraula/engage-engine/internal/shared/response"
)

type mockUserRepo struct {
	usersByUsername map[string]*domain.User
	usersByEmail    map[string]*domain.User
	usersByID       map[string]*domain.User
	followingMap    map[string]bool
}

func newMockUserRepo() *mockUserRepo {
	return &mockUserRepo{
		usersByUsername: make(map[string]*domain.User),
		usersByEmail:    make(map[string]*domain.User),
		usersByID:       make(map[string]*domain.User),
		followingMap:    make(map[string]bool),
	}
}

func (m *mockUserRepo) Create(ctx context.Context, u *domain.User) error {
	u.ID = "generated-uuid-" + u.Username
	m.usersByID[u.ID] = u
	m.usersByUsername[u.Username] = u
	m.usersByEmail[u.Email] = u
	return nil
}

func (m *mockUserRepo) GetByID(ctx context.Context, id string) (*domain.User, error) {
	if u, ok := m.usersByID[id]; ok {
		return u, nil
	}
	return nil, domain.ErrUserNotFound
}

func (m *mockUserRepo) GetByUsername(ctx context.Context, username string) (*domain.User, error) {
	if u, ok := m.usersByUsername[username]; ok {
		return u, nil
	}
	return nil, domain.ErrUserNotFound
}

func (m *mockUserRepo) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	if u, ok := m.usersByEmail[email]; ok {
		return u, nil
	}
	return nil, domain.ErrUserNotFound
}

func (m *mockUserRepo) GetByIdentity(ctx context.Context, identity string) (*domain.User, error) {
	if u, ok := m.usersByUsername[identity]; ok {
		return u, nil
	}
	if u, ok := m.usersByEmail[identity]; ok {
		return u, nil
	}
	return nil, domain.ErrUserNotFound
}

func (m *mockUserRepo) Update(ctx context.Context, u *domain.User) error {
	m.usersByID[u.ID] = u
	return nil
}

func (m *mockUserRepo) IsFollowing(ctx context.Context, followerID, followeeID string) (bool, error) {
	return m.followingMap[followerID+":"+followeeID], nil
}

const testSecret = "jwt-secret-key-32bytes-for-tests!"

func setupUserRouter() (*http.ServeMux, *mockUserRepo) {
	repo := newMockUserRepo()
	uc := usecase.NewUserUseCase(repo, testSecret, 1*time.Hour)
	h := handler.NewUserHandler(uc)

	mux := http.NewServeMux()
	handler.RegisterRoutes(mux, h, middleware.RequireAuth(testSecret), middleware.OptionalAuth(testSecret))
	return mux, repo
}

func TestUserHandler_Register(t *testing.T) {
	mux, _ := setupUserRouter()

	t.Run("successful registration", func(t *testing.T) {
		body := `{"username":"sarah","email":"sarah@example.com","password":"Password123","display_name":"Sarah Connor"}`
		req := httptest.NewRequest("POST", "/api/v1/users/register", bytes.NewBufferString(body))
		rec := httptest.NewRecorder()

		mux.ServeHTTP(rec, req)

		if rec.Code != http.StatusCreated {
			t.Fatalf("expected status 201, got %d, body: %s", rec.Code, rec.Body.String())
		}

		var resp response.Envelope
		if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}
		if !resp.Success {
			t.Errorf("expected success true, got false")
		}
	})

	t.Run("successful registration with underscore and dot in username", func(t *testing.T) {
		body := `{"username":"sophia_dev","email":"sophia@example.com","password":"Password123!","display_name":"Sophia Chen"}`
		req := httptest.NewRequest("POST", "/api/v1/users/register", bytes.NewBufferString(body))
		rec := httptest.NewRecorder()

		mux.ServeHTTP(rec, req)

		if rec.Code != http.StatusCreated {
			t.Fatalf("expected status 201, got %d, body: %s", rec.Code, rec.Body.String())
		}
	})

	t.Run("validation error on short password", func(t *testing.T) {
		body := `{"username":"sarah2","email":"sarah2@example.com","password":"123","display_name":"Sarah"}`
		req := httptest.NewRequest("POST", "/api/v1/users/register", bytes.NewBufferString(body))
		rec := httptest.NewRecorder()

		mux.ServeHTTP(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected status 400, got %d", rec.Code)
		}
	})
}

func TestUserHandler_LoginAndMe(t *testing.T) {
	mux, repo := setupUserRouter()

	// Pre-populate user
	user, _ := domain.NewUser("johndoe", "john@example.com", "Password123", "John Doe")
	_ = repo.Create(context.Background(), user)

	// Login
	loginBody := `{"identity":"johndoe","password":"Password123"}`
	req := httptest.NewRequest("POST", "/api/v1/users/login", bytes.NewBufferString(loginBody))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d, body: %s", rec.Code, rec.Body.String())
	}

	var loginResp struct {
		Success bool                `json:"success"`
		Data    usecase.LoginOutput `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &loginResp); err != nil {
		t.Fatalf("failed to parse login response: %v", err)
	}

	token := loginResp.Data.AccessToken
	if token == "" {
		t.Fatalf("expected non-empty access token")
	}

	// GET /api/v1/users/me with token
	reqMe := httptest.NewRequest("GET", "/api/v1/users/me", nil)
	reqMe.Header.Set("Authorization", "Bearer "+token)
	recMe := httptest.NewRecorder()
	mux.ServeHTTP(recMe, reqMe)

	if recMe.Code != http.StatusOK {
		t.Fatalf("expected status 200 on /me, got %d", recMe.Code)
	}

	// GET /api/v1/users/me without token -> 401
	reqAnon := httptest.NewRequest("GET", "/api/v1/users/me", nil)
	recAnon := httptest.NewRecorder()
	mux.ServeHTTP(recAnon, reqAnon)

	if recAnon.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401 on unauthenticated /me, got %d", recAnon.Code)
	}
}

func TestUserHandler_GetByUsername(t *testing.T) {
	mux, repo := setupUserRouter()

	user, _ := domain.NewUser("johndoe", "john@example.com", "Password123", "John Doe")
	_ = repo.Create(context.Background(), user)

	req := httptest.NewRequest("GET", "/api/v1/users/johndoe", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	// Non-existent user -> 404
	req404 := httptest.NewRequest("GET", "/api/v1/users/notfounduser", nil)
	rec404 := httptest.NewRecorder()
	mux.ServeHTTP(rec404, req404)

	if rec404.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d", rec404.Code)
	}
}
