ALTER TABLE patient_profiles
    ALTER COLUMN nik TYPE TEXT,
    ALTER COLUMN insurance_number TYPE TEXT;

ALTER TABLE patient_profiles
    ADD COLUMN nik_hash VARCHAR(64);

ALTER TABLE patient_profiles
    ALTER COLUMN nik_hash SET NOT NULL,
    ADD CONSTRAINT patient_profiles_nik_hash_key UNIQUE (nik_hash);

CREATE INDEX idx_patient_profiles_nik_hash ON patient_profiles(nik_hash);

DROP INDEX IF EXISTS idx_patient_profiles_nik;
