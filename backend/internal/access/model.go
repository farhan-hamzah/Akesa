package access

import (
	"time"

	"github.com/farhan-hamzah/Akesa/backend/internal/datacategory"
	"github.com/google/uuid"
)

type Status string

const (
	StatusPending  Status = "PENDING"
	StatusApproved Status = "APPROVED"
	StatusRejected Status = "REJECTED"
	StatusRevoked  Status = "REVOKED"
)

type Request struct {
	ID          uuid.UUID               `json:"id"`
	HospitalID  uuid.UUID               `json:"hospitalId"`
	PatientID   uuid.UUID               `json:"patientId"`
	RequestedBy uuid.UUID               `json:"requestedBy"`
	Purpose     string                  `json:"purpose"`
	Categories  []datacategory.Category `json:"categories"`
	Status      Status                  `json:"status"`
	DecidedAt   *time.Time              `json:"decidedAt,omitempty"`
	CreatedAt   time.Time               `json:"createdAt"`
	UpdatedAt   time.Time               `json:"updatedAt"`
}

func (r *Request) IsGrantedNow() bool {
	return r.Status == StatusApproved
}
