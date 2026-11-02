package repository

import (
	"context"
	"errors"

	"github.com/anjal-niraula/engage-engine/internal/modules/user/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type postgresUserRepository struct {
	pool *pgxpool.Pool
}

// NewPostgresUserRepository creates a new PostgreSQL repository for users.
func NewPostgresUserRepository(pool *pgxpool.Pool) domain.UserRepository {
	return &postgresUserRepository{pool: pool}
}

func (r *postgresUserRepository) Create(ctx context.Context, u *domain.User) error {
	query := `
		INSERT INTO users (username, email, password_hash, display_name, bio, avatar_url, website, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id, followers_count, following_count, posts_count;
	`

	return r.pool.QueryRow(ctx, query,
		u.Username,
		u.Email,
		u.PasswordHash,
		u.DisplayName,
		u.Bio,
		u.AvatarURL,
		u.Website,
		u.CreatedAt,
		u.UpdatedAt,
	).Scan(&u.ID, &u.FollowersCount, &u.FollowingCount, &u.PostsCount)
}

func (r *postgresUserRepository) GetByID(ctx context.Context, id string) (*domain.User, error) {
	query := `
		SELECT id, username, email, password_hash, display_name, bio, avatar_url, website,
		       followers_count, following_count, posts_count, created_at, updated_at
		FROM users
		WHERE id = $1;
	`
	return r.scanUser(r.pool.QueryRow(ctx, query, id))
}

func (r *postgresUserRepository) GetByUsername(ctx context.Context, username string) (*domain.User, error) {
	query := `
		SELECT id, username, email, password_hash, display_name, bio, avatar_url, website,
		       followers_count, following_count, posts_count, created_at, updated_at
		FROM users
		WHERE username = $1;
	`
	return r.scanUser(r.pool.QueryRow(ctx, query, username))
}

func (r *postgresUserRepository) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	query := `
		SELECT id, username, email, password_hash, display_name, bio, avatar_url, website,
		       followers_count, following_count, posts_count, created_at, updated_at
		FROM users
		WHERE email = $1;
	`
	return r.scanUser(r.pool.QueryRow(ctx, query, email))
}

func (r *postgresUserRepository) GetByIdentity(ctx context.Context, identity string) (*domain.User, error) {
	query := `
		SELECT id, username, email, password_hash, display_name, bio, avatar_url, website,
		       followers_count, following_count, posts_count, created_at, updated_at
		FROM users
		WHERE username = $1 OR email = $1
		LIMIT 1;
	`
	return r.scanUser(r.pool.QueryRow(ctx, query, identity))
}

func (r *postgresUserRepository) Update(ctx context.Context, u *domain.User) error {
	query := `
		UPDATE users
		SET display_name = $1, bio = $2, avatar_url = $3, website = $4, updated_at = $5
		WHERE id = $6;
	`
	tag, err := r.pool.Exec(ctx, query,
		u.DisplayName,
		u.Bio,
		u.AvatarURL,
		u.Website,
		u.UpdatedAt,
		u.ID,
	)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrUserNotFound
	}
	return nil
}

func (r *postgresUserRepository) IsFollowing(ctx context.Context, followerID, followeeID string) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM follows WHERE follower_id = $1 AND followee_id = $2);`
	var exists bool
	err := r.pool.QueryRow(ctx, query, followerID, followeeID).Scan(&exists)
	return exists, err
}

func (r *postgresUserRepository) scanUser(row pgx.Row) (*domain.User, error) {
	var u domain.User
	err := row.Scan(
		&u.ID,
		&u.Username,
		&u.Email,
		&u.PasswordHash,
		&u.DisplayName,
		&u.Bio,
		&u.AvatarURL,
		&u.Website,
		&u.FollowersCount,
		&u.FollowingCount,
		&u.PostsCount,
		&u.CreatedAt,
		&u.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrUserNotFound
		}
		return nil, err
	}
	return &u, nil
}
