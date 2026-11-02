package usecase

import (
	"context"
	"strings"
	"time"

	"github.com/anjal-niraula/engage-engine/internal/modules/user/domain"
	"github.com/anjal-niraula/engage-engine/internal/shared/middleware"
)

// UserUseCase defines application business logic operations for users.
type UserUseCase interface {
	Register(ctx context.Context, input RegisterInput) (*UserOutput, error)
	Login(ctx context.Context, input LoginInput) (*LoginOutput, error)
	GetProfile(ctx context.Context, userID string) (*UserOutput, error)
	UpdateProfile(ctx context.Context, userID string, input UpdateProfileInput) (*UserOutput, error)
	GetPublicProfile(ctx context.Context, username string, viewerID string) (*PublicProfileOutput, error)
}

type userUseCase struct {
	userRepo  domain.UserRepository
	jwtSecret string
	tokenTTL  time.Duration
}

// NewUserUseCase constructs an instance of UserUseCase with dependencies.
func NewUserUseCase(userRepo domain.UserRepository, jwtSecret string, tokenTTL time.Duration) UserUseCase {
	if tokenTTL <= 0 {
		tokenTTL = 24 * time.Hour
	}
	return &userUseCase{
		userRepo:  userRepo,
		jwtSecret: jwtSecret,
		tokenTTL:  tokenTTL,
	}
}

func (uc *userUseCase) Register(ctx context.Context, input RegisterInput) (*UserOutput, error) {
	cleanUsername := strings.ToLower(strings.TrimSpace(input.Username))
	cleanEmail := strings.ToLower(strings.TrimSpace(input.Email))

	// Check if username already exists
	if existing, err := uc.userRepo.GetByUsername(ctx, cleanUsername); err == nil && existing != nil {
		return nil, domain.ErrUsernameTaken
	}

	// Check if email already exists
	if existing, err := uc.userRepo.GetByEmail(ctx, cleanEmail); err == nil && existing != nil {
		return nil, domain.ErrEmailTaken
	}

	user, err := domain.NewUser(cleanUsername, cleanEmail, input.Password, strings.TrimSpace(input.DisplayName))
	if err != nil {
		return nil, err
	}

	if err := uc.userRepo.Create(ctx, user); err != nil {
		return nil, err
	}

	return toUserOutput(user), nil
}

func (uc *userUseCase) Login(ctx context.Context, input LoginInput) (*LoginOutput, error) {
	identity := strings.TrimSpace(input.Identity)
	user, err := uc.userRepo.GetByIdentity(ctx, identity)
	if err != nil || user == nil {
		return nil, domain.ErrInvalidCredentials
	}

	if !user.CheckPassword(input.Password) {
		return nil, domain.ErrInvalidCredentials
	}

	token, err := middleware.GenerateToken(user.ID, uc.jwtSecret, uc.tokenTTL)
	if err != nil {
		return nil, err
	}

	return &LoginOutput{
		AccessToken: token,
		TokenType:   "Bearer",
		ExpiresIn:   int64(uc.tokenTTL.Seconds()),
		User:        toUserOutput(user),
	}, nil
}

func (uc *userUseCase) GetProfile(ctx context.Context, userID string) (*UserOutput, error) {
	user, err := uc.userRepo.GetByID(ctx, userID)
	if err != nil || user == nil {
		return nil, domain.ErrUserNotFound
	}

	return toUserOutput(user), nil
}

func (uc *userUseCase) UpdateProfile(ctx context.Context, userID string, input UpdateProfileInput) (*UserOutput, error) {
	user, err := uc.userRepo.GetByID(ctx, userID)
	if err != nil || user == nil {
		return nil, domain.ErrUserNotFound
	}

	if input.DisplayName != nil {
		trimmed := strings.TrimSpace(*input.DisplayName)
		if trimmed != "" {
			user.DisplayName = trimmed
		}
	}
	if input.Bio != nil {
		user.Bio = input.Bio
	}
	if input.AvatarURL != nil {
		user.AvatarURL = input.AvatarURL
	}
	if input.Website != nil {
		user.Website = input.Website
	}

	user.UpdatedAt = time.Now().UTC()

	if err := uc.userRepo.Update(ctx, user); err != nil {
		return nil, err
	}

	return toUserOutput(user), nil
}

func (uc *userUseCase) GetPublicProfile(ctx context.Context, username string, viewerID string) (*PublicProfileOutput, error) {
	cleanUsername := strings.ToLower(strings.TrimSpace(username))
	user, err := uc.userRepo.GetByUsername(ctx, cleanUsername)
	if err != nil || user == nil {
		return nil, domain.ErrUserNotFound
	}

	isFollowing := false
	if viewerID != "" && viewerID != user.ID {
		isFollowing, _ = uc.userRepo.IsFollowing(ctx, viewerID, user.ID)
	}

	return &PublicProfileOutput{
		ID:             user.ID,
		Username:       user.Username,
		DisplayName:    user.DisplayName,
		Bio:            user.Bio,
		AvatarURL:      user.AvatarURL,
		Website:        user.Website,
		FollowersCount: user.FollowersCount,
		FollowingCount: user.FollowingCount,
		PostsCount:     user.PostsCount,
		IsFollowing:    isFollowing,
		CreatedAt:      user.CreatedAt,
	}, nil
}

func toUserOutput(u *domain.User) *UserOutput {
	return &UserOutput{
		ID:             u.ID,
		Username:       u.Username,
		Email:          u.Email,
		DisplayName:    u.DisplayName,
		Bio:            u.Bio,
		AvatarURL:      u.AvatarURL,
		Website:        u.Website,
		FollowersCount: u.FollowersCount,
		FollowingCount: u.FollowingCount,
		PostsCount:     u.PostsCount,
		CreatedAt:      u.CreatedAt,
	}
}
