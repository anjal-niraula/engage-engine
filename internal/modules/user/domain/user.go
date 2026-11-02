package domain

import (
	"time"

	"golang.org/x/crypto/bcrypt"
)

// User represents the core user account entity in the system.
type User struct {
	ID             string
	Username       string
	Email          string
	PasswordHash   string
	DisplayName    string
	Bio            *string
	AvatarURL      *string
	Website        *string
	FollowersCount int64
	FollowingCount int64
	PostsCount     int64
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// NewUser creates a new User domain model with a hashed password.
func NewUser(username, email, plainPassword, displayName string) (*User, error) {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(plainPassword), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	return &User{
		Username:     username,
		Email:        email,
		PasswordHash: string(hashedPassword),
		DisplayName:  displayName,
		CreatedAt:    now,
		UpdatedAt:    now,
	}, nil
}

// CheckPassword verifies whether the supplied password matches the bcrypt hash.
func (u *User) CheckPassword(plainPassword string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(plainPassword))
	return err == nil
}
