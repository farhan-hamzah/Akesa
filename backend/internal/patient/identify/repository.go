package identity

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

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

const selectColumns = `
	id,
	patient_id,
	status,
	document_type,
	document_storage_key,
	selfie_storage_key,
	extracted_nik,
	extracted_full_name,
	extracted_date_of_birth,
	extracted_gender,
	document_status,
	liveness_status,
	face_match_status,
	face_match_score,
	provider,
	provider_reference,
	failure_reason,
	verified_at,
	expires_at,
	created_at,
	updated_at
`

func (r *Repository) Create(
	ctx context.Context,
	verification *Verification,
) (*Verification, error) {
	query := `
		INSERT INTO identity_verifications (
			id,
			patient_id,
			status,
			document_type,
			expires_at
		)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING ` + selectColumns

	created := &Verification{}

	err := r.scanRow(
		r.db.QueryRow(
			ctx,
			query,
			verification.ID,
			verification.PatientID,
			verification.Status,
			verification.DocumentType,
			verification.ExpiresAt,
		),
		created,
	)
	if err != nil {
		return nil, fmt.Errorf("create identity verification: %w", err)
	}

	return created, nil
}

func (r *Repository) FindByID(
	ctx context.Context,
	id uuid.UUID,
) (*Verification, error) {
	query := `
		SELECT ` + selectColumns + `
		FROM identity_verifications
		WHERE id = $1
	`

	verification := &Verification{}

	err := r.scanRow(
		r.db.QueryRow(ctx, query, id),
		verification,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrVerificationNotFound
		}

		return nil, fmt.Errorf("find identity verification: %w", err)
	}

	return verification, nil
}

func (r *Repository) FindLatestByPatientID(
	ctx context.Context,
	patientID uuid.UUID,
) (*Verification, error) {
	const query = `
        SELECT
            id,
            patient_id,
            status,
            document_type,
            document_storage_key,
			selfie_storage_key,
            extracted_nik,
            extracted_full_name,
            extracted_date_of_birth,
            extracted_gender,
            document_status,
            liveness_status,
            face_match_status,
            face_match_score,
            provider,
            provider_reference,
            failure_reason,
            verified_at,
            expires_at,
            created_at,
            updated_at
        FROM identity_verifications
        WHERE patient_id = $1
        ORDER BY created_at DESC
        LIMIT 1
    `

	var verification Verification

	err := r.db.QueryRow(ctx, query, patientID).Scan(
		&verification.ID,
		&verification.PatientID,
		&verification.Status,
		&verification.DocumentType,
		&verification.DocumentStorageKey,
		&verification.SelfieStorageKey,
		&verification.ExtractedNIK,
		&verification.ExtractedFullName,
		&verification.ExtractedDateOfBirth,
		&verification.ExtractedGender,
		&verification.DocumentStatus,
		&verification.LivenessStatus,
		&verification.FaceMatchStatus,
		&verification.FaceMatchScore,
		&verification.Provider,
		&verification.ProviderReference,
		&verification.FailureReason,
		&verification.VerifiedAt,
		&verification.ExpiresAt,
		&verification.CreatedAt,
		&verification.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrVerificationNotFound
		}

		return nil, fmt.Errorf("find latest identity verification: %w", err)
	}

	return &verification, nil
}

func (r *Repository) UpdateStatus(
	ctx context.Context,
	id uuid.UUID,
	status VerificationStatus,
	failureReason *string,
	verifiedAt *time.Time,
) (*Verification, error) {
	query := `
		UPDATE identity_verifications
		SET
			status = $2,
			failure_reason = $3,
			verified_at = $4,
			updated_at = NOW()
		WHERE id = $1
		RETURNING ` + selectColumns

	verification := &Verification{}

	err := r.scanRow(
		r.db.QueryRow(
			ctx,
			query,
			id,
			status,
			failureReason,
			verifiedAt,
		),
		verification,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrVerificationNotFound
		}

		return nil, fmt.Errorf("update identity verification status: %w", err)
	}

	return verification, nil
}

func (r *Repository) scanRow(
	row pgx.Row,
	verification *Verification,
) error {
	err := row.Scan(
		&verification.ID,
		&verification.PatientID,
		&verification.Status,
		&verification.DocumentType,
		&verification.DocumentStorageKey,
		&verification.SelfieStorageKey,
		&verification.ExtractedNIK,
		&verification.ExtractedFullName,
		&verification.ExtractedDateOfBirth,
		&verification.ExtractedGender,
		&verification.DocumentStatus,
		&verification.LivenessStatus,
		&verification.FaceMatchStatus,
		&verification.FaceMatchScore,
		&verification.Provider,
		&verification.ProviderReference,
		&verification.FailureReason,
		&verification.VerifiedAt,
		&verification.ExpiresAt,
		&verification.CreatedAt,
		&verification.UpdatedAt,
	)

	return err
}

func (r *Repository) UpdateDocument(
	ctx context.Context,
	id uuid.UUID,
	storageKey string,
	status VerificationStatus,
) (*Verification, error) {
	const query = `
    UPDATE identity_verifications
    SET
        document_storage_key = $2,
        status = $3,
        updated_at = NOW()
    WHERE id = $1
      AND status IN ('PENDING', 'PROCESSING')
      AND $2 IS NOT NULL
      AND $2 <> ''
    RETURNING ` + selectColumns

	verification := &Verification{}

	err := r.scanRow(
		r.db.QueryRow(ctx, query, id, storageKey, status),
		verification,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrVerificationNotFound
		}

		return nil, fmt.Errorf(
			"update identity verification document: %w",
			err,
		)
	}

	return verification, nil
}

func (r *Repository) UpdateSelfie(
	ctx context.Context,
	id uuid.UUID,
	storageKey *string,
	status VerificationStatus,
) (*Verification, error) {
	const query = `
        UPDATE identity_verifications
        SET
            selfie_storage_key = $2,
            status = $3,
            updated_at = NOW()
        WHERE id = $1
          AND document_storage_key IS NOT NULL
          AND document_storage_key <> ''
          AND status IN ('PENDING', 'PROCESSING')
          AND $2 IS NOT NULL
          AND $2 <> ''
        RETURNING ` + selectColumns

	verification := &Verification{}

	err := r.scanRow(
		r.db.QueryRow(ctx, query, id, storageKey, status),
		verification,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrVerificationNotFound
		}

		return nil, fmt.Errorf(
			"update identity verification selfie: %w",
			err,
		)
	}

	return verification, nil
}

func (r *Repository) MarkVerified(
	ctx context.Context,
	verificationID uuid.UUID,
	verifiedAt time.Time,
) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin verification transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	var patientID uuid.UUID

	const updateVerification = `
        UPDATE identity_verifications
        SET
            status = 'VERIFIED',
            verified_at = $2,
            failure_reason = NULL,
            updated_at = NOW()
        WHERE id = $1
          AND status = 'MANUAL_REVIEW'
          AND document_storage_key IS NOT NULL
          AND document_storage_key <> ''
          AND selfie_storage_key IS NOT NULL
          AND selfie_storage_key <> ''
        RETURNING patient_id
    `

	err = tx.QueryRow(
		ctx,
		updateVerification,
		verificationID,
		verifiedAt,
	).Scan(&patientID)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return fmt.Errorf(
				"verification not found or not eligible for approval: %w",
				ErrVerificationNotFound,
			)
		}

		return fmt.Errorf("mark verification as verified: %w", err)
	}

	const updatePatient = `
    UPDATE patient_profiles
    SET
        identity_verification_status = 'VERIFIED',
        profile_completed = (
            NULLIF(BTRIM(full_name), '') IS NOT NULL
            AND NULLIF(BTRIM(nik), '') IS NOT NULL
            AND date_of_birth IS NOT NULL
            AND NULLIF(BTRIM(gender), '') IS NOT NULL
            AND NULLIF(BTRIM(phone_number), '') IS NOT NULL
            AND NULLIF(BTRIM(address), '') IS NOT NULL
        ),
        updated_at = NOW()
    WHERE id = $1
`

	result, err := tx.Exec(ctx, updatePatient, patientID)
	if err != nil {
		return fmt.Errorf("update patient verification status: %w", err)
	}

	if result.RowsAffected() != 1 {
		return fmt.Errorf(
			"patient profile not found for verification %s",
			verificationID,
		)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit verification transaction: %w", err)
	}

	return nil
}

func (r *Repository) MarkRejected(
	ctx context.Context,
	verificationID uuid.UUID,
	reason string,
) error {
	reason = strings.TrimSpace(reason)
	if reason == "" {
		return errors.New("rejection reason is required")
	}

	const query = `
        UPDATE identity_verifications
        SET
            status = 'DOCUMENT_REJECTED',
            failure_reason = $2,
            verified_at = NULL,
            updated_at = NOW()
        WHERE id = $1
          AND status = 'MANUAL_REVIEW'
          AND document_storage_key IS NOT NULL
          AND document_storage_key <> ''
          AND selfie_storage_key IS NOT NULL
          AND selfie_storage_key <> ''
    `

	result, err := r.db.Exec(
		ctx,
		query,
		verificationID,
		reason,
	)
	if err != nil {
		return fmt.Errorf("mark verification as rejected: %w", err)
	}

	if result.RowsAffected() != 1 {
		return fmt.Errorf(
			"verification not found or not eligible for rejection: %w",
			ErrVerificationNotFound,
		)
	}

	return nil
}

func (r *Repository) ListPendingReview(
	ctx context.Context,
) ([]AdminIdentityQueueItem, error) {
	const query = `
		SELECT
			iv.id,
			iv.patient_id,
			iv.status,
			iv.document_type,
			iv.document_status,
			iv.liveness_status,
			iv.face_match_status,
			iv.created_at,
			pp.patient_code,
			pp.full_name,
			'****************' AS masked_nik,
			pp.date_of_birth::text,
			pp.gender
		FROM identity_verifications iv
		INNER JOIN patient_profiles pp
			ON pp.id = iv.patient_id
		WHERE iv.status = 'MANUAL_REVIEW'
			AND iv.document_storage_key IS NOT NULL
			AND iv.document_storage_key <> ''
			AND iv.selfie_storage_key IS NOT NULL
			AND iv.selfie_storage_key <> ''
		ORDER BY iv.created_at ASC
	`

	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]AdminIdentityQueueItem, 0)

	for rows.Next() {
		var item AdminIdentityQueueItem
		var createdAt time.Time
		var fullName string
		var dateOfBirth string
		var gender string

		if err := rows.Scan(
			&item.ID,
			&item.PatientID,
			&item.Status,
			&item.DocumentType,
			&item.DocumentStatus,
			&item.LivenessStatus,
			&item.FaceMatchStatus,
			&createdAt,
			&item.Patient.PatientCode,
			&fullName,
			&item.Patient.MaskedNIK,
			&dateOfBirth,
			&gender,
		); err != nil {
			return nil, err
		}

		item.CreatedAt = createdAt.Format(time.RFC3339)
		item.Patient.FullName = &fullName
		item.Patient.DateOfBirth = &dateOfBirth
		item.Patient.Gender = &gender

		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return items, nil
}
