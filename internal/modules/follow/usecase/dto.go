package usecase

import "time"

// FollowToggleOutput represents the result of a follow or unfollow operation.
type FollowToggleOutput struct {
	UserID      string `json:"user_id"`
	IsFollowing bool   `json:"is_following"`
}

// FollowUserOutput represents a user profile in followers or following lists.
type FollowUserOutput struct {
	ID          string    `json:"id"`
	Username    string    `json:"username"`
	DisplayName string    `json:"display_name"`
	AvatarURL   *string   `json:"avatar_url"`
	FollowedAt  time.Time `json:"followed_at"`
}
