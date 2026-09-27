CREATE TABLE identity_verifications (
    id UUID PRIMARY KEY,

    patient_id UUID NOT NULL
        REFERENCES patient_profiles(id)
        ON DELETE CASCADE,

    status VARCHAR(30) NOT NULL DEFAULT 'PENDING',
    document_type VARCHAR(30) NOT NULL DEFAULT 'KTP',
    document_storage_key TEXT,
    extracted_nik TEXT,
    extracted_full_name TEXT,
    extracted_date_of_birth DATE,
    extracted_gender VARCHAR(10),
    document_status VARCHAR(30),
    liveness_status VARCHAR(30),
    face_match_status VARCHAR(30),
    face_match_score DECIMAL(5,4),
    provider VARCHAR(50),
    provider_reference VARCHAR(255),
    failure_reason VARCHAR(100),
    verified_at TIMESTAMPTZ,
    expires_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT identity_verifications_status_check
        CHECK (
            status IN (
                'PENDING',
                'PROCESSING',
                'DOCUMENT_REJECTED',
                'DATA_MISMATCH',
                'LIVENESS_FAILED',
                'FACE_MISMATCH',
                'MANUAL_REVIEW',
                'VERIFIED',
                'EXPIRED'
            )
        ),

    CONSTRAINT identity_verifications_document_type_check
        CHECK (
            document_type IN ('KTP')
        ),

    CONSTRAINT identity_verifications_face_match_score_check
        CHECK (
            face_match_score IS NULL
            OR (
                face_match_score >= 0
                AND face_match_score <= 1
            )
        )
);

CREATE INDEX idx_identity_verifications_patient_id
    ON identity_verifications(patient_id);

CREATE INDEX idx_identity_verifications_status
    ON identity_verifications(status);

CREATE INDEX idx_identity_verifications_expires_at
    ON identity_verifications(expires_at);

CREATE INDEX idx_identity_verifications_provider_reference
    ON identity_verifications(provider_reference);