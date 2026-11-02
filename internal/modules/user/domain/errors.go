package domain

import "errors"

var (
	// ErrUserNotFound is returned when a user record does not exist.
	ErrUserNotFound = errors.New("user not found")

	// ErrUsernameTaken is returned when a requested username already exists.
	ErrUsernameTaken = errors.New("username is already taken")

	// ErrEmailTaken is returned when a requested email address already exists.
	ErrEmailTaken = errors.New("email is already registered")

	// ErrInvalidCredentials is returned when login authentication fails.
	ErrInvalidCredentials = errors.New("invalid email/username or password")
)

