package handler

import "net/http"

// RegisterRoutes mounts all notification module endpoints on the ServeMux.
func RegisterRoutes(
	mux *http.ServeMux,
	h *NotificationHandler,
	requireAuth func(http.Handler) http.Handler,
	optionalAuth func(http.Handler) http.Handler,
) {
	mux.Handle("GET /api/v1/notifications", requireAuth(http.HandlerFunc(h.ListNotifications)))
	mux.Handle("PATCH /api/v1/notifications/{id}/read", requireAuth(http.HandlerFunc(h.MarkAsRead)))
}
