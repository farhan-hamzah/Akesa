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
		`INSERT INTO hospitals (id, name, address, phone, email, registration_number, npwp, license_document_url, pic_user_id, status)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)
		 RETURNING id, name, address, phone, email, registration_number, COALESCE(npwp, ''), COALESCE(license_document_url, ''), pic_user_id, status, created_at, updated_at`,
		uuid.New(), in.Name, in.Address, in.Phone, in.Email, in.RegistrationNumber, in.NPWP, in.LicenseDocumentURL, in.PICUserID, StatusPending,
	).Scan(&h.ID, &h.Name, &h.Address, &h.Phone, &h.Email, &h.RegistrationNumber, &h.NPWP, &h.LicenseDocumentURL, &h.PICUserID, &h.Status, &h.CreatedAt, &h.UpdatedAt)

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
		`SELECT id, name, address, phone, email, registration_number, COALESCE(npwp, ''), COALESCE(license_document_url, ''), pic_user_id, status, created_at, updated_at
		 FROM hospitals WHERE id = $1`,
		id,
	).Scan(&h.ID, &h.Name, &h.Address, &h.Phone, &h.Email, &h.RegistrationNumber, &h.NPWP, &h.LicenseDocumentURL, &h.PICUserID, &h.Status, &h.CreatedAt, &h.UpdatedAt)

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
		`SELECT id, name, address, phone, email, registration_number, COALESCE(npwp, ''), COALESCE(license_document_url, ''), pic_user_id, status, created_at, updated_at
		 FROM hospitals ORDER BY created_at DESC`,
	)
	if err != nil {
		return nil, fmt.Errorf("list hospitals: %w", err)
	}
	defer rows.Close()

	var hospitals []*Hospital
	for rows.Next() {
		h := &Hospital{}
		if err := rows.Scan(&h.ID, &h.Name, &h.Address, &h.Phone, &h.Email, &h.RegistrationNumber, &h.NPWP, &h.LicenseDocumentURL, &h.PICUserID, &h.Status, &h.CreatedAt, &h.UpdatedAt); err != nil {
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
		 RETURNING id, name, address, phone, email, registration_number, COALESCE(npwp, ''), COALESCE(license_document_url, ''), pic_user_id, status, created_at, updated_at`,
		id, status,
	).Scan(&h.ID, &h.Name, &h.Address, &h.Phone, &h.Email, &h.RegistrationNumber, &h.NPWP, &h.LicenseDocumentURL, &h.PICUserID, &h.Status, &h.CreatedAt, &h.UpdatedAt)

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

// CreateInvitation inserts a new hospital registration invitation.
func (r *Repository) CreateInvitation(ctx context.Context, inv *Invitation) error {
	_, err := r.db.Exec(ctx,
		`INSERT INTO hospital_invitations (id, key_code, target_email, created_by, is_used, expires_at, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		inv.ID, inv.KeyCode, inv.TargetEmail, inv.CreatedBy, inv.IsUsed, inv.ExpiresAt, inv.CreatedAt, inv.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("insert invitation: %w", err)
	}
	return nil
}

// FindInvitationByKey retrieves an invitation by its unique key code.
func (r *Repository) FindInvitationByKey(ctx context.Context, keyCode string) (*Invitation, error) {
	inv := &Invitation{}
	err := r.db.QueryRow(ctx,
		`SELECT id, key_code, target_email, created_by, used_by, is_used, expires_at, created_at, updated_at
		 FROM hospital_invitations WHERE key_code = $1`,
		keyCode,
	).Scan(&inv.ID, &inv.KeyCode, &inv.TargetEmail, &inv.CreatedBy, &inv.UsedBy, &inv.IsUsed, &inv.ExpiresAt, &inv.CreatedAt, &inv.UpdatedAt)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrInvitationNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("query invitation: %w", err)
	}
	return inv, nil
}

// CreateApplicationTx atomically validates the invitation, inserts the hospital_application
// with status PENDING_REVIEW, and marks the invitation as used.
func (r *Repository) CreateApplicationTx(
	ctx context.Context,
	keyCode string,
	in HospitalInput,
	applicantUserID uuid.UUID,
	picPosition string,
	picFullName string,
) (*HospitalApplication, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	// 1. Lock and validate invitation
	inv := &Invitation{}
	err = tx.QueryRow(ctx,
		`SELECT id, key_code, target_email, created_by, used_by, is_used, expires_at, created_at, updated_at
		 FROM hospital_invitations
		 WHERE key_code = $1
		 FOR UPDATE`,
		keyCode,
	).Scan(&inv.ID, &inv.KeyCode, &inv.TargetEmail, &inv.CreatedBy, &inv.UsedBy, &inv.IsUsed, &inv.ExpiresAt, &inv.CreatedAt, &inv.UpdatedAt)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrInvitationNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("lock invitation: %w", err)
	}

	if inv.IsUsed {
		return nil, ErrInvitationAlreadyUsed
	}
	if inv.IsExpired() {
		return nil, ErrInvitationExpired
	}

	// 2. Insert into hospital_applications (status: PENDING_REVIEW)
	row := tx.QueryRow(ctx,
		`INSERT INTO hospital_applications (
			id, invitation_id, applicant_user_id,
			name, address, phone, email, registration_number,
			npwp, license_document_url, pic_full_name, pic_position,
			status, admin_notes, created_at, updated_at
		 ) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,'PENDING_REVIEW','',NOW(),NOW())
		 RETURNING id, invitation_id, applicant_user_id, name, address, phone, email, registration_number,
		           COALESCE(npwp, ''), COALESCE(license_document_url, ''), pic_full_name, pic_position,
		           status, COALESCE(admin_notes, ''), created_at, updated_at`,
		uuid.New(), inv.ID, applicantUserID,
		in.Name, in.Address, in.Phone, in.Email, in.RegistrationNumber,
		in.NPWP, in.LicenseDocumentURL, picFullName, picPosition,
	)

	app, err := scanApplication(row)
	if err != nil {
		return nil, fmt.Errorf("insert hospital application: %w", err)
	}

	// 3. Mark invitation as used
	_, err = tx.Exec(ctx,
		`UPDATE hospital_invitations
		 SET is_used = TRUE, used_by = $2, updated_at = NOW()
		 WHERE id = $1`,
		inv.ID, applicantUserID,
	)
	if err != nil {
		return nil, fmt.Errorf("update invitation used: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit tx: %w", err)
	}

	return app, nil
}

// FindApplicationByApplicantID finds the most recent application for a given user.
func (r *Repository) FindApplicationByApplicantID(ctx context.Context, applicantUserID uuid.UUID) (*HospitalApplication, error) {
	row := r.db.QueryRow(ctx,
		`SELECT id, invitation_id, applicant_user_id, name, address, phone, email, registration_number,
		        COALESCE(npwp, ''), COALESCE(license_document_url, ''), pic_full_name, pic_position,
		        status, COALESCE(admin_notes, ''), created_at, updated_at
		 FROM hospital_applications
		 WHERE applicant_user_id = $1
		 ORDER BY created_at DESC
		 LIMIT 1`,
		applicantUserID,
	)

	app, err := scanApplication(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrApplicationNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("query application by applicant: %w", err)
	}
	return app, nil
}

// FindApplicationByID finds an application by its UUID.
func (r *Repository) FindApplicationByID(ctx context.Context, id uuid.UUID) (*HospitalApplication, error) {
	row := r.db.QueryRow(ctx,
		`SELECT id, invitation_id, applicant_user_id, name, address, phone, email, registration_number,
		        COALESCE(npwp, ''), COALESCE(license_document_url, ''), pic_full_name, pic_position,
		        status, COALESCE(admin_notes, ''), created_at, updated_at
		 FROM hospital_applications
		 WHERE id = $1`,
		id,
	)

	app, err := scanApplication(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrApplicationNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("query application by id: %w", err)
	}
	return app, nil
}

// ListApplications lists hospital applications, optionally filtered by status.
func (r *Repository) ListApplications(ctx context.Context, status ApplicationStatus) ([]*HospitalApplication, error) {
	var rows pgx.Rows
	var err error

	if status != "" {
		rows, err = r.db.Query(ctx,
			`SELECT id, invitation_id, applicant_user_id, name, address, phone, email, registration_number,
			        COALESCE(npwp, ''), COALESCE(license_document_url, ''), pic_full_name, pic_position,
			        status, COALESCE(admin_notes, ''), created_at, updated_at
			 FROM hospital_applications
			 WHERE status = $1
			 ORDER BY created_at DESC`,
			status,
		)
	} else {
		rows, err = r.db.Query(ctx,
			`SELECT id, invitation_id, applicant_user_id, name, address, phone, email, registration_number,
			        COALESCE(npwp, ''), COALESCE(license_document_url, ''), pic_full_name, pic_position,
			        status, COALESCE(admin_notes, ''), created_at, updated_at
			 FROM hospital_applications
			 ORDER BY created_at DESC`,
		)
	}

	if err != nil {
		return nil, fmt.Errorf("list applications: %w", err)
	}
	defer rows.Close()

	var apps []*HospitalApplication
	for rows.Next() {
		app, err := scanApplication(rows)
		if err != nil {
			return nil, fmt.Errorf("scan application: %w", err)
		}
		apps = append(apps, app)
	}

	return apps, rows.Err()
}

// UpdateApplicationContentTx updates the content of an application when status is REVISION_REQUIRED,
// and moves its status back to PENDING_REVIEW.
func (r *Repository) UpdateApplicationContentTx(
	ctx context.Context,
	applicantUserID uuid.UUID,
	in HospitalInput,
	picPosition string,
	picFullName string,
) (*HospitalApplication, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	// 1. Lock existing application
	var currentID uuid.UUID
	var currentStatus string
	err = tx.QueryRow(ctx,
		`SELECT id, status FROM hospital_applications
		 WHERE applicant_user_id = $1
		 ORDER BY created_at DESC
		 LIMIT 1
		 FOR UPDATE`,
		applicantUserID,
	).Scan(&currentID, &currentStatus)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrApplicationNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("lock application for update: %w", err)
	}

	if ApplicationStatus(currentStatus) == ApplicationApproved || ApplicationStatus(currentStatus) == ApplicationRejected {
		return nil, ErrApplicationAlreadyFinalized
	}
	if ApplicationStatus(currentStatus) != ApplicationRevisionRequired {
		return nil, ErrApplicationNotRevisionRequired
	}

	// 2. Update application content and reset status to PENDING_REVIEW
	row := tx.QueryRow(ctx,
		`UPDATE hospital_applications
		 SET name = $2, address = $3, phone = $4, email = $5, registration_number = $6,
		     npwp = $7, license_document_url = $8, pic_position = $9, pic_full_name = $10,
		     status = 'PENDING_REVIEW', updated_at = NOW()
		 WHERE id = $1
		 RETURNING id, invitation_id, applicant_user_id, name, address, phone, email, registration_number,
		           COALESCE(npwp, ''), COALESCE(license_document_url, ''), pic_full_name, pic_position,
		           status, COALESCE(admin_notes, ''), created_at, updated_at`,
		currentID, in.Name, in.Address, in.Phone, in.Email, in.RegistrationNumber,
		in.NPWP, in.LicenseDocumentURL, picPosition, picFullName,
	)

	app, err := scanApplication(row)
	if err != nil {
		return nil, fmt.Errorf("update application content: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit tx: %w", err)
	}

	return app, nil
}

// UpdateApplicationStatus updates status and admin notes of an application.
func (r *Repository) UpdateApplicationStatus(
	ctx context.Context,
	id uuid.UUID,
	newStatus ApplicationStatus,
	adminNotes string,
) (*HospitalApplication, error) {
	row := r.db.QueryRow(ctx,
		`UPDATE hospital_applications
		 SET status = $2, admin_notes = $3, updated_at = NOW()
		 WHERE id = $1 AND status NOT IN ('APPROVED', 'REJECTED')
		 RETURNING id, invitation_id, applicant_user_id, name, address, phone, email, registration_number,
		           COALESCE(npwp, ''), COALESCE(license_document_url, ''), pic_full_name, pic_position,
		           status, COALESCE(admin_notes, ''), created_at, updated_at`,
		id, newStatus, adminNotes,
	)

	app, err := scanApplication(row)
	if errors.Is(err, pgx.ErrNoRows) {
		// check if it exists at all
		_, checkErr := r.FindApplicationByID(ctx, id)
		if errors.Is(checkErr, ErrApplicationNotFound) {
			return nil, ErrApplicationNotFound
		}
		return nil, ErrApplicationAlreadyFinalized
	}
	if err != nil {
		return nil, fmt.Errorf("update application status: %w", err)
	}

	return app, nil
}

// ApproveApplicationTx atomically marks application APPROVED, inserts into hospitals table (ACTIVE),
// and creates the staff record for the applicant user.
func (r *Repository) ApproveApplicationTx(ctx context.Context, id uuid.UUID) (*Hospital, *HospitalApplication, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	// 1. Lock application
	row := tx.QueryRow(ctx,
		`SELECT id, invitation_id, applicant_user_id, name, address, phone, email, registration_number,
		        COALESCE(npwp, ''), COALESCE(license_document_url, ''), pic_full_name, pic_position,
		        status, COALESCE(admin_notes, ''), created_at, updated_at
		 FROM hospital_applications
		 WHERE id = $1
		 FOR UPDATE`,
		id,
	)
	app, err := scanApplication(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil, ErrApplicationNotFound
	}
	if err != nil {
		return nil, nil, fmt.Errorf("lock application: %w", err)
	}

	if app.Status == ApplicationApproved || app.Status == ApplicationRejected {
		return nil, nil, ErrApplicationAlreadyFinalized
	}

	// 2. Insert into hospitals table as ACTIVE
	h := &Hospital{}
	err = tx.QueryRow(ctx,
		`INSERT INTO hospitals (id, name, address, phone, email, registration_number, npwp, license_document_url, pic_user_id, status)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)
		 RETURNING id, name, address, phone, email, registration_number, COALESCE(npwp, ''), COALESCE(license_document_url, ''), pic_user_id, status, created_at, updated_at`,
		uuid.New(), app.Name, app.Address, app.Phone, app.Email, app.RegistrationNumber, app.NPWP, app.LicenseDocumentURL, app.ApplicantUserID, StatusActive,
	).Scan(&h.ID, &h.Name, &h.Address, &h.Phone, &h.Email, &h.RegistrationNumber, &h.NPWP, &h.LicenseDocumentURL, &h.PICUserID, &h.Status, &h.CreatedAt, &h.UpdatedAt)

	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == uniqueViolationCode {
			if pgErr.ConstraintName == "hospitals_email_key" {
				return nil, nil, ErrEmailTaken
			}
			if pgErr.ConstraintName == "hospitals_registration_number_key" {
				return nil, nil, ErrRegistrationTaken
			}
		}
		return nil, nil, fmt.Errorf("insert hospital on approval: %w", err)
	}

	// 3. Create staff record
	_, err = tx.Exec(ctx,
		`INSERT INTO hospital_staff (id, user_id, hospital_id, full_name, position)
		 VALUES ($1,$2,$3,$4,$5)
		 ON CONFLICT (user_id) DO UPDATE
		 SET hospital_id = EXCLUDED.hospital_id, full_name = EXCLUDED.full_name, position = EXCLUDED.position, updated_at = NOW()`,
		uuid.New(), app.ApplicantUserID, h.ID, app.PICFullName, app.PICPosition,
	)
	if err != nil {
		return nil, nil, fmt.Errorf("link staff on approval: %w", err)
	}

	// 4. Mark application as APPROVED
	_, err = tx.Exec(ctx,
		`UPDATE hospital_applications
		 SET status = 'APPROVED', updated_at = NOW()
		 WHERE id = $1`,
		app.ID,
	)
	if err != nil {
		return nil, nil, fmt.Errorf("update application to approved: %w", err)
	}
	app.Status = ApplicationApproved

	if err := tx.Commit(ctx); err != nil {
		return nil, nil, fmt.Errorf("commit tx: %w", err)
	}

	return h, app, nil
}

func scanApplication(row interface{ Scan(...any) error }) (*HospitalApplication, error) {
	app := &HospitalApplication{}
	var statusStr string
	err := row.Scan(
		&app.ID,
		&app.InvitationID,
		&app.ApplicantUserID,
		&app.Name,
		&app.Address,
		&app.Phone,
		&app.Email,
		&app.RegistrationNumber,
		&app.NPWP,
		&app.LicenseDocumentURL,
		&app.PICFullName,
		&app.PICPosition,
		&statusStr,
		&app.AdminNotes,
		&app.CreatedAt,
		&app.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	app.Status = ApplicationStatus(statusStr)
	return app, nil
}
