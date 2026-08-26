package user

import (
	"context"
)

type Service struct {
	repository *Repository
}

func NewService(repository *Repository) *Service {
	return &Service{
		repository: repository,
	}
}

func (s *Service) GetByClerkUserID(
	ctx context.Context,
	clerkUserID string,
) (*User, error) {
	user, err := s.repository.FindByClerkUserID(
		ctx,
		clerkUserID,
	)

	if err != nil {
		return nil, err
	}

	return user, nil
}

func (s *Service) CreateFromClerk(
	ctx context.Context,
	clerkUserID string,
) (*User, error) {
	return s.repository.Create(
		ctx,
		clerkUserID,
	)
}
