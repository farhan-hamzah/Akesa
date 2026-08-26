package audit

import (
	"context"
	"encoding/json"

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

func (r *Repository) Create(ctx context.Context, logEntry *AuditLog) error {

	metadata, err := json.Marshal(logEntry.Metadata)

	if err != nil {
		return err
	}

	_, err = r.db.Exec(
		ctx,
		`
		INSERT INTO audit_logs (
			id,
			user_id,
			patient_id,
			action,
			resource_type,
			resource_id,
			metadata,
			ip_address,
			user_agent
		)
		VALUES (
			$1,
			$2,
			$3,
			$4,
			$5,
			$6,
			$7,
			$8,
			$9
		)
		`,
		logEntry.ID,
		logEntry.UserID,
		logEntry.PatientID,
		logEntry.Action,
		logEntry.ResourceType,
		logEntry.ResourceID,
		metadata,
		logEntry.IPAddress,
		logEntry.UserAgent,
	)

	return err
}
