package domain

import "time"

// FollowUser represents a follower or followee in relationship listings.
type FollowUser struct {
	ID          string
	Username    string
	DisplayName string
	AvatarURL   *string
	FollowedAt  time.Time
}
