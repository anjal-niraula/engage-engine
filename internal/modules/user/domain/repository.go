package domain

import "context"

// UserRepository defines the persistence port for the User entity.
type UserRepository interface {
	Create(ctx context.Context, user *User) error
	GetByID(ctx context.Context, id string) (*User, error)
	GetByUsername(ctx context.Context, username string) (*User, error)
	GetByEmail(ctx context.Context, email string) (*User, error)
	GetByIdentity(ctx context.Context, identity string) (*User, error)
	Update(ctx context.Context, user *User) error
	IsFollowing(ctx context.Context, followerID, followeeID string) (bool, error)
}

