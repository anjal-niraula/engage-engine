package handler

import (
	"errors"
	"net/http"

	"github.com/anjal-niraula/engage-engine/internal/modules/analytics/domain"
	"github.com/anjal-niraula/engage-engine/internal/modules/analytics/usecase"
	sharedErrors "github.com/anjal-niraula/engage-engine/internal/shared/errors"
	"github.com/anjal-niraula/engage-engine/internal/shared/middleware"
	"github.com/anjal-niraula/engage-engine/internal/shared/response"
	"github.com/anjal-niraula/engage-engine/internal/shared/validation"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

type AnalyticsHandler struct {
	uc     usecase.AnalyticsUseCase
	logger *zerolog.Logger
}

func NewAnalyticsHandler(uc usecase.AnalyticsUseCase, logger *zerolog.Logger) *AnalyticsHandler {
	return &AnalyticsHandler{uc: uc, logger: logger}
}

// TrackView handles POST /api/v1/views (fire-and-forget 202 Accepted)
func (h *AnalyticsHandler) TrackView(w http.ResponseWriter, r *http.Request) {
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

	err = h.uc.RecordView(r.Context(), usecase.RecordViewInput{
		PostID:     postID,
		SessionID:  req.SessionID,
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
func (h *AnalyticsHandler) GetPostStats(w http.ResponseWriter, r *http.Request) {
	postIDStr := r.PathValue("id")
	if !validation.IsValidUUID(postIDStr) {
		response.WriteError(w, sharedErrors.NewBadRequestError("Invalid post ID, UUID expected", false, nil, nil, nil))
		return
	}
	postID, err := uuid.Parse(postIDStr)
	if err != nil {
		response.WriteError(w, sharedErrors.NewBadRequestError("Invalid post ID, UUID expected", false, nil, nil, nil))
		return
	}

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

// GetPostAnalytics handles GET /api/v1/posts/{id}/analytics
func (h *AnalyticsHandler) GetPostAnalytics(w http.ResponseWriter, r *http.Request) {
	userIDStr, ok := middleware.GetUserID(r.Context())
	if !ok {
		response.WriteError(w, sharedErrors.NewUnauthorizedError("Authentication required", false))
		return
	}
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		response.WriteError(w, sharedErrors.NewUnauthorizedError("Invalid user identity", false))
		return
	}

	postIDStr := r.PathValue("id")
	if !validation.IsValidUUID(postIDStr) {
		response.WriteError(w, sharedErrors.NewBadRequestError("Invalid post ID, UUID expected", false, nil, nil, nil))
		return
	}
	postID, err := uuid.Parse(postIDStr)
	if err != nil {
		response.WriteError(w, sharedErrors.NewBadRequestError("Invalid post ID, UUID expected", false, nil, nil, nil))
		return
	}

	analytics, err := h.uc.GetPostAnalytics(r.Context(), postID, userID)
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrPostNotFound):
			response.WriteError(w, sharedErrors.NewNotFoundError("Post not found", false, nil))
		case errors.Is(err, domain.ErrForbidden):
			response.WriteError(w, sharedErrors.NewForbiddenError("You are not authorized to view this post's analytics", false))
		default:
			h.logger.Error().Err(err).Msg("failed to get post analytics")
			response.WriteError(w, sharedErrors.NewInternalServerError())
		}
		return
	}

	response.WriteSuccess(w, http.StatusOK, analytics)
}

// GetUserAnalytics handles GET /api/v1/users/me/analytics
func (h *AnalyticsHandler) GetUserAnalytics(w http.ResponseWriter, r *http.Request) {
	userIDStr, ok := middleware.GetUserID(r.Context())
	if !ok {
		response.WriteError(w, sharedErrors.NewUnauthorizedError("Authentication required", false))
		return
	}
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		response.WriteError(w, sharedErrors.NewUnauthorizedError("Invalid user identity", false))
		return
	}

	period := r.URL.Query().Get("range")
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
