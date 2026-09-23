package patientqr

import (
	"context"

	"github.com/google/uuid"
)

type Repository interface {
	FindByUserID(ctx context.Context, userID uuid.UUID) (*Credential, error)
	FindByTokenHash(ctx context.Context, tokenHash string) (*Credential, error)
	Create(ctx context.Context, credential *Credential) error
	Update(ctx context.Context, credential *Credential) error
	Deactivate(ctx context.Context, userID uuid.UUID) error
}
