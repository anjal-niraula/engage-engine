package domain

import "errors"

var (
	// ErrCannotFollowSelf is returned when a user attempts to follow themselves.
	ErrCannotFollowSelf = errors.New("cannot follow yourself")

	// ErrTargetUserNotFound is returned when the user to follow does not exist.
	ErrTargetUserNotFound = errors.New("target user not found")
)
