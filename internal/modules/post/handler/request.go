package handler

// CreatePostRequest defines JSON validation tags for post publishing.
type CreatePostRequest struct {
	Caption   *string  `json:"caption" validate:"omitempty,max=2200"`
	MediaURLs []string `json:"media_urls" validate:"required,min=1,max=10,dive,url"`
	MediaType string   `json:"media_type" validate:"required,oneof=image video carousel"`
	Location  *string  `json:"location" validate:"omitempty,max=100"`
}
