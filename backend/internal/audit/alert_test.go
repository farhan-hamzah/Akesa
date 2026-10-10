package audit

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/farhan-hamzah/Akesa/backend/internal/auth"
	"github.com/farhan-hamzah/Akesa/backend/internal/patient"
	"github.com/google/uuid"
)

type mockPatientResolver struct {
	profile *patient.Profile
	err     error
}

func (m *mockPatientResolver) GetByUserID(ctx context.Context, userID uuid.UUID) (*patient.Profile, error) {
	return m.profile, m.err
}

func TestActionConstants(t *testing.T) {
	if ActionIntegrityViolationBlocked != "INTEGRITY_VIOLATION_BLOCKED" {
		t.Fatalf("expected ActionIntegrityViolationBlocked to be INTEGRITY_VIOLATION_BLOCKED, got %s", ActionIntegrityViolationBlocked)
	}
}

func TestSecurityAlertsHandler_Empty(t *testing.T) {
	// A handler with nil db repo won't be called directly for DB in this unit test,
	// but we can verify the Handler struct initialization and resolver binding.
	resolver := &mockPatientResolver{
		profile: &patient.Profile{
			ID:     uuid.New(),
			UserID: uuid.New(),
		},
	}

	h := NewHandler(nil, resolver)
	if h.patientResolver == nil {
		t.Fatalf("expected patientResolver to be set")
	}
}

func TestMyProfileHistoryHandler_Unauthorized(t *testing.T) {
	h := NewHandler(nil, nil)
	req := httptest.NewRequest("GET", "/api/v1/patient/history", nil)
	w := httptest.NewRecorder()

	h.MyProfileHistory(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 Unauthorized for unauthenticated request, got %d", w.Code)
	}
}

func TestMyProfileHistoryHandler_ProfileNotFound(t *testing.T) {
	userID := uuid.New()
	resolver := &mockPatientResolver{
		profile: nil,
		err:     patient.ErrProfileNotFound,
	}

	h := NewHandler(nil, resolver)
	req := httptest.NewRequest("GET", "/api/v1/patient/history", nil)
	req = req.WithContext(auth.WithAuthUser(req.Context(), &auth.AuthUser{ID: userID, Role: "PATIENT", Status: "ACTIVE"}))
	w := httptest.NewRecorder()

	h.MyProfileHistory(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404 Not Found when profile does not exist, got %d", w.Code)
	}
}

func TestTamperAuditPayloadIntegrity(t *testing.T) {
	patientID := uuid.New()
	hospitalID := uuid.New()
	staffID := uuid.New()
	requestID := uuid.New()
	now := time.Now().UTC()

	payloadMap := map[string]any{
		"error":        "hash_mismatch",
		"currentHash":  "current-bad-hash",
		"recordedHash": "ledger-valid-hash",
		"patientId":    patientID.String(),
		"hospitalId":   hospitalID.String(),
		"reason":       "Real-time patient profile hash does not match anchored ledger hash",
	}

	payloadBytes, err := json.Marshal(payloadMap)
	if err != nil {
		t.Fatalf("marshal payload failed: %v", err)
	}

	tamperHash := computeHash(genesisHash, "ACCESS_REQUEST", requestID, ActionIntegrityViolationBlocked, staffID, payloadBytes, now)
	if tamperHash == "" {
		t.Fatalf("tamper hash must not be empty")
	}

	// Verify unmarshaling back to ensure keys are intact
	var parsed map[string]any
	if err := json.Unmarshal(payloadBytes, &parsed); err != nil {
		t.Fatalf("unmarshal payload failed: %v", err)
	}

	if parsed["patientId"] != patientID.String() {
		t.Errorf("expected patientId %s, got %v", patientID, parsed["patientId"])
	}
	if parsed["hospitalId"] != hospitalID.String() {
		t.Errorf("expected hospitalId %s, got %v", hospitalID, parsed["hospitalId"])
	}
	if parsed["error"] != "hash_mismatch" {
		t.Errorf("expected error hash_mismatch, got %v", parsed["error"])
	}
}

func TestLogPayloadJSONSerialization(t *testing.T) {
	logEntry := Log{
		ID:         uuid.New(),
		EntityType: "ACCESS_REQUEST",
		Action:     "DATA_ACCESSED",
		Payload:    json.RawMessage(`{"hospitalId":"123","categories":["IDENTITY"]}`),
	}

	bytes, err := json.Marshal(logEntry)
	if err != nil {
		t.Fatalf("failed to marshal log: %v", err)
	}

	var raw map[string]any
	if err := json.Unmarshal(bytes, &raw); err != nil {
		t.Fatalf("failed to unmarshal log JSON: %v", err)
	}

	payloadMap, ok := raw["payload"].(map[string]any)
	if !ok {
		t.Fatalf("expected payload to be a JSON object, but got: %T (%v)", raw["payload"], raw["payload"])
	}

	if payloadMap["hospitalId"] != "123" {
		t.Errorf("expected hospitalId to be 123, got %v", payloadMap["hospitalId"])
	}
}
