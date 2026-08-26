package audit

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

func (s *Service) Log(ctx context.Context, entry *AuditLog) error {

	if entry.ID == uuid.Nil {
		entry.ID = uuid.New()
	}

	return s.repository.Create(
		ctx,
		entry,
	)
}
