package access

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/farhan-hamzah/Akesa/backend/internal/datacategory"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

func (r *Repository) Create(
	ctx context.Context,
	hospitalID, patientID, requestedBy uuid.UUID,
	purpose string,
	categories []datacategory.Category,
) (*Request, error) {
	row := r.db.QueryRow(ctx,
		`INSERT INTO access_requests (id, hospital_id, patient_id, requested_by, purpose, categories, status)
		 VALUES ($1,$2,$3,$4,$5,$6,$7)
		 RETURNING id, hospital_id, patient_id, requested_by, purpose, categories, status, decided_at, created_at, updated_at`,
		uuid.New(), hospitalID, patientID, requestedBy, purpose, toStringSlice(categories), StatusPending,
	)

	req, err := scanRequest(row)
	if err != nil {
		return nil, fmt.Errorf("insert access request: %w", err)
	}

	return req, nil
}

func (r *Repository) FindByID(ctx context.Context, id uuid.UUID) (*Request, error) {
	row := r.db.QueryRow(ctx,
		`SELECT id, hospital_id, patient_id, requested_by, purpose, categories, status, decided_at, created_at, updated_at
		 FROM access_requests WHERE id = $1`,
		id,
	)

	req, err := scanRequest(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrRequestNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("query access request: %w", err)
	}

	return req, nil
}

func (r *Repository) ListByPatient(ctx context.Context, patientID uuid.UUID) ([]*Request, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, hospital_id, patient_id, requested_by, purpose, categories, status, decided_at, created_at, updated_at
		 FROM access_requests WHERE patient_id = $1 ORDER BY created_at DESC`,
		patientID,
	)
	if err != nil {
		return nil, fmt.Errorf("list access requests by patient: %w", err)
	}
	defer rows.Close()

	return scanRequestRows(rows)
}

func (r *Repository) ListByHospital(ctx context.Context, hospitalID uuid.UUID) ([]*Request, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, hospital_id, patient_id, requested_by, purpose, categories, status, decided_at, created_at, updated_at
		 FROM access_requests WHERE hospital_id = $1 ORDER BY created_at DESC`,
		hospitalID,
	)
	if err != nil {
		return nil, fmt.Errorf("list access requests by hospital: %w", err)
	}
	defer rows.Close()

	return scanRequestRows(rows)
}

// UpdateStatus performs an atomic, guarded transition: it only succeeds if
// the row's current status matches expectedCurrent, which prevents a
// double-approve / double-revoke race between two concurrent requests for
// the same access request.
func (r *Repository) UpdateStatus(ctx context.Context, id uuid.UUID, expectedCurrent, next Status) (*Request, error) {
	var decidedAt *time.Time
	if next == StatusApproved || next == StatusRejected {
		now := time.Now().UTC()
		decidedAt = &now
	}

	row := r.db.QueryRow(ctx,
		`UPDATE access_requests SET status = $3, decided_at = COALESCE($4, decided_at), updated_at = NOW()
		 WHERE id = $1 AND status = $2
		 RETURNING id, hospital_id, patient_id, requested_by, purpose, categories, status, decided_at, created_at, updated_at`,
		id, expectedCurrent, next, decidedAt,
	)

	req, err := scanRequest(row)
	if errors.Is(err, pgx.ErrNoRows) {
		// Either the id doesn't exist at all, or it exists but wasn't in
		// the expected state - distinguish the two for a clearer error.
		if _, findErr := r.FindByID(ctx, id); errors.Is(findErr, ErrRequestNotFound) {
			return nil, ErrRequestNotFound
		}
		return nil, ErrNotPending
	}
	if err != nil {
		return nil, fmt.Errorf("update access request status: %w", err)
	}

	return req, nil
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanRequest(row rowScanner) (*Request, error) {
	req := &Request{}
	var rawCategories []string

	err := row.Scan(
		&req.ID, &req.HospitalID, &req.PatientID, &req.RequestedBy, &req.Purpose,
		&rawCategories, &req.Status, &req.DecidedAt, &req.CreatedAt, &req.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	req.Categories = toCategorySlice(rawCategories)
	return req, nil
}

func scanRequestRows(rows pgx.Rows) ([]*Request, error) {
	var requests []*Request
	for rows.Next() {
		req, err := scanRequest(rows)
		if err != nil {
			return nil, fmt.Errorf("scan access request: %w", err)
		}
		requests = append(requests, req)
	}
	return requests, rows.Err()
}

func toStringSlice(categories []datacategory.Category) []string {
	out := make([]string, len(categories))
	for i, c := range categories {
		out[i] = string(c)
	}
	return out
}

func toCategorySlice(raw []string) []datacategory.Category {
	out := make([]datacategory.Category, len(raw))
	for i, v := range raw {
		out[i] = datacategory.Category(v)
	}
	return out
}
