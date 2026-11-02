package handler

import "net/http"

// RegisterRoutes mounts all comment module endpoints on the ServeMux.
func RegisterRoutes(
	mux *http.ServeMux,
	h *CommentHandler,
	requireAuth func(http.Handler) http.Handler,
	optionalAuth func(http.Handler) http.Handler,
) {
	mux.Handle("POST /api/v1/posts/{id}/comments", requireAuth(http.HandlerFunc(h.CreateComment)))
	mux.Handle("GET /api/v1/posts/{id}/comments", optionalAuth(http.HandlerFunc(h.ListComments)))
	mux.Handle("GET /api/v1/comments/{id}/replies", optionalAuth(http.HandlerFunc(h.ListReplies)))
	mux.Handle("DELETE /api/v1/comments/{id}", requireAuth(http.HandlerFunc(h.DeleteComment)))
}
