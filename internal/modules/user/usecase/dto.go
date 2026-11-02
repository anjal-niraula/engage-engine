package usecase

import "time"

// RegisterInput represents the input data required to register a user.
type RegisterInput struct {
	Username    string
	Email       string
	Password    string
	DisplayName string
}

// LoginInput represents credentials provided for authentication.
type LoginInput struct {
	Identity string // Username or Email
	Password string
}

// UpdateProfileInput represents fields that can be updated on a user profile.
type UpdateProfileInput struct {
	DisplayName *string
	Bio         *string
	AvatarURL   *string
	Website     *string
}

// UserOutput represents the public user data returned in responses.
type UserOutput struct {
	ID             string    `json:"id"`
	Username       string    `json:"username"`
	Email          string    `json:"email"`
	DisplayName    string    `json:"display_name"`
	Bio            *string   `json:"bio"`
	AvatarURL      *string   `json:"avatar_url"`
	Website        *string   `json:"website"`
	FollowersCount int64     `json:"followers_count"`
	FollowingCount int64     `json:"following_count"`
	PostsCount     int64     `json:"posts_count"`
	CreatedAt      time.Time `json:"created_at"`
}

// LoginOutput carries the authentication token and authenticated user summary.
type LoginOutput struct {
	AccessToken string      `json:"access_token"`
	TokenType   string      `json:"token_type"`
	ExpiresIn   int64       `json:"expires_in"`
	User        *UserOutput `json:"user"`
}

// PublicProfileOutput represents a user profile viewed by another user or guest.
type PublicProfileOutput struct {
	ID             string    `json:"id"`
	Username       string    `json:"username"`
	DisplayName    string    `json:"display_name"`
	Bio            *string   `json:"bio"`
	AvatarURL      *string   `json:"avatar_url"`
	Website        *string   `json:"website"`
	FollowersCount int64     `json:"followers_count"`
	FollowingCount int64     `json:"following_count"`
	PostsCount     int64     `json:"posts_count"`
	IsFollowing    bool      `json:"is_following"`
	CreatedAt      time.Time `json:"created_at"`
}

