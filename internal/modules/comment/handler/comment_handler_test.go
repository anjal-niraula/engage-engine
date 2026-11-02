package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/anjal-niraula/engage-engine/internal/modules/comment/domain"
	"github.com/anjal-niraula/engage-engine/internal/modules/comment/handler"
	"github.com/anjal-niraula/engage-engine/internal/modules/comment/usecase"
	"github.com/anjal-niraula/engage-engine/internal/shared/middleware"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type mockCommentUseCase struct {
	mock.Mock
}

func (m *mockCommentUseCase) AddComment(ctx context.Context, input usecase.CreateCommentInput) (*usecase.CommentOutput, error) {
	args := m.Called(ctx, input)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*usecase.CommentOutput), args.Error(1)
}

func (m *mockCommentUseCase) ListComments(ctx context.Context, postID uuid.UUID, cursor string, limit int) ([]usecase.CommentOutput, string, bool, error) {
	args := m.Called(ctx, postID, cursor, limit)
	return args.Get(0).([]usecase.CommentOutput), args.String(1), args.Bool(2), args.Error(3)
}

func (m *mockCommentUseCase) ListReplies(ctx context.Context, parentID uuid.UUID, cursor string, limit int) ([]usecase.CommentOutput, string, bool, error) {
	args := m.Called(ctx, parentID, cursor, limit)
	return args.Get(0).([]usecase.CommentOutput), args.String(1), args.Bool(2), args.Error(3)
}

func (m *mockCommentUseCase) DeleteComment(ctx context.Context, commentID uuid.UUID, requestingUserID uuid.UUID) error {
	args := m.Called(ctx, commentID, requestingUserID)
	return args.Error(0)
}

func TestCommentHandler_Endpoints(t *testing.T) {
	logger := zerolog.Nop()
	uc := new(mockCommentUseCase)
	h := handler.NewCommentHandler(uc, &logger)

	tokenSecret := "test-secret-at-least-32-chars-long!"
	userID := uuid.New()
	token, _ := middleware.GenerateToken(userID.String(), tokenSecret, time.Hour)

	mux := http.NewServeMux()
	handler.RegisterRoutes(mux, h, middleware.RequireAuth(tokenSecret), middleware.OptionalAuth(tokenSecret))

	postID := uuid.New()

	t.Run("CreateComment", func(t *testing.T) {
		reqBody := `{"content": "Awesome picture!"}`
		req := httptest.NewRequest(http.MethodPost, "/api/v1/posts/"+postID.String()+"/comments", bytes.NewBufferString(reqBody))
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		uc.On("AddComment", mock.Anything, mock.MatchedBy(func(in usecase.CreateCommentInput) bool {
			return in.PostID == postID && in.Content == "Awesome picture!" && in.AuthorID == userID
		})).Return(&usecase.CommentOutput{
			ID:        uuid.New(),
			PostID:    postID,
			Content:   "Awesome picture!",
			CreatedAt: time.Now().UTC(),
			Author: domain.CommentAuthor{
				Username: "testuser",
			},
		}, nil).Once()

		mux.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusCreated, rec.Code)

		var resp map[string]interface{}
		err := json.NewDecoder(rec.Body).Decode(&resp)
		assert.NoError(t, err)
		assert.True(t, resp["success"].(bool))
	})

	t.Run("ListComments", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/posts/"+postID.String()+"/comments", nil)
		rec := httptest.NewRecorder()

		uc.On("ListComments", mock.Anything, postID, "", 20).Return([]usecase.CommentOutput{}, "", false, nil).Once()

		mux.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusOK, rec.Code)
	})

	t.Run("DeleteComment", func(t *testing.T) {
		commentID := uuid.New()
		req := httptest.NewRequest(http.MethodDelete, "/api/v1/comments/"+commentID.String(), nil)
		req.Header.Set("Authorization", "Bearer "+token)
		rec := httptest.NewRecorder()

		uc.On("DeleteComment", mock.Anything, commentID, userID).Return(nil).Once()

		mux.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusNoContent, rec.Code)
	})
}
