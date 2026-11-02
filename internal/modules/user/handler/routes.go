package handler

import "net/http"

// RegisterRoutes registers all user module endpoints onto the HTTP ServeMux.
func RegisterRoutes(
	mux *http.ServeMux,
	h *UserHandler,
	requireAuth func(http.Handler) http.Handler,
	optionalAuth func(http.Handler) http.Handler,
) {
	mux.HandleFunc("POST /api/v1/users/register", h.Register)
	mux.HandleFunc("POST /api/v1/users/login", h.Login)

	mux.Handle("GET /api/v1/users/me", requireAuth(http.HandlerFunc(h.GetMe)))
	mux.Handle("PATCH /api/v1/users/me", requireAuth(http.HandlerFunc(h.UpdateMe)))

	mux.Handle("GET /api/v1/users/{username}", optionalAuth(http.HandlerFunc(h.GetByUsername)))
}

