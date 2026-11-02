package pagination

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"
)

var (
	// ErrInvalidCursor indicates the cursor token could not be parsed or decoded.
	ErrInvalidCursor = errors.New("invalid pagination cursor")
)

// CursorPayload represents the keyset values encoded in the cursor token.
type CursorPayload struct {
	ID        string    `json:"id"`
	CreatedAt time.Time `json:"created_at"`
}

// EncodeCursor serializes a record ID and timestamp into a URL-safe Base64 token.
func EncodeCursor(id string, createdAt time.Time) string {
	payload := CursorPayload{
		ID:        id,
		CreatedAt: createdAt.UTC(),
	}
	bytes, err := json.Marshal(payload)
	if err != nil {
		return ""
	}
	return base64.RawURLEncoding.EncodeToString(bytes)
}

// DecodeCursor parses and validates a URL-safe Base64 cursor string into a CursorPayload.
func DecodeCursor(cursorStr string) (*CursorPayload, error) {
	if cursorStr == "" {
		return nil, nil
	}

	bytes, err := base64.RawURLEncoding.DecodeString(cursorStr)
	if err != nil {
		return nil, ErrInvalidCursor
	}

	var payload CursorPayload
	if err := json.Unmarshal(bytes, &payload); err != nil {
		return nil, ErrInvalidCursor
	}

	if payload.ID == "" || payload.CreatedAt.IsZero() {
		return nil, ErrInvalidCursor
	}

	return &payload, nil
}

// ParsePaginationParams extracts and validates the limit and cursor query parameters from an HTTP request.
func ParsePaginationParams(r *http.Request, defaultLimit, maxLimit int) (int, string, error) {
	limit := defaultLimit
	if param := r.URL.Query().Get("limit"); param != "" {
		parsedLimit, err := strconv.Atoi(param)
		if err != nil || parsedLimit <= 0 {
			return 0, "", errors.New("limit must be a positive integer")
		}
		if parsedLimit > maxLimit {
			limit = maxLimit
		} else {
			limit = parsedLimit
		}
	}

	cursor := r.URL.Query().Get("cursor")
	return limit, cursor, nil
}

