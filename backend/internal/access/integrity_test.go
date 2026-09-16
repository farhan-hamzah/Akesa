package access

import (
	"errors"
	"testing"
)

func TestIntegrityErrors(t *testing.T) {
	err := ErrDataTampered
	if !errors.Is(err, ErrDataTampered) {
		t.Fatalf("expected ErrDataTampered match")
	}
	if err.Error() != "patient profile data integrity violation - hash mismatch" {
		t.Fatalf("unexpected error message: %s", err.Error())
	}
}
