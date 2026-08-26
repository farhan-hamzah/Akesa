package hospital

import (
	"context"
	"strings"

	"github.com/google/uuid"
)

// UserRoleUpdater is the slice of user.Service this package depends on -
// promoting a synced account to HOSPITAL_STAFF once an admin links it to
// a hospital. Defined on the consumer side to avoid a hospital<->user
// import cycle.
type UserRoleUpdater interface {
	UpdateRole(ctx context.Context, userID uuid.UUID, role string) error
}

type Service struct {
	repository *Repository
	users      UserRoleUpdater
}

func NewService(repository *Repository, users UserRoleUpdater) *Service {
	return &Service{repository: repository, users: users}
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
	return s.repository.UpdateStatus(ctx, id, StatusActive)
}

func (s *Service) Deactivate(ctx context.Context, id uuid.UUID) (*Hospital, error) {
	return s.repository.UpdateStatus(ctx, id, StatusInactive)
}

// AddStaff links an already-Clerk-authenticated user to a hospital and
// promotes their role to HOSPITAL_STAFF. The user must have already
// called /api/v1/users/sync at least once so the userID exists.
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
