package handler

import "net/http"

// RegisterRoutes mounts all analytics and views endpoints on the ServeMux.
func RegisterRoutes(
	mux *http.ServeMux,
	h *AnalyticsHandler,
	requireAuth func(http.Handler) http.Handler,
	optionalAuth func(http.Handler) http.Handler,
) {
	mux.Handle("POST /api/v1/views", optionalAuth(http.HandlerFunc(h.TrackView)))
	mux.Handle("GET /api/v1/posts/{id}/stats", optionalAuth(http.HandlerFunc(h.GetPostStats)))
	mux.Handle("GET /api/v1/posts/{id}/analytics", requireAuth(http.HandlerFunc(h.GetPostAnalytics)))
	mux.Handle("GET /api/v1/users/me/analytics", requireAuth(http.HandlerFunc(h.GetUserAnalytics)))
}
