package patientqr

import "github.com/google/uuid"

type Credential struct {
	ID          uuid.UUID `json:"id"`
	UserID      uuid.UUID `json:"userId"`
	DisplayCode string    `json:"displayCode"`
	TokenHash   string    `json:"-"`
	Token       string    `json:"-"`
	IsActive    bool      `json:"isActive"`
}
