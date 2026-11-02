package repository

import (
	"context"
	"time"

	"github.com/anjal-niraula/engage-engine/internal/modules/notification/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresNotificationRepo struct {
	pool *pgxpool.Pool
}

func NewPostgresNotificationRepo(pool *pgxpool.Pool) *PostgresNotificationRepo {
	return &PostgresNotificationRepo{pool: pool}
}

func (r *PostgresNotificationRepo) Create(ctx context.Context, n *domain.Notification) error {
	query := `
		INSERT INTO notifications (id, user_id, actor_id, type, target_id, target_type, is_read, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`
	_, err := r.pool.Exec(ctx, query,
		n.ID,
		n.UserID,
		n.ActorID,
		string(n.Type),
		n.TargetID,
		n.TargetType,
		n.IsRead,
		n.CreatedAt,
	)
	return err
}

func (r *PostgresNotificationRepo) ListByUser(ctx context.Context, userID uuid.UUID, cursorID string, cursorTime *time.Time, limit int) ([]*domain.Notification, error) {
	var rows pgx.Rows
	var err error

	if cursorTime == nil || cursorID == "" {
		query := `
			SELECT n.id, n.user_id, n.actor_id, n.type, n.target_id, n.target_type, n.is_read, n.created_at,
			       u.id, u.username, u.display_name, u.avatar_url
			FROM notifications n
			JOIN users u ON n.actor_id = u.id
			WHERE n.user_id = $1
			ORDER BY n.created_at DESC, n.id DESC
			LIMIT $2
		`
		rows, err = r.pool.Query(ctx, query, userID, limit)
	} else {
		cursorUUID, parseErr := uuid.Parse(cursorID)
		if parseErr != nil {
			cursorUUID = uuid.Nil
		}
		query := `
			SELECT n.id, n.user_id, n.actor_id, n.type, n.target_id, n.target_type, n.is_read, n.created_at,
			       u.id, u.username, u.display_name, u.avatar_url
			FROM notifications n
			JOIN users u ON n.actor_id = u.id
			WHERE n.user_id = $1
			  AND (n.created_at, n.id) < ($2, $3)
			ORDER BY n.created_at DESC, n.id DESC
			LIMIT $4
		`
		rows, err = r.pool.Query(ctx, query, userID, *cursorTime, cursorUUID, limit)
	}

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var notifications []*domain.Notification
	for rows.Next() {
		var n domain.Notification
		var typeStr string
		var actor domain.NotificationActor

		err := rows.Scan(
			&n.ID,
			&n.UserID,
			&n.ActorID,
			&typeStr,
			&n.TargetID,
			&n.TargetType,
			&n.IsRead,
			&n.CreatedAt,
			&actor.ID,
			&actor.Username,
			&actor.DisplayName,
			&actor.AvatarURL,
		)
		if err != nil {
			return nil, err
		}

		n.Type = domain.NotificationType(typeStr)
		n.Actor = &actor
		notifications = append(notifications, &n)
	}

	return notifications, nil
}

func (r *PostgresNotificationRepo) MarkAsRead(ctx context.Context, id uuid.UUID, userID uuid.UUID) error {
	query := `UPDATE notifications SET is_read = TRUE WHERE id = $1 AND user_id = $2`
	ct, err := r.pool.Exec(ctx, query, id, userID)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return domain.ErrNotificationNotFound
	}
	return nil
}
