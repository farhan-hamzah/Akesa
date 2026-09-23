package audit

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestAuditHashChainContinuity(t *testing.T) {
	patientID := uuid.New()
	doctorID := uuid.New()
	now := time.Now().UTC()

	h1 := computeHash(genesisHash, "PATIENT_PROFILE", patientID, ActionProfileUpdated, patientID, []byte(`{"profileHash":"hash-v1"}`), now)

	h2 := computeHash(h1, "ACCESS_REQUEST", patientID, ActionAccessApproved, doctorID, []byte(`{"categories":["identitas"]}`), now.Add(time.Minute))

	h3 := computeHash(h2, "ACCESS_REQUEST", patientID, ActionDataAccessed, doctorID, []byte(`{"dataHash":"hash-v1"}`), now.Add(2*time.Minute))

	if h1 == "" || h2 == "" || h3 == "" {
		t.Fatalf("hash computation returned empty string")
	}

	if h1 == h2 || h2 == h3 {
		t.Fatalf("hashes should be unique across blocks")
	}

	tamperedH1 := computeHash(genesisHash, "PATIENT_PROFILE", patientID, ActionProfileUpdated, patientID, []byte(`{"profileHash":"HACKED-TAMPERED-DATA"}`), now)

	if tamperedH1 == h1 {
		t.Fatalf("tampered payload must produce different hash")
	}

	tamperedH2 := computeHash(tamperedH1, "ACCESS_REQUEST", patientID, ActionAccessApproved, doctorID, []byte(`{"categories":["identitas"]}`), now.Add(time.Minute))

	if tamperedH2 == h2 {
		t.Fatalf("chain failure: Block 2 hash must not match when previous block was tampered")
	}
}
