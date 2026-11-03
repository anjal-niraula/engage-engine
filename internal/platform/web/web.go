package web

import (
	_ "embed"
	"net/http"
)

//go:embed index.html
var indexHTML []byte

// RegisterRoutes mounts the interactive Instagram-style web UI on the router.
func RegisterRoutes(mux *http.ServeMux) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Cache-Control", "no-cache")
		_, _ = w.Write(indexHTML)
	})

	mux.Handle("GET /{$}", handler)
	mux.Handle("GET /app", handler)
	mux.Handle("GET /u/{username...}", handler)
	mux.Handle("GET /p/{postId...}", handler)
}

