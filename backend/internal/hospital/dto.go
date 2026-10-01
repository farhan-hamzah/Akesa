package hospital

import (
	"strings"
	"time"

	"github.com/google/uuid"
)

type HospitalInput struct {
	Name               string
	Address            string
	Phone              string
	Email              string
	RegistrationNumber string
	NPWP               string
	LicenseDocumentURL string
	PICUserID          *uuid.UUID
}

type HospitalRequest struct {
	Name               string `json:"name"`
	Address            string `json:"address"`
	Phone              string `json:"phone"`
	Email              string `json:"email"`
	RegistrationNumber string `json:"registrationNumber"`
	NPWP               string `json:"npwp"`
	LicenseDocumentURL string `json:"licenseDocumentUrl"`
}

func (req HospitalRequest) Validate() (HospitalInput, error) {
	name := strings.TrimSpace(req.Name)
	address := strings.TrimSpace(req.Address)
	phone := strings.TrimSpace(req.Phone)
	email := strings.TrimSpace(strings.ToLower(req.Email))
	regNumber := strings.TrimSpace(req.RegistrationNumber)
	npwp := strings.TrimSpace(req.NPWP)
	licenseDocURL := strings.TrimSpace(req.LicenseDocumentURL)

	if name == "" {
		return HospitalInput{}, fieldErr("name", "wajib diisi")
	}
	if address == "" {
		return HospitalInput{}, fieldErr("address", "wajib diisi")
	}
	if phone == "" {
		return HospitalInput{}, fieldErr("phone", "wajib diisi")
	}
	if !strings.Contains(email, "@") {
		return HospitalInput{}, fieldErr("email", "format email tidak valid")
	}
	if regNumber == "" {
		return HospitalInput{}, fieldErr("registrationNumber", "wajib diisi")
	}

	return HospitalInput{
		Name:               name,
		Address:            address,
		Phone:              phone,
		Email:              email,
		RegistrationNumber: regNumber,
		NPWP:               npwp,
		LicenseDocumentURL: licenseDocURL,
	}, nil
}

type GenerateInvitationRequest struct {
	Email          string `json:"email"`
	ExpiresInHours int    `json:"expiresInHours"`
}

func (req GenerateInvitationRequest) Validate() (string, int, error) {
	email := strings.TrimSpace(strings.ToLower(req.Email))
	if !strings.Contains(email, "@") {
		return "", 0, fieldErr("email", "format email tidak valid")
	}
	expiresIn := req.ExpiresInHours
	if expiresIn <= 0 {
		expiresIn = 72 // default to 72 hours (3 days)
	}
	return email, expiresIn, nil
}

type GenerateInvitationResponse struct {
	KeyCode     string    `json:"keyCode"`
	MagicLink   string    `json:"magicLink"`
	TargetEmail string    `json:"targetEmail"`
	ExpiresAt   time.Time `json:"expiresAt"`
}

type ValidateInvitationResponse struct {
	Valid       bool      `json:"valid"`
	TargetEmail string    `json:"targetEmail"`
	ExpiresAt   time.Time `json:"expiresAt"`
}

type RegisterWithKeyRequest struct {
	KeyCode            string `json:"keyCode"`
	Name               string `json:"name"`
	Address            string `json:"address"`
	Phone              string `json:"phone"`
	Email              string `json:"email"`
	RegistrationNumber string `json:"registrationNumber"`
	NPWP               string `json:"npwp"`
	LicenseDocumentURL string `json:"licenseDocumentUrl"`
	PICPosition        string `json:"picPosition"`
	PICFullName        string `json:"picFullName"`
}

func (req RegisterWithKeyRequest) Validate() (string, HospitalInput, string, string, error) {
	keyCode := strings.TrimSpace(req.KeyCode)
	if keyCode == "" {
		return "", HospitalInput{}, "", "", fieldErr("keyCode", "kunci registrasi wajib diisi")
	}

	hReq := HospitalRequest{
		Name:               req.Name,
		Address:            req.Address,
		Phone:              req.Phone,
		Email:              req.Email,
		RegistrationNumber: req.RegistrationNumber,
		NPWP:               req.NPWP,
		LicenseDocumentURL: req.LicenseDocumentURL,
	}
	in, err := hReq.Validate()
	if err != nil {
		return "", HospitalInput{}, "", "", err
	}

	picPos := strings.TrimSpace(req.PICPosition)
	if picPos == "" {
		picPos = "PIC Rumah Sakit"
	}

	picName := strings.TrimSpace(req.PICFullName)
	if picName == "" {
		picName = "Perwakilan Rumah Sakit"
	}

	return keyCode, in, picPos, picName, nil
}

type UpdateApplicationRequest struct {
	Name               string `json:"name"`
	Address            string `json:"address"`
	Phone              string `json:"phone"`
	Email              string `json:"email"`
	RegistrationNumber string `json:"registrationNumber"`
	NPWP               string `json:"npwp"`
	LicenseDocumentURL string `json:"licenseDocumentUrl"`
	PICPosition        string `json:"picPosition"`
	PICFullName        string `json:"picFullName"`
}

func (req UpdateApplicationRequest) Validate() (HospitalInput, string, string, error) {
	hReq := HospitalRequest{
		Name:               req.Name,
		Address:            req.Address,
		Phone:              req.Phone,
		Email:              req.Email,
		RegistrationNumber: req.RegistrationNumber,
		NPWP:               req.NPWP,
		LicenseDocumentURL: req.LicenseDocumentURL,
	}
	in, err := hReq.Validate()
	if err != nil {
		return HospitalInput{}, "", "", err
	}

	picPos := strings.TrimSpace(req.PICPosition)
	if picPos == "" {
		picPos = "PIC Rumah Sakit"
	}

	picName := strings.TrimSpace(req.PICFullName)
	if picName == "" {
		picName = "Perwakilan Rumah Sakit"
	}

	return in, picPos, picName, nil
}

type AdminReviewRequest struct {
	Notes string `json:"notes"`
}

func (req AdminReviewRequest) Validate(action string) (string, error) {
	notes := strings.TrimSpace(req.Notes)
	if notes == "" {
		return "", fieldErr("notes", "catatan/alasan wajib diisi saat "+action)
	}
	return notes, nil
}

type StaffRequest struct {
	UserID   string `json:"userId"`
	FullName string `json:"fullName"`
	Position string `json:"position"`
}

func fieldErr(field, message string) error {
	return &ValidationError{Field: field, Message: message}
}

type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	return e.Field + ": " + e.Message
}
