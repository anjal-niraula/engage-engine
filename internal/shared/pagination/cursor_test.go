package pagination_test

import (
	"net/http/httptest"
	"testing"
	"time"

	"github.com/anjal-niraula/engage-engine/internal/shared/pagination"
)

func TestEncodeAndDecodeCursor(t *testing.T) {
	id := "c5d0e4f3-0000-4000-8000-333344445555"
	now := time.Date(2026, 10, 31, 20, 0, 0, 123456000, time.UTC)

	token := pagination.EncodeCursor(id, now)
	if token == "" {
		t.Fatalf("expected non-empty cursor token")
	}

	decoded, err := pagination.DecodeCursor(token)
	if err != nil {
		t.Fatalf("unexpected error decoding cursor: %v", err)
	}

	if decoded.ID != id {
		t.Errorf("expected ID %s, got %s", id, decoded.ID)
	}
	if !decoded.CreatedAt.Equal(now) {
		t.Errorf("expected CreatedAt %v, got %v", now, decoded.CreatedAt)
	}
}

func TestDecodeCursor_Empty(t *testing.T) {
	decoded, err := pagination.DecodeCursor("")
	if err != nil {
		t.Fatalf("expected nil error on empty cursor, got %v", err)
	}
	if decoded != nil {
		t.Errorf("expected nil payload on empty cursor, got %+v", decoded)
	}
}

func TestDecodeCursor_Invalid(t *testing.T) {
	testCases := []string{
		"not-valid-base64@@",
		"e30=", // empty JSON "{}"
	}

	for _, tc := range testCases {
		_, err := pagination.DecodeCursor(tc)
		if err == nil {
			t.Errorf("expected error for invalid cursor '%s', got nil", tc)
		}
	}
}

func TestParsePaginationParams(t *testing.T) {
	t.Run("default params", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/posts", nil)
		limit, cursor, err := pagination.ParsePaginationParams(req, 20, 50)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if limit != 20 {
			t.Errorf("expected limit 20, got %d", limit)
		}
		if cursor != "" {
			t.Errorf("expected empty cursor, got %s", cursor)
		}
	})

	t.Run("custom valid limit and cursor", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/posts?limit=15&cursor=abc123token", nil)
		limit, cursor, err := pagination.ParsePaginationParams(req, 20, 50)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if limit != 15 {
			t.Errorf("expected limit 15, got %d", limit)
		}
		if cursor != "abc123token" {
			t.Errorf("expected cursor 'abc123token', got %s", cursor)
		}
	})

	t.Run("limit capped at maxLimit", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/posts?limit=200", nil)
		limit, _, err := pagination.ParsePaginationParams(req, 20, 50)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if limit != 50 {
			t.Errorf("expected limit capped at 50, got %d", limit)
		}
	})

	t.Run("invalid limit error", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/posts?limit=-5", nil)
		_, _, err := pagination.ParsePaginationParams(req, 20, 50)
		if err == nil {
			t.Errorf("expected error for negative limit, got nil")
		}
	})
}
