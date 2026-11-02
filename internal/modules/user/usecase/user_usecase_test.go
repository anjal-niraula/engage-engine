package usecase_test

import (
	"context"
	"testing"
	"time"

	"github.com/anjal-niraula/engage-engine/internal/modules/user/domain"
	"github.com/anjal-niraula/engage-engine/internal/modules/user/usecase"
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

const testJWTSecret = "test-secret-key-32bytes-for-jwt-signing"

func TestUserUseCase_RegisterAndLogin(t *testing.T) {
	repo := newMockUserRepo()
	uc := usecase.NewUserUseCase(repo, testJWTSecret, 1*time.Hour)

	ctx := context.Background()

	// 1. Register User
	userOut, err := uc.Register(ctx, usecase.RegisterInput{
		Username:    "johndoe",
		Email:       "john@example.com",
		Password:    "password123",
		DisplayName: "John Doe",
	})
	if err != nil {
		t.Fatalf("unexpected registration error: %v", err)
	}

	if userOut.Username != "johndoe" || userOut.Email != "john@example.com" {
		t.Errorf("unexpected user output: %+v", userOut)
	}

	// 2. Register Duplicate Username
	_, err = uc.Register(ctx, usecase.RegisterInput{
		Username:    "johndoe",
		Email:       "other@example.com",
		Password:    "password123",
		DisplayName: "Other Doe",
	})
	if err != domain.ErrUsernameTaken {
		t.Errorf("expected ErrUsernameTaken, got %v", err)
	}

	// 3. Register Duplicate Email
	_, err = uc.Register(ctx, usecase.RegisterInput{
		Username:    "janedoe",
		Email:       "john@example.com",
		Password:    "password123",
		DisplayName: "Jane Doe",
	})
	if err != domain.ErrEmailTaken {
		t.Errorf("expected ErrEmailTaken, got %v", err)
	}

	// 4. Successful Login
	loginOut, err := uc.Login(ctx, usecase.LoginInput{
		Identity: "johndoe",
		Password: "password123",
	})
	if err != nil {
		t.Fatalf("unexpected login error: %v", err)
	}
	if loginOut.AccessToken == "" || loginOut.User.ID != userOut.ID {
		t.Errorf("unexpected login output: %+v", loginOut)
	}

	// 5. Invalid Password
	_, err = uc.Login(ctx, usecase.LoginInput{
		Identity: "johndoe",
		Password: "wrongpassword",
	})
	if err != domain.ErrInvalidCredentials {
		t.Errorf("expected ErrInvalidCredentials, got %v", err)
	}
}

func TestUserUseCase_GetAndUpdateProfile(t *testing.T) {
	repo := newMockUserRepo()
	uc := usecase.NewUserUseCase(repo, testJWTSecret, 1*time.Hour)
	ctx := context.Background()

	userOut, _ := uc.Register(ctx, usecase.RegisterInput{
		Username:    "alice",
		Email:       "alice@example.com",
		Password:    "password123",
		DisplayName: "Alice Wonder",
	})

	// Get Profile
	profile, err := uc.GetProfile(ctx, userOut.ID)
	if err != nil {
		t.Fatalf("unexpected error getting profile: %v", err)
	}
	if profile.DisplayName != "Alice Wonder" {
		t.Errorf("expected 'Alice Wonder', got %s", profile.DisplayName)
	}

	// Update Profile
	newBio := "Hello world"
	updated, err := uc.UpdateProfile(ctx, userOut.ID, usecase.UpdateProfileInput{
		Bio: &newBio,
	})
	if err != nil {
		t.Fatalf("unexpected error updating profile: %v", err)
	}
	if updated.Bio == nil || *updated.Bio != newBio {
		t.Errorf("expected updated bio, got %v", updated.Bio)
	}
}

func TestUserUseCase_GetPublicProfile(t *testing.T) {
	repo := newMockUserRepo()
	uc := usecase.NewUserUseCase(repo, testJWTSecret, 1*time.Hour)
	ctx := context.Background()

	alice, _ := uc.Register(ctx, usecase.RegisterInput{
		Username:    "alice",
		Email:       "alice@example.com",
		Password:    "password123",
		DisplayName: "Alice Wonder",
	})

	bob, _ := uc.Register(ctx, usecase.RegisterInput{
		Username:    "bob",
		Email:       "bob@example.com",
		Password:    "password123",
		DisplayName: "Bob Builder",
	})

	repo.followingMap[bob.ID+":"+alice.ID] = true

	// Bob views Alice (isFollowing should be true)
	publicProf, err := uc.GetPublicProfile(ctx, "alice", bob.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !publicProf.IsFollowing {
		t.Errorf("expected is_following to be true")
	}

	// Anonymous views Alice (isFollowing should be false)
	anonProf, err := uc.GetPublicProfile(ctx, "alice", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if anonProf.IsFollowing {
		t.Errorf("expected is_following to be false for anonymous")
	}
}

