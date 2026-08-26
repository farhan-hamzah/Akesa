ALTER TABLE patient_profiles DROP CONSTRAINT IF EXISTS patient_profiles_nik_hash_key;
DROP INDEX IF EXISTS idx_patient_profiles_nik_hash;
ALTER TABLE patient_profiles DROP COLUMN IF EXISTS nik_hash;

ALTER TABLE patient_profiles
    ALTER COLUMN nik TYPE VARCHAR(16),
    ALTER COLUMN insurance_number TYPE VARCHAR(100);

CREATE INDEX idx_patient_profiles_nik ON patient_profiles(nik);
