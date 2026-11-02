package repository

import (
	"context"
	"errors"
	"time"

	"github.com/anjal-niraula/engage-engine/internal/modules/comment/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresCommentRepo struct {
	pool *pgxpool.Pool
}

func NewPostgresCommentRepo(pool *pgxpool.Pool) *PostgresCommentRepo {
	return &PostgresCommentRepo{pool: pool}
}

func (r *PostgresCommentRepo) Create(ctx context.Context, comment *domain.Comment) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	// Check that post exists and is not deleted
	var postExists bool
	err = tx.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM posts WHERE id = $1 AND deleted_at IS NULL)", comment.PostID).Scan(&postExists)
	if err != nil {
		return err
	}
	if !postExists {
		return domain.ErrPostNotFound
	}

	query := `
		INSERT INTO comments (id, post_id, author_id, parent_id, content, likes_count, replies_count, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`
	_, err = tx.Exec(ctx, query,
		comment.ID,
		comment.PostID,
		comment.AuthorID,
		comment.ParentID,
		comment.Content,
		comment.LikesCount,
		comment.RepliesCount,
		comment.CreatedAt,
		comment.UpdatedAt,
	)
	if err != nil {
		return err
	}

	_, err = tx.Exec(ctx, `UPDATE posts SET comments_count = comments_count + 1 WHERE id = $1`, comment.PostID)
	if err != nil {
		return err
	}

	if comment.ParentID != nil {
		_, err = tx.Exec(ctx, `UPDATE comments SET replies_count = replies_count + 1 WHERE id = $1`, *comment.ParentID)
		if err != nil {
			return err
		}
	}

	return tx.Commit(ctx)
}

func (r *PostgresCommentRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.Comment, error) {
	query := `
		SELECT id, post_id, author_id, parent_id, content, likes_count, replies_count, created_at, updated_at, deleted_at
		FROM comments
		WHERE id = $1 AND deleted_at IS NULL
	`
	row := r.pool.QueryRow(ctx, query, id)

	var c domain.Comment
	err := row.Scan(
		&c.ID,
		&c.PostID,
		&c.AuthorID,
		&c.ParentID,
		&c.Content,
		&c.LikesCount,
		&c.RepliesCount,
		&c.CreatedAt,
		&c.UpdatedAt,
		&c.DeletedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &c, nil
}

func (r *PostgresCommentRepo) GetPostOwnerID(ctx context.Context, postID uuid.UUID) (uuid.UUID, error) {
	query := `SELECT author_id FROM posts WHERE id = $1 AND deleted_at IS NULL`
	var authorID uuid.UUID
	err := r.pool.QueryRow(ctx, query, postID).Scan(&authorID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return uuid.Nil, domain.ErrPostNotFound
		}
		return uuid.Nil, err
	}
	return authorID, nil
}

func (r *PostgresCommentRepo) GetAuthor(ctx context.Context, userID uuid.UUID) (*domain.CommentAuthor, error) {
	query := `SELECT id, username, display_name, avatar_url FROM users WHERE id = $1`
	var a domain.CommentAuthor
	err := r.pool.QueryRow(ctx, query, userID).Scan(&a.ID, &a.Username, &a.DisplayName, &a.AvatarURL)
	if err != nil {
		return nil, err
	}
	return &a, nil
}

func (r *PostgresCommentRepo) ListByPost(ctx context.Context, postID uuid.UUID, cursorTime time.Time, cursorID uuid.UUID, limit int) ([]domain.CommentWithAuthor, bool, error) {
	fetchLimit := limit + 1
	var rows pgx.Rows
	var err error

	if cursorTime.IsZero() {
		query := `
			SELECT c.id, c.post_id, c.author_id, c.parent_id, c.content, c.likes_count, c.replies_count, c.created_at, c.updated_at,
			       u.id, u.username, u.display_name, u.avatar_url
			FROM comments c
			JOIN users u ON c.author_id = u.id
			WHERE c.post_id = $1 AND c.parent_id IS NULL AND c.deleted_at IS NULL
			ORDER BY c.created_at ASC, c.id ASC
			LIMIT $2
		`
		rows, err = r.pool.Query(ctx, query, postID, fetchLimit)
	} else {
		query := `
			SELECT c.id, c.post_id, c.author_id, c.parent_id, c.content, c.likes_count, c.replies_count, c.created_at, c.updated_at,
			       u.id, u.username, u.display_name, u.avatar_url
			FROM comments c
			JOIN users u ON c.author_id = u.id
			WHERE c.post_id = $1 AND c.parent_id IS NULL AND c.deleted_at IS NULL
			  AND (c.created_at, c.id) > ($2, $3)
			ORDER BY c.created_at ASC, c.id ASC
			LIMIT $4
		`
		rows, err = r.pool.Query(ctx, query, postID, cursorTime, cursorID, fetchLimit)
	}

	if err != nil {
		return nil, false, err
	}
	defer rows.Close()

	var items []domain.CommentWithAuthor
	for rows.Next() {
		var it domain.CommentWithAuthor
		err := rows.Scan(
			&it.Comment.ID,
			&it.Comment.PostID,
			&it.Comment.AuthorID,
			&it.Comment.ParentID,
			&it.Comment.Content,
			&it.Comment.LikesCount,
			&it.Comment.RepliesCount,
			&it.Comment.CreatedAt,
			&it.Comment.UpdatedAt,
			&it.Author.ID,
			&it.Author.Username,
			&it.Author.DisplayName,
			&it.Author.AvatarURL,
		)
		if err != nil {
			return nil, false, err
		}
		items = append(items, it)
	}

	hasMore := len(items) > limit
	if hasMore {
		items = items[:limit]
	}

	return items, hasMore, nil
}

func (r *PostgresCommentRepo) ListReplies(ctx context.Context, parentID uuid.UUID, cursorTime time.Time, cursorID uuid.UUID, limit int) ([]domain.CommentWithAuthor, bool, error) {
	fetchLimit := limit + 1
	var rows pgx.Rows
	var err error

	if cursorTime.IsZero() {
		query := `
			SELECT c.id, c.post_id, c.author_id, c.parent_id, c.content, c.likes_count, c.replies_count, c.created_at, c.updated_at,
			       u.id, u.username, u.display_name, u.avatar_url
			FROM comments c
			JOIN users u ON c.author_id = u.id
			WHERE c.parent_id = $1 AND c.deleted_at IS NULL
			ORDER BY c.created_at ASC, c.id ASC
			LIMIT $2
		`
		rows, err = r.pool.Query(ctx, query, parentID, fetchLimit)
	} else {
		query := `
			SELECT c.id, c.post_id, c.author_id, c.parent_id, c.content, c.likes_count, c.replies_count, c.created_at, c.updated_at,
			       u.id, u.username, u.display_name, u.avatar_url
			FROM comments c
			JOIN users u ON c.author_id = u.id
			WHERE c.parent_id = $1 AND c.deleted_at IS NULL
			  AND (c.created_at, c.id) > ($2, $3)
			ORDER BY c.created_at ASC, c.id ASC
			LIMIT $4
		`
		rows, err = r.pool.Query(ctx, query, parentID, cursorTime, cursorID, fetchLimit)
	}

	if err != nil {
		return nil, false, err
	}
	defer rows.Close()

	var items []domain.CommentWithAuthor
	for rows.Next() {
		var it domain.CommentWithAuthor
		err := rows.Scan(
			&it.Comment.ID,
			&it.Comment.PostID,
			&it.Comment.AuthorID,
			&it.Comment.ParentID,
			&it.Comment.Content,
			&it.Comment.LikesCount,
			&it.Comment.RepliesCount,
			&it.Comment.CreatedAt,
			&it.Comment.UpdatedAt,
			&it.Author.ID,
			&it.Author.Username,
			&it.Author.DisplayName,
			&it.Author.AvatarURL,
		)
		if err != nil {
			return nil, false, err
		}
		items = append(items, it)
	}

	hasMore := len(items) > limit
	if hasMore {
		items = items[:limit]
	}

	return items, hasMore, nil
}

func (r *PostgresCommentRepo) SoftDelete(ctx context.Context, comment *domain.Comment) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx, `UPDATE comments SET deleted_at = NOW() WHERE id = $1`, comment.ID)
	if err != nil {
		return err
	}

	_, err = tx.Exec(ctx, `UPDATE posts SET comments_count = GREATEST(comments_count - 1, 0) WHERE id = $1`, comment.PostID)
	if err != nil {
		return err
	}

	if comment.ParentID != nil {
		_, err = tx.Exec(ctx, `UPDATE comments SET replies_count = GREATEST(replies_count - 1, 0) WHERE id = $1`, *comment.ParentID)
		if err != nil {
			return err
		}
	}

	return tx.Commit(ctx)
}
