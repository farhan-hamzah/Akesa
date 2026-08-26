package hospital

import "strings"

type HospitalInput struct {
	Name               string
	Address            string
	Phone              string
	Email              string
	RegistrationNumber string
}

type HospitalRequest struct {
	Name               string `json:"name"`
	Address            string `json:"address"`
	Phone              string `json:"phone"`
	Email              string `json:"email"`
	RegistrationNumber string `json:"registrationNumber"`
}

func (req HospitalRequest) Validate() (HospitalInput, error) {
	name := strings.TrimSpace(req.Name)
	address := strings.TrimSpace(req.Address)
	phone := strings.TrimSpace(req.Phone)
	email := strings.TrimSpace(strings.ToLower(req.Email))
	regNumber := strings.TrimSpace(req.RegistrationNumber)

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
	}, nil
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
