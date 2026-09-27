package patient

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
)

type AuditRecorder interface {
	RecordProfileHash(ctx context.Context, patientID uuid.UUID, profileHash string) error
}

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

func (s *Service) ComputeHash(p *Profile) string {
	return s.hashProfile(p)
}

func (s *Service) hashProfile(p *Profile) string {
	canonical := p.FullName + "|" +
		p.NIK + "|" +
		p.DateOfBirth.Format("2006-01-02") + "|" +
		string(p.Gender) + "|" +
		p.PhoneNumber + "|" +
		p.Address + "|" +
		derefOrEmpty(p.BloodType) + "|" +
		derefOrEmpty(p.DrugAllergy) + "|" +
		derefOrEmpty(p.MedicalHistory) + "|" +
		derefOrEmpty(p.InsuranceNumber) + "|" +
		derefOrEmpty(p.EmergencyContactName) + "|" +
		derefOrEmpty(p.EmergencyContactPhone)

	return s.hasher.Hash(canonical)
}
