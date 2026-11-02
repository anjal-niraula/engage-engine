package usecase

import "time"

// LikeToggleOutput represents the result of a like or unlike action.
type LikeToggleOutput struct {
	TargetID   string `json:"target_id"`
	Liked      bool   `json:"liked"`
	LikesCount int64  `json:"likes_count"`
}

// LikerUserOutput represents public user info in the likers list.
type LikerUserOutput struct {
	UserID      string    `json:"id"`
	Username    string    `json:"username"`
	DisplayName string    `json:"display_name"`
	AvatarURL   *string   `json:"avatar_url"`
	LikedAt     time.Time `json:"liked_at"`
}
