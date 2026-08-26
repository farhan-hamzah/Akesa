package patient

import (
	"context"

	"github.com/google/uuid"
)

type Service struct {
	repository *Repository
}

func NewService(repository *Repository) *Service {
	return &Service{
		repository: repository,
	}
}

func (s *Service) GetByUserID(ctx context.Context, userID uuid.UUID) (*Patient, error) {
	return s.repository.FindByUserID(ctx, userID)
}

func (s *Service) GetByID(ctx context.Context, id uuid.UUID) (*Patient, error) {
	return s.repository.FindByID(ctx, id)
}
