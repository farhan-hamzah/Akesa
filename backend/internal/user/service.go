package user

import (
	"context"
	"errors"

	"github.com/farhan-hamzah/Akesa/backend/internal/auth"
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

func (s *Service) GetByClerkUserID(ctx context.Context, clerkUserID string) (*User, error) {
	return s.repository.FindByClerkUserID(
		ctx,
		clerkUserID,
	)
}

func (s *Service) GetByID(ctx context.Context, id uuid.UUID) (*User, error) {
	return s.repository.FindByID(ctx, id)
}

func (s *Service) CreateFromClerk(ctx context.Context, clerkUserID string) (*User, error) {
	return s.repository.Create(
		ctx,
		clerkUserID,
	)
}

// LoadAuthUser implements auth.UserLoader - this is what turns a bare
// Clerk-authenticated request into a request carrying our own user id,
// role and status, which auth.RequireRole then checks. This is the single
// source of truth for "what role does this caller actually have" - never
// trust a role claimed by the client.
func (s *Service) LoadAuthUser(ctx context.Context, clerkUserID string) (*auth.AuthUser, error) {
	u, err := s.repository.FindByClerkUserID(ctx, clerkUserID)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			return nil, auth.ErrUserNotFound
		}
		return nil, err
	}

	return &auth.AuthUser{
		ID:     u.ID,
		Role:   string(u.Role),
		Status: string(u.Status),
	}, nil
}

// UpdateRole implements hospital.UserRoleUpdater.
func (s *Service) UpdateRole(ctx context.Context, userID uuid.UUID, role string) error {
	r := Role(role)
	if !IsValidRole(r) {
		return ErrInvalidRole
	}
	return s.repository.UpdateRole(ctx, userID, r)
}

func (s *Service) UpdateStatus(ctx context.Context, userID uuid.UUID, status Status) error {
	return s.repository.UpdateStatus(ctx, userID, status)
}
