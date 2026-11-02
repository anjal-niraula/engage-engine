package docs

import (
	_ "embed"
	"net/http"
)

//go:embed openapi.json
var openAPISpec []byte

//go:embed openapi.html
var openAPIHTML []byte

// RegisterRoutes registers the interactive API documentation and spec endpoints.
func RegisterRoutes(mux *http.ServeMux) {
	// 1. Raw OpenAPI 3.0 specification JSON
	mux.HandleFunc("GET /docs/openapi.json", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "no-cache")
		_, _ = w.Write(openAPISpec)
	})

	// 2. Interactive Scalar UI documentation page
	mux.HandleFunc("GET /docs", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Cache-Control", "no-cache")
		_, _ = w.Write(openAPIHTML)
	})
}

