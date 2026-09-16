package patient

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
)

// AuditRecorder is the slice of audit.Service this package depends on.
// Defined here (the consumer) rather than in the audit package - so
// patient never has to import audit.
type AuditRecorder interface {
	RecordProfileHash(ctx context.Context, patientID uuid.UUID, profileHash string) error
}

// Hasher produces a one-way, keyed hash of a string. In production this
// is *crypto.KeyedHasher (HMAC-SHA256 with a secret key) - deliberately
// NOT a plain sha256(), because the profile includes the NIK, and a plain
// hash of a small, structured value like a NIK can be brute-forced offline
// by anyone who gets hold of the hash. See internal/crypto/hash.go.
type Hasher interface {
	Hash(value string) string
}

type Service struct {
	repository *Repository
	audit      AuditRecorder
	hasher     Hasher
}

func NewService(repository *Repository, audit AuditRecorder, hasher Hasher) *Service {
	return &Service{repository: repository, audit: audit, hasher: hasher}
}

func (s *Service) GetByUserID(ctx context.Context, userID uuid.UUID) (*Profile, error) {
	return s.repository.FindByUserID(ctx, userID)
}

func (s *Service) GetByPatientCode(ctx context.Context, patientCode string) (*Profile, error) {
	return s.repository.FindByPatientCode(ctx, patientCode)
}

func (s *Service) GetByID(ctx context.Context, id uuid.UUID) (*Profile, error) {
	return s.repository.FindByID(ctx, id)
}

func (s *Service) Create(ctx context.Context, userID uuid.UUID, in ProfileInput) (*Profile, error) {
	if _, err := s.repository.FindByUserID(ctx, userID); err == nil {
		return nil, ErrProfileExists
	} else if !errors.Is(err, ErrProfileNotFound) {
		return nil, err
	}

	profile, err := s.repository.Create(ctx, userID, in)
	if err != nil {
		return nil, err
	}

	if err := s.recordHash(ctx, profile); err != nil {
		// The profile write already succeeded; audit logging failure
		// should not roll that back, but it must be visible to operators.
		return profile, fmt.Errorf("profile saved but audit logging failed: %w", err)
	}

	return profile, nil
}

func (s *Service) Update(ctx context.Context, userID uuid.UUID, in ProfileInput) (*Profile, error) {
	profile, err := s.repository.Update(ctx, userID, in)
	if err != nil {
		return nil, err
	}

	if err := s.recordHash(ctx, profile); err != nil {
		return profile, fmt.Errorf("profile saved but audit logging failed: %w", err)
	}

	return profile, nil
}

func (s *Service) recordHash(ctx context.Context, profile *Profile) error {
	if s.audit == nil {
		return nil
	}
	return s.audit.RecordProfileHash(ctx, profile.ID, s.hashProfile(profile))
}

// ComputeHash exposes the deterministic profile hash computation for integrity verification.
func (s *Service) ComputeHash(p *Profile) string {
	return s.hashProfile(p)
}

// hashProfile produces a deterministic, keyed content hash of the profile
// so the audit chain can prove "this exact version of the data existed at
// this time" without ever storing the personal data - or a
// brute-forceable hash of it - in the chain itself.
func (s *Service) hashProfile(p *Profile) string {
	canonical := p.FullName + "|" +
		p.NIK + "|" +
		p.DateOfBirth.Format("2006-01-02") + "|" +
		string(p.Gender) + "|" +
		p.PhoneNumber + "|" +
		p.Address + "|" +
		derefOrEmpty(p.BloodType) + "|" +
		derefOrEmpty(p.InsuranceNumber) + "|" +
		derefOrEmpty(p.EmergencyContactName) + "|" +
		derefOrEmpty(p.EmergencyContactPhone)

	return s.hasher.Hash(canonical)
}
