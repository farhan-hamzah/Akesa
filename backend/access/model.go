package access

import (
	"time"

	"github.com/google/uuid"
)

type Status string

const (
	StatusPending  Status = "PENDING"
	StatusApproved Status = "APPROVED"
	StatusRejected Status = "REJECTED"
	StatusRevoked  Status = "REVOKED"
)

type AccessRequest struct {
	ID          uuid.UUID  `json:"id"`
	PatientID   uuid.UUID  `json:"patientId"`
	HospitalID  uuid.UUID  `json:"hospitalId"`
	RequestedBy uuid.UUID  `json:"requestedBy"`
	Reason      string     `json:"reason"`
	Status      Status     `json:"status"`
	RequestedAt time.Time  `json:"requestedAt"`
	ReviewedAt  *time.Time `json:"reviewedAt,omitempty"`
	ReviewedBy  *uuid.UUID `json:"reviewedBy,omitempty"`
	CreatedAt   time.Time  `json:"createdAt"`
	UpdatedAt   time.Time  `json:"updatedAt"`
}
