package response

import (
	"encoding/json"
	"net/http"

	"github.com/anjal-niraula/engage-engine/internal/shared/errors"
)

// Meta contains pagination and collection metadata.
type Meta struct {
	Cursor  *string `json:"cursor,omitempty"`
	HasMore bool    `json:"has_more"`
	Limit   int     `json:"limit,omitempty"`
}

// Envelope is the standard JSON success response format.
type Envelope struct {
	Success bool  `json:"success"`
	Data    any   `json:"data"`
	Meta    *Meta `json:"meta,omitempty"`
}

// WriteSuccess writes a standard JSON success response with HTTP status code and payload data.
func WriteSuccess(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)

	resp := Envelope{
		Success: true,
		Data:    data,
	}

	_ = json.NewEncoder(w).Encode(resp)
}

// WritePaginated writes a standard paginated response containing cursor metadata.
func WritePaginated(w http.ResponseWriter, status int, data any, cursor *string, hasMore bool, limit int) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)

	resp := Envelope{
		Success: true,
		Data:    data,
		Meta: &Meta{
			Cursor:  cursor,
			HasMore: hasMore,
			Limit:   limit,
		},
	}

	_ = json.NewEncoder(w).Encode(resp)
}

// WriteNoContent writes an empty HTTP 204 No Content response.
func WriteNoContent(w http.ResponseWriter) {
	w.WriteHeader(http.StatusNoContent)
}

// WriteError serializes a standardized JSON error response.
// If the given error implements *errors.HTTPError, its structured status and field errors are returned.
// Otherwise, an HTTP 500 Internal Server Error is returned as a fallback.
func WriteError(w http.ResponseWriter, err error) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")

	if httpErr, ok := err.(*errors.HTTPError); ok {
		w.WriteHeader(httpErr.Status)
		_ = json.NewEncoder(w).Encode(httpErr)
		return
	}

	internalErr := errors.NewInternalServerError()
	w.WriteHeader(internalErr.Status)
	_ = json.NewEncoder(w).Encode(internalErr)
}

