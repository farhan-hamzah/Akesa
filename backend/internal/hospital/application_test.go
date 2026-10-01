package hospital

import (
	"testing"
)

func TestApplicationStatus_IsValid(t *testing.T) {
	tests := []struct {
		status ApplicationStatus
		valid  bool
	}{
		{ApplicationPendingReview, true},
		{ApplicationRevisionRequired, true},
		{ApplicationRejected, true},
		{ApplicationApproved, true},
		{ApplicationStatus("UNKNOWN"), false},
		{ApplicationStatus(""), false},
		{ApplicationStatus("pending_review"), false},
	}

	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			if got := tt.status.IsValid(); got != tt.valid {
				t.Errorf("ApplicationStatus(%q).IsValid() = %v, want %v", tt.status, got, tt.valid)
			}
		})
	}
}

func TestUpdateApplicationRequest_Validate(t *testing.T) {
	tests := []struct {
		name        string
		req         UpdateApplicationRequest
		wantErr     bool
		wantPicPos  string
		wantPicName string
	}{
		{
			name: "valid request with explicit PIC details",
			req: UpdateApplicationRequest{
				Name:               "RS Harapan Bangsa",
				Address:            "Jl. Sudirman No. 10",
				Phone:              "021-1234567",
				Email:              "kontak@harapanbangsa.com",
				RegistrationNumber: "REG-999888",
				NPWP:               "01.234.567.8-000.000",
				LicenseDocumentURL: "https://storage.akesa.id/docs/revised.pdf",
				PICPosition:        "Direktur Utama",
				PICFullName:        "Dr. Budi Santoso",
			},
			wantErr:     false,
			wantPicPos:  "Direktur Utama",
			wantPicName: "Dr. Budi Santoso",
		},
		{
			name: "valid request with default PIC details",
			req: UpdateApplicationRequest{
				Name:               "RS Harapan Bangsa",
				Address:            "Jl. Sudirman No. 10",
				Phone:              "021-1234567",
				Email:              "kontak@harapanbangsa.com",
				RegistrationNumber: "REG-999888",
			},
			wantErr:     false,
			wantPicPos:  "PIC Rumah Sakit",
			wantPicName: "Perwakilan Rumah Sakit",
		},
		{
			name: "missing name",
			req: UpdateApplicationRequest{
				Address:            "Jl. Sudirman No. 10",
				Phone:              "021-1234567",
				Email:              "kontak@harapanbangsa.com",
				RegistrationNumber: "REG-999888",
			},
			wantErr: true,
		},
		{
			name: "invalid email",
			req: UpdateApplicationRequest{
				Name:               "RS Harapan Bangsa",
				Address:            "Jl. Sudirman No. 10",
				Phone:              "021-1234567",
				Email:              "invalid-email-format",
				RegistrationNumber: "REG-999888",
			},
			wantErr: true,
		},
		{
			name: "missing registration number",
			req: UpdateApplicationRequest{
				Name:    "RS Harapan Bangsa",
				Address: "Jl. Sudirman No. 10",
				Phone:   "021-1234567",
				Email:   "kontak@harapanbangsa.com",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input, picPos, picName, err := tt.req.Validate()
			if (err != nil) != tt.wantErr {
				t.Fatalf("UpdateApplicationRequest.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr {
				if picPos != tt.wantPicPos {
					t.Errorf("picPos = %v, want %v", picPos, tt.wantPicPos)
				}
				if picName != tt.wantPicName {
					t.Errorf("picName = %v, want %v", picName, tt.wantPicName)
				}
				if input.Name != tt.req.Name {
					t.Errorf("input.Name = %v, want %v", input.Name, tt.req.Name)
				}
			}
		})
	}
}

func TestAdminReviewRequest_Validate(t *testing.T) {
	tests := []struct {
		name    string
		req     AdminReviewRequest
		action  string
		wantErr bool
	}{
		{
			name:    "valid review with notes",
			req:     AdminReviewRequest{Notes: "Mohon upload ulang dokumen izin operasional yang jelas"},
			action:  "meminta revisi",
			wantErr: false,
		},
		{
			name:    "empty notes",
			req:     AdminReviewRequest{Notes: ""},
			action:  "meminta revisi",
			wantErr: true,
		},
		{
			name:    "whitespace only notes",
			req:     AdminReviewRequest{Notes: "   \t\n  "},
			action:  "menolak pendaftaran",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			notes, err := tt.req.Validate(tt.action)
			if (err != nil) != tt.wantErr {
				t.Fatalf("AdminReviewRequest.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && notes == "" {
				t.Errorf("expected non-empty notes")
			}
		})
	}
}
