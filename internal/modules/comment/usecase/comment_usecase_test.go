package usecase_test

import (
	"context"
	"testing"
	"time"

	"github.com/anjal-niraula/engage-engine/internal/modules/comment/domain"
	"github.com/anjal-niraula/engage-engine/internal/modules/comment/usecase"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type mockCommentRepo struct {
	mock.Mock
}

func (m *mockCommentRepo) Create(ctx context.Context, comment *domain.Comment) error {
	args := m.Called(ctx, comment)
	return args.Error(0)
}

func (m *mockCommentRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.Comment, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Comment), args.Error(1)
}

func (m *mockCommentRepo) GetPostOwnerID(ctx context.Context, postID uuid.UUID) (uuid.UUID, error) {
	args := m.Called(ctx, postID)
	return args.Get(0).(uuid.UUID), args.Error(1)
}

func (m *mockCommentRepo) GetAuthor(ctx context.Context, userID uuid.UUID) (*domain.CommentAuthor, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.CommentAuthor), args.Error(1)
}

func (m *mockCommentRepo) ListByPost(ctx context.Context, postID uuid.UUID, cursorTime time.Time, cursorID uuid.UUID, limit int) ([]domain.CommentWithAuthor, bool, error) {
	args := m.Called(ctx, postID, cursorTime, cursorID, limit)
	return args.Get(0).([]domain.CommentWithAuthor), args.Bool(1), args.Error(2)
}

func (m *mockCommentRepo) ListReplies(ctx context.Context, parentID uuid.UUID, cursorTime time.Time, cursorID uuid.UUID, limit int) ([]domain.CommentWithAuthor, bool, error) {
	args := m.Called(ctx, parentID, cursorTime, cursorID, limit)
	return args.Get(0).([]domain.CommentWithAuthor), args.Bool(1), args.Error(2)
}

func (m *mockCommentRepo) SoftDelete(ctx context.Context, comment *domain.Comment) error {
	args := m.Called(ctx, comment)
	return args.Error(0)
}

func TestCommentUseCase_AddComment(t *testing.T) {
	repo := new(mockCommentRepo)
	uc := usecase.NewCommentUseCase(repo)

	postID := uuid.New()
	authorID := uuid.New()

	author := &domain.CommentAuthor{
		ID:          authorID,
		Username:    "alice",
		DisplayName: "Alice",
	}

	repo.On("Create", mock.Anything, mock.MatchedBy(func(c *domain.Comment) bool {
		return c.PostID == postID && c.AuthorID == authorID && c.Content == "Great post!"
	})).Return(nil)
	repo.On("GetAuthor", mock.Anything, authorID).Return(author, nil)

	out, err := uc.AddComment(context.Background(), usecase.CreateCommentInput{
		PostID:   postID,
		AuthorID: authorID,
		Content:  "Great post!",
	})

	assert.NoError(t, err)
	assert.NotNil(t, out)
	assert.Equal(t, "Great post!", out.Content)
	assert.Equal(t, author.Username, out.Author.Username)
	repo.AssertExpectations(t)
}

func TestCommentUseCase_DeleteComment_Authorization(t *testing.T) {
	repo := new(mockCommentRepo)
	uc := usecase.NewCommentUseCase(repo)

	postID := uuid.New()
	postOwnerID := uuid.New()
	commentAuthorID := uuid.New()
	randomUserID := uuid.New()
	commentID := uuid.New()

	comment := &domain.Comment{
		ID:       commentID,
		PostID:   postID,
		AuthorID: commentAuthorID,
	}

	repo.On("GetByID", mock.Anything, commentID).Return(comment, nil)
	repo.On("GetPostOwnerID", mock.Anything, postID).Return(postOwnerID, nil)

	// Random user cannot delete
	err := uc.DeleteComment(context.Background(), commentID, randomUserID)
	assert.ErrorIs(t, err, domain.ErrForbiddenCommentAction)

	// Comment author can delete
	repo.On("SoftDelete", mock.Anything, comment).Return(nil).Once()
	err = uc.DeleteComment(context.Background(), commentID, commentAuthorID)
	assert.NoError(t, err)

	// Post owner can delete
	repo.On("SoftDelete", mock.Anything, comment).Return(nil).Once()
	err = uc.DeleteComment(context.Background(), commentID, postOwnerID)
	assert.NoError(t, err)

	repo.AssertExpectations(t)
}
