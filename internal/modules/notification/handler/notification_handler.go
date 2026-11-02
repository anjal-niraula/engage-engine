package handler

import (
	"errors"
	"net/http"

	"github.com/anjal-niraula/engage-engine/internal/modules/notification/domain"
	"github.com/anjal-niraula/engage-engine/internal/modules/notification/usecase"
	sharedErrors "github.com/anjal-niraula/engage-engine/internal/shared/errors"
	"github.com/anjal-niraula/engage-engine/internal/shared/middleware"
	"github.com/anjal-niraula/engage-engine/internal/shared/pagination"
	"github.com/anjal-niraula/engage-engine/internal/shared/response"
	"github.com/anjal-niraula/engage-engine/internal/shared/validation"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

type NotificationHandler struct {
	uc     usecase.NotificationUseCase
	logger *zerolog.Logger
}

func NewNotificationHandler(uc usecase.NotificationUseCase, logger *zerolog.Logger) *NotificationHandler {
	return &NotificationHandler{uc: uc, logger: logger}
}

func (h *NotificationHandler) ListNotifications(w http.ResponseWriter, r *http.Request) {
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

	limit, cursor, err := pagination.ParsePaginationParams(r, 20, 50)
	if err != nil {
		response.WriteError(w, sharedErrors.NewBadRequestError(err.Error(), false, nil, nil, nil))
		return
	}

	notifications, nextCursor, hasMore, err := h.uc.ListNotifications(r.Context(), userID, limit, cursor)
	if err != nil {
		h.logger.Error().Err(err).Msg("failed to list notifications")
		response.WriteError(w, sharedErrors.NewInternalServerError())
		return
	}

	response.WritePaginated(w, http.StatusOK, notifications, nextCursor, hasMore, limit)
}

func (h *NotificationHandler) MarkAsRead(w http.ResponseWriter, r *http.Request) {
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

	idStr := r.PathValue("id")
	if !validation.IsValidUUID(idStr) {
		response.WriteError(w, sharedErrors.NewBadRequestError("Invalid notification ID, UUID expected", false, nil, nil, nil))
		return
	}
	notifID, err := uuid.Parse(idStr)
	if err != nil {
		response.WriteError(w, sharedErrors.NewBadRequestError("Invalid notification ID, UUID expected", false, nil, nil, nil))
		return
	}

	err = h.uc.MarkAsRead(r.Context(), notifID, userID)
	if err != nil {
		if errors.Is(err, domain.ErrNotificationNotFound) {
			response.WriteError(w, sharedErrors.NewNotFoundError("Notification not found", false, nil))
			return
		}
		h.logger.Error().Err(err).Msg("failed to mark notification as read")
		response.WriteError(w, sharedErrors.NewInternalServerError())
		return
	}

	response.WriteSuccess(w, http.StatusOK, map[string]interface{}{
		"id":      notifID,
		"is_read": true,
	})
}
