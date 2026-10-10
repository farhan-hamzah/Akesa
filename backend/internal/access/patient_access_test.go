package access

import (
	"context"
	"errors"
	"testing"

	"github.com/farhan-hamzah/Akesa/backend/internal/patient"
	"github.com/google/uuid"
)

type mockPatientLookup struct {
	profile *patient.Profile
	err     error
}

func (m *mockPatientLookup) GetByPatientCode(ctx context.Context, code string) (*patient.Profile, error) {
	return m.profile, m.err
}

func (m *mockPatientLookup) GetByID(ctx context.Context, id uuid.UUID) (*patient.Profile, error) {
	return m.profile, m.err
}

func (m *mockPatientLookup) GetByUserID(ctx context.Context, userID uuid.UUID) (*patient.Profile, error) {
	return m.profile, m.err
}

func (m *mockPatientLookup) ComputeHash(profile *patient.Profile) string {
	return "mock-hash"
}

func TestListForPatient_NoProfileReturnsEmpty(t *testing.T) {
	mockPatients := &mockPatientLookup{
		err: patient.ErrProfileNotFound,
	}

	service := NewService(nil, mockPatients, nil, nil)
	userID := uuid.New()

	requests, err := service.ListForPatient(context.Background(), userID)
	if err != nil {
		t.Fatalf("expected no error when profile not found, got %v", err)
	}
	if len(requests) != 0 {
		t.Fatalf("expected empty requests list, got %d items", len(requests))
	}
}

func TestOwnedPendingRequest_NoProfileReturnsErrNotOwner(t *testing.T) {
	mockPatients := &mockPatientLookup{
		err: patient.ErrProfileNotFound,
	}

	service := NewService(nil, mockPatients, nil, nil)
	userID := uuid.New()
	reqID := uuid.New()

	_, err := service.ownedPendingRequest(context.Background(), userID, reqID)
	if !errors.Is(err, ErrNotOwner) {
		t.Fatalf("expected ErrNotOwner, got %v", err)
	}
}

func TestRevoke_NoProfileReturnsErrNotOwner(t *testing.T) {
	mockPatients := &mockPatientLookup{
		err: patient.ErrProfileNotFound,
	}

	service := NewService(nil, mockPatients, nil, nil)
	userID := uuid.New()
	reqID := uuid.New()

	_, err := service.Revoke(context.Background(), userID, reqID)
	if !errors.Is(err, ErrNotOwner) {
		t.Fatalf("expected ErrNotOwner, got %v", err)
	}
}
