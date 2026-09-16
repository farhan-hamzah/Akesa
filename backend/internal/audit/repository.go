package audit

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// genesisHash is the prevHash used for the very first entry in the chain.
const genesisHash = "0000000000000000000000000000000000000000000000000000000000000"

// chainName is a single global chain for the prototype. If per-patient
// chains are ever needed, this can become a parameter.
const chainName = "global"

type Repository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

// InsertWithChain appends a new entry to the hash chain inside a
// transaction: it locks the current chain tip (SELECT ... FOR UPDATE),
// computes the new entry's hash on top of it, inserts the entry, and
// advances the tip - so concurrent writers can never fork the chain.
func (r *Repository) InsertWithChain(
	ctx context.Context,
	entityType string,
	entityID uuid.UUID,
	action string,
	actorID uuid.UUID,
	payload []byte,
) (*Log, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck // no-op if committed

	var prevHash string
	err = tx.QueryRow(
		ctx,
		`SELECT last_hash FROM audit_chain_state WHERE chain_name = $1 FOR UPDATE`,
		chainName,
	).Scan(&prevHash)

	if err != nil {
		// First-ever entry: seed the chain state row.
		prevHash = genesisHash
		if _, insertErr := tx.Exec(
			ctx,
			`INSERT INTO audit_chain_state (chain_name, last_hash) VALUES ($1, $2)
			 ON CONFLICT (chain_name) DO NOTHING`,
			chainName, prevHash,
		); insertErr != nil {
			return nil, fmt.Errorf("seed chain state: %w", insertErr)
		}
	}

	now := time.Now().UTC()
	hash := computeHash(prevHash, entityType, entityID, action, actorID, payload, now)

	entry := &Log{
		ID:         uuid.New(),
		EntityType: entityType,
		EntityID:   entityID,
		Action:     action,
		ActorID:    actorID,
		Payload:    payload,
		PrevHash:   prevHash,
		Hash:       hash,
		CreatedAt:  now,
	}

	_, err = tx.Exec(
		ctx,
		`INSERT INTO audit_logs (
			id, entity_type, entity_id, action, actor_id, payload, prev_hash, hash, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
		entry.ID, entry.EntityType, entry.EntityID, entry.Action, entry.ActorID,
		entry.Payload, entry.PrevHash, entry.Hash, entry.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("insert audit log: %w", err)
	}

	_, err = tx.Exec(
		ctx,
		`UPDATE audit_chain_state SET last_hash = $1 WHERE chain_name = $2`,
		entry.Hash, chainName,
	)
	if err != nil {
		return nil, fmt.Errorf("advance chain state: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit tx: %w", err)
	}

	return entry, nil
}

// ListByEntity returns the audit trail for a single entity (e.g. one
// patient's profile, or one access request), oldest first, so a patient
// can review "riwayat penggunaan data" (SRS).
func (r *Repository) ListByEntity(ctx context.Context, entityType string, entityID uuid.UUID) ([]*Log, error) {
	rows, err := r.db.Query(
		ctx,
		`SELECT id, entity_type, entity_id, action, actor_id, payload, prev_hash, hash, created_at
		 FROM audit_logs
		 WHERE entity_type = $1 AND entity_id = $2
		 ORDER BY created_at ASC`,
		entityType, entityID,
	)
	if err != nil {
		return nil, fmt.Errorf("query audit logs: %w", err)
	}
	defer rows.Close()

	var logs []*Log
	for rows.Next() {
		entry := &Log{}
		if err := rows.Scan(
			&entry.ID, &entry.EntityType, &entry.EntityID, &entry.Action, &entry.ActorID,
			&entry.Payload, &entry.PrevHash, &entry.Hash, &entry.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan audit log: %w", err)
		}
		logs = append(logs, entry)
	}

	return logs, rows.Err()
}

// VerifyChainIntegrity walks the entire audit chain in chronological order
// from genesis to the current tip, re-computing each block's SHA-256 hash.
// If any row has been altered, deleted, or inserted out of order, it returns
// false and a detailed error identifying the corrupted entry.
func (r *Repository) VerifyChainIntegrity(ctx context.Context) (bool, int, error) {
	rows, err := r.db.Query(
		ctx,
		`SELECT id, entity_type, entity_id, action, actor_id, payload, prev_hash, hash, created_at
		 FROM audit_logs
		 ORDER BY created_at ASC`,
	)
	if err != nil {
		return false, 0, fmt.Errorf("query audit logs: %w", err)
	}
	defer rows.Close()

	expectedPrevHash := genesisHash
	count := 0

	for rows.Next() {
		entry := &Log{}
		if err := rows.Scan(
			&entry.ID, &entry.EntityType, &entry.EntityID, &entry.Action, &entry.ActorID,
			&entry.Payload, &entry.PrevHash, &entry.Hash, &entry.CreatedAt,
		); err != nil {
			return false, count, fmt.Errorf("scan audit log at index %d: %w", count, err)
		}

		if entry.PrevHash != expectedPrevHash {
			return false, count, fmt.Errorf("chain broken at entry %s (index %d): expected prev_hash %s, got %s",
				entry.ID, count, expectedPrevHash, entry.PrevHash)
		}

		computed := computeHash(
			entry.PrevHash,
			entry.EntityType,
			entry.EntityID,
			entry.Action,
			entry.ActorID,
			entry.Payload,
			entry.CreatedAt,
		)
		if entry.Hash != computed {
			return false, count, fmt.Errorf("tamper detected at entry %s (index %d): expected hash %s, got %s",
				entry.ID, count, computed, entry.Hash)
		}

		expectedPrevHash = entry.Hash
		count++
	}

	if err := rows.Err(); err != nil {
		return false, count, fmt.Errorf("iterate audit logs: %w", err)
	}

	if count > 0 {
		var lastHash string
		err := r.db.QueryRow(ctx, `SELECT last_hash FROM audit_chain_state WHERE chain_name = $1`, chainName).Scan(&lastHash)
		if err != nil {
			return false, count, fmt.Errorf("query chain tip: %w", err)
		}
		if lastHash != expectedPrevHash {
			return false, count, fmt.Errorf("chain tip mismatch: state has %s, computed chain ended at %s", lastHash, expectedPrevHash)
		}
	}

	return true, count, nil
}

// GetLatestEntityLog returns the most recent audit log entry for a specific entity and action.
func (r *Repository) GetLatestEntityLog(ctx context.Context, entityType string, entityID uuid.UUID, action string) (*Log, error) {
	entry := &Log{}
	err := r.db.QueryRow(
		ctx,
		`SELECT id, entity_type, entity_id, action, actor_id, payload, prev_hash, hash, created_at
		 FROM audit_logs
		 WHERE entity_type = $1 AND entity_id = $2 AND action = $3
		 ORDER BY created_at DESC
		 LIMIT 1`,
		entityType, entityID, action,
	).Scan(
		&entry.ID, &entry.EntityType, &entry.EntityID, &entry.Action, &entry.ActorID,
		&entry.Payload, &entry.PrevHash, &entry.Hash, &entry.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return entry, nil
}

func computeHash(
	prevHash string,
	entityType string,
	entityID uuid.UUID,
	action string,
	actorID uuid.UUID,
	payload []byte,
	ts time.Time,
) string {
	h := sha256.New()
	h.Write([]byte(prevHash))
	h.Write([]byte(entityType))
	h.Write([]byte(entityID.String()))
	h.Write([]byte(action))
	h.Write([]byte(actorID.String()))
	h.Write(payload)
	h.Write([]byte(ts.Format(time.RFC3339Nano)))
	return hex.EncodeToString(h.Sum(nil))
}
