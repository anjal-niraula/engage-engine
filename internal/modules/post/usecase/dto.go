package usecase

import "time"

// CreatePostInput holds the data required to publish a new post.
type CreatePostInput struct {
	Caption   *string
	MediaURLs []string
	MediaType string
	Location  *string
}

// AuthorOutput represents public author details attached to a post.
type AuthorOutput struct {
	ID          string  `json:"id"`
	Username    string  `json:"username"`
	DisplayName string  `json:"display_name"`
	AvatarURL   *string `json:"avatar_url"`
}

// PostOutput represents the post data returned to API consumers.
type PostOutput struct {
	ID            string        `json:"id"`
	AuthorID      string        `json:"author_id"`
	Caption       *string       `json:"caption"`
	MediaURLs     []string      `json:"media_urls"`
	MediaType     string        `json:"media_type"`
	Location      *string       `json:"location"`
	LikesCount    int64         `json:"likes_count"`
	CommentsCount int64         `json:"comments_count"`
	ViewsCount    int64         `json:"views_count"`
	HasLiked      bool          `json:"has_liked"`
	Author        *AuthorOutput `json:"author,omitempty"`
	CreatedAt     time.Time     `json:"created_at"`
	UpdatedAt     time.Time     `json:"updated_at"`
}
