package audit

import (
	"time"

	"github.com/google/uuid"
)

const (
	ActionProfileUpdated  = "PROFILE_UPDATED"
	ActionAccessRequested = "ACCESS_REQUESTED"
	ActionAccessApproved  = "ACCESS_APPROVED"
	ActionAccessRejected  = "ACCESS_REJECTED"
	ActionAccessRevoked   = "ACCESS_REVOKED"
	ActionDataAccessed              = "DATA_ACCESSED"
	ActionIntegrityViolationBlocked = "INTEGRITY_VIOLATION_BLOCKED"
)

type Log struct {
	ID         uuid.UUID `json:"id"`
	EntityType string    `json:"entityType"`
	EntityID   uuid.UUID `json:"entityId"`
	Action     string    `json:"action"`
	ActorID    uuid.UUID `json:"actorId"`
	Payload    []byte    `json:"payload"` // small JSON metadata only, never raw PII
	PrevHash   string    `json:"prevHash"`
	Hash       string    `json:"hash"`
	CreatedAt  time.Time `json:"createdAt"`
}
