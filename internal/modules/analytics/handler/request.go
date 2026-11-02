package handler

type TrackViewRequest struct {
	PostID     string `json:"post_id" validate:"required,uuid"`
	SessionID  string `json:"session_id" validate:"required,min=3,max=100"`
	DurationMS int    `json:"duration_ms" validate:"gte=0"`
}
