-- Encrypt-at-rest for NIK and insurance number. The app now stores
-- AES-256-GCM ciphertext in these columns (base64-encoded, so widened to
-- TEXT), plus a keyed HMAC hash of the NIK (nik_hash) so we can still
-- enforce "one NIK, one profile" and do exact-match lookups without ever
-- decrypting rows in bulk. See internal/crypto for the implementation.
--
-- IMPORTANT: this migration assumes the table has no pre-existing
-- plaintext data (true for a fresh prototype database). If you already
-- have real rows in patient_profiles, you MUST write and run a one-off
-- backfill script (encrypt the existing nik/insurance_number values and
-- compute nik_hash for each row) BEFORE this migration, otherwise the
-- NOT NULL / UNIQUE constraints below will fail, or old plaintext values
-- will fail to decrypt at read time afterwards.

ALTER TABLE patient_profiles
    ALTER COLUMN nik TYPE TEXT,
    ALTER COLUMN insurance_number TYPE TEXT;

ALTER TABLE patient_profiles
    ADD COLUMN nik_hash VARCHAR(64);

ALTER TABLE patient_profiles
    ALTER COLUMN nik_hash SET NOT NULL,
    ADD CONSTRAINT patient_profiles_nik_hash_key UNIQUE (nik_hash);

CREATE INDEX idx_patient_profiles_nik_hash ON patient_profiles(nik_hash);

-- The plaintext-oriented index from migration 000003 is no longer useful
-- now that `nik` holds ciphertext (every value looks equally random to
-- the index) - lookups now go through nik_hash instead.
DROP INDEX IF EXISTS idx_patient_profiles_nik;
