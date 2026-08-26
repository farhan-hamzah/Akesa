// Package audit implements the "permissioned blockchain proof of concept"
// described in the SRS: an append-only, hash-chained log of profile
// versions, consent decisions, revocations and data access events.
//
// For the prototype this chain is stored in Postgres as a tamper-evident
// hash chain (each entry's hash depends on the previous entry's hash), NOT
// on an actual distributed ledger - no patient personal data is ever put
// into the chain, only hashes and small non-sensitive metadata, matching
// "data pribadi pasien tidak disimpan secara langsung pada blockchain".
//
// The Recorder interface below is what the rest of the codebase depends
// on. When the team is ready to plug in a real permissioned blockchain
// client (e.g. Hyperledger Fabric), implement Recorder with that client
// and swap it in server wiring - no other package needs to change.
package audit

import (
	"time"

	"github.com/google/uuid"
)

// Well-known action names. Keep these stable - they end up inside the hash
// chain and any consumer (e.g. a future verification UI) will match on them.
const (
	ActionProfileUpdated  = "PROFILE_UPDATED"
	ActionAccessRequested = "ACCESS_REQUESTED"
	ActionAccessApproved  = "ACCESS_APPROVED"
	ActionAccessRejected  = "ACCESS_REJECTED"
	ActionAccessRevoked   = "ACCESS_REVOKED"
	ActionDataAccessed    = "DATA_ACCESSED"
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
