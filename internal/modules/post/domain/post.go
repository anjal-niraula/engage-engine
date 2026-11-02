package domain

import "time"

// Post represents a user post with media and engagement counters.
type Post struct {
	ID            string
	AuthorID      string
	Caption       *string
	MediaURLs     []string
	MediaType     string
	Location      *string
	LikesCount    int64
	CommentsCount int64
	ViewsCount    int64
	CreatedAt     time.Time
	UpdatedAt     time.Time
	DeletedAt     *time.Time
}

// Author represents the public snapshot of a user who created a post.
type Author struct {
	ID          string
	Username    string
	DisplayName string
	AvatarURL   *string
}

// NewPost initializes a new Post domain entity.
func NewPost(authorID string, caption *string, mediaURLs []string, mediaType string, location *string) *Post {
	now := time.Now().UTC()
	return &Post{
		AuthorID:  authorID,
		Caption:   caption,
		MediaURLs: mediaURLs,
		MediaType: mediaType,
		Location:  location,
		CreatedAt: now,
		UpdatedAt: now,
	}
}
