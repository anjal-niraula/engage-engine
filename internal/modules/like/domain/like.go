package domain

import (
	"errors"
	"time"
)

var (
	// ErrTargetNotFound indicates the target post or comment does not exist.
	ErrTargetNotFound = errors.New("target resource not found")
)

// LikerUser represents a user who liked a post or comment.
type LikerUser struct {
	UserID      string
	Username    string
	DisplayName string
	AvatarURL   *string
	LikedAt     time.Time
}
