package handler

import "net/http"

// RegisterRoutes mounts all like module endpoints on the ServeMux.
func RegisterRoutes(
	mux *http.ServeMux,
	h *LikeHandler,
	requireAuth func(http.Handler) http.Handler,
	optionalAuth func(http.Handler) http.Handler,
) {
	mux.Handle("POST /api/v1/posts/{id}/likes", requireAuth(http.HandlerFunc(h.LikePost)))
	mux.Handle("DELETE /api/v1/posts/{id}/likes", requireAuth(http.HandlerFunc(h.UnlikePost)))
	mux.Handle("GET /api/v1/posts/{id}/likes", optionalAuth(http.HandlerFunc(h.ListPostLikes)))

	mux.Handle("POST /api/v1/comments/{id}/likes", requireAuth(http.HandlerFunc(h.LikeComment)))
	mux.Handle("DELETE /api/v1/comments/{id}/likes", requireAuth(http.HandlerFunc(h.UnlikeComment)))
}
