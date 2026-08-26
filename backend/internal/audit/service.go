package audit

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
)

type Service struct {
	repository *Repository
}

func NewService(repository *Repository) *Service {
	return &Service{repository: repository}
}

// Append is the low-level entry point: log any event about any entity.
func (s *Service) Append(
	ctx context.Context,
	entityType string,
	entityID uuid.UUID,
	action string,
	actorID uuid.UUID,
	payload any,
) (*Log, error) {
	data, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal audit payload: %w", err)
	}

	return s.repository.InsertWithChain(ctx, entityType, entityID, action, actorID, data)
}

// RecordProfileHash implements patient.AuditRecorder: it logs that a
// patient's profile changed, storing only the hash of the new profile
// content (never the content itself).
func (s *Service) RecordProfileHash(ctx context.Context, patientID uuid.UUID, profileHash string) error {
	_, err := s.Append(
		ctx,
		"PATIENT_PROFILE",
		patientID,
		ActionProfileUpdated,
		patientID,
		map[string]string{"profileHash": profileHash},
	)
	return err
}

// RecordAccessEvent implements access.AuditRecorder: it logs consent
// lifecycle and data-access events for an access request.
func (s *Service) RecordAccessEvent(
	ctx context.Context,
	entityType string,
	entityID uuid.UUID,
	action string,
	actorID uuid.UUID,
	payload map[string]any,
) error {
	_, err := s.Append(ctx, entityType, entityID, action, actorID, payload)
	return err
}

// History returns the full, ordered audit trail for one entity - used by
// the "riwayat penggunaan data" endpoint.
func (s *Service) History(ctx context.Context, entityType string, entityID uuid.UUID) ([]*Log, error) {
	return s.repository.ListByEntity(ctx, entityType, entityID)
}
