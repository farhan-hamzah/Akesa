package hospital

import (
	"context"
	"crypto/rand"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/google/uuid"
)

type UserRoleUpdater interface {
	UpdateRole(ctx context.Context, userID uuid.UUID, role string) error
}

type Service struct {
	repository       *Repository
	users            UserRoleUpdater
	magicLinkBaseURL string
}

func NewService(repository *Repository, users UserRoleUpdater, magicLinkBaseURL string) *Service {
	if strings.TrimSpace(magicLinkBaseURL) == "" {
		magicLinkBaseURL = "akesa://register-hospital"
	}
	return &Service{
		repository:       repository,
		users:            users,
		magicLinkBaseURL: strings.TrimRight(magicLinkBaseURL, "/"),
	}
}

func (s *Service) Create(ctx context.Context, in HospitalInput) (*Hospital, error) {
	return s.repository.Create(ctx, in)
}

func (s *Service) Get(ctx context.Context, id uuid.UUID) (*Hospital, error) {
	return s.repository.FindByID(ctx, id)
}

func (s *Service) List(ctx context.Context) ([]*Hospital, error) {
	return s.repository.List(ctx)
}

func (s *Service) Verify(ctx context.Context, id uuid.UUID) (*Hospital, error) {
	return s.repository.UpdateStatus(ctx, id, StatusVerified)
}

func (s *Service) Activate(ctx context.Context, id uuid.UUID) (*Hospital, error) {
	h, err := s.repository.UpdateStatus(ctx, id, StatusActive)
	if err != nil {
		return nil, err
	}

	// Promote PIC user to HOSPITAL_STAFF upon hospital activation by Admin
	if h.PICUserID != nil && s.users != nil {
		_ = s.users.UpdateRole(ctx, *h.PICUserID, "HOSPITAL_STAFF")
	}

	return h, nil
}

func (s *Service) Deactivate(ctx context.Context, id uuid.UUID) (*Hospital, error) {
	return s.repository.UpdateStatus(ctx, id, StatusInactive)
}

func (s *Service) AddStaff(ctx context.Context, hospitalID, userID uuid.UUID, fullName, position string) (*Staff, error) {
	hospital, err := s.repository.FindByID(ctx, hospitalID)
	if err != nil {
		return nil, err
	}
	if hospital.Status != StatusActive && hospital.Status != StatusVerified {
		return nil, ErrHospitalNotActive
	}

	staff, err := s.repository.CreateStaff(ctx, userID, hospitalID, strings.TrimSpace(fullName), strings.TrimSpace(position))
	if err != nil {
		return nil, err
	}

	if err := s.users.UpdateRole(ctx, userID, "HOSPITAL_STAFF"); err != nil {
		return staff, err
	}

	return staff, nil
}

func (s *Service) GetStaffByUserID(ctx context.Context, userID uuid.UUID) (*Staff, error) {
	return s.repository.FindStaffByUserID(ctx, userID)
}

// GenerateInvitation creates a new one-time invitation key and magic link for a prospective hospital.
func (s *Service) GenerateInvitation(ctx context.Context, adminUserID uuid.UUID, req GenerateInvitationRequest) (*GenerateInvitationResponse, error) {
	email, expiresInHours, err := req.Validate()
	if err != nil {
		return nil, err
	}

	keyCode, err := generateSecureKeyCode()
	if err != nil {
		return nil, fmt.Errorf("generate key code: %w", err)
	}

	now := time.Now()
	expiresAt := now.Add(time.Duration(expiresInHours) * time.Hour)

	inv := &Invitation{
		ID:          uuid.New(),
		KeyCode:     keyCode,
		TargetEmail: email,
		CreatedBy:   adminUserID,
		IsUsed:      false,
		ExpiresAt:   expiresAt,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if err := s.repository.CreateInvitation(ctx, inv); err != nil {
		return nil, err
	}

	magicLink := fmt.Sprintf("%s?key=%s", s.magicLinkBaseURL, keyCode)

	return &GenerateInvitationResponse{
		KeyCode:     keyCode,
		MagicLink:   magicLink,
		TargetEmail: email,
		ExpiresAt:   expiresAt,
	}, nil
}

// ValidateInvitation checks if an invitation key exists, has not expired, and has not been used.
func (s *Service) ValidateInvitation(ctx context.Context, keyCode string) (*ValidateInvitationResponse, error) {
	keyCode = strings.TrimSpace(keyCode)
	if keyCode == "" {
		return nil, ErrInvitationNotFound
	}

	inv, err := s.repository.FindInvitationByKey(ctx, keyCode)
	if err != nil {
		return nil, err
	}

	if inv.IsUsed {
		return nil, ErrInvitationAlreadyUsed
	}
	if inv.IsExpired() {
		return nil, ErrInvitationExpired
	}

	return &ValidateInvitationResponse{
		Valid:       true,
		TargetEmail: inv.TargetEmail,
		ExpiresAt:   inv.ExpiresAt,
	}, nil
}

// RegisterWithKey registers a new hospital application using an invitation key.
func (s *Service) RegisterWithKey(ctx context.Context, userID uuid.UUID, req RegisterWithKeyRequest) (*HospitalApplication, error) {
	keyCode, in, picPos, picName, err := req.Validate()
	if err != nil {
		return nil, err
	}

	return s.repository.CreateApplicationTx(ctx, keyCode, in, userID, picPos, picName)
}

// GetMyApplication returns the most recent hospital application submitted by the logged-in user.
func (s *Service) GetMyApplication(ctx context.Context, userID uuid.UUID) (*HospitalApplication, error) {
	return s.repository.FindApplicationByApplicantID(ctx, userID)
}

// ResubmitApplication updates and resubmits an application that is currently in REVISION_REQUIRED status.
func (s *Service) ResubmitApplication(ctx context.Context, userID uuid.UUID, req UpdateApplicationRequest) (*HospitalApplication, error) {
	in, picPos, picName, err := req.Validate()
	if err != nil {
		return nil, err
	}

	return s.repository.UpdateApplicationContentTx(ctx, userID, in, picPos, picName)
}

// ListApplications returns all hospital applications, optionally filtered by status (Admin).
func (s *Service) ListApplications(ctx context.Context, status ApplicationStatus) ([]*HospitalApplication, error) {
	return s.repository.ListApplications(ctx, status)
}

// GetApplication returns an application by its ID (Admin).
func (s *Service) GetApplication(ctx context.Context, id uuid.UUID) (*HospitalApplication, error) {
	return s.repository.FindApplicationByID(ctx, id)
}

// RequestRevision transitions an application to REVISION_REQUIRED with admin feedback notes.
func (s *Service) RequestRevision(ctx context.Context, id uuid.UUID, notes string) (*HospitalApplication, error) {
	notes = strings.TrimSpace(notes)
	if notes == "" {
		return nil, fieldErr("notes", "catatan revisi wajib diisi")
	}
	return s.repository.UpdateApplicationStatus(ctx, id, ApplicationRevisionRequired, notes)
}

// RejectApplication transitions an application to REJECTED with reason notes.
func (s *Service) RejectApplication(ctx context.Context, id uuid.UUID, reason string) (*HospitalApplication, error) {
	reason = strings.TrimSpace(reason)
	if reason == "" {
		return nil, fieldErr("notes", "alasan penolakan wajib diisi")
	}
	return s.repository.UpdateApplicationStatus(ctx, id, ApplicationRejected, reason)
}

// ApproveApplication approves an application, copies it to hospitals table as ACTIVE,
// creates the staff record, and promotes the applicant user to HOSPITAL_STAFF.
func (s *Service) ApproveApplication(ctx context.Context, id uuid.UUID) (*Hospital, error) {
	h, app, err := s.repository.ApproveApplicationTx(ctx, id)
	if err != nil {
		return nil, err
	}

	// Promote applicant user to HOSPITAL_STAFF upon approval
	if s.users != nil {
		_ = s.users.UpdateRole(ctx, app.ApplicantUserID, "HOSPITAL_STAFF")
	}

	return h, nil
}

// generateSecureKeyCode creates a human-friendly unique key code (e.g. AKESA-7X9K-3B2M)
func generateSecureKeyCode() (string, error) {
	const charset = "23456789ABCDEFGHJKLMNPQRSTUVWXYZ"
	segment1 := make([]byte, 4)
	segment2 := make([]byte, 4)

	for i := range segment1 {
		num, err := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
		if err != nil {
			return "", err
		}
		segment1[i] = charset[num.Int64()]
	}
	for i := range segment2 {
		num, err := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
		if err != nil {
			return "", err
		}
		segment2[i] = charset[num.Int64()]
	}

	return fmt.Sprintf("AKESA-%s-%s", string(segment1), string(segment2)), nil
}
