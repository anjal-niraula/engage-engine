package docs_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/anjal-niraula/engage-engine/internal/platform/docs"
	"github.com/stretchr/testify/assert"
)

func TestDocsRoutes(t *testing.T) {
	mux := http.NewServeMux()
	docs.RegisterRoutes(mux)

	t.Run("GET /docs returns HTML UI", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/docs", nil)
		rec := httptest.NewRecorder()

		mux.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Contains(t, rec.Header().Get("Content-Type"), "text/html")
		assert.Contains(t, rec.Body.String(), "api-reference")
	})

	t.Run("GET /docs/openapi.json returns valid OpenAPI 3 spec", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/docs/openapi.json", nil)
		rec := httptest.NewRecorder()

		mux.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Contains(t, rec.Header().Get("Content-Type"), "application/json")

		var spec map[string]interface{}
		err := json.Unmarshal(rec.Body.Bytes(), &spec)
		assert.NoError(t, err)
		assert.Equal(t, "3.0.3", spec["openapi"])
		assert.NotNil(t, spec["paths"])
	})
}

