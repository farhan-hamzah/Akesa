package audit

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/farhan-hamzah/Akesa/backend/internal/blockchain"
	"github.com/google/uuid"
)

type Service struct {
	repository *Repository
	bcClient   blockchain.Client
}

func NewService(repository *Repository, bcClient blockchain.Client) *Service {
	return &Service{
		repository: repository,
		bcClient:   bcClient,
	}
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
	// Layer 1: Record in Postgres hash chain
	_, err := s.Append(
		ctx,
		"PATIENT_PROFILE",
		patientID,
		ActionProfileUpdated,
		patientID,
		map[string]string{"profileHash": profileHash},
	)
	if err != nil {
		return err
	}

	// Layer 2: Record on EVM Smart Contract (if configured & online)
	if s.bcClient != nil && s.bcClient.IsEnabled() {
		go func() {
			bgCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			txHash, bcErr := s.bcClient.RecordProfileHash(bgCtx, patientID, profileHash)
			if bcErr != nil {
				log.Printf("[blockchain] warning: failed to sync profile hash on-chain (fallback active): %v", bcErr)
			} else {
				log.Printf("[blockchain] profile hash anchored on-chain! txHash=%s", txHash)
			}
		}()
	}

	return nil
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
	// Layer 1: Record in Postgres hash chain
	_, err := s.Append(ctx, entityType, entityID, action, actorID, payload)
	if err != nil {
		return err
	}

	// Layer 2: Record on EVM Smart Contract (if configured & online)
	if s.bcClient != nil && s.bcClient.IsEnabled() {
		go func() {
			bgCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			var status uint8
			switch action {
			case ActionAccessApproved:
				status = 2
			case ActionAccessRejected:
				status = 3
			case ActionAccessRevoked:
				status = 4
			}

			if status > 0 {
				metaBytes, _ := json.Marshal(payload)
				txHash, bcErr := s.bcClient.RecordConsent(bgCtx, entityID, actorID, uuid.Nil, status, string(metaBytes))
				if bcErr != nil {
					log.Printf("[blockchain] warning: failed to sync consent on-chain (fallback active): %v", bcErr)
				} else {
					log.Printf("[blockchain] consent anchored on-chain! txHash=%s", txHash)
				}
			} else if action == ActionDataAccessed {
				metaBytes, _ := json.Marshal(payload)
				txHash, bcErr := s.bcClient.RecordDataAccess(bgCtx, entityID, actorID, string(metaBytes))
				if bcErr != nil {
					log.Printf("[blockchain] warning: failed to sync data access on-chain (fallback active): %v", bcErr)
				} else {
					log.Printf("[blockchain] data access anchored on-chain! txHash=%s", txHash)
				}
			}
		}()
	}

	return nil
}

// History returns the full, ordered audit trail for one entity - used by
// the "riwayat penggunaan data" endpoint.
func (s *Service) History(ctx context.Context, entityType string, entityID uuid.UUID) ([]*Log, error) {
	return s.repository.ListByEntity(ctx, entityType, entityID)
}

// VerifyChainIntegrity verifies the entire database audit trail against SHA-256 hash chains.
func (s *Service) VerifyChainIntegrity(ctx context.Context) (bool, int, error) {
	return s.repository.VerifyChainIntegrity(ctx)
}

// VerifyProfileOnChain validates patient profile hash against smart contract or local fallback.
func (s *Service) VerifyProfileOnChain(ctx context.Context, patientID uuid.UUID, currentHash string) (bool, string, error) {
	if s.bcClient == nil || !s.bcClient.IsEnabled() {
		// Fallback to local cryptographic ledger verification
		latestLog, err := s.repository.GetLatestEntityLog(ctx, "PATIENT_PROFILE", patientID, ActionProfileUpdated)
		if err != nil {
			return false, "", fmt.Errorf("local ledger verification: %w", err)
		}
		var payload map[string]string
		if err := json.Unmarshal(latestLog.Payload, &payload); err != nil {
			return false, "", fmt.Errorf("unmarshal audit payload: %w", err)
		}
		recordedHash := payload["profileHash"]
		return recordedHash == currentHash, recordedHash, nil
	}
	return s.bcClient.VerifyProfileHash(ctx, patientID, currentHash)
}

// GetLatestProfileHash returns the latest recorded profile hash for a patient from the audit ledger.
func (s *Service) GetLatestProfileHash(ctx context.Context, patientID uuid.UUID) (string, error) {
	latestLog, err := s.repository.GetLatestEntityLog(ctx, "PATIENT_PROFILE", patientID, ActionProfileUpdated)
	if err != nil {
		return "", err
	}
	var payload map[string]string
	if err := json.Unmarshal(latestLog.Payload, &payload); err != nil {
		return "", fmt.Errorf("unmarshal audit payload: %w", err)
	}
	return payload["profileHash"], nil
}
