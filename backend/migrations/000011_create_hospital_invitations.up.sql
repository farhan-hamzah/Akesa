CREATE TABLE hospital_invitations (
    id UUID PRIMARY KEY,
    key_code VARCHAR(64) UNIQUE NOT NULL,
    target_email VARCHAR(255) NOT NULL,
    created_by UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    used_by UUID REFERENCES users(id) ON DELETE SET NULL,
    is_used BOOLEAN NOT NULL DEFAULT FALSE,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_hospital_invitations_key_code ON hospital_invitations(key_code);
CREATE INDEX idx_hospital_invitations_target_email ON hospital_invitations(target_email);
