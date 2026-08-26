package user

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{
		db: db,
	}
}

func (r *Repository) FindByClerkUserID(ctx context.Context, clerkUserID string) (*User, error) {
	user := &User{}

	err := r.db.QueryRow(
		ctx,
		`
		SELECT
			id,
			clerk_user_id,
			role,
			status,
			created_at,
			updated_at
		FROM users
		WHERE clerk_user_id = $1
		`,
		clerkUserID,
	).Scan(
		&user.ID,
		&user.ClerkUserID,
		&user.Role,
		&user.Status,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrUserNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("query user by clerk id: %w", err)
	}

	return user, nil
}

func (r *Repository) FindByID(ctx context.Context, id uuid.UUID) (*User, error) {
	user := &User{}

	err := r.db.QueryRow(
		ctx,
		`
		SELECT
			id,
			clerk_user_id,
			role,
			status,
			created_at,
			updated_at
		FROM users
		WHERE id = $1
		`,
		id,
	).Scan(
		&user.ID,
		&user.ClerkUserID,
		&user.Role,
		&user.Status,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrUserNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("query user by id: %w", err)
	}

	return user, nil
}

func (r *Repository) Create(ctx context.Context, clerkUserID string) (*User, error) {
	user := &User{}

	err := r.db.QueryRow(
		ctx,
		`
		INSERT INTO users (
			id,
			clerk_user_id,
			role,
			status
		)
		VALUES (
			$1,
			$2,
			$3,
			$4
		)
		RETURNING
			id,
			clerk_user_id,
			role,
			status,
			created_at,
			updated_at
		`,
		uuid.New(),
		clerkUserID,
		RolePatient,
		StatusActive,
	).Scan(
		&user.ID,
		&user.ClerkUserID,
		&user.Role,
		&user.Status,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("insert user: %w", err)
	}

	return user, nil
}

// UpdateRole is admin-only in practice (enforced at the handler/service
// layer, not here) - it lets an Administrator promote a synced account to
// HOSPITAL_STAFF or ADMIN, e.g. when linking staff to a hospital.
func (r *Repository) UpdateRole(ctx context.Context, id uuid.UUID, role Role) error {
	tag, err := r.db.Exec(
		ctx,
		`UPDATE users SET role = $2, updated_at = NOW() WHERE id = $1`,
		id, role,
	)
	if err != nil {
		return fmt.Errorf("update user role: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrUserNotFound
	}
	return nil
}

func (r *Repository) UpdateStatus(ctx context.Context, id uuid.UUID, status Status) error {
	tag, err := r.db.Exec(
		ctx,
		`UPDATE users SET status = $2, updated_at = NOW() WHERE id = $1`,
		id, status,
	)
	if err != nil {
		return fmt.Errorf("update user status: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrUserNotFound
	}
	return nil
}
