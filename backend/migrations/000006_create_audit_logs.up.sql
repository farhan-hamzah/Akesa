-- audit_logs is the append-only, hash-chained log described in the SRS as
-- the "permissioned blockchain" proof of concept. Only hashes and small
-- non-sensitive metadata are stored here - never raw patient data.
CREATE TABLE audit_logs (
    id UUID PRIMARY KEY,
    entity_type VARCHAR(50) NOT NULL,
    entity_id UUID NOT NULL,
    action VARCHAR(50) NOT NULL,
    actor_id UUID NOT NULL,
    payload JSONB NOT NULL DEFAULT '{}',
    prev_hash VARCHAR(64) NOT NULL,
    hash VARCHAR(64) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_audit_logs_entity ON audit_logs(entity_type, entity_id);
CREATE INDEX idx_audit_logs_created_at ON audit_logs(created_at);

-- audit_chain_state tracks the current tip of each hash chain. A single
-- 'global' row is used for the prototype; the schema allows splitting
-- into per-patient chains later without a breaking migration.
CREATE TABLE audit_chain_state (
    chain_name VARCHAR(50) PRIMARY KEY,
    last_hash VARCHAR(64) NOT NULL
);
