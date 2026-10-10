-- --------------------------------------------------------
-- Host:                         aws-0-ap-northeast-1.pooler.supabase.com
-- Server version:               PostgreSQL 17.6 on aarch64-unknown-linux-gnu, compiled by gcc (GCC) 15.2.0, 64-bit
-- Server OS:                    
-- HeidiSQL Version:             12.1.0.6537
-- --------------------------------------------------------

/*!40101 SET @OLD_CHARACTER_SET_CLIENT=@@CHARACTER_SET_CLIENT */;
/*!40101 SET NAMES  */;
/*!40103 SET @OLD_TIME_ZONE=@@TIME_ZONE */;
/*!40103 SET TIME_ZONE='+00:00' */;
/*!40014 SET @OLD_FOREIGN_KEY_CHECKS=@@FOREIGN_KEY_CHECKS, FOREIGN_KEY_CHECKS=0 */;
/*!40101 SET @OLD_SQL_MODE=@@SQL_MODE, SQL_MODE='NO_AUTO_VALUE_ON_ZERO' */;
/*!40111 SET @OLD_SQL_NOTES=@@SQL_NOTES, SQL_NOTES=0 */;

-- Dumping structure for function public.rls_auto_enable
DELIMITER //
CREATE FUNCTION "rls_auto_enable"() RETURNS UNKNOWN AS $$ 
DECLARE
  cmd record;
BEGIN
  FOR cmd IN
    SELECT *
    FROM pg_event_trigger_ddl_commands()
    WHERE command_tag IN ('CREATE TABLE', 'CREATE TABLE AS', 'SELECT INTO')
      AND object_type IN ('table','partitioned table')
  LOOP
     IF cmd.schema_name IS NOT NULL AND cmd.schema_name IN ('public') AND cmd.schema_name NOT IN ('pg_catalog','information_schema') AND cmd.schema_name NOT LIKE 'pg_toast%' AND cmd.schema_name NOT LIKE 'pg_temp%' THEN
      BEGIN
        EXECUTE format('alter table if exists %s enable row level security', cmd.object_identity);
        RAISE LOG 'rls_auto_enable: enabled RLS on %', cmd.object_identity;
      EXCEPTION
        WHEN OTHERS THEN
          RAISE LOG 'rls_auto_enable: failed to enable RLS on %', cmd.object_identity;
      END;
     ELSE
        RAISE LOG 'rls_auto_enable: skip % (either system schema or not in enforced list: %.)', cmd.object_identity, cmd.schema_name;
     END IF;
  END LOOP;
END;
 $$//
DELIMITER ;

-- Dumping structure for table public.access_requests
CREATE TABLE IF NOT EXISTS "access_requests" (
	"id" UUID NOT NULL,
	"hospital_id" UUID NOT NULL,
	"patient_id" UUID NOT NULL,
	"requested_by" UUID NOT NULL,
	"purpose" TEXT NOT NULL,
	"categories" UNKNOWN NOT NULL,
	"status" VARCHAR(20) NOT NULL DEFAULT 'PENDING',
	"decided_at" TIMESTAMPTZ NULL DEFAULT NULL,
	"created_at" TIMESTAMPTZ NOT NULL DEFAULT 'now()',
	"updated_at" TIMESTAMPTZ NOT NULL DEFAULT 'now()',
	PRIMARY KEY ("id"),
	INDEX "idx_access_requests_patient_id" ("patient_id"),
	INDEX "idx_access_requests_hospital_id" ("hospital_id"),
	INDEX "idx_access_requests_status" ("status"),
	CONSTRAINT "access_requests_hospital_id_fkey" FOREIGN KEY ("hospital_id") REFERENCES "hospitals" ("id") ON UPDATE NO ACTION ON DELETE CASCADE,
	CONSTRAINT "access_requests_patient_id_fkey" FOREIGN KEY ("patient_id") REFERENCES "patient_profiles" ("id") ON UPDATE NO ACTION ON DELETE CASCADE,
	CONSTRAINT "access_requests_requested_by_fkey" FOREIGN KEY ("requested_by") REFERENCES "users" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION
);

-- Dumping data for table public.access_requests: -1 rows
/*!40000 ALTER TABLE "access_requests" DISABLE KEYS */;
REPLACE INTO "access_requests" ("id", "hospital_id", "patient_id", "requested_by", "purpose", "categories", "status", "decided_at", "created_at", "updated_at") VALUES
	('3efac1ba-5b4a-43bc-824b-eeb926209ddb', '85ce395a-541c-431d-b371-9286a86f6386', 'c6acab0f-8506-4f72-8a82-e19be15de335', '76568a30-17db-40ee-a4a3-04af16793773', 'Pemeriksaana pasien', '{IDENTITY,CONTACT,MEDICAL_BASIC}', 'REVOKED', '2026-10-08 05:44:39.509851+00', '2026-10-08 05:14:15.383579+00', '2026-10-08 10:19:39.721265+00');
/*!40000 ALTER TABLE "access_requests" ENABLE KEYS */;

-- Dumping structure for table public.audit_chain_state
CREATE TABLE IF NOT EXISTS "audit_chain_state" (
	"chain_name" VARCHAR(50) NOT NULL,
	"last_hash" VARCHAR(64) NOT NULL,
	PRIMARY KEY ("chain_name")
);

-- Dumping data for table public.audit_chain_state: -1 rows
/*!40000 ALTER TABLE "audit_chain_state" DISABLE KEYS */;
REPLACE INTO "audit_chain_state" ("chain_name", "last_hash") VALUES
	('global', '9e88b42918d7beee9772a9ed2c9c4496d226bdcf06a92841199e9f172d6bb705');
/*!40000 ALTER TABLE "audit_chain_state" ENABLE KEYS */;

-- Dumping structure for table public.audit_logs
CREATE TABLE IF NOT EXISTS "audit_logs" (
	"id" UUID NOT NULL,
	"entity_type" VARCHAR(50) NOT NULL,
	"entity_id" UUID NOT NULL,
	"action" VARCHAR(50) NOT NULL,
	"actor_id" UUID NOT NULL,
	"payload" JSONB NOT NULL DEFAULT '{}',
	"prev_hash" VARCHAR(64) NOT NULL,
	"hash" VARCHAR(64) NOT NULL,
	"created_at" TIMESTAMPTZ NOT NULL DEFAULT 'now()',
	PRIMARY KEY ("id"),
	INDEX "idx_audit_logs_entity" ("entity_type", "entity_id"),
	INDEX "idx_audit_logs_created_at" ("created_at")
);

-- Dumping data for table public.audit_logs: -1 rows
/*!40000 ALTER TABLE "audit_logs" DISABLE KEYS */;
REPLACE INTO "audit_logs" ("id", "entity_type", "entity_id", "action", "actor_id", "payload", "prev_hash", "hash", "created_at") VALUES
	('f61cbba4-996e-40b2-ad2e-4d26b1297b44', 'PATIENT_PROFILE', 'c6acab0f-8506-4f72-8a82-e19be15de335', 'PROFILE_UPDATED', 'c6acab0f-8506-4f72-8a82-e19be15de335', '{"profileHash": "0865878564c8ce24943219abf43686cb0bb383eeb5e8e8e019e002b3a73c5213"}', '0000000000000000000000000000000000000000000000000000000000000', 'e985274695f3b373039e315aedf1c34b872078940a962d424f90f00a55668340', '2026-09-22 13:54:46.95829+00'),
	('af2d14f6-6083-4305-a91d-891bf0da41f7', 'ACCESS_REQUEST', '3efac1ba-5b4a-43bc-824b-eeb926209ddb', 'ACCESS_REQUESTED', '76568a30-17db-40ee-a4a3-04af16793773', '{"categories": ["IDENTITY", "CONTACT", "MEDICAL_BASIC"], "hospitalId": "85ce395a-541c-431d-b371-9286a86f6386"}', 'e985274695f3b373039e315aedf1c34b872078940a962d424f90f00a55668340', '9811edbf0ad76234a3de8f4c2b9328afe9a1d3cd966888b5d09ad770ab14bafa', '2026-10-08 05:14:14.58933+00'),
	('feb12886-a6be-4ae4-8ffe-776f9f9db196', 'ACCESS_REQUEST', '3efac1ba-5b4a-43bc-824b-eeb926209ddb', 'ACCESS_APPROVED', '40551d73-8ad5-43f4-a4dd-c44f046718d0', '{"categories": ["IDENTITY", "CONTACT", "MEDICAL_BASIC"]}', '9811edbf0ad76234a3de8f4c2b9328afe9a1d3cd966888b5d09ad770ab14bafa', '7f1724fbe19e782b970d0962e9bbaa1329967153794f0907b24fa2bfc4a7db09', '2026-10-08 05:44:39.95499+00'),
	('416e9f98-055d-41e7-8dbd-46ccaf03cae1', 'ACCESS_REQUEST', '3efac1ba-5b4a-43bc-824b-eeb926209ddb', 'INTEGRITY_VIOLATION_BLOCKED', '76568a30-17db-40ee-a4a3-04af16793773', '{"error": "hash_mismatch", "currentHash": "57bfd030a34c5575b66dce4ad46eeea3aad1a0761da2b86e47aa9b7c53bea980", "recordedHash": "0865878564c8ce24943219abf43686cb0bb383eeb5e8e8e019e002b3a73c5213"}', '7f1724fbe19e782b970d0962e9bbaa1329967153794f0907b24fa2bfc4a7db09', '0d8b47e09b087565356668488f5816318d81c29752ff0ecbcb6937f8adf97f31', '2026-10-08 05:45:36.800489+00'),
	('f817170d-0fd7-451b-af13-aee4ff203579', 'ACCESS_REQUEST', '3efac1ba-5b4a-43bc-824b-eeb926209ddb', 'INTEGRITY_VIOLATION_BLOCKED', '76568a30-17db-40ee-a4a3-04af16793773', '{"error": "hash_mismatch", "currentHash": "57bfd030a34c5575b66dce4ad46eeea3aad1a0761da2b86e47aa9b7c53bea980", "recordedHash": "0x0000000000000000000000000000000000000000000000000000000000000000"}', '0d8b47e09b087565356668488f5816318d81c29752ff0ecbcb6937f8adf97f31', '1519354000730ca55ebe760240b58e2c4abe871c1e8043c007762e4d3426de89', '2026-10-08 06:03:14.246338+00'),
	('3b55efd0-028f-44c7-a8b9-6f3f032cb4ae', 'PATIENT_PROFILE', 'c6acab0f-8506-4f72-8a82-e19be15de335', 'PROFILE_UPDATED', 'c6acab0f-8506-4f72-8a82-e19be15de335', '{"profileHash": "57bfd030a34c5575b66dce4ad46eeea3aad1a0761da2b86e47aa9b7c53bea980"}', '1519354000730ca55ebe760240b58e2c4abe871c1e8043c007762e4d3426de89', '9ce19d78fb387680163e3cf7ce55292c05fb749e9d60d3df6909aa94fbc74a8f', '2026-10-08 06:20:14.521535+00'),
	('670b990e-2213-426a-8ad6-545a652bfc29', 'ACCESS_REQUEST', '3efac1ba-5b4a-43bc-824b-eeb926209ddb', 'DATA_ACCESSED', '76568a30-17db-40ee-a4a3-04af16793773', '{"dataHash": "57bfd030a34c5575b66dce4ad46eeea3aad1a0761da2b86e47aa9b7c53bea980", "categories": ["IDENTITY", "CONTACT", "MEDICAL_BASIC"]}', '9ce19d78fb387680163e3cf7ce55292c05fb749e9d60d3df6909aa94fbc74a8f', '137f11d99212ec515a4fffe2947b6176e5566169e142b38956a95165f885e6d9', '2026-10-08 06:22:27.495875+00'),
	('33213388-b157-4391-a828-6d66df5143dd', 'ACCESS_REQUEST', '3efac1ba-5b4a-43bc-824b-eeb926209ddb', 'DATA_ACCESSED', '76568a30-17db-40ee-a4a3-04af16793773', '{"dataHash": "57bfd030a34c5575b66dce4ad46eeea3aad1a0761da2b86e47aa9b7c53bea980", "categories": ["IDENTITY", "CONTACT", "MEDICAL_BASIC"]}', '137f11d99212ec515a4fffe2947b6176e5566169e142b38956a95165f885e6d9', 'b97910fec601fb27576fd9beeecd6d34a95abae471c494e0f48be926abcaa093', '2026-10-08 06:23:05.902642+00'),
	('e96da060-d1df-49b5-bcba-3ad04e9bb723', 'ACCESS_REQUEST', '3efac1ba-5b4a-43bc-824b-eeb926209ddb', 'DATA_ACCESSED', '76568a30-17db-40ee-a4a3-04af16793773', '{"dataHash": "57bfd030a34c5575b66dce4ad46eeea3aad1a0761da2b86e47aa9b7c53bea980", "categories": ["IDENTITY", "CONTACT", "MEDICAL_BASIC"]}', 'b97910fec601fb27576fd9beeecd6d34a95abae471c494e0f48be926abcaa093', '9a250d39ced0f2c3c4c25af81b2dc48e44e59684533f3fa1e8addafc14a0329d', '2026-10-08 10:16:30.293183+00'),
	('a141dfa0-57ac-459f-afc1-a7fb18d00760', 'ACCESS_REQUEST', '3efac1ba-5b4a-43bc-824b-eeb926209ddb', 'DATA_ACCESSED', '76568a30-17db-40ee-a4a3-04af16793773', '{"dataHash": "57bfd030a34c5575b66dce4ad46eeea3aad1a0761da2b86e47aa9b7c53bea980", "categories": ["IDENTITY", "CONTACT", "MEDICAL_BASIC"]}', '9a250d39ced0f2c3c4c25af81b2dc48e44e59684533f3fa1e8addafc14a0329d', 'e76ef2fd93e6ac8b4d93c0b77e2f7d4ae0f098b25bf5a12230ecdebea9d98a12', '2026-10-08 10:16:42.631643+00'),
	('569539b0-0b84-43fe-bc29-8cb4ccedaab8', 'ACCESS_REQUEST', '3efac1ba-5b4a-43bc-824b-eeb926209ddb', 'ACCESS_REVOKED', '40551d73-8ad5-43f4-a4dd-c44f046718d0', 'null', 'e76ef2fd93e6ac8b4d93c0b77e2f7d4ae0f098b25bf5a12230ecdebea9d98a12', '9e88b42918d7beee9772a9ed2c9c4496d226bdcf06a92841199e9f172d6bb705', '2026-10-08 10:19:39.737698+00');
/*!40000 ALTER TABLE "audit_logs" ENABLE KEYS */;

-- Dumping structure for table public.hospitals
CREATE TABLE IF NOT EXISTS "hospitals" (
	"id" UUID NOT NULL,
	"name" VARCHAR(255) NOT NULL,
	"address" TEXT NOT NULL,
	"phone" VARCHAR(50) NOT NULL,
	"email" VARCHAR(255) NOT NULL,
	"registration_number" VARCHAR(100) NOT NULL,
	"status" VARCHAR(30) NOT NULL DEFAULT 'PENDING',
	"created_at" TIMESTAMPTZ NOT NULL DEFAULT 'now()',
	"updated_at" TIMESTAMPTZ NOT NULL DEFAULT 'now()',
	PRIMARY KEY ("id"),
	UNIQUE INDEX "hospitals_email_key" ("email"),
	UNIQUE INDEX "hospitals_registration_number_key" ("registration_number"),
	INDEX "idx_hospitals_status" ("status")
);

-- Dumping data for table public.hospitals: -1 rows
/*!40000 ALTER TABLE "hospitals" DISABLE KEYS */;
REPLACE INTO "hospitals" ("id", "name", "address", "phone", "email", "registration_number", "status", "created_at", "updated_at") VALUES
	('85ce395a-541c-431d-b371-9286a86f6386', 'RS Telkom Medika', 'Gedung Business Center, Jl. Telekomunikasi, Sukapura, Dayeuhkolot, Bandung Regency, West Java 40257', '081111500115', 'admin@telkommedika.id', 'RS-2026-001', 'ACTIVE', '2026-09-12 12:01:51.046007+00', '2026-09-12 12:03:29.595756+00');
/*!40000 ALTER TABLE "hospitals" ENABLE KEYS */;

-- Dumping structure for table public.hospital_staff
CREATE TABLE IF NOT EXISTS "hospital_staff" (
	"id" UUID NOT NULL,
	"user_id" UUID NOT NULL,
	"hospital_id" UUID NOT NULL,
	"full_name" VARCHAR(255) NOT NULL,
	"position" VARCHAR(100) NOT NULL DEFAULT '',
	"created_at" TIMESTAMPTZ NOT NULL DEFAULT 'now()',
	"updated_at" TIMESTAMPTZ NOT NULL DEFAULT 'now()',
	PRIMARY KEY ("id"),
	UNIQUE INDEX "hospital_staff_user_id_key" ("user_id"),
	INDEX "idx_hospital_staff_hospital_id" ("hospital_id"),
	CONSTRAINT "hospital_staff_hospital_id_fkey" FOREIGN KEY ("hospital_id") REFERENCES "hospitals" ("id") ON UPDATE NO ACTION ON DELETE CASCADE,
	CONSTRAINT "hospital_staff_user_id_fkey" FOREIGN KEY ("user_id") REFERENCES "users" ("id") ON UPDATE NO ACTION ON DELETE CASCADE
);

-- Dumping data for table public.hospital_staff: -1 rows
/*!40000 ALTER TABLE "hospital_staff" DISABLE KEYS */;
REPLACE INTO "hospital_staff" ("id", "user_id", "hospital_id", "full_name", "position", "created_at", "updated_at") VALUES
	('93233718-c615-48a5-b16a-3c9767323227', '76568a30-17db-40ee-a4a3-04af16793773', '85ce395a-541c-431d-b371-9286a86f6386', 'Bala Lazuardi', 'Petugas Pendaftaran', '2026-09-12 12:05:37.538202+00', '2026-09-12 12:05:37.538202+00');
/*!40000 ALTER TABLE "hospital_staff" ENABLE KEYS */;

-- Dumping structure for table public.identity_verifications
CREATE TABLE IF NOT EXISTS "identity_verifications" (
	"id" UUID NOT NULL,
	"patient_id" UUID NOT NULL,
	"status" VARCHAR(30) NOT NULL DEFAULT 'PENDING',
	"document_type" VARCHAR(30) NOT NULL DEFAULT 'KTP',
	"document_storage_key" TEXT NULL DEFAULT NULL,
	"extracted_nik" TEXT NULL DEFAULT NULL,
	"extracted_full_name" TEXT NULL DEFAULT NULL,
	"extracted_date_of_birth" DATE NULL DEFAULT NULL,
	"extracted_gender" VARCHAR(10) NULL DEFAULT NULL,
	"document_status" VARCHAR(30) NULL DEFAULT NULL,
	"liveness_status" VARCHAR(30) NULL DEFAULT NULL,
	"face_match_status" VARCHAR(30) NULL DEFAULT NULL,
	"face_match_score" NUMERIC(5,4) NULL DEFAULT NULL,
	"provider" VARCHAR(50) NULL DEFAULT NULL,
	"provider_reference" VARCHAR(255) NULL DEFAULT NULL,
	"failure_reason" VARCHAR(100) NULL DEFAULT NULL,
	"verified_at" TIMESTAMPTZ NULL DEFAULT NULL,
	"expires_at" TIMESTAMPTZ NULL DEFAULT NULL,
	"created_at" TIMESTAMPTZ NOT NULL DEFAULT 'now()',
	"updated_at" TIMESTAMPTZ NOT NULL DEFAULT 'now()',
	"selfie_storage_key" TEXT NULL DEFAULT NULL,
	PRIMARY KEY ("id"),
	INDEX "idx_identity_verifications_patient_id" ("patient_id"),
	INDEX "idx_identity_verifications_status" ("status"),
	INDEX "idx_identity_verifications_expires_at" ("expires_at"),
	INDEX "idx_identity_verifications_provider_reference" ("provider_reference"),
	CONSTRAINT "identity_verifications_patient_id_fkey" FOREIGN KEY ("patient_id") REFERENCES "patient_profiles" ("id") ON UPDATE NO ACTION ON DELETE CASCADE,
	CONSTRAINT "identity_verifications_document_type_check" CHECK (((document_type)::text = 'KTP'::text)),
	CONSTRAINT "identity_verifications_face_match_score_check" CHECK (((face_match_score IS NULL) OR ((face_match_score >= (0)::numeric) AND (face_match_score <= (1)::numeric)))),
	CONSTRAINT "identity_verifications_status_check" CHECK (((status)::text = ANY ((ARRAY['PENDING'::character varying, 'PROCESSING'::character varying, 'DOCUMENT_REJECTED'::character varying, 'DATA_MISMATCH'::character varying, 'LIVENESS_FAILED'::character varying, 'FACE_MISMATCH'::character varying, 'MANUAL_REVIEW'::character varying, 'VERIFIED'::character varying, 'EXPIRED'::character varying])::text[])))
);

-- Dumping data for table public.identity_verifications: -1 rows
/*!40000 ALTER TABLE "identity_verifications" DISABLE KEYS */;
REPLACE INTO "identity_verifications" ("id", "patient_id", "status", "document_type", "document_storage_key", "extracted_nik", "extracted_full_name", "extracted_date_of_birth", "extracted_gender", "document_status", "liveness_status", "face_match_status", "face_match_score", "provider", "provider_reference", "failure_reason", "verified_at", "expires_at", "created_at", "updated_at", "selfie_storage_key") VALUES
	('05ebea76-4a78-433b-acc4-b09c13f0d125', 'c6acab0f-8506-4f72-8a82-e19be15de335', 'EXPIRED', 'KTP', 'identity/c6acab0f-8506-4f72-8a82-e19be15de335/05ebea76-4a78-433b-acc4-b09c13f0d125/0b208ae6-a370-4229-8de3-4b105e194cb6.jpg', NULL, NULL, NULL, NULL, NULL, 'MANUAL_REVIEW', NULL, NULL, NULL, NULL, NULL, NULL, '2026-09-29 13:59:11.12121+00', '2026-09-29 13:44:11.246665+00', '2026-10-01 12:24:06.532567+00', 'identity/c6acab0f-8506-4f72-8a82-e19be15de335/05ebea76-4a78-433b-acc4-b09c13f0d125/selfie/b52e7c0b-8f75-492a-b9fe-5c3b41b22f43.jpg'),
	('4811e3d3-eab7-451d-8eae-961b9919a4f9', 'c6acab0f-8506-4f72-8a82-e19be15de335', 'PENDING', 'KTP', NULL, NULL, NULL, NULL, NULL, NULL, NULL, NULL, NULL, NULL, NULL, NULL, NULL, '2026-10-08 10:49:25.058573+00', '2026-10-08 10:34:27.291143+00', '2026-10-08 10:34:27.291143+00', NULL);
/*!40000 ALTER TABLE "identity_verifications" ENABLE KEYS */;

-- Dumping structure for table public.patient_profiles
CREATE TABLE IF NOT EXISTS "patient_profiles" (
	"id" UUID NOT NULL,
	"user_id" UUID NOT NULL,
	"patient_code" VARCHAR(20) NOT NULL,
	"full_name" VARCHAR(255) NOT NULL,
	"nik" TEXT NOT NULL,
	"date_of_birth" DATE NOT NULL,
	"gender" VARCHAR(10) NOT NULL,
	"phone_number" VARCHAR(30) NOT NULL,
	"address" TEXT NOT NULL,
	"blood_type" VARCHAR(5) NULL DEFAULT NULL,
	"insurance_number" TEXT NULL DEFAULT NULL,
	"emergency_contact_name" VARCHAR(255) NULL DEFAULT NULL,
	"emergency_contact_phone" VARCHAR(30) NULL DEFAULT NULL,
	"created_at" TIMESTAMPTZ NOT NULL DEFAULT 'now()',
	"updated_at" TIMESTAMPTZ NOT NULL DEFAULT 'now()',
	"nik_hash" VARCHAR(64) NOT NULL,
	"drug_allergy" TEXT NULL DEFAULT NULL,
	"medical_history" TEXT NULL DEFAULT NULL,
	PRIMARY KEY ("id"),
	UNIQUE INDEX "patient_profiles_user_id_key" ("user_id"),
	UNIQUE INDEX "patient_profiles_patient_code_key" ("patient_code"),
	INDEX "idx_patient_profiles_patient_code" ("patient_code"),
	UNIQUE INDEX "patient_profiles_nik_hash_key" ("nik_hash"),
	INDEX "idx_patient_profiles_nik_hash" ("nik_hash"),
	CONSTRAINT "patient_profiles_user_id_fkey" FOREIGN KEY ("user_id") REFERENCES "users" ("id") ON UPDATE NO ACTION ON DELETE CASCADE,
	CONSTRAINT "patient_profiles_blood_type_check" CHECK (((blood_type IS NULL) OR ((blood_type)::text = ANY ((ARRAY['A+'::character varying, 'A-'::character varying, 'B+'::character varying, 'B-'::character varying, 'AB+'::character varying, 'AB-'::character varying, 'O+'::character varying, 'O-'::character varying])::text[]))))
);

-- Dumping data for table public.patient_profiles: -1 rows
/*!40000 ALTER TABLE "patient_profiles" DISABLE KEYS */;
REPLACE INTO "patient_profiles" ("id", "user_id", "patient_code", "full_name", "nik", "date_of_birth", "gender", "phone_number", "address", "blood_type", "insurance_number", "emergency_contact_name", "emergency_contact_phone", "created_at", "updated_at", "nik_hash", "drug_allergy", "medical_history") VALUES
	('c6acab0f-8506-4f72-8a82-e19be15de335', '40551d73-8ad5-43f4-a4dd-c44f046718d0', 'AKS-dca183a1', 'Bala Lazuardi', 'vJpL/mAIhiFpv6pBd0vhuoyyJNgTFj6+sBNZEO0Duybf3Ip3Sd2nmaC6ozA=', '1998-06-12', 'MALE', '081234567890', 'Jl. Telekomunikasi, Bandung', NULL, NULL, NULL, NULL, '2026-09-22 13:54:45.41624+00', '2026-10-08 06:20:14.963843+00', '1ce6d61b4be8574dca383e02e5962f5a8ea0dc72d85d9e7e13fe3aac661b0754', NULL, NULL);
/*!40000 ALTER TABLE "patient_profiles" ENABLE KEYS */;

-- Dumping structure for table public.patient_qr_credentials
CREATE TABLE IF NOT EXISTS "patient_qr_credentials" (
	"id" UUID NOT NULL DEFAULT 'gen_random_uuid()',
	"user_id" UUID NOT NULL,
	"display_code" VARCHAR(32) NOT NULL,
	"token_hash" VARCHAR(64) NOT NULL,
	"is_active" BOOLEAN NOT NULL DEFAULT 'true',
	"created_at" TIMESTAMPTZ NOT NULL DEFAULT 'now()',
	"updated_at" TIMESTAMPTZ NOT NULL DEFAULT 'now()',
	INDEX "idx_patient_qr_token_hash" ("token_hash"),
	PRIMARY KEY ("id"),
	UNIQUE INDEX "patient_qr_credentials_user_id_key" ("user_id"),
	UNIQUE INDEX "patient_qr_credentials_display_code_key" ("display_code"),
	UNIQUE INDEX "patient_qr_credentials_token_hash_key" ("token_hash"),
	INDEX "idx_patient_qr_user_id" ("user_id"),
	CONSTRAINT "patient_qr_credentials_user_id_fkey" FOREIGN KEY ("user_id") REFERENCES "users" ("id") ON UPDATE NO ACTION ON DELETE CASCADE
);

-- Dumping data for table public.patient_qr_credentials: -1 rows
/*!40000 ALTER TABLE "patient_qr_credentials" DISABLE KEYS */;
REPLACE INTO "patient_qr_credentials" ("id", "user_id", "display_code", "token_hash", "is_active", "created_at", "updated_at") VALUES
	('17ab866f-1c58-4810-a656-cbe65f8c1a9c', '40551d73-8ad5-43f4-a4dd-c44f046718d0', '65F7CFFA', 'b361bab29d3a1b08f1c498832bbdbb22631cd607327f8476a6730db43dd36068', 'true', '2026-09-23 15:06:19.242158+00', '2026-10-08 10:20:01.27562+00');
/*!40000 ALTER TABLE "patient_qr_credentials" ENABLE KEYS */;

-- Dumping structure for table public.schema_migrations
CREATE TABLE IF NOT EXISTS "schema_migrations" (
	"version" BIGINT NOT NULL,
	"dirty" BOOLEAN NOT NULL,
	PRIMARY KEY ("version")
);

-- Dumping data for table public.schema_migrations: -1 rows
/*!40000 ALTER TABLE "schema_migrations" DISABLE KEYS */;
REPLACE INTO "schema_migrations" ("version", "dirty") VALUES
	(8, 'false');
/*!40000 ALTER TABLE "schema_migrations" ENABLE KEYS */;

-- Dumping structure for table public.users
CREATE TABLE IF NOT EXISTS "users" (
	"id" UUID NOT NULL,
	"clerk_user_id" VARCHAR(255) NOT NULL,
	"role" VARCHAR(30) NOT NULL,
	"status" VARCHAR(30) NOT NULL DEFAULT 'ACTIVE',
	"created_at" TIMESTAMPTZ NOT NULL DEFAULT 'now()',
	"updated_at" TIMESTAMPTZ NOT NULL DEFAULT 'now()',
	PRIMARY KEY ("id"),
	UNIQUE INDEX "users_clerk_user_id_key" ("clerk_user_id")
);

-- Dumping data for table public.users: -1 rows
/*!40000 ALTER TABLE "users" DISABLE KEYS */;
REPLACE INTO "users" ("id", "clerk_user_id", "role", "status", "created_at", "updated_at") VALUES
	('76a3916b-ebf3-4221-8c30-60b646b8cc47', 'user_3IRmuwqtYLPkOaHzI2N6OX2QtsB', 'ADMIN', 'ACTIVE', '2026-09-12 11:21:38.431286+00', '2026-09-12 11:21:38.431286+00'),
	('40551d73-8ad5-43f4-a4dd-c44f046718d0', 'user_3IaWnMdZ4ZX8DTeFYHEXvwyHb9u', 'PATIENT', 'ACTIVE', '2026-09-12 11:26:20.672214+00', '2026-09-12 11:26:20.672214+00'),
	('76568a30-17db-40ee-a4a3-04af16793773', 'user_3IaWzgsGnLN5L0xcHnJFxTlrlVs', 'HOSPITAL_STAFF', 'ACTIVE', '2026-09-12 11:30:25.693088+00', '2026-09-12 12:05:37.727098+00');
/*!40000 ALTER TABLE "users" ENABLE KEYS */;

/*!40103 SET TIME_ZONE=IFNULL(@OLD_TIME_ZONE, 'system') */;
/*!40101 SET SQL_MODE=IFNULL(@OLD_SQL_MODE, '') */;
/*!40014 SET FOREIGN_KEY_CHECKS=IFNULL(@OLD_FOREIGN_KEY_CHECKS, 1) */;
/*!40101 SET CHARACTER_SET_CLIENT=@OLD_CHARACTER_SET_CLIENT */;
/*!40111 SET SQL_NOTES=IFNULL(@OLD_SQL_NOTES, 1) */;
