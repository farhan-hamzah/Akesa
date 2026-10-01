package hospital

import (
	"strings"
	"testing"
	"time"
)

func TestGenerateSecureKeyCode(t *testing.T) {
	seen := make(map[string]bool)
	for i := 0; i < 100; i++ {
		code, err := generateSecureKeyCode()
		if err != nil {
			t.Fatalf("failed to generate key: %v", err)
		}
		if !strings.HasPrefix(code, "AKESA-") {
			t.Errorf("expected code to start with 'AKESA-', got %s", code)
		}
		parts := strings.Split(code, "-")
		if len(parts) != 3 {
			t.Errorf("expected 3 parts separated by hyphen, got %d in %s", len(parts), code)
		}
		if len(parts[1]) != 4 || len(parts[2]) != 4 {
			t.Errorf("expected 4 chars per segment, got %s", code)
		}
		if seen[code] {
			t.Errorf("duplicate code generated: %s", code)
		}
		seen[code] = true
	}
}

func TestInvitation_IsExpired(t *testing.T) {
	invNotExpired := &Invitation{
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}
	if invNotExpired.IsExpired() {
		t.Errorf("expected invitation not to be expired")
	}

	invExpired := &Invitation{
		ExpiresAt: time.Now().Add(-1 * time.Minute),
	}
	if !invExpired.IsExpired() {
		t.Errorf("expected invitation to be expired")
	}
}

func TestRegisterWithKeyRequest_Validate(t *testing.T) {
	tests := []struct {
		name    string
		req     RegisterWithKeyRequest
		wantErr bool
	}{
		{
			name: "valid request",
			req: RegisterWithKeyRequest{
				KeyCode:            "AKESA-7X9K-3B2M",
				Name:               "RSUP Dr. Sardjito",
				Address:            "Jl. Kesehatan No. 1, Yogyakarta",
				Phone:              "0274-587333",
				Email:              "kontak@sardjito.co.id",
				RegistrationNumber: "3471012",
				NPWP:               "01.234.567.8-901.000",
				LicenseDocumentURL: "https://storage.akesa.id/docs/license-sardjito.pdf",
			},
			wantErr: false,
		},
		{
			name: "missing key code",
			req: RegisterWithKeyRequest{
				Name:               "RSUP Dr. Sardjito",
				Address:            "Jl. Kesehatan",
				Phone:              "0274",
				Email:              "kontak@sardjito.co.id",
				RegistrationNumber: "3471012",
			},
			wantErr: true,
		},
		{
			name: "missing name",
			req: RegisterWithKeyRequest{
				KeyCode:            "AKESA-7X9K-3B2M",
				Address:            "Jl. Kesehatan",
				Phone:              "0274",
				Email:              "kontak@sardjito.co.id",
				RegistrationNumber: "3471012",
			},
			wantErr: true,
		},
		{
			name: "invalid email",
			req: RegisterWithKeyRequest{
				KeyCode:            "AKESA-7X9K-3B2M",
				Name:               "RSUP Dr. Sardjito",
				Address:            "Jl. Kesehatan",
				Phone:              "0274",
				Email:              "invalid-email",
				RegistrationNumber: "3471012",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, _, _, err := tt.req.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("RegisterWithKeyRequest.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
