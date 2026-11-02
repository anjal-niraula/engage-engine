package handler

import (
	"errors"
	"net/http"

	"github.com/anjal-niraula/engage-engine/internal/modules/comment/domain"
	"github.com/anjal-niraula/engage-engine/internal/modules/comment/usecase"
	sharedErrors "github.com/anjal-niraula/engage-engine/internal/shared/errors"
	"github.com/anjal-niraula/engage-engine/internal/shared/middleware"
	"github.com/anjal-niraula/engage-engine/internal/shared/pagination"
	"github.com/anjal-niraula/engage-engine/internal/shared/response"
	"github.com/anjal-niraula/engage-engine/internal/shared/validation"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

type CommentHandler struct {
	uc     usecase.UseCase
	logger *zerolog.Logger
}

func NewCommentHandler(uc usecase.UseCase, logger *zerolog.Logger) *CommentHandler {
	return &CommentHandler{uc: uc, logger: logger}
}

func (h *CommentHandler) CreateComment(w http.ResponseWriter, r *http.Request) {
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

	var req CreateCommentRequest
	if err := validation.DecodeAndValidate(r, &req); err != nil {
		response.WriteError(w, err)
		return
	}

	var parentID *uuid.UUID
	if req.ParentID != nil && *req.ParentID != "" {
		if !validation.IsValidUUID(*req.ParentID) {
			response.WriteError(w, sharedErrors.NewBadRequestError("Invalid parent comment ID, UUID expected", false, nil, nil, nil))
			return
		}
		pID, err := uuid.Parse(*req.ParentID)
		if err != nil {
			response.WriteError(w, sharedErrors.NewBadRequestError("Invalid parent comment ID, UUID expected", false, nil, nil, nil))
			return
		}
		parentID = &pID
	}

	out, err := h.uc.AddComment(r.Context(), usecase.CreateCommentInput{
		PostID:   postID,
		AuthorID: userID,
		ParentID: parentID,
		Content:  req.Content,
	})
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrPostNotFound):
			response.WriteError(w, sharedErrors.NewNotFoundError("Post not found", false, nil))
		case errors.Is(err, domain.ErrParentNotFound):
			response.WriteError(w, sharedErrors.NewNotFoundError("Parent comment not found", false, nil))
		case errors.Is(err, domain.ErrMismatchedParentPost):
			response.WriteError(w, sharedErrors.NewBadRequestError("Parent comment belongs to a different post", false, nil, nil, nil))
		default:
			h.logger.Error().Err(err).Msg("failed to create comment")
			response.WriteError(w, sharedErrors.NewInternalServerError())
		}
		return
	}

	response.WriteSuccess(w, http.StatusCreated, out)
}

func (h *CommentHandler) ListComments(w http.ResponseWriter, r *http.Request) {
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

	limit, cursor, err := pagination.ParsePaginationParams(r, 20, 50)
	if err != nil {
		response.WriteError(w, sharedErrors.NewBadRequestError(err.Error(), false, nil, nil, nil))
		return
	}

	comments, nextCursor, hasMore, err := h.uc.ListComments(r.Context(), postID, cursor, limit)
	if err != nil {
		h.logger.Error().Err(err).Msg("failed to list comments")
		response.WriteError(w, sharedErrors.NewInternalServerError())
		return
	}

	var nextCursorPtr *string
	if nextCursor != "" {
		nextCursorPtr = &nextCursor
	}

	response.WritePaginated(w, http.StatusOK, comments, nextCursorPtr, hasMore, limit)
}

func (h *CommentHandler) ListReplies(w http.ResponseWriter, r *http.Request) {
	commentIDStr := r.PathValue("id")
	if !validation.IsValidUUID(commentIDStr) {
		response.WriteError(w, sharedErrors.NewBadRequestError("Invalid comment ID, UUID expected", false, nil, nil, nil))
		return
	}
	commentID, err := uuid.Parse(commentIDStr)
	if err != nil {
		response.WriteError(w, sharedErrors.NewBadRequestError("Invalid comment ID, UUID expected", false, nil, nil, nil))
		return
	}

	limit, cursor, err := pagination.ParsePaginationParams(r, 10, 50)
	if err != nil {
		response.WriteError(w, sharedErrors.NewBadRequestError(err.Error(), false, nil, nil, nil))
		return
	}

	replies, nextCursor, hasMore, err := h.uc.ListReplies(r.Context(), commentID, cursor, limit)
	if err != nil {
		if errors.Is(err, domain.ErrParentNotFound) {
			response.WriteError(w, sharedErrors.NewNotFoundError("Comment not found", false, nil))
			return
		}
		h.logger.Error().Err(err).Msg("failed to list replies")
		response.WriteError(w, sharedErrors.NewInternalServerError())
		return
	}

	var nextCursorPtr *string
	if nextCursor != "" {
		nextCursorPtr = &nextCursor
	}

	response.WritePaginated(w, http.StatusOK, replies, nextCursorPtr, hasMore, limit)
}

func (h *CommentHandler) DeleteComment(w http.ResponseWriter, r *http.Request) {
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

	commentIDStr := r.PathValue("id")
	if !validation.IsValidUUID(commentIDStr) {
		response.WriteError(w, sharedErrors.NewBadRequestError("Invalid comment ID, UUID expected", false, nil, nil, nil))
		return
	}
	commentID, err := uuid.Parse(commentIDStr)
	if err != nil {
		response.WriteError(w, sharedErrors.NewBadRequestError("Invalid comment ID, UUID expected", false, nil, nil, nil))
		return
	}

	err = h.uc.DeleteComment(r.Context(), commentID, userID)
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrCommentNotFound):
			response.WriteError(w, sharedErrors.NewNotFoundError("Comment not found", false, nil))
		case errors.Is(err, domain.ErrForbiddenCommentAction):
			response.WriteError(w, sharedErrors.NewForbiddenError("You are not authorized to delete this comment", false))
		default:
			h.logger.Error().Err(err).Msg("failed to delete comment")
			response.WriteError(w, sharedErrors.NewInternalServerError())
		}
		return
	}

	response.WriteNoContent(w)
}
