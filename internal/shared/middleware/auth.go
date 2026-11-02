package middleware

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	sharedErrors "github.com/anjal-niraula/engage-engine/internal/shared/errors"
	"github.com/anjal-niraula/engage-engine/internal/shared/response"
)

type contextKey string

const (
	userIDKey contextKey = "user_id"
)

// UserClaims defines JWT payload for authentication.
type UserClaims struct {
	UserID string `json:"user_id"`
	jwt.RegisteredClaims
}

// GenerateToken issues a signed JWT for the specified user ID.
func GenerateToken(userID string, secret string, ttl time.Duration) (string, error) {
	if secret == "" {
		return "", errors.New("JWT secret key cannot be empty")
	}

	claims := UserClaims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID,
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(ttl)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

// ValidateToken verifies the signature and expiration of a JWT token string.
func ValidateToken(tokenStr string, secret string) (string, error) {
	if secret == "" {
		return "", errors.New("JWT secret key cannot be empty")
	}

	token, err := jwt.ParseWithClaims(tokenStr, &UserClaims{}, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return []byte(secret), nil
	})

	if err != nil {
		return "", err
	}

	if claims, ok := token.Claims.(*UserClaims); ok && token.Valid {
		return claims.UserID, nil
	}

	return "", errors.New("invalid token claims")
}

// WithUserID injects a user ID into context.
func WithUserID(ctx context.Context, userID string) context.Context {
	return context.WithValue(ctx, userIDKey, userID)
}

// GetUserID retrieves the authenticated user ID from context.
func GetUserID(ctx context.Context) (string, bool) {
	val, ok := ctx.Value(userIDKey).(string)
	return val, ok && val != ""
}

// RequireAuth enforces a valid Bearer JWT on protected endpoints.
func RequireAuth(secret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				response.WriteError(w, sharedErrors.NewUnauthorizedError("Authentication required", false))
				return
			}

			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
				response.WriteError(w, sharedErrors.NewUnauthorizedError("Malformed authorization header, expected Bearer <token>", false))
				return
			}

			userID, err := ValidateToken(parts[1], secret)
			if err != nil {
				response.WriteError(w, sharedErrors.NewUnauthorizedError("Invalid or expired authentication token", false))
				return
			}

			ctx := WithUserID(r.Context(), userID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// OptionalAuth extracts a Bearer JWT if provided, allowing anonymous requests through if absent.
func OptionalAuth(secret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				next.ServeHTTP(w, r)
				return
			}

			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) == 2 && strings.EqualFold(parts[0], "Bearer") {
				if userID, err := ValidateToken(parts[1], secret); err == nil {
					ctx := WithUserID(r.Context(), userID)
					next.ServeHTTP(w, r.WithContext(ctx))
					return
				}
			}

			next.ServeHTTP(w, r)
		})
	}
}

