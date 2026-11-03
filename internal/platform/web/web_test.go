package web_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/anjal-niraula/engage-engine/internal/platform/web"
	"github.com/stretchr/testify/assert"
)

func TestWebRoutes(t *testing.T) {
	mux := http.NewServeMux()
	web.RegisterRoutes(mux)

	t.Run("GET / returns Instagram UI", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()

		mux.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Contains(t, rec.Header().Get("Content-Type"), "text/html")
		assert.Contains(t, rec.Body.String(), "EngageEngine")
		assert.Contains(t, rec.Body.String(), "Cuckoo Filter")
	})

	t.Run("GET /app returns Instagram UI", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/app", nil)
		rec := httptest.NewRecorder()

		mux.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Contains(t, rec.Header().Get("Content-Type"), "text/html")
	})
}

