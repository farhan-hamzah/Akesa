package patient

import (
	"strings"
	"time"
)

// ProfileInput is the validated shape used for both create and update -
// keeping one struct for both avoids drift between the two code paths.
type ProfileInput struct {
	FullName              string
	NIK                   string
	DateOfBirth           time.Time
	Gender                Gender
	PhoneNumber           string
	Address               string
	BloodType             *string
	InsuranceNumber       *string
	EmergencyContactName  *string
	EmergencyContactPhone *string
}

// ProfileRequest is the raw wire format accepted from the mobile app.
type ProfileRequest struct {
	FullName              string  `json:"fullName"`
	NIK                   string  `json:"nik"`
	DateOfBirth           string  `json:"dateOfBirth"` // "YYYY-MM-DD"
	Gender                string  `json:"gender"`
	PhoneNumber           string  `json:"phoneNumber"`
	Address               string  `json:"address"`
	BloodType             *string `json:"bloodType,omitempty"`
	InsuranceNumber       *string `json:"insuranceNumber,omitempty"`
	EmergencyContactName  *string `json:"emergencyContactName,omitempty"`
	EmergencyContactPhone *string `json:"emergencyContactPhone,omitempty"`
}

// Validate parses and sanity-checks the request, returning a ProfileInput
// ready for the service layer. Keeping validation here (not in the
// handler) means it's unit-testable without spinning up an http.Request.
func (req ProfileRequest) Validate() (ProfileInput, error) {
	fullName := strings.TrimSpace(req.FullName)
	nik := strings.TrimSpace(req.NIK)
	phone := strings.TrimSpace(req.PhoneNumber)
	address := strings.TrimSpace(req.Address)

	if fullName == "" {
		return ProfileInput{}, fieldErr("fullName", "wajib diisi")
	}
	if len(nik) != 16 {
		return ProfileInput{}, fieldErr("nik", "harus terdiri dari 16 digit")
	}
	if phone == "" {
		return ProfileInput{}, fieldErr("phoneNumber", "wajib diisi")
	}
	if address == "" {
		return ProfileInput{}, fieldErr("address", "wajib diisi")
	}

	gender := Gender(strings.ToUpper(strings.TrimSpace(req.Gender)))
	if gender != GenderMale && gender != GenderFemale {
		return ProfileInput{}, fieldErr("gender", "harus MALE atau FEMALE")
	}

	dob, err := time.Parse("2006-01-02", req.DateOfBirth)
	if err != nil {
		return ProfileInput{}, fieldErr("dateOfBirth", "format harus YYYY-MM-DD")
	}
	if dob.After(time.Now()) {
		return ProfileInput{}, fieldErr("dateOfBirth", "tidak boleh di masa depan")
	}

	return ProfileInput{
		FullName:              fullName,
		NIK:                   nik,
		DateOfBirth:           dob,
		Gender:                gender,
		PhoneNumber:           phone,
		Address:               address,
		BloodType:             trimPtr(req.BloodType),
		InsuranceNumber:       trimPtr(req.InsuranceNumber),
		EmergencyContactName:  trimPtr(req.EmergencyContactName),
		EmergencyContactPhone: trimPtr(req.EmergencyContactPhone),
	}, nil
}

func trimPtr(s *string) *string {
	if s == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*s)
	if trimmed == "" {
		return nil
	}
	return &trimmed
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
