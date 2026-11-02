package repository

import (
	"context"
	"errors"
	"time"

	"github.com/anjal-niraula/engage-engine/internal/modules/post/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type postgresPostRepository struct {
	pool *pgxpool.Pool
}

// NewPostgresPostRepository constructs an instance of PostRepository with PostgreSQL.
func NewPostgresPostRepository(pool *pgxpool.Pool) domain.PostRepository {
	return &postgresPostRepository{pool: pool}
}

func (r *postgresPostRepository) Create(ctx context.Context, p *domain.Post) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	query := `
		INSERT INTO posts (author_id, caption, media_urls, media_type, location, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, likes_count, comments_count, views_count;
	`
	err = tx.QueryRow(ctx, query,
		p.AuthorID,
		p.Caption,
		p.MediaURLs,
		p.MediaType,
		p.Location,
		p.CreatedAt,
		p.UpdatedAt,
	).Scan(&p.ID, &p.LikesCount, &p.CommentsCount, &p.ViewsCount)
	if err != nil {
		return err
	}

	_, err = tx.Exec(ctx, `UPDATE users SET posts_count = posts_count + 1 WHERE id = $1`, p.AuthorID)
	if err != nil {
		return err
	}

	return tx.Commit(ctx)
}

func (r *postgresPostRepository) GetByID(ctx context.Context, id string) (*domain.Post, error) {
	query := `
		SELECT id, author_id, caption, media_urls, media_type, location,
		       likes_count, comments_count, views_count, created_at, updated_at, deleted_at
		FROM posts
		WHERE id = $1 AND deleted_at IS NULL;
	`
	return r.scanPost(r.pool.QueryRow(ctx, query, id))
}

func (r *postgresPostRepository) Delete(ctx context.Context, id string, authorID string) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	// Check post existence and ownership
	var actualAuthorID string
	err = tx.QueryRow(ctx, `SELECT author_id FROM posts WHERE id = $1 AND deleted_at IS NULL;`, id).Scan(&actualAuthorID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.ErrPostNotFound
		}
		return err
	}

	if actualAuthorID != authorID {
		return domain.ErrForbiddenPostAction
	}

	// Soft delete
	_, err = tx.Exec(ctx, `UPDATE posts SET deleted_at = NOW() WHERE id = $1;`, id)
	if err != nil {
		return err
	}

	_, err = tx.Exec(ctx, `UPDATE users SET posts_count = GREATEST(posts_count - 1, 0) WHERE id = $1;`, authorID)
	if err != nil {
		return err
	}

	return tx.Commit(ctx)
}

func (r *postgresPostRepository) ListByAuthorID(ctx context.Context, authorID string, limit int, cursorID string, cursorTime *time.Time) ([]*domain.Post, error) {
	var rows pgx.Rows
	var err error

	if cursorID != "" && cursorTime != nil {
		query := `
			SELECT id, author_id, caption, media_urls, media_type, location,
			       likes_count, comments_count, views_count, created_at, updated_at, deleted_at
			FROM posts
			WHERE author_id = $1 AND deleted_at IS NULL
			  AND (created_at, id) < ($2, $3::uuid)
			ORDER BY created_at DESC, id DESC
			LIMIT $4;
		`
		rows, err = r.pool.Query(ctx, query, authorID, cursorTime, cursorID, limit)
	} else {
		query := `
			SELECT id, author_id, caption, media_urls, media_type, location,
			       likes_count, comments_count, views_count, created_at, updated_at, deleted_at
			FROM posts
			WHERE author_id = $1 AND deleted_at IS NULL
			ORDER BY created_at DESC, id DESC
			LIMIT $2;
		`
		rows, err = r.pool.Query(ctx, query, authorID, limit)
	}

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return r.scanPosts(rows)
}

func (r *postgresPostRepository) ListFeed(ctx context.Context, userID string, limit int, cursorID string, cursorTime *time.Time) ([]*domain.Post, error) {
	var rows pgx.Rows
	var err error

	if cursorID != "" && cursorTime != nil {
		query := `
			SELECT p.id, p.author_id, p.caption, p.media_urls, p.media_type, p.location,
			       p.likes_count, p.comments_count, p.views_count, p.created_at, p.updated_at, p.deleted_at
			FROM posts p
			WHERE p.deleted_at IS NULL
			  AND (p.author_id = $1 OR p.author_id IN (SELECT followee_id FROM follows WHERE follower_id = $1))
			  AND (p.created_at, p.id) < ($2, $3::uuid)
			ORDER BY p.created_at DESC, p.id DESC
			LIMIT $4;
		`
		rows, err = r.pool.Query(ctx, query, userID, cursorTime, cursorID, limit)
	} else {
		query := `
			SELECT p.id, p.author_id, p.caption, p.media_urls, p.media_type, p.location,
			       p.likes_count, p.comments_count, p.views_count, p.created_at, p.updated_at, p.deleted_at
			FROM posts p
			WHERE p.deleted_at IS NULL
			  AND (p.author_id = $1 OR p.author_id IN (SELECT followee_id FROM follows WHERE follower_id = $1))
			ORDER BY p.created_at DESC, p.id DESC
			LIMIT $2;
		`
		rows, err = r.pool.Query(ctx, query, userID, limit)
	}

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return r.scanPosts(rows)
}

func (r *postgresPostRepository) GetAuthorByUsername(ctx context.Context, username string) (*domain.Author, error) {
	query := `SELECT id, username, display_name, avatar_url FROM users WHERE username = $1;`
	var a domain.Author
	err := r.pool.QueryRow(ctx, query, username).Scan(&a.ID, &a.Username, &a.DisplayName, &a.AvatarURL)
	if err != nil {
		return nil, err
	}
	return &a, nil
}

func (r *postgresPostRepository) BatchGetAuthors(ctx context.Context, authorIDs []string) (map[string]*domain.Author, error) {
	results := make(map[string]*domain.Author)
	if len(authorIDs) == 0 {
		return results, nil
	}

	query := `SELECT id, username, display_name, avatar_url FROM users WHERE id = ANY($1);`
	rows, err := r.pool.Query(ctx, query, authorIDs)
	if err != nil {
		return results, err
	}
	defer rows.Close()

	for rows.Next() {
		var a domain.Author
		if err := rows.Scan(&a.ID, &a.Username, &a.DisplayName, &a.AvatarURL); err == nil {
			results[a.ID] = &a
		}
	}

	return results, nil
}

func (r *postgresPostRepository) BatchCheckLiked(ctx context.Context, userID string, postIDs []string) (map[string]bool, error) {
	results := make(map[string]bool)
	if len(postIDs) == 0 || userID == "" {
		return results, nil
	}

	query := `SELECT post_id FROM post_likes WHERE user_id = $1 AND post_id = ANY($2);`
	rows, err := r.pool.Query(ctx, query, userID, postIDs)
	if err != nil {
		return results, err
	}
	defer rows.Close()

	for rows.Next() {
		var postID string
		if err := rows.Scan(&postID); err == nil {
			results[postID] = true
		}
	}

	return results, nil
}

func (r *postgresPostRepository) scanPost(row pgx.Row) (*domain.Post, error) {
	var p domain.Post
	err := row.Scan(
		&p.ID,
		&p.AuthorID,
		&p.Caption,
		&p.MediaURLs,
		&p.MediaType,
		&p.Location,
		&p.LikesCount,
		&p.CommentsCount,
		&p.ViewsCount,
		&p.CreatedAt,
		&p.UpdatedAt,
		&p.DeletedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrPostNotFound
		}
		return nil, err
	}
	return &p, nil
}

func (r *postgresPostRepository) scanPosts(rows pgx.Rows) ([]*domain.Post, error) {
	posts := make([]*domain.Post, 0)
	for rows.Next() {
		var p domain.Post
		err := rows.Scan(
			&p.ID,
			&p.AuthorID,
			&p.Caption,
			&p.MediaURLs,
			&p.MediaType,
			&p.Location,
			&p.LikesCount,
			&p.CommentsCount,
			&p.ViewsCount,
			&p.CreatedAt,
			&p.UpdatedAt,
			&p.DeletedAt,
		)
		if err != nil {
			return nil, err
		}
		posts = append(posts, &p)
	}
	return posts, rows.Err()
}
