package handler

import (
	"errors"
	"net/http"

	"github.com/anjal-niraula/engage-engine/internal/modules/like/domain"
	"github.com/anjal-niraula/engage-engine/internal/modules/like/usecase"
	sharedErrors "github.com/anjal-niraula/engage-engine/internal/shared/errors"
	"github.com/anjal-niraula/engage-engine/internal/shared/middleware"
	"github.com/anjal-niraula/engage-engine/internal/shared/pagination"
	"github.com/anjal-niraula/engage-engine/internal/shared/response"
	"github.com/anjal-niraula/engage-engine/internal/shared/validation"
)

// LikeHandler coordinates HTTP requests for liking and unliking content.
type LikeHandler struct {
	useCase usecase.LikeUseCase
}

// NewLikeHandler creates a new LikeHandler.
func NewLikeHandler(uc usecase.LikeUseCase) *LikeHandler {
	return &LikeHandler{useCase: uc}
}

// LikePost handles POST /api/v1/posts/{id}/likes
func (h *LikeHandler) LikePost(w http.ResponseWriter, r *http.Request) {
	postID := r.PathValue("id")
	if !validation.IsValidUUID(postID) {
		response.WriteError(w, sharedErrors.NewBadRequestError("Invalid post ID, UUID expected", false, nil, nil, nil))
		return
	}

	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		response.WriteError(w, sharedErrors.NewUnauthorizedError("Authentication required", false))
		return
	}

	output, err := h.useCase.LikePost(r.Context(), userID, postID)
	if err != nil {
		if errors.Is(err, domain.ErrTargetNotFound) {
			response.WriteError(w, sharedErrors.NewNotFoundError("Post not found", false, nil))
			return
		}
		response.WriteError(w, sharedErrors.NewInternalServerError())
		return
	}

	response.WriteSuccess(w, http.StatusOK, output)
}

// UnlikePost handles DELETE /api/v1/posts/{id}/likes
func (h *LikeHandler) UnlikePost(w http.ResponseWriter, r *http.Request) {
	postID := r.PathValue("id")
	if !validation.IsValidUUID(postID) {
		response.WriteError(w, sharedErrors.NewBadRequestError("Invalid post ID, UUID expected", false, nil, nil, nil))
		return
	}

	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		response.WriteError(w, sharedErrors.NewUnauthorizedError("Authentication required", false))
		return
	}

	output, err := h.useCase.UnlikePost(r.Context(), userID, postID)
	if err != nil {
		if errors.Is(err, domain.ErrTargetNotFound) {
			response.WriteError(w, sharedErrors.NewNotFoundError("Post not found", false, nil))
			return
		}
		response.WriteError(w, sharedErrors.NewInternalServerError())
		return
	}

	response.WriteSuccess(w, http.StatusOK, output)
}

// ListPostLikes handles GET /api/v1/posts/{id}/likes
func (h *LikeHandler) ListPostLikes(w http.ResponseWriter, r *http.Request) {
	postID := r.PathValue("id")
	if !validation.IsValidUUID(postID) {
		response.WriteError(w, sharedErrors.NewBadRequestError("Invalid post ID, UUID expected", false, nil, nil, nil))
		return
	}

	limit, cursor, err := pagination.ParsePaginationParams(r, 20, 50)
	if err != nil {
		response.WriteError(w, sharedErrors.NewBadRequestError(err.Error(), false, nil, nil, nil))
		return
	}

	likers, nextCursor, hasMore, err := h.useCase.ListPostLikes(r.Context(), postID, limit, cursor)
	if err != nil {
		response.WriteError(w, sharedErrors.NewInternalServerError())
		return
	}

	response.WritePaginated(w, http.StatusOK, likers, nextCursor, hasMore, limit)
}

// LikeComment handles POST /api/v1/comments/{id}/likes
func (h *LikeHandler) LikeComment(w http.ResponseWriter, r *http.Request) {
	commentID := r.PathValue("id")
	if !validation.IsValidUUID(commentID) {
		response.WriteError(w, sharedErrors.NewBadRequestError("Invalid comment ID, UUID expected", false, nil, nil, nil))
		return
	}

	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		response.WriteError(w, sharedErrors.NewUnauthorizedError("Authentication required", false))
		return
	}

	output, err := h.useCase.LikeComment(r.Context(), userID, commentID)
	if err != nil {
		if errors.Is(err, domain.ErrTargetNotFound) {
			response.WriteError(w, sharedErrors.NewNotFoundError("Comment not found", false, nil))
			return
		}
		response.WriteError(w, sharedErrors.NewInternalServerError())
		return
	}

	response.WriteSuccess(w, http.StatusOK, output)
}

// UnlikeComment handles DELETE /api/v1/comments/{id}/likes
func (h *LikeHandler) UnlikeComment(w http.ResponseWriter, r *http.Request) {
	commentID := r.PathValue("id")
	if !validation.IsValidUUID(commentID) {
		response.WriteError(w, sharedErrors.NewBadRequestError("Invalid comment ID, UUID expected", false, nil, nil, nil))
		return
	}

	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		response.WriteError(w, sharedErrors.NewUnauthorizedError("Authentication required", false))
		return
	}

	output, err := h.useCase.UnlikeComment(r.Context(), userID, commentID)
	if err != nil {
		if errors.Is(err, domain.ErrTargetNotFound) {
			response.WriteError(w, sharedErrors.NewNotFoundError("Comment not found", false, nil))
			return
		}
		response.WriteError(w, sharedErrors.NewInternalServerError())
		return
	}

	response.WriteSuccess(w, http.StatusOK, output)
}
