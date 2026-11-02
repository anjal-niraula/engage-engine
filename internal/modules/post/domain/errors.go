package domain

import "errors"

var (
	// ErrPostNotFound is returned when a requested post does not exist or has been deleted.
	ErrPostNotFound = errors.New("post not found")

	// ErrForbiddenPostAction is returned when a user attempts to modify or delete a post they do not own.
	ErrForbiddenPostAction = errors.New("you are not authorized to modify this post")
)
