package handler

import "net/http"

// RegisterRoutes registers all follow module endpoints on the ServeMux.
func RegisterRoutes(
	mux *http.ServeMux,
	h *FollowHandler,
	requireAuth func(http.Handler) http.Handler,
	optionalAuth func(http.Handler) http.Handler,
) {
	mux.Handle("POST /api/v1/users/{id}/follow", requireAuth(http.HandlerFunc(h.Follow)))
	mux.Handle("DELETE /api/v1/users/{id}/follow", requireAuth(http.HandlerFunc(h.Unfollow)))

	mux.Handle("GET /api/v1/users/{id}/followers", optionalAuth(http.HandlerFunc(h.ListFollowers)))
	mux.Handle("GET /api/v1/users/{id}/following", optionalAuth(http.HandlerFunc(h.ListFollowing)))
}
