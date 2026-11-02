package response_test

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	sharedErrors "github.com/anjal-niraula/engage-engine/internal/shared/errors"
	"github.com/anjal-niraula/engage-engine/internal/shared/response"
)

func TestWriteSuccess(t *testing.T) {
	rec := httptest.NewRecorder()
	payload := map[string]string{"username": "testuser"}

	response.WriteSuccess(rec, http.StatusOK, payload)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	var body response.Envelope
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to decode response body: %v", err)
	}

	if !body.Success {
		t.Errorf("expected success true, got false")
	}
	dataMap, ok := body.Data.(map[string]any)
	if !ok || dataMap["username"] != "testuser" {
		t.Errorf("expected data.username to be 'testuser', got %v", body.Data)
	}
}

func TestWritePaginated(t *testing.T) {
	rec := httptest.NewRecorder()
	payload := []string{"item1", "item2"}
	cursor := "eyJuZXh0Ijo0Mn0="

	response.WritePaginated(rec, http.StatusOK, payload, &cursor, true, 20)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	var body response.Envelope
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to decode response body: %v", err)
	}

	if body.Meta == nil {
		t.Fatalf("expected meta object, got nil")
	}
	if body.Meta.Cursor == nil || *body.Meta.Cursor != cursor {
		t.Errorf("expected cursor %s, got %v", cursor, body.Meta.Cursor)
	}
	if !body.Meta.HasMore {
		t.Errorf("expected has_more true, got false")
	}
	if body.Meta.Limit != 20 {
		t.Errorf("expected limit 20, got %d", body.Meta.Limit)
	}
}

func TestWriteNoContent(t *testing.T) {
	rec := httptest.NewRecorder()
	response.WriteNoContent(rec)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected status 204, got %d", rec.Code)
	}
	if rec.Body.Len() != 0 {
		t.Errorf("expected empty body, got %s", rec.Body.String())
	}
}

func TestWriteError_HTTPError(t *testing.T) {
	rec := httptest.NewRecorder()
	fieldErrors := []sharedErrors.FieldError{
		{Field: "email", Error: "must be valid"},
	}
	httpErr := sharedErrors.NewBadRequestError("Invalid input", false, nil, fieldErrors, nil)

	response.WriteError(rec, httpErr)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", rec.Code)
	}

	var decoded sharedErrors.HTTPError
	if err := json.Unmarshal(rec.Body.Bytes(), &decoded); err != nil {
		t.Fatalf("failed to decode error body: %v", err)
	}

	if decoded.Code != "BAD_REQUEST" {
		t.Errorf("expected BAD_REQUEST, got %s", decoded.Code)
	}
	if len(decoded.Errors) != 1 || decoded.Errors[0].Field != "email" {
		t.Errorf("expected field error on email, got %+v", decoded.Errors)
	}
}

func TestWriteError_Fallback(t *testing.T) {
	rec := httptest.NewRecorder()
	stdErr := errors.New("database connection lost")

	response.WriteError(rec, stdErr)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected status 500, got %d", rec.Code)
	}

	var decoded sharedErrors.HTTPError
	if err := json.Unmarshal(rec.Body.Bytes(), &decoded); err != nil {
		t.Fatalf("failed to decode fallback error body: %v", err)
	}

	if decoded.Code != "INTERNAL_SERVER_ERROR" {
		t.Errorf("expected code INTERNAL_SERVER_ERROR, got %s", decoded.Code)
	}
}
