package repository

import (
	"context"
	"errors"
	"time"

	"github.com/anjal-niraula/engage-engine/internal/modules/follow/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type postgresFollowRepository struct {
	pool *pgxpool.Pool
}

// NewPostgresFollowRepository constructs a PostgreSQL follow repository.
func NewPostgresFollowRepository(pool *pgxpool.Pool) domain.FollowRepository {
	return &postgresFollowRepository{pool: pool}
}

func (r *postgresFollowRepository) Follow(ctx context.Context, followerID, followeeID string) (bool, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return false, err
	}
	defer tx.Rollback(ctx)

	// Verify target followee exists
	var exists bool
	err = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM users WHERE id = $1);`, followeeID).Scan(&exists)
	if err != nil || !exists {
		return false, domain.ErrTargetUserNotFound
	}

	var insertedID string
	err = tx.QueryRow(ctx, `
		INSERT INTO follows (follower_id, followee_id)
		VALUES ($1, $2)
		ON CONFLICT (follower_id, followee_id) DO NOTHING
		RETURNING id;
	`, followerID, followeeID).Scan(&insertedID)

	isNewFollow := err == nil && insertedID != ""

	if isNewFollow {
		if _, err := tx.Exec(ctx, `UPDATE users SET following_count = following_count + 1 WHERE id = $1;`, followerID); err != nil {
			return false, err
		}
		if _, err := tx.Exec(ctx, `UPDATE users SET followers_count = followers_count + 1 WHERE id = $1;`, followeeID); err != nil {
			return false, err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return false, err
	}

	return isNewFollow, nil
}

func (r *postgresFollowRepository) Unfollow(ctx context.Context, followerID, followeeID string) (bool, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return false, err
	}
	defer tx.Rollback(ctx)

	tag, err := tx.Exec(ctx, `DELETE FROM follows WHERE follower_id = $1 AND followee_id = $2;`, followerID, followeeID)
	if err != nil {
		return false, err
	}

	wasDeleted := tag.RowsAffected() > 0

	if wasDeleted {
		if _, err := tx.Exec(ctx, `UPDATE users SET following_count = GREATEST(following_count - 1, 0) WHERE id = $1;`, followerID); err != nil {
			return false, err
		}
		if _, err := tx.Exec(ctx, `UPDATE users SET followers_count = GREATEST(followers_count - 1, 0) WHERE id = $1;`, followeeID); err != nil {
			return false, err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return false, err
	}

	return wasDeleted, nil
}

func (r *postgresFollowRepository) ListFollowers(ctx context.Context, userID string, limit int, cursorID string, cursorTime *time.Time) ([]*domain.FollowUser, error) {
	var rows pgx.Rows
	var err error

	if cursorID != "" && cursorTime != nil {
		query := `
			SELECT u.id, u.username, u.display_name, u.avatar_url, f.created_at
			FROM follows f
			JOIN users u ON u.id = f.follower_id
			WHERE f.followee_id = $1
			  AND (f.created_at, f.follower_id) < ($2, $3::uuid)
			ORDER BY f.created_at DESC, f.follower_id DESC
			LIMIT $4;
		`
		rows, err = r.pool.Query(ctx, query, userID, cursorTime, cursorID, limit)
	} else {
		query := `
			SELECT u.id, u.username, u.display_name, u.avatar_url, f.created_at
			FROM follows f
			JOIN users u ON u.id = f.follower_id
			WHERE f.followee_id = $1
			ORDER BY f.created_at DESC, f.follower_id DESC
			LIMIT $2;
		`
		rows, err = r.pool.Query(ctx, query, userID, limit)
	}

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return r.scanFollowUsers(rows)
}

func (r *postgresFollowRepository) ListFollowing(ctx context.Context, userID string, limit int, cursorID string, cursorTime *time.Time) ([]*domain.FollowUser, error) {
	var rows pgx.Rows
	var err error

	if cursorID != "" && cursorTime != nil {
		query := `
			SELECT u.id, u.username, u.display_name, u.avatar_url, f.created_at
			FROM follows f
			JOIN users u ON u.id = f.followee_id
			WHERE f.follower_id = $1
			  AND (f.created_at, f.followee_id) < ($2, $3::uuid)
			ORDER BY f.created_at DESC, f.followee_id DESC
			LIMIT $4;
		`
		rows, err = r.pool.Query(ctx, query, userID, cursorTime, cursorID, limit)
	} else {
		query := `
			SELECT u.id, u.username, u.display_name, u.avatar_url, f.created_at
			FROM follows f
			JOIN users u ON u.id = f.followee_id
			WHERE f.follower_id = $1
			ORDER BY f.created_at DESC, f.followee_id DESC
			LIMIT $2;
		`
		rows, err = r.pool.Query(ctx, query, userID, limit)
	}

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return r.scanFollowUsers(rows)
}

func (r *postgresFollowRepository) scanFollowUsers(rows pgx.Rows) ([]*domain.FollowUser, error) {
	users := make([]*domain.FollowUser, 0)
	for rows.Next() {
		var u domain.FollowUser
		if err := rows.Scan(&u.ID, &u.Username, &u.DisplayName, &u.AvatarURL, &u.FollowedAt); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return users, nil
			}
			return nil, err
		}
		users = append(users, &u)
	}
	return users, rows.Err()
}
