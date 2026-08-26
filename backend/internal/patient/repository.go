package patient

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

func (r *Repository) FindByUserID(ctx context.Context, userID uuid.UUID) (*Patient, error) {
	patient := &Patient{}

	err := r.db.QueryRow(
		ctx,
		`
		SELECT
			id,
			user_id,
			nik_hash,
			full_name,
			date_of_birth,
			gender,
			phone,
			address,
			created_at,
			updated_at
		FROM patients
		WHERE user_id = $1
		`,
		userID,
	).Scan(
		&patient.ID,
		&patient.UserID,
		&patient.NIKHash,
		&patient.FullName,
		&patient.DateOfBirth,
		&patient.Gender,
		&patient.Phone,
		&patient.Address,
		&patient.CreatedAt,
		&patient.UpdatedAt,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrPatientNotFound
		}

		return nil, err
	}

	return patient, nil
}

func (r *Repository) FindByID(ctx context.Context, id uuid.UUID) (*Patient, error) {
	patient := &Patient{}

	err := r.db.QueryRow(
		ctx,
		`
		SELECT
			id,
			user_id,
			nik_hash,
			full_name,
			date_of_birth,
			gender,
			phone,
			address,
			created_at,
			updated_at
		FROM patients
		WHERE id = $1
		`,
		id,
	).Scan(
		&patient.ID,
		&patient.UserID,
		&patient.NIKHash,
		&patient.FullName,
		&patient.DateOfBirth,
		&patient.Gender,
		&patient.Phone,
		&patient.Address,
		&patient.CreatedAt,
		&patient.UpdatedAt,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrPatientNotFound
		}

		return nil, err
	}

	return patient, nil
}
