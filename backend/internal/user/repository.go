package user

import (
	"context"

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

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrUserNotFound
		}

		return nil, err
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

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrUserNotFound
		}

		return nil, err
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
		return nil, err
	}

	return user, nil
}
