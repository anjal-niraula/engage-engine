package domain

import "errors"

var (
	ErrPostNotFound = errors.New("post not found")
	ErrForbidden    = errors.New("you are not authorized to view analytics for this resource")
)
