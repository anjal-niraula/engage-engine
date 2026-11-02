package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/anjal-niraula/engage-engine/internal/shared/middleware"
)

const testSecret = "test-secret-32-bytes-long-key-for-test"

func TestTokenGenerationAndValidation(t *testing.T) {
	userID := "c5d0e4f3-0000-4000-8000-333344445555"

	token, err := middleware.GenerateToken(userID, testSecret, 1*time.Hour)
	if err != nil {
		t.Fatalf("unexpected error generating token: %v", err)
	}

	extractedID, err := middleware.ValidateToken(token, testSecret)
	if err != nil {
		t.Fatalf("unexpected error validating token: %v", err)
	}

	if extractedID != userID {
		t.Errorf("expected userID %s, got %s", userID, extractedID)
	}
}

func TestRequireAuthMiddleware_Success(t *testing.T) {
	userID := "c5d0e4f3-0000-4000-8000-333344445555"
	token, _ := middleware.GenerateToken(userID, testSecret, 1*time.Hour)

	var recordedUserID string
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id, _ := middleware.GetUserID(r.Context())
		recordedUserID = id
		w.WriteHeader(http.StatusOK)
	})

	mw := middleware.RequireAuth(testSecret)(handler)

	req := httptest.NewRequest("GET", "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()

	mw.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
	if recordedUserID != userID {
		t.Errorf("expected recorded userID %s, got %s", userID, recordedUserID)
	}
}

func TestRequireAuthMiddleware_MissingHeader(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	mw := middleware.RequireAuth(testSecret)(handler)

	req := httptest.NewRequest("GET", "/protected", nil)
	rec := httptest.NewRecorder()

	mw.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d", rec.Code)
	}
}

func TestRequireAuthMiddleware_InvalidToken(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	mw := middleware.RequireAuth(testSecret)(handler)

	req := httptest.NewRequest("GET", "/protected", nil)
	req.Header.Set("Authorization", "Bearer invalid.jwt.token")
	rec := httptest.NewRecorder()

	mw.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d", rec.Code)
	}
}

func TestOptionalAuthMiddleware_WithAndWithoutToken(t *testing.T) {
	userID := "c5d0e4f3-0000-4000-8000-333344445555"
	token, _ := middleware.GenerateToken(userID, testSecret, 1*time.Hour)

	var recordedUserID string
	var hasUser bool
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		recordedUserID, hasUser = middleware.GetUserID(r.Context())
		w.WriteHeader(http.StatusOK)
	})

	mw := middleware.OptionalAuth(testSecret)(handler)

	// 1. Without Token
	req1 := httptest.NewRequest("GET", "/public", nil)
	rec1 := httptest.NewRecorder()
	mw.ServeHTTP(rec1, req1)
	if rec1.Code != http.StatusOK || hasUser {
		t.Errorf("expected public request to succeed without user, got status %d, hasUser %v", rec1.Code, hasUser)
	}

	// 2. With Valid Token
	req2 := httptest.NewRequest("GET", "/public", nil)
	req2.Header.Set("Authorization", "Bearer "+token)
	rec2 := httptest.NewRecorder()
	mw.ServeHTTP(rec2, req2)
	if rec2.Code != http.StatusOK || !hasUser || recordedUserID != userID {
		t.Errorf("expected user in context, got status %d, hasUser %v, id %s", rec2.Code, hasUser, recordedUserID)
	}
}

