package validation_test

import (
	"bytes"
	"net/http/httptest"
	"testing"

	"github.com/anjal-niraula/engage-engine/internal/shared/validation"
)

type TestUserPayload struct {
	Username string `json:"username" validate:"required,min=3,max=30"`
	Email    string `json:"email" validate:"required,email"`
	Age      int    `json:"age" validate:"min=18"`
	Role     string `json:"role" validate:"oneof=admin member"`
}

func TestDecodeAndValidate_Success(t *testing.T) {
	jsonBody := `{"username":"johndoe","email":"john@example.com","age":25,"role":"member"}`
	req := httptest.NewRequest("POST", "/test", bytes.NewBufferString(jsonBody))

	var payload TestUserPayload
	err := validation.DecodeAndValidate(req, &payload)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	if payload.Username != "johndoe" || payload.Email != "john@example.com" {
		t.Errorf("unexpected payload content: %+v", payload)
	}
}

func TestDecodeAndValidate_FieldValidationFailure(t *testing.T) {
	jsonBody := `{"username":"jo","email":"invalid-email","age":12,"role":"guest"}`
	req := httptest.NewRequest("POST", "/test", bytes.NewBufferString(jsonBody))

	var payload TestUserPayload
	httpErr := validation.DecodeAndValidate(req, &payload)
	if httpErr == nil {
		t.Fatalf("expected validation error, got nil")
	}

	if httpErr.Status != 400 {
		t.Errorf("expected status 400, got %d", httpErr.Status)
	}

	if len(httpErr.Errors) != 4 {
		t.Errorf("expected 4 field errors, got %d: %+v", len(httpErr.Errors), httpErr.Errors)
	}
}

func TestDecodeAndValidate_MalformedJSON(t *testing.T) {
	jsonBody := `{"username":"incomplete`
	req := httptest.NewRequest("POST", "/test", bytes.NewBufferString(jsonBody))

	var payload TestUserPayload
	httpErr := validation.DecodeAndValidate(req, &payload)
	if httpErr == nil {
		t.Fatalf("expected error for malformed json, got nil")
	}

	if httpErr.Status != 400 {
		t.Errorf("expected status 400, got %d", httpErr.Status)
	}
}

func TestDecodeAndValidate_UnknownField(t *testing.T) {
	jsonBody := `{"username":"johndoe","email":"john@example.com","age":25,"role":"member","extra":"forbidden"}`
	req := httptest.NewRequest("POST", "/test", bytes.NewBufferString(jsonBody))

	var payload TestUserPayload
	httpErr := validation.DecodeAndValidate(req, &payload)
	if httpErr == nil {
		t.Fatalf("expected error for unknown field, got nil")
	}
}

func TestIsValidUUID(t *testing.T) {
	valid := "c5d0e4f3-0000-4000-8000-333344445555"
	invalid := "c5d0e4f3-0000-4000-8000"

	if !validation.IsValidUUID(valid) {
		t.Errorf("expected %s to be valid UUID", valid)
	}
	if validation.IsValidUUID(invalid) {
		t.Errorf("expected %s to be invalid UUID", invalid)
	}
}
