CREATE TABLE patient_qr_credentials (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    user_id UUID NOT NULL UNIQUE
        REFERENCES users(id)
        ON DELETE CASCADE,

    display_code VARCHAR(32) NOT NULL UNIQUE,

    token_hash VARCHAR(64) NOT NULL UNIQUE,

    is_active BOOLEAN NOT NULL DEFAULT TRUE,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_patient_qr_token_hash
    ON patient_qr_credentials(token_hash);

CREATE INDEX idx_patient_qr_user_id
    ON patient_qr_credentials(user_id);