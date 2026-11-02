package domain

import "errors"

var (
	ErrCommentNotFound        = errors.New("comment not found")
	ErrPostNotFound           = errors.New("post not found")
	ErrParentNotFound         = errors.New("parent comment not found")
	ErrMismatchedParentPost   = errors.New("parent comment does not belong to this post")
	ErrForbiddenCommentAction = errors.New("you are not authorized to delete this comment")
)
