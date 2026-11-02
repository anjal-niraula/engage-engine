package usecase

import (
	"context"
	"strings"
	"time"

	"github.com/anjal-niraula/engage-engine/internal/modules/post/domain"
	"github.com/anjal-niraula/engage-engine/internal/shared/pagination"
)

// PostUseCase defines application operations for post creation, retrieval, and feeds.
type PostUseCase interface {
	CreatePost(ctx context.Context, authorID string, input CreatePostInput) (*PostOutput, error)
	GetPost(ctx context.Context, postID string, viewerID string) (*PostOutput, error)
	DeletePost(ctx context.Context, postID string, authorID string) error
	ListUserPosts(ctx context.Context, username string, limit int, cursorStr string) ([]*PostOutput, *string, bool, error)
	GetFeed(ctx context.Context, userID string, limit int, cursorStr string) ([]*PostOutput, *string, bool, error)
}

type postUseCase struct {
	postRepo domain.PostRepository
}

// NewPostUseCase constructs an instance of PostUseCase.
func NewPostUseCase(postRepo domain.PostRepository) PostUseCase {
	return &postUseCase{postRepo: postRepo}
}

func (uc *postUseCase) CreatePost(ctx context.Context, authorID string, input CreatePostInput) (*PostOutput, error) {
	post := domain.NewPost(authorID, input.Caption, input.MediaURLs, input.MediaType, input.Location)
	if err := uc.postRepo.Create(ctx, post); err != nil {
		return nil, err
	}

	authors, _ := uc.postRepo.BatchGetAuthors(ctx, []string{authorID})
	var authorOut *AuthorOutput
	if author, ok := authors[authorID]; ok {
		authorOut = &AuthorOutput{
			ID:          author.ID,
			Username:    author.Username,
			DisplayName: author.DisplayName,
			AvatarURL:   author.AvatarURL,
		}
	}

	return toPostOutput(post, false, authorOut), nil
}

func (uc *postUseCase) GetPost(ctx context.Context, postID string, viewerID string) (*PostOutput, error) {
	post, err := uc.postRepo.GetByID(ctx, postID)
	if err != nil {
		return nil, err
	}

	authors, _ := uc.postRepo.BatchGetAuthors(ctx, []string{post.AuthorID})
	var authorOut *AuthorOutput
	if author, ok := authors[post.AuthorID]; ok {
		authorOut = &AuthorOutput{
			ID:          author.ID,
			Username:    author.Username,
			DisplayName: author.DisplayName,
			AvatarURL:   author.AvatarURL,
		}
	}

	hasLiked := false
	if viewerID != "" {
		likedMap, _ := uc.postRepo.BatchCheckLiked(ctx, viewerID, []string{postID})
		hasLiked = likedMap[postID]
	}

	return toPostOutput(post, hasLiked, authorOut), nil
}

func (uc *postUseCase) DeletePost(ctx context.Context, postID string, authorID string) error {
	return uc.postRepo.Delete(ctx, postID, authorID)
}

func (uc *postUseCase) ListUserPosts(ctx context.Context, username string, limit int, cursorStr string) ([]*PostOutput, *string, bool, error) {
	author, err := uc.postRepo.GetAuthorByUsername(ctx, strings.ToLower(username))
	if err != nil {
		return nil, nil, false, domain.ErrPostNotFound
	}

	var cursorID string
	var cursorTime *time.Time
	if cursorStr != "" {
		decoded, err := pagination.DecodeCursor(cursorStr)
		if err == nil && decoded != nil {
			cursorID = decoded.ID
			cursorTime = &decoded.CreatedAt
		}
	}

	posts, err := uc.postRepo.ListByAuthorID(ctx, author.ID, limit+1, cursorID, cursorTime)
	if err != nil {
		return nil, nil, false, err
	}

	hasMore := len(posts) > limit
	if hasMore {
		posts = posts[:limit]
	}

	var nextCursor *string
	if len(posts) > 0 && hasMore {
		last := posts[len(posts)-1]
		token := pagination.EncodeCursor(last.ID, last.CreatedAt)
		nextCursor = &token
	}

	authorOut := &AuthorOutput{
		ID:          author.ID,
		Username:    author.Username,
		DisplayName: author.DisplayName,
		AvatarURL:   author.AvatarURL,
	}

	output := make([]*PostOutput, 0, len(posts))
	for _, p := range posts {
		output = append(output, toPostOutput(p, false, authorOut))
	}

	return output, nextCursor, hasMore, nil
}

func (uc *postUseCase) GetFeed(ctx context.Context, userID string, limit int, cursorStr string) ([]*PostOutput, *string, bool, error) {
	var cursorID string
	var cursorTime *time.Time
	if cursorStr != "" {
		decoded, err := pagination.DecodeCursor(cursorStr)
		if err == nil && decoded != nil {
			cursorID = decoded.ID
			cursorTime = &decoded.CreatedAt
		}
	}

	posts, err := uc.postRepo.ListFeed(ctx, userID, limit+1, cursorID, cursorTime)
	if err != nil {
		return nil, nil, false, err
	}

	hasMore := len(posts) > limit
	if hasMore {
		posts = posts[:limit]
	}

	var nextCursor *string
	if len(posts) > 0 && hasMore {
		last := posts[len(posts)-1]
		token := pagination.EncodeCursor(last.ID, last.CreatedAt)
		nextCursor = &token
	}

	// Batch hydrate distinct authors and like states (0 N+1 issues)
	authorIDs := make([]string, 0, len(posts))
	postIDs := make([]string, 0, len(posts))
	for _, p := range posts {
		authorIDs = append(authorIDs, p.AuthorID)
		postIDs = append(postIDs, p.ID)
	}

	authorsMap, _ := uc.postRepo.BatchGetAuthors(ctx, authorIDs)
	likedMap, _ := uc.postRepo.BatchCheckLiked(ctx, userID, postIDs)

	output := make([]*PostOutput, 0, len(posts))
	for _, p := range posts {
		var authorOut *AuthorOutput
		if a, ok := authorsMap[p.AuthorID]; ok {
			authorOut = &AuthorOutput{
				ID:          a.ID,
				Username:    a.Username,
				DisplayName: a.DisplayName,
				AvatarURL:   a.AvatarURL,
			}
		}
		output = append(output, toPostOutput(p, likedMap[p.ID], authorOut))
	}

	return output, nextCursor, hasMore, nil
}

func toPostOutput(p *domain.Post, hasLiked bool, author *AuthorOutput) *PostOutput {
	return &PostOutput{
		ID:            p.ID,
		AuthorID:      p.AuthorID,
		Caption:       p.Caption,
		MediaURLs:     p.MediaURLs,
		MediaType:     p.MediaType,
		Location:      p.Location,
		LikesCount:    p.LikesCount,
		CommentsCount: p.CommentsCount,
		ViewsCount:    p.ViewsCount,
		HasLiked:      hasLiked,
		Author:        author,
		CreatedAt:     p.CreatedAt,
		UpdatedAt:     p.UpdatedAt,
	}
}
