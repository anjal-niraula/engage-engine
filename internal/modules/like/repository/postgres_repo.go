package repository

import (
	"context"
	"errors"
	"time"

	"github.com/anjal-niraula/engage-engine/internal/modules/like/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type postgresLikeRepository struct {
	pool *pgxpool.Pool
}

// NewPostgresLikeRepository creates a PostgreSQL repository for like operations.
func NewPostgresLikeRepository(pool *pgxpool.Pool) domain.LikeRepository {
	return &postgresLikeRepository{pool: pool}
}

func (r *postgresLikeRepository) LikePost(ctx context.Context, userID, postID string) (bool, int64, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return false, 0, err
	}
	defer tx.Rollback(ctx)

	// Verify target post exists
	var currentLikes int64
	err = tx.QueryRow(ctx, `SELECT likes_count FROM posts WHERE id = $1 AND deleted_at IS NULL;`, postID).Scan(&currentLikes)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, 0, domain.ErrTargetNotFound
		}
		return false, 0, err
	}

	// Idempotent insert
	var insertedID string
	err = tx.QueryRow(ctx, `
		INSERT INTO post_likes (user_id, post_id)
		VALUES ($1, $2)
		ON CONFLICT (user_id, post_id) DO NOTHING
		RETURNING id;
	`, userID, postID).Scan(&insertedID)

	isNewLike := err == nil && insertedID != ""

	if isNewLike {
		err = tx.QueryRow(ctx, `
			UPDATE posts
			SET likes_count = likes_count + 1
			WHERE id = $1
			RETURNING likes_count;
		`, postID).Scan(&currentLikes)
		if err != nil {
			return false, 0, err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return false, 0, err
	}

	return isNewLike, currentLikes, nil
}

func (r *postgresLikeRepository) UnlikePost(ctx context.Context, userID, postID string) (bool, int64, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return false, 0, err
	}
	defer tx.Rollback(ctx)

	// Verify post exists
	var currentLikes int64
	err = tx.QueryRow(ctx, `SELECT likes_count FROM posts WHERE id = $1 AND deleted_at IS NULL;`, postID).Scan(&currentLikes)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, 0, domain.ErrTargetNotFound
		}
		return false, 0, err
	}

	tag, err := tx.Exec(ctx, `DELETE FROM post_likes WHERE user_id = $1 AND post_id = $2;`, userID, postID)
	if err != nil {
		return false, 0, err
	}

	wasDeleted := tag.RowsAffected() > 0

	if wasDeleted {
		err = tx.QueryRow(ctx, `
			UPDATE posts
			SET likes_count = GREATEST(likes_count - 1, 0)
			WHERE id = $1
			RETURNING likes_count;
		`, postID).Scan(&currentLikes)
		if err != nil {
			return false, 0, err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return false, 0, err
	}

	return wasDeleted, currentLikes, nil
}

func (r *postgresLikeRepository) ListPostLikes(ctx context.Context, postID string, limit int, cursorID string, cursorTime *time.Time) ([]*domain.LikerUser, error) {
	var rows pgx.Rows
	var err error

	if cursorID != "" && cursorTime != nil {
		query := `
			SELECT pl.user_id, u.username, u.display_name, u.avatar_url, pl.created_at
			FROM post_likes pl
			JOIN users u ON u.id = pl.user_id
			WHERE pl.post_id = $1
			  AND (pl.created_at, pl.user_id) < ($2, $3::uuid)
			ORDER BY pl.created_at DESC, pl.user_id DESC
			LIMIT $4;
		`
		rows, err = r.pool.Query(ctx, query, postID, cursorTime, cursorID, limit)
	} else {
		query := `
			SELECT pl.user_id, u.username, u.display_name, u.avatar_url, pl.created_at
			FROM post_likes pl
			JOIN users u ON u.id = pl.user_id
			WHERE pl.post_id = $1
			ORDER BY pl.created_at DESC, pl.user_id DESC
			LIMIT $2;
		`
		rows, err = r.pool.Query(ctx, query, postID, limit)
	}

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	likers := make([]*domain.LikerUser, 0)
	for rows.Next() {
		var l domain.LikerUser
		if err := rows.Scan(&l.UserID, &l.Username, &l.DisplayName, &l.AvatarURL, &l.LikedAt); err != nil {
			return nil, err
		}
		likers = append(likers, &l)
	}

	return likers, rows.Err()
}

func (r *postgresLikeRepository) LikeComment(ctx context.Context, userID, commentID string) (bool, int64, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return false, 0, err
	}
	defer tx.Rollback(ctx)

	var currentLikes int64
	err = tx.QueryRow(ctx, `SELECT likes_count FROM comments WHERE id = $1 AND deleted_at IS NULL;`, commentID).Scan(&currentLikes)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, 0, domain.ErrTargetNotFound
		}
		return false, 0, err
	}

	var insertedID string
	err = tx.QueryRow(ctx, `
		INSERT INTO comment_likes (user_id, comment_id)
		VALUES ($1, $2)
		ON CONFLICT (user_id, comment_id) DO NOTHING
		RETURNING id;
	`, userID, commentID).Scan(&insertedID)

	isNewLike := err == nil && insertedID != ""

	if isNewLike {
		err = tx.QueryRow(ctx, `
			UPDATE comments
			SET likes_count = likes_count + 1
			WHERE id = $1
			RETURNING likes_count;
		`, commentID).Scan(&currentLikes)
		if err != nil {
			return false, 0, err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return false, 0, err
	}

	return isNewLike, currentLikes, nil
}

func (r *postgresLikeRepository) UnlikeComment(ctx context.Context, userID, commentID string) (bool, int64, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return false, 0, err
	}
	defer tx.Rollback(ctx)

	var currentLikes int64
	err = tx.QueryRow(ctx, `SELECT likes_count FROM comments WHERE id = $1 AND deleted_at IS NULL;`, commentID).Scan(&currentLikes)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, 0, domain.ErrTargetNotFound
		}
		return false, 0, err
	}

	tag, err := tx.Exec(ctx, `DELETE FROM comment_likes WHERE user_id = $1 AND comment_id = $2;`, userID, commentID)
	if err != nil {
		return false, 0, err
	}

	wasDeleted := tag.RowsAffected() > 0

	if wasDeleted {
		err = tx.QueryRow(ctx, `
			UPDATE comments
			SET likes_count = GREATEST(likes_count - 1, 0)
			WHERE id = $1
			RETURNING likes_count;
		`, commentID).Scan(&currentLikes)
		if err != nil {
			return false, 0, err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return false, 0, err
	}

	return wasDeleted, currentLikes, nil
}
