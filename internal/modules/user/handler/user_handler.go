package handler

import (
	"errors"
	"net/http"

	"github.com/anjal-niraula/engage-engine/internal/modules/user/domain"
	"github.com/anjal-niraula/engage-engine/internal/modules/user/usecase"
	sharedErrors "github.com/anjal-niraula/engage-engine/internal/shared/errors"
	"github.com/anjal-niraula/engage-engine/internal/shared/middleware"
	"github.com/anjal-niraula/engage-engine/internal/shared/response"
	"github.com/anjal-niraula/engage-engine/internal/shared/validation"
)

// UserHandler handles HTTP transport requests for user operations.
type UserHandler struct {
	useCase usecase.UserUseCase
}

// NewUserHandler creates a new UserHandler instance.
func NewUserHandler(uc usecase.UserUseCase) *UserHandler {
	return &UserHandler{useCase: uc}
}

// Register handles POST /api/v1/users/register
func (h *UserHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req RegisterRequest
	if err := validation.DecodeAndValidate(r, &req); err != nil {
		response.WriteError(w, err)
		return
	}

	userOutput, err := h.useCase.Register(r.Context(), usecase.RegisterInput{
		Username:    req.Username,
		Email:       req.Email,
		Password:    req.Password,
		DisplayName: req.DisplayName,
	})
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrUsernameTaken):
			response.WriteError(w, sharedErrors.NewConflictError("Username is already taken", false, nil))
		case errors.Is(err, domain.ErrEmailTaken):
			response.WriteError(w, sharedErrors.NewConflictError("Email address is already in use", false, nil))
		default:
			response.WriteError(w, sharedErrors.NewInternalServerError())
		}
		return
	}

	response.WriteSuccess(w, http.StatusCreated, userOutput)
}

// Login handles POST /api/v1/users/login
func (h *UserHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := validation.DecodeAndValidate(r, &req); err != nil {
		response.WriteError(w, err)
		return
	}

	loginOutput, err := h.useCase.Login(r.Context(), usecase.LoginInput{
		Identity: req.Identity,
		Password: req.Password,
	})
	if err != nil {
		if errors.Is(err, domain.ErrInvalidCredentials) {
			response.WriteError(w, sharedErrors.NewUnauthorizedError("Invalid email/username or password", false))
			return
		}
		response.WriteError(w, sharedErrors.NewInternalServerError())
		return
	}

	response.WriteSuccess(w, http.StatusOK, loginOutput)
}

// GetMe handles GET /api/v1/users/me
func (h *UserHandler) GetMe(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		response.WriteError(w, sharedErrors.NewUnauthorizedError("Authentication required", false))
		return
	}

	userOutput, err := h.useCase.GetProfile(r.Context(), userID)
	if err != nil {
		if errors.Is(err, domain.ErrUserNotFound) {
			response.WriteError(w, sharedErrors.NewNotFoundError("User profile not found", false, nil))
			return
		}
		response.WriteError(w, sharedErrors.NewInternalServerError())
		return
	}

	response.WriteSuccess(w, http.StatusOK, userOutput)
}

// UpdateMe handles PATCH /api/v1/users/me
func (h *UserHandler) UpdateMe(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		response.WriteError(w, sharedErrors.NewUnauthorizedError("Authentication required", false))
		return
	}

	var req UpdateProfileRequest
	if err := validation.DecodeAndValidate(r, &req); err != nil {
		response.WriteError(w, err)
		return
	}

	userOutput, err := h.useCase.UpdateProfile(r.Context(), userID, usecase.UpdateProfileInput{
		DisplayName: req.DisplayName,
		Bio:         req.Bio,
		AvatarURL:   req.AvatarURL,
		Website:     req.Website,
	})
	if err != nil {
		if errors.Is(err, domain.ErrUserNotFound) {
			response.WriteError(w, sharedErrors.NewNotFoundError("User profile not found", false, nil))
			return
		}
		response.WriteError(w, sharedErrors.NewInternalServerError())
		return
	}

	response.WriteSuccess(w, http.StatusOK, userOutput)
}

// GetByUsername handles GET /api/v1/users/{username}
func (h *UserHandler) GetByUsername(w http.ResponseWriter, r *http.Request) {
	username := r.PathValue("username")
	if username == "" {
		response.WriteError(w, sharedErrors.NewBadRequestError("Username path parameter is required", false, nil, nil, nil))
		return
	}

	viewerID, _ := middleware.GetUserID(r.Context())

	profile, err := h.useCase.GetPublicProfile(r.Context(), username, viewerID)
	if err != nil {
		if errors.Is(err, domain.ErrUserNotFound) {
			response.WriteError(w, sharedErrors.NewNotFoundError("User not found", false, nil))
			return
		}
		response.WriteError(w, sharedErrors.NewInternalServerError())
		return
	}

	response.WriteSuccess(w, http.StatusOK, profile)
}

