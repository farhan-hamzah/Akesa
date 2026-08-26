package hospital

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

const uniqueViolationCode = "23505"

type Repository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

func (r *Repository) Create(ctx context.Context, in HospitalInput) (*Hospital, error) {
	h := &Hospital{}

	err := r.db.QueryRow(ctx,
		`INSERT INTO hospitals (id, name, address, phone, email, registration_number, status)
		 VALUES ($1,$2,$3,$4,$5,$6,$7)
		 RETURNING id, name, address, phone, email, registration_number, status, created_at, updated_at`,
		uuid.New(), in.Name, in.Address, in.Phone, in.Email, in.RegistrationNumber, StatusPending,
	).Scan(&h.ID, &h.Name, &h.Address, &h.Phone, &h.Email, &h.RegistrationNumber, &h.Status, &h.CreatedAt, &h.UpdatedAt)

	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == uniqueViolationCode {
			if pgErr.ConstraintName == "hospitals_email_key" {
				return nil, ErrEmailTaken
			}
			if pgErr.ConstraintName == "hospitals_registration_number_key" {
				return nil, ErrRegistrationTaken
			}
		}
		return nil, fmt.Errorf("insert hospital: %w", err)
	}

	return h, nil
}

func (r *Repository) FindByID(ctx context.Context, id uuid.UUID) (*Hospital, error) {
	h := &Hospital{}

	err := r.db.QueryRow(ctx,
		`SELECT id, name, address, phone, email, registration_number, status, created_at, updated_at
		 FROM hospitals WHERE id = $1`,
		id,
	).Scan(&h.ID, &h.Name, &h.Address, &h.Phone, &h.Email, &h.RegistrationNumber, &h.Status, &h.CreatedAt, &h.UpdatedAt)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("query hospital: %w", err)
	}

	return h, nil
}

func (r *Repository) List(ctx context.Context) ([]*Hospital, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, name, address, phone, email, registration_number, status, created_at, updated_at
		 FROM hospitals ORDER BY created_at DESC`,
	)
	if err != nil {
		return nil, fmt.Errorf("list hospitals: %w", err)
	}
	defer rows.Close()

	var hospitals []*Hospital
	for rows.Next() {
		h := &Hospital{}
		if err := rows.Scan(&h.ID, &h.Name, &h.Address, &h.Phone, &h.Email, &h.RegistrationNumber, &h.Status, &h.CreatedAt, &h.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan hospital: %w", err)
		}
		hospitals = append(hospitals, h)
	}

	return hospitals, rows.Err()
}

func (r *Repository) UpdateStatus(ctx context.Context, id uuid.UUID, status Status) (*Hospital, error) {
	h := &Hospital{}

	err := r.db.QueryRow(ctx,
		`UPDATE hospitals SET status = $2, updated_at = NOW()
		 WHERE id = $1
		 RETURNING id, name, address, phone, email, registration_number, status, created_at, updated_at`,
		id, status,
	).Scan(&h.ID, &h.Name, &h.Address, &h.Phone, &h.Email, &h.RegistrationNumber, &h.Status, &h.CreatedAt, &h.UpdatedAt)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("update hospital status: %w", err)
	}

	return h, nil
}

func (r *Repository) CreateStaff(ctx context.Context, userID, hospitalID uuid.UUID, fullName, position string) (*Staff, error) {
	s := &Staff{}

	err := r.db.QueryRow(ctx,
		`INSERT INTO hospital_staff (id, user_id, hospital_id, full_name, position)
		 VALUES ($1,$2,$3,$4,$5)
		 RETURNING id, user_id, hospital_id, full_name, position, created_at, updated_at`,
		uuid.New(), userID, hospitalID, fullName, position,
	).Scan(&s.ID, &s.UserID, &s.HospitalID, &s.FullName, &s.Position, &s.CreatedAt, &s.UpdatedAt)

	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == uniqueViolationCode {
			return nil, ErrStaffAlreadyLinked
		}
		return nil, fmt.Errorf("insert hospital staff: %w", err)
	}

	return s, nil
}

func (r *Repository) FindStaffByUserID(ctx context.Context, userID uuid.UUID) (*Staff, error) {
	s := &Staff{}

	err := r.db.QueryRow(ctx,
		`SELECT id, user_id, hospital_id, full_name, position, created_at, updated_at
		 FROM hospital_staff WHERE user_id = $1`,
		userID,
	).Scan(&s.ID, &s.UserID, &s.HospitalID, &s.FullName, &s.Position, &s.CreatedAt, &s.UpdatedAt)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrStaffNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("query hospital staff: %w", err)
	}

	return s, nil
}
