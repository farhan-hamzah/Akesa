package access

import (
	"strings"

	"github.com/farhan-hamzah/Akesa/backend/internal/datacategory"
)

type CreateRequestInput struct {
	PatientCode string
	Purpose     string
	Categories  []datacategory.Category
}

type CreateRequestBody struct {
	PatientCode string   `json:"patientCode"`
	Purpose     string   `json:"purpose"`
	Categories  []string `json:"categories"`
}

func (b CreateRequestBody) Validate() (CreateRequestInput, error) {
	code := strings.TrimSpace(b.PatientCode)
	purpose := strings.TrimSpace(b.Purpose)

	if code == "" {
		return CreateRequestInput{}, fieldErr("patientCode", "wajib diisi")
	}
	if purpose == "" {
		return CreateRequestInput{}, ErrEmptyPurpose
	}

	categories := make([]datacategory.Category, len(b.Categories))
	for i, c := range b.Categories {
		categories[i] = datacategory.Category(strings.ToUpper(strings.TrimSpace(c)))
	}

	validated, err := datacategory.ValidateAll(categories)
	if err != nil {
		return CreateRequestInput{}, err
	}

	return CreateRequestInput{PatientCode: code, Purpose: purpose, Categories: validated}, nil
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
