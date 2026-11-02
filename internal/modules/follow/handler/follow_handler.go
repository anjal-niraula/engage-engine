package handler

import (
	"errors"
	"net/http"

	"github.com/anjal-niraula/engage-engine/internal/modules/follow/domain"
	"github.com/anjal-niraula/engage-engine/internal/modules/follow/usecase"
	sharedErrors "github.com/anjal-niraula/engage-engine/internal/shared/errors"
	"github.com/anjal-niraula/engage-engine/internal/shared/middleware"
	"github.com/anjal-niraula/engage-engine/internal/shared/pagination"
	"github.com/anjal-niraula/engage-engine/internal/shared/response"
	"github.com/anjal-niraula/engage-engine/internal/shared/validation"
)

// FollowHandler coordinates HTTP delivery for follower and following relationships.
type FollowHandler struct {
	useCase usecase.FollowUseCase
}

// NewFollowHandler constructs a new FollowHandler.
func NewFollowHandler(uc usecase.FollowUseCase) *FollowHandler {
	return &FollowHandler{useCase: uc}
}

// Follow handles POST /api/v1/users/{id}/follow
func (h *FollowHandler) Follow(w http.ResponseWriter, r *http.Request) {
	targetUserID := r.PathValue("id")
	if !validation.IsValidUUID(targetUserID) {
		response.WriteError(w, sharedErrors.NewBadRequestError("Invalid user ID, UUID expected", false, nil, nil, nil))
		return
	}

	followerID, ok := middleware.GetUserID(r.Context())
	if !ok {
		response.WriteError(w, sharedErrors.NewUnauthorizedError("Authentication required", false))
		return
	}

	output, err := h.useCase.Follow(r.Context(), followerID, targetUserID)
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrCannotFollowSelf):
			response.WriteError(w, sharedErrors.NewBadRequestError("You cannot follow yourself", false, nil, nil, nil))
		case errors.Is(err, domain.ErrTargetUserNotFound):
			response.WriteError(w, sharedErrors.NewNotFoundError("User not found", false, nil))
		default:
			response.WriteError(w, sharedErrors.NewInternalServerError())
		}
		return
	}

	response.WriteSuccess(w, http.StatusOK, output)
}

// Unfollow handles DELETE /api/v1/users/{id}/follow
func (h *FollowHandler) Unfollow(w http.ResponseWriter, r *http.Request) {
	targetUserID := r.PathValue("id")
	if !validation.IsValidUUID(targetUserID) {
		response.WriteError(w, sharedErrors.NewBadRequestError("Invalid user ID, UUID expected", false, nil, nil, nil))
		return
	}

	followerID, ok := middleware.GetUserID(r.Context())
	if !ok {
		response.WriteError(w, sharedErrors.NewUnauthorizedError("Authentication required", false))
		return
	}

	output, err := h.useCase.Unfollow(r.Context(), followerID, targetUserID)
	if err != nil {
		if errors.Is(err, domain.ErrCannotFollowSelf) {
			response.WriteError(w, sharedErrors.NewBadRequestError("You cannot unfollow yourself", false, nil, nil, nil))
			return
		}
		response.WriteError(w, sharedErrors.NewInternalServerError())
		return
	}

	response.WriteSuccess(w, http.StatusOK, output)
}

// ListFollowers handles GET /api/v1/users/{id}/followers
func (h *FollowHandler) ListFollowers(w http.ResponseWriter, r *http.Request) {
	targetUserID := r.PathValue("id")
	if !validation.IsValidUUID(targetUserID) {
		response.WriteError(w, sharedErrors.NewBadRequestError("Invalid user ID, UUID expected", false, nil, nil, nil))
		return
	}

	limit, cursor, err := pagination.ParsePaginationParams(r, 20, 50)
	if err != nil {
		response.WriteError(w, sharedErrors.NewBadRequestError(err.Error(), false, nil, nil, nil))
		return
	}

	users, nextCursor, hasMore, err := h.useCase.ListFollowers(r.Context(), targetUserID, limit, cursor)
	if err != nil {
		response.WriteError(w, sharedErrors.NewInternalServerError())
		return
	}

	response.WritePaginated(w, http.StatusOK, users, nextCursor, hasMore, limit)
}

// ListFollowing handles GET /api/v1/users/{id}/following
func (h *FollowHandler) ListFollowing(w http.ResponseWriter, r *http.Request) {
	targetUserID := r.PathValue("id")
	if !validation.IsValidUUID(targetUserID) {
		response.WriteError(w, sharedErrors.NewBadRequestError("Invalid user ID, UUID expected", false, nil, nil, nil))
		return
	}

	limit, cursor, err := pagination.ParsePaginationParams(r, 20, 50)
	if err != nil {
		response.WriteError(w, sharedErrors.NewBadRequestError(err.Error(), false, nil, nil, nil))
		return
	}

	users, nextCursor, hasMore, err := h.useCase.ListFollowing(r.Context(), targetUserID, limit, cursor)
	if err != nil {
		response.WriteError(w, sharedErrors.NewInternalServerError())
		return
	}

	response.WritePaginated(w, http.StatusOK, users, nextCursor, hasMore, limit)
}
