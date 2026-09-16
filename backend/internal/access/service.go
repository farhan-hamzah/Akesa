package access

import (
	"context"
	"errors"
	"log"

	"github.com/farhan-hamzah/Akesa/backend/internal/hospital"
	"github.com/farhan-hamzah/Akesa/backend/internal/patient"
	"github.com/google/uuid"
)

// PatientLookup is the slice of patient.Service this package depends on.
type PatientLookup interface {
	GetByPatientCode(ctx context.Context, patientCode string) (*patient.Profile, error)
	GetByID(ctx context.Context, id uuid.UUID) (*patient.Profile, error)
	ComputeHash(profile *patient.Profile) string
}

// StaffLookup is the slice of hospital.Service this package depends on, to
// resolve which hospital a calling staff member belongs to.
type StaffLookup interface {
	GetStaffByUserID(ctx context.Context, userID uuid.UUID) (*hospital.Staff, error)
}

// AuditRecorder is the slice of audit.Service this package depends on.
type AuditRecorder interface {
	RecordAccessEvent(ctx context.Context, entityType string, entityID uuid.UUID, action string, actorID uuid.UUID, payload map[string]any) error
	VerifyProfileOnChain(ctx context.Context, patientID uuid.UUID, currentHash string) (bool, string, error)
	GetLatestProfileHash(ctx context.Context, patientID uuid.UUID) (string, error)
}

const entityTypeAccessRequest = "ACCESS_REQUEST"

// Audit action name literals mirrored from the audit package to avoid a
// hard dependency on it - see audit.Action* constants for the canonical
// definitions used when writing the chain.
const (
	actionRequested = "ACCESS_REQUESTED"
	actionApproved  = "ACCESS_APPROVED"
	actionRejected  = "ACCESS_REJECTED"
	actionRevoked   = "ACCESS_REVOKED"
	actionAccessed  = "DATA_ACCESSED"
)

type Service struct {
	repository *Repository
	patients   PatientLookup
	staff      StaffLookup
	audit      AuditRecorder
}

func NewService(repository *Repository, patients PatientLookup, staff StaffLookup, audit AuditRecorder) *Service {
	return &Service{repository: repository, patients: patients, staff: staff, audit: audit}
}

// CreateRequest - a hospital staff member requests access to a patient's
// data. The staff's own hospital (never a client-supplied hospital id) is
// used, so a staff member can never request access on behalf of another
// hospital.
func (s *Service) CreateRequest(ctx context.Context, staffUserID uuid.UUID, in CreateRequestInput) (*Request, error) {
	staff, err := s.staff.GetStaffByUserID(ctx, staffUserID)
	if err != nil {
		return nil, err
	}

	patientProfile, err := s.patients.GetByPatientCode(ctx, in.PatientCode)
	if err != nil {
		if errors.Is(err, patient.ErrProfileNotFound) {
			return nil, ErrPatientNotFound
		}
		return nil, err
	}

	req, err := s.repository.Create(ctx, staff.HospitalID, patientProfile.ID, staffUserID, in.Purpose, in.Categories)
	if err != nil {
		return nil, err
	}

	s.recordEvent(ctx, req.ID, actionRequested, staffUserID, map[string]any{
		"hospitalId": staff.HospitalID,
		"categories": in.Categories,
	})

	return req, nil
}

func (s *Service) ListForPatient(ctx context.Context, patientID uuid.UUID) ([]*Request, error) {
	return s.repository.ListByPatient(ctx, patientID)
}

func (s *Service) ListForHospitalStaff(ctx context.Context, staffUserID uuid.UUID) ([]*Request, error) {
	staff, err := s.staff.GetStaffByUserID(ctx, staffUserID)
	if err != nil {
		return nil, err
	}
	return s.repository.ListByHospital(ctx, staff.HospitalID)
}

// Approve - only the owning patient may approve their own request.
func (s *Service) Approve(ctx context.Context, patientID, requestID uuid.UUID) (*Request, error) {
	req, err := s.ownedPendingRequest(ctx, patientID, requestID)
	if err != nil {
		return nil, err
	}

	updated, err := s.repository.UpdateStatus(ctx, req.ID, StatusPending, StatusApproved)
	if err != nil {
		return nil, err
	}

	s.recordEvent(ctx, updated.ID, actionApproved, patientID, map[string]any{"categories": updated.Categories})
	return updated, nil
}

func (s *Service) Reject(ctx context.Context, patientID, requestID uuid.UUID) (*Request, error) {
	req, err := s.ownedPendingRequest(ctx, patientID, requestID)
	if err != nil {
		return nil, err
	}

	updated, err := s.repository.UpdateStatus(ctx, req.ID, StatusPending, StatusRejected)
	if err != nil {
		return nil, err
	}

	s.recordEvent(ctx, updated.ID, actionRejected, patientID, nil)
	return updated, nil
}

// Revoke - a patient can revoke access they previously approved, at any
// time. This is the "mencabut akses" right from the SRS.
func (s *Service) Revoke(ctx context.Context, patientID, requestID uuid.UUID) (*Request, error) {
	req, err := s.repository.FindByID(ctx, requestID)
	if err != nil {
		return nil, err
	}
	if req.PatientID != patientID {
		return nil, ErrNotOwner
	}
	if req.Status != StatusApproved {
		return nil, ErrNotApproved
	}

	updated, err := s.repository.UpdateStatus(ctx, req.ID, StatusApproved, StatusRevoked)
	if err != nil {
		return nil, err
	}

	s.recordEvent(ctx, updated.ID, actionRevoked, patientID, nil)
	return updated, nil
}

// FetchApprovedData is what a hospital staff member calls to read the
// patient data they've been granted - it re-checks, on every call, that
// the request belongs to the caller's hospital and is currently approved,
// and it logs a DATA_ACCESSED audit event every time data is read.
func (s *Service) FetchApprovedData(ctx context.Context, staffUserID, requestID uuid.UUID) (map[string]any, error) {
	staff, err := s.staff.GetStaffByUserID(ctx, staffUserID)
	if err != nil {
		return nil, err
	}

	req, err := s.repository.FindByID(ctx, requestID)
	if err != nil {
		return nil, err
	}
	if req.HospitalID != staff.HospitalID {
		return nil, ErrNotOwner
	}
	if !req.IsGrantedNow() {
		return nil, ErrNotApproved
	}

	patientProfile, err := s.patients.GetByID(ctx, req.PatientID)
	if err != nil {
		return nil, err
	}

	// FR-BC-03 & FR-BC-04: Calculate real-time hash of patient data and verify
	// against blockchain ledger before disclosing data to hospital staff.
	currentHash := s.patients.ComputeHash(patientProfile)
	if s.audit != nil {
		valid, recordedHash, err := s.audit.VerifyProfileOnChain(ctx, req.PatientID, currentHash)
		if err != nil {
			log.Printf("[integrity] blockchain verify error (%v), falling back to local audit ledger", err)
			latestHash, lerr := s.audit.GetLatestProfileHash(ctx, req.PatientID)
			if lerr == nil && latestHash != "" {
				valid = (latestHash == currentHash)
				recordedHash = latestHash
			}
		}

		if !valid && recordedHash != "" {
			log.Printf("[SECURITY ALERT - FR-BC-04] Patient data integrity violation! PatientID=%s currentHash=%s recordedHash=%s",
				req.PatientID, currentHash, recordedHash)
			s.recordEvent(ctx, req.ID, "INTEGRITY_VIOLATION_BLOCKED", staffUserID, map[string]any{
				"error":        "hash_mismatch",
				"currentHash":  currentHash,
				"recordedHash": recordedHash,
			})
			return nil, ErrDataTampered
		}
	}

	view := patientProfile.FilteredView(req.Categories)

	s.recordEvent(ctx, req.ID, actionAccessed, staffUserID, map[string]any{
		"categories": req.Categories,
		"dataHash":   currentHash,
	})

	return view, nil
}

func (s *Service) ownedPendingRequest(ctx context.Context, patientID, requestID uuid.UUID) (*Request, error) {
	req, err := s.repository.FindByID(ctx, requestID)
	if err != nil {
		return nil, err
	}
	if req.PatientID != patientID {
		return nil, ErrNotOwner
	}
	if req.Status != StatusPending {
		return nil, ErrNotPending
	}
	return req, nil
}

func (s *Service) recordEvent(ctx context.Context, requestID uuid.UUID, action string, actorID uuid.UUID, payload map[string]any) {
	if s.audit == nil {
		return
	}
	// An audit logging failure must never block or roll back a decision
	// the patient/staff already made - the underlying write already
	// committed. In production this should also push to a metric/alert,
	// not just a log line.
	if err := s.audit.RecordAccessEvent(ctx, entityTypeAccessRequest, requestID, action, actorID, payload); err != nil {
		log.Printf("audit: failed to record %s for access_request=%s: %v", action, requestID, err)
	}
}
