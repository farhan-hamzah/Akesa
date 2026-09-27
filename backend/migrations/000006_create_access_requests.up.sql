CREATE TABLE access_requests (
    id UUID PRIMARY KEY,
    hospital_id UUID NOT NULL REFERENCES hospitals(id) ON DELETE CASCADE,
    patient_id UUID NOT NULL REFERENCES patient_profiles(id) ON DELETE CASCADE,
    requested_by UUID NOT NULL REFERENCES users(id),
    purpose TEXT NOT NULL,
    categories TEXT[] NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'PENDING',
    decided_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_access_requests_patient_id ON access_requests(patient_id);
CREATE INDEX idx_access_requests_hospital_id ON access_requests(hospital_id);
CREATE INDEX idx_access_requests_status ON access_requests(status);
