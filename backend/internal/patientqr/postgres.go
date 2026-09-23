package patientqr

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type postgresRepository struct {
	db *pgxpool.Pool
}

var _ Repository = (*postgresRepository)(nil)

func NewRepository(db *pgxpool.Pool) Repository {
	return &postgresRepository{
		db: db,
	}
}

func (r *postgresRepository) FindByUserID(
	ctx context.Context,
	userID uuid.UUID,
) (*Credential, error) {
	const query = `
		SELECT
			id,
			user_id,
			display_code,
			token_hash,
			is_active
		FROM patient_qr_credentials
		WHERE user_id = $1
	`

	var credential Credential

	err := r.db.QueryRow(
		ctx,
		query,
		userID,
	).Scan(
		&credential.ID,
		&credential.UserID,
		&credential.DisplayCode,
		&credential.TokenHash,
		&credential.IsActive,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, pgx.ErrNoRows
		}

		return nil, err
	}

	return &credential, nil
}

func (r *postgresRepository) FindByTokenHash(
	ctx context.Context,
	tokenHash string,
) (*Credential, error) {
	const query = `
		SELECT
			id,
			user_id,
			display_code,
			token_hash,
			is_active
		FROM patient_qr_credentials
		WHERE token_hash = $1
	`

	var credential Credential

	err := r.db.QueryRow(
		ctx,
		query,
		tokenHash,
	).Scan(
		&credential.ID,
		&credential.UserID,
		&credential.DisplayCode,
		&credential.TokenHash,
		&credential.IsActive,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, pgx.ErrNoRows
		}

		return nil, err
	}

	return &credential, nil
}

func (r *postgresRepository) Create(
	ctx context.Context,
	credential *Credential,
) error {
	const query = `
		INSERT INTO patient_qr_credentials (
			id,
			user_id,
			display_code,
			token_hash,
			is_active
		)
		VALUES ($1, $2, $3, $4, $5)
	`

	_, err := r.db.Exec(
		ctx,
		query,
		credential.ID,
		credential.UserID,
		credential.DisplayCode,
		credential.TokenHash,
		credential.IsActive,
	)

	return err
}

func (r *postgresRepository) Update(
	ctx context.Context,
	credential *Credential,
) error {
	const query = `
		UPDATE patient_qr_credentials
		SET
			display_code = $2,
			token_hash = $3,
			is_active = $4,
			updated_at = NOW()
		WHERE user_id = $1
	`

	_, err := r.db.Exec(
		ctx,
		query,
		credential.UserID,
		credential.DisplayCode,
		credential.TokenHash,
		credential.IsActive,
	)

	return err
}

func (r *postgresRepository) Deactivate(
	ctx context.Context,
	userID uuid.UUID,
) error {
	const query = `
		UPDATE patient_qr_credentials
		SET
			is_active = FALSE,
			updated_at = NOW()
		WHERE user_id = $1
	`

	_, err := r.db.Exec(
		ctx,
		query,
		userID,
	)

	return err
}
