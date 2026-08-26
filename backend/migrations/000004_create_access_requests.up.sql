CREATE TABLE access_requests (
    id UUID PRIMARY KEY,

    patient_id UUID NOT NULL
        REFERENCES patients(id)
        ON DELETE CASCADE,

    hospital_id UUID NOT NULL
        REFERENCES hospitals(id)
        ON DELETE CASCADE,

    requested_by UUID NOT NULL
        REFERENCES users(id)
        ON DELETE RESTRICT,

    reason TEXT NOT NULL,

    status VARCHAR(30) NOT NULL DEFAULT 'PENDING',

    requested_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    reviewed_at TIMESTAMPTZ,

    reviewed_by UUID
        REFERENCES users(id)
        ON DELETE SET NULL,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_access_requests_patient_id
    ON access_requests(patient_id);

CREATE INDEX idx_access_requests_hospital_id
    ON access_requests(hospital_id);

CREATE INDEX idx_access_requests_requested_by
    ON access_requests(requested_by);

CREATE INDEX idx_access_requests_status
    ON access_requests(status);