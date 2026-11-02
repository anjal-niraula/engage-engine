package handler

// RegisterRequest holds validated fields for user registration.
type RegisterRequest struct {
	Username    string `json:"username" validate:"required,min=3,max=30,alphanum"`
	Email       string `json:"email" validate:"required,email,max=255"`
	Password    string `json:"password" validate:"required,min=8,max=72"`
	DisplayName string `json:"display_name" validate:"required,min=1,max=100"`
}

// LoginRequest holds credentials for user login.
type LoginRequest struct {
	Identity string `json:"identity" validate:"required"`
	Password string `json:"password" validate:"required"`
}

// UpdateProfileRequest holds optional fields for partial profile updates.
type UpdateProfileRequest struct {
	DisplayName *string `json:"display_name" validate:"omitempty,min=1,max=100"`
	Bio         *string `json:"bio" validate:"omitempty,max=500"`
	AvatarURL   *string `json:"avatar_url" validate:"omitempty,url,max=512"`
	Website     *string `json:"website" validate:"omitempty,url,max=255"`
}

