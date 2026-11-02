package errors

import (
	"strings"
)

// FieldError describes a validation failure on a specific input field.
type FieldError struct {
	Field string `json:"field"`
	Error string `json:"error"`
}

// ActionType defines client actions upon encountering an error.
type ActionType string

const (
	ActionTypeRedirect ActionType = "redirect"
)

// Action gives clients instructions on how to handle the error.
type Action struct {
	Type    ActionType `json:"type"`
	Message string     `json:"message"`
	Value   string     `json:"value"`
}

// HTTPError is a structured error carrying HTTP status codes, error codes, and field errors.
type HTTPError struct {
	Code     string       `json:"code"`
	Message  string       `json:"message"`
	Status   int          `json:"status"`
	Override bool         `json:"override"`
	Errors   []FieldError `json:"errors,omitempty"`
	Action   *Action      `json:"action,omitempty"`
}

func (e *HTTPError) Error() string {
	return e.Message
}

func (e *HTTPError) Is(target error) bool {
	_, ok := target.(*HTTPError)
	return ok
}

func (e *HTTPError) WithMessage(message string) *HTTPError {
	return &HTTPError{
		Code:     e.Code,
		Message:  message,
		Status:   e.Status,
		Override: e.Override,
		Errors:   e.Errors,
		Action:   e.Action,
	}
}

// MakeUpperCaseWithUnderscores converts human-readable strings to UPPER_CASE_SNAKE_CASE.
func MakeUpperCaseWithUnderscores(str string) string {
	return strings.ToUpper(strings.ReplaceAll(str, " ", "_"))
}
