package handler

import (
	"errors"
	"net/http"

	"github.com/anjal-niraula/engage-engine/internal/modules/views/domain"
	"github.com/anjal-niraula/engage-engine/internal/modules/views/usecase"
	sharedErrors "github.com/anjal-niraula/engage-engine/internal/shared/errors"
	"github.com/anjal-niraula/engage-engine/internal/shared/middleware"
	"github.com/anjal-niraula/engage-engine/internal/shared/response"
	"github.com/anjal-niraula/engage-engine/internal/shared/validation"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

type ViewsHandler struct {
	uc     usecase.ViewUseCase
	logger *zerolog.Logger
}

type AnalyticsHandler = ViewsHandler

func NewViewsHandler(uc usecase.ViewUseCase, logger *zerolog.Logger) *ViewsHandler {
	return &ViewsHandler{uc: uc, logger: logger}
}

func NewAnalyticsHandler(uc usecase.ViewUseCase, logger *zerolog.Logger) *ViewsHandler {
	return NewViewsHandler(uc, logger)
}

// TrackView handles POST /api/v1/views (fire-and-forget 202 Accepted)
func (h *ViewsHandler) TrackView(w http.ResponseWriter, r *http.Request) {
	var req TrackViewRequest
	if err := validation.DecodeAndValidate(r, &req); err != nil {
		response.WriteError(w, err)
		return
	}

	postID, err := uuid.Parse(req.PostID)
	if err != nil {
		response.WriteError(w, sharedErrors.NewBadRequestError("Invalid post ID", false, nil, nil, nil))
		return
	}

	// Canonical Viewer Identity for Redis HyperLogLog deduplication:
	// 1. Authenticated user (JWT): "user:<userID>"
	// 2. Client header (x-user-id): "user:<x-user-id>"
	// 3. Persistent Client Session: "anon:<sessionID>"
	// 4. Remote IP fallback: "anon:<IP>"
	// This guarantees that repeated visits by the SAME user NEVER inflate the unique viewer count in HLL!
	viewerIdentity := ""
	if userID, ok := middleware.GetUserID(r.Context()); ok && userID != "" {
		viewerIdentity = "user:" + userID
	} else if clientUser := r.Header.Get("x-user-id"); clientUser != "" {
		viewerIdentity = "user:" + clientUser
	} else if req.SessionID != "" {
		viewerIdentity = req.SessionID
	} else {
		viewerIdentity = "anon:" + r.RemoteAddr
	}

	err = h.uc.RecordView(r.Context(), usecase.RecordViewInput{
		PostID:     postID,
		SessionID:  viewerIdentity,
		DurationMS: req.DurationMS,
	})
	if err != nil {
		h.logger.Error().Err(err).Msg("failed to record view")
		response.WriteError(w, sharedErrors.NewInternalServerError())
		return
	}

	response.WriteSuccess(w, http.StatusAccepted, map[string]string{
		"status": "queued",
	})
}

// GetPostStats handles GET /api/v1/posts/{id}/stats
func (h *ViewsHandler) GetPostStats(w http.ResponseWriter, r *http.Request) {
	postIDStr := r.PathValue("id")
	if !validation.IsValidUUID(postIDStr) {
		response.WriteError(w, sharedErrors.NewBadRequestError("Invalid post ID, UUID expected", false, nil, nil, nil))
		return
	}

	postID, _ := uuid.Parse(postIDStr)
	stats, err := h.uc.GetPostStats(r.Context(), postID)
	if err != nil {
		if errors.Is(err, domain.ErrPostNotFound) {
			response.WriteError(w, sharedErrors.NewNotFoundError("Post not found", false, nil))
			return
		}
		h.logger.Error().Err(err).Msg("failed to get post stats")
		response.WriteError(w, sharedErrors.NewInternalServerError())
		return
	}

	response.WriteSuccess(w, http.StatusOK, stats)
}

// GetPostAnalytics handles GET /api/v1/posts/{id}/analytics (Owner only)
func (h *ViewsHandler) GetPostAnalytics(w http.ResponseWriter, r *http.Request) {
	postIDStr := r.PathValue("id")
	if !validation.IsValidUUID(postIDStr) {
		response.WriteError(w, sharedErrors.NewBadRequestError("Invalid post ID, UUID expected", false, nil, nil, nil))
		return
	}

	userIDStr, ok := middleware.GetUserID(r.Context())
	if !ok {
		response.WriteError(w, sharedErrors.NewUnauthorizedError("Authentication required", false))
		return
	}

	postID, _ := uuid.Parse(postIDStr)
	requestingUserID, err := uuid.Parse(userIDStr)
	if err != nil {
		response.WriteError(w, sharedErrors.NewBadRequestError("Invalid authenticated user ID", false, nil, nil, nil))
		return
	}

	analytics, err := h.uc.GetPostAnalytics(r.Context(), postID, requestingUserID)
	if err != nil {
		if errors.Is(err, domain.ErrPostNotFound) {
			response.WriteError(w, sharedErrors.NewNotFoundError("Post not found", false, nil))
			return
		}
		if errors.Is(err, domain.ErrForbidden) {
			response.WriteError(w, sharedErrors.NewForbiddenError("You are not authorized to view analytics for this post", false))
			return
		}
		h.logger.Error().Err(err).Msg("failed to get post analytics")
		response.WriteError(w, sharedErrors.NewInternalServerError())
		return
	}

	response.WriteSuccess(w, http.StatusOK, analytics)
}

// GetUserAnalytics handles GET /api/v1/users/me/analytics
func (h *ViewsHandler) GetUserAnalytics(w http.ResponseWriter, r *http.Request) {
	userIDStr, ok := middleware.GetUserID(r.Context())
	if !ok {
		response.WriteError(w, sharedErrors.NewUnauthorizedError("Authentication required", false))
		return
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		response.WriteError(w, sharedErrors.NewBadRequestError("Invalid authenticated user ID", false, nil, nil, nil))
		return
	}

	period := r.URL.Query().Get("period")
	if period == "" {
		period = "30d"
	}
	analytics, err := h.uc.GetUserAnalytics(r.Context(), userID, period)
	if err != nil {
		h.logger.Error().Err(err).Msg("failed to get creator analytics")
		response.WriteError(w, sharedErrors.NewInternalServerError())
		return
	}

	response.WriteSuccess(w, http.StatusOK, analytics)
}
