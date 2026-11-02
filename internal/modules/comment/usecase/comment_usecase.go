package usecase

import (
	"context"
	"strings"
	"time"

	"github.com/anjal-niraula/engage-engine/internal/modules/comment/domain"
	"github.com/anjal-niraula/engage-engine/internal/shared/pagination"
	"github.com/google/uuid"
)

type UseCase interface {
	AddComment(ctx context.Context, input CreateCommentInput) (*CommentOutput, error)
	ListComments(ctx context.Context, postID uuid.UUID, cursor string, limit int) ([]CommentOutput, string, bool, error)
	ListReplies(ctx context.Context, parentID uuid.UUID, cursor string, limit int) ([]CommentOutput, string, bool, error)
	DeleteComment(ctx context.Context, commentID uuid.UUID, requestingUserID uuid.UUID) error
}

type commentUseCase struct {
	repo domain.CommentRepository
}

func NewCommentUseCase(repo domain.CommentRepository) UseCase {
	return &commentUseCase{repo: repo}
}

func (u *commentUseCase) AddComment(ctx context.Context, input CreateCommentInput) (*CommentOutput, error) {
	input.Content = strings.TrimSpace(input.Content)
	if input.Content == "" {
		return nil, domain.ErrCommentNotFound
	}

	if input.ParentID != nil {
		parent, err := u.repo.GetByID(ctx, *input.ParentID)
		if err != nil {
			return nil, err
		}
		if parent == nil {
			return nil, domain.ErrParentNotFound
		}
		if parent.PostID != input.PostID {
			return nil, domain.ErrMismatchedParentPost
		}
	}

	now := time.Now().UTC()
	comment := &domain.Comment{
		ID:        uuid.New(),
		PostID:    input.PostID,
		AuthorID:  input.AuthorID,
		ParentID:  input.ParentID,
		Content:   input.Content,
		CreatedAt: now,
		UpdatedAt: now,
	}

	if err := u.repo.Create(ctx, comment); err != nil {
		return nil, err
	}

	author, err := u.repo.GetAuthor(ctx, input.AuthorID)
	if err != nil {
		return nil, err
	}

	return &CommentOutput{
		ID:           comment.ID,
		PostID:       comment.PostID,
		ParentID:     comment.ParentID,
		Content:      comment.Content,
		LikesCount:   0,
		RepliesCount: 0,
		CreatedAt:    comment.CreatedAt,
		Author:       *author,
	}, nil
}

func (u *commentUseCase) ListComments(ctx context.Context, postID uuid.UUID, cursor string, limit int) ([]CommentOutput, string, bool, error) {
	if limit <= 0 || limit > 50 {
		limit = 20
	}

	var curTime time.Time
	var curID uuid.UUID
	if cursor != "" {
		c, err := pagination.DecodeCursor(cursor)
		if err == nil && c != nil {
			curTime = c.CreatedAt
			if parsed, err := uuid.Parse(c.ID); err == nil {
				curID = parsed
			}
		}
	}

	items, hasMore, err := u.repo.ListByPost(ctx, postID, curTime, curID, limit)
	if err != nil {
		return nil, "", false, err
	}

	outputs := make([]CommentOutput, 0, len(items))
	for _, it := range items {
		outputs = append(outputs, CommentOutput{
			ID:           it.Comment.ID,
			PostID:       it.Comment.PostID,
			ParentID:     it.Comment.ParentID,
			Content:      it.Comment.Content,
			LikesCount:   it.Comment.LikesCount,
			RepliesCount: it.Comment.RepliesCount,
			CreatedAt:    it.Comment.CreatedAt,
			Author:       it.Author,
		})
	}

	var nextCursor string
	if len(items) > 0 && hasMore {
		last := items[len(items)-1]
		nextCursor = pagination.EncodeCursor(last.Comment.ID.String(), last.Comment.CreatedAt)
	}

	return outputs, nextCursor, hasMore, nil
}

func (u *commentUseCase) ListReplies(ctx context.Context, parentID uuid.UUID, cursor string, limit int) ([]CommentOutput, string, bool, error) {
	if limit <= 0 || limit > 50 {
		limit = 10
	}

	parent, err := u.repo.GetByID(ctx, parentID)
	if err != nil {
		return nil, "", false, err
	}
	if parent == nil {
		return nil, "", false, domain.ErrParentNotFound
	}

	var curTime time.Time
	var curID uuid.UUID
	if cursor != "" {
		c, err := pagination.DecodeCursor(cursor)
		if err == nil && c != nil {
			curTime = c.CreatedAt
			if parsed, err := uuid.Parse(c.ID); err == nil {
				curID = parsed
			}
		}
	}

	items, hasMore, err := u.repo.ListReplies(ctx, parentID, curTime, curID, limit)
	if err != nil {
		return nil, "", false, err
	}

	outputs := make([]CommentOutput, 0, len(items))
	for _, it := range items {
		outputs = append(outputs, CommentOutput{
			ID:           it.Comment.ID,
			PostID:       it.Comment.PostID,
			ParentID:     it.Comment.ParentID,
			Content:      it.Comment.Content,
			LikesCount:   it.Comment.LikesCount,
			RepliesCount: it.Comment.RepliesCount,
			CreatedAt:    it.Comment.CreatedAt,
			Author:       it.Author,
		})
	}

	var nextCursor string
	if len(items) > 0 && hasMore {
		last := items[len(items)-1]
		nextCursor = pagination.EncodeCursor(last.Comment.ID.String(), last.Comment.CreatedAt)
	}

	return outputs, nextCursor, hasMore, nil
}

func (u *commentUseCase) DeleteComment(ctx context.Context, commentID uuid.UUID, requestingUserID uuid.UUID) error {
	comment, err := u.repo.GetByID(ctx, commentID)
	if err != nil {
		return err
	}
	if comment == nil {
		return domain.ErrCommentNotFound
	}

	postOwnerID, err := u.repo.GetPostOwnerID(ctx, comment.PostID)
	if err != nil {
		return err
	}

	if requestingUserID != comment.AuthorID && requestingUserID != postOwnerID {
		return domain.ErrForbiddenCommentAction
	}

	return u.repo.SoftDelete(ctx, comment)
}
