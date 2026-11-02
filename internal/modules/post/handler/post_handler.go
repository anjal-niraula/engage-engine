package handler

import (
	"errors"
	"net/http"

	"github.com/anjal-niraula/engage-engine/internal/modules/post/domain"
	"github.com/anjal-niraula/engage-engine/internal/modules/post/usecase"
	sharedErrors "github.com/anjal-niraula/engage-engine/internal/shared/errors"
	"github.com/anjal-niraula/engage-engine/internal/shared/middleware"
	"github.com/anjal-niraula/engage-engine/internal/shared/pagination"
	"github.com/anjal-niraula/engage-engine/internal/shared/response"
	"github.com/anjal-niraula/engage-engine/internal/shared/validation"
)

// PostHandler handles HTTP transport for post publishing, feeds, and retrieval.
type PostHandler struct {
	useCase usecase.PostUseCase
}

// NewPostHandler constructs a new PostHandler.
func NewPostHandler(uc usecase.PostUseCase) *PostHandler {
	return &PostHandler{useCase: uc}
}

// Create handles POST /api/v1/posts
func (h *PostHandler) Create(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		response.WriteError(w, sharedErrors.NewUnauthorizedError("Authentication required", false))
		return
	}

	var req CreatePostRequest
	if err := validation.DecodeAndValidate(r, &req); err != nil {
		response.WriteError(w, err)
		return
	}

	postOut, err := h.useCase.CreatePost(r.Context(), userID, usecase.CreatePostInput{
		Caption:   req.Caption,
		MediaURLs: req.MediaURLs,
		MediaType: req.MediaType,
		Location:  req.Location,
	})
	if err != nil {
		response.WriteError(w, sharedErrors.NewInternalServerError())
		return
	}

	response.WriteSuccess(w, http.StatusCreated, postOut)
}

// GetByID handles GET /api/v1/posts/{id}
func (h *PostHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if !validation.IsValidUUID(id) {
		response.WriteError(w, sharedErrors.NewBadRequestError("Invalid post ID format, UUID expected", false, nil, nil, nil))
		return
	}

	viewerID, _ := middleware.GetUserID(r.Context())

	postOut, err := h.useCase.GetPost(r.Context(), id, viewerID)
	if err != nil {
		if errors.Is(err, domain.ErrPostNotFound) {
			response.WriteError(w, sharedErrors.NewNotFoundError("Post not found", false, nil))
			return
		}
		response.WriteError(w, sharedErrors.NewInternalServerError())
		return
	}

	response.WriteSuccess(w, http.StatusOK, postOut)
}

// Delete handles DELETE /api/v1/posts/{id}
func (h *PostHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if !validation.IsValidUUID(id) {
		response.WriteError(w, sharedErrors.NewBadRequestError("Invalid post ID format, UUID expected", false, nil, nil, nil))
		return
	}

	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		response.WriteError(w, sharedErrors.NewUnauthorizedError("Authentication required", false))
		return
	}

	err := h.useCase.DeletePost(r.Context(), id, userID)
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrPostNotFound):
			response.WriteError(w, sharedErrors.NewNotFoundError("Post not found", false, nil))
		case errors.Is(err, domain.ErrForbiddenPostAction):
			response.WriteError(w, sharedErrors.NewForbiddenError("You are not authorized to delete this post", false))
		default:
			response.WriteError(w, sharedErrors.NewInternalServerError())
		}
		return
	}

	response.WriteNoContent(w)
}

// ListUserPosts handles GET /api/v1/users/{username}/posts
func (h *PostHandler) ListUserPosts(w http.ResponseWriter, r *http.Request) {
	username := r.PathValue("username")
	if username == "" {
		response.WriteError(w, sharedErrors.NewBadRequestError("Username path parameter is required", false, nil, nil, nil))
		return
	}

	limit, cursor, err := pagination.ParsePaginationParams(r, 12, 50)
	if err != nil {
		response.WriteError(w, sharedErrors.NewBadRequestError(err.Error(), false, nil, nil, nil))
		return
	}

	posts, nextCursor, hasMore, err := h.useCase.ListUserPosts(r.Context(), username, limit, cursor)
	if err != nil {
		if errors.Is(err, domain.ErrPostNotFound) {
			response.WriteError(w, sharedErrors.NewNotFoundError("User not found", false, nil))
			return
		}
		response.WriteError(w, sharedErrors.NewInternalServerError())
		return
	}

	response.WritePaginated(w, http.StatusOK, posts, nextCursor, hasMore, limit)
}

// GetFeed handles GET /api/v1/feed
func (h *PostHandler) GetFeed(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		response.WriteError(w, sharedErrors.NewUnauthorizedError("Authentication required", false))
		return
	}

	limit, cursor, err := pagination.ParsePaginationParams(r, 10, 30)
	if err != nil {
		response.WriteError(w, sharedErrors.NewBadRequestError(err.Error(), false, nil, nil, nil))
		return
	}

	posts, nextCursor, hasMore, err := h.useCase.GetFeed(r.Context(), userID, limit, cursor)
	if err != nil {
		response.WriteError(w, sharedErrors.NewInternalServerError())
		return
	}

	response.WritePaginated(w, http.StatusOK, posts, nextCursor, hasMore, limit)
}
