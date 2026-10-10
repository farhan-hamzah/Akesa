package notification

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrNotificationNotFound = errors.New("notification not found")

type Repository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

func (r *Repository) Create(ctx context.Context, n *Notification) error {
	query := `
		INSERT INTO notifications (
			id, user_id, title, message, category, severity, is_read, action_url, metadata, created_at, read_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11
		)
	`
	_, err := r.db.Exec(ctx, query,
		n.ID,
		n.UserID,
		n.Title,
		n.Message,
		n.Category,
		n.Severity,
		n.IsRead,
		n.ActionURL,
		n.Metadata,
		n.CreatedAt,
		n.ReadAt,
	)
	if err != nil {
		return fmt.Errorf("create notification: %w", err)
	}
	return nil
}

func (r *Repository) ListByUserID(ctx context.Context, userID uuid.UUID, unreadOnly bool, limit, offset int) ([]*Notification, int, error) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}

	countQuery := `SELECT COUNT(*) FROM notifications WHERE user_id = $1`
	if unreadOnly {
		countQuery += ` AND is_read = FALSE`
	}

	var total int
	if err := r.db.QueryRow(ctx, countQuery, userID).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count notifications: %w", err)
	}

	dataQuery := `
		SELECT id, user_id, title, message, category, severity, is_read, action_url, metadata, created_at, read_at
		FROM notifications
		WHERE user_id = $1
	`
	if unreadOnly {
		dataQuery += ` AND is_read = FALSE`
	}
	dataQuery += ` ORDER BY created_at DESC LIMIT $2 OFFSET $3`

	rows, err := r.db.Query(ctx, dataQuery, userID, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("query notifications: %w", err)
	}
	defer rows.Close()

	items := make([]*Notification, 0)
	for rows.Next() {
		n := &Notification{}
		if err := rows.Scan(
			&n.ID,
			&n.UserID,
			&n.Title,
			&n.Message,
			&n.Category,
			&n.Severity,
			&n.IsRead,
			&n.ActionURL,
			&n.Metadata,
			&n.CreatedAt,
			&n.ReadAt,
		); err != nil {
			return nil, 0, fmt.Errorf("scan notification: %w", err)
		}
		items = append(items, n)
	}

	return items, total, rows.Err()
}

func (r *Repository) CountUnread(ctx context.Context, userID uuid.UUID) (int, error) {
	query := `SELECT COUNT(*) FROM notifications WHERE user_id = $1 AND is_read = FALSE`
	var count int
	if err := r.db.QueryRow(ctx, query, userID).Scan(&count); err != nil {
		return 0, fmt.Errorf("count unread: %w", err)
	}
	return count, nil
}

func (r *Repository) MarkAsRead(ctx context.Context, id, userID uuid.UUID) (*Notification, error) {
	now := time.Now().UTC()
	query := `
		UPDATE notifications
		SET is_read = TRUE, read_at = COALESCE(read_at, $3)
		WHERE id = $1 AND user_id = $2
		RETURNING id, user_id, title, message, category, severity, is_read, action_url, metadata, created_at, read_at
	`
	n := &Notification{}
	err := r.db.QueryRow(ctx, query, id, userID, now).Scan(
		&n.ID,
		&n.UserID,
		&n.Title,
		&n.Message,
		&n.Category,
		&n.Severity,
		&n.IsRead,
		&n.ActionURL,
		&n.Metadata,
		&n.CreatedAt,
		&n.ReadAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotificationNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("mark notification as read: %w", err)
	}
	return n, nil
}

func (r *Repository) MarkAllAsRead(ctx context.Context, userID uuid.UUID) error {
	now := time.Now().UTC()
	query := `
		UPDATE notifications
		SET is_read = TRUE, read_at = COALESCE(read_at, $2)
		WHERE user_id = $1 AND is_read = FALSE
	`
	_, err := r.db.Exec(ctx, query, userID, now)
	if err != nil {
		return fmt.Errorf("mark all as read: %w", err)
	}
	return nil
}

func (r *Repository) UpsertDeviceToken(ctx context.Context, dt *DeviceToken) error {
	query := `
		INSERT INTO user_device_tokens (id, user_id, token, platform, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (token) DO UPDATE
		SET user_id = EXCLUDED.user_id,
		    platform = EXCLUDED.platform,
		    updated_at = EXCLUDED.updated_at
	`
	_, err := r.db.Exec(ctx, query, dt.ID, dt.UserID, dt.Token, dt.Platform, dt.CreatedAt, dt.UpdatedAt)
	if err != nil {
		return fmt.Errorf("upsert device token: %w", err)
	}
	return nil
}

func (r *Repository) DeleteDeviceToken(ctx context.Context, userID uuid.UUID, token string) error {
	query := `DELETE FROM user_device_tokens WHERE user_id = $1 AND token = $2`
	_, err := r.db.Exec(ctx, query, userID, token)
	if err != nil {
		return fmt.Errorf("delete device token: %w", err)
	}
	return nil
}

func (r *Repository) GetDeviceTokensByUserID(ctx context.Context, userID uuid.UUID) ([]string, error) {
	query := `SELECT token FROM user_device_tokens WHERE user_id = $1 ORDER BY updated_at DESC`
	rows, err := r.db.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("query tokens by user: %w", err)
	}
	defer rows.Close()

	var tokens []string
	for rows.Next() {
		var tok string
		if err := rows.Scan(&tok); err != nil {
			return nil, fmt.Errorf("scan token: %w", err)
		}
		tokens = append(tokens, tok)
	}
	return tokens, rows.Err()
}

func (r *Repository) GetUsersByRole(ctx context.Context, role string) ([]uuid.UUID, error) {
	query := `SELECT id FROM users WHERE role = $1 AND status = 'ACTIVE'`
	rows, err := r.db.Query(ctx, query, role)
	if err != nil {
		return nil, fmt.Errorf("query users by role: %w", err)
	}
	defer rows.Close()

	var ids []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scan user id: %w", err)
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}
