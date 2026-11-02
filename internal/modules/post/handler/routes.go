package handler

import "net/http"

// RegisterRoutes registers all post module endpoints onto the HTTP ServeMux.
func RegisterRoutes(
	mux *http.ServeMux,
	h *PostHandler,
	requireAuth func(http.Handler) http.Handler,
	optionalAuth func(http.Handler) http.Handler,
) {
	mux.Handle("POST /api/v1/posts", requireAuth(http.HandlerFunc(h.Create)))
	mux.Handle("GET /api/v1/posts/{id}", optionalAuth(http.HandlerFunc(h.GetByID)))
	mux.Handle("DELETE /api/v1/posts/{id}", requireAuth(http.HandlerFunc(h.Delete)))

	mux.Handle("GET /api/v1/users/{username}/posts", optionalAuth(http.HandlerFunc(h.ListUserPosts)))
	mux.Handle("GET /api/v1/feed", requireAuth(http.HandlerFunc(h.GetFeed)))
}
