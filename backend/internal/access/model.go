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

// Request is one hospital's request to view a scoped set of a patient's
// data categories, and the patient's decision on it. This is the central
// object of the whole system: "rumah sakit dapat mengajukan permintaan
// akses ... pasien memiliki hak untuk menyetujui atau menolak".
type Request struct {
	ID          uuid.UUID               `json:"id"`
	HospitalID  uuid.UUID               `json:"hospitalId"`
	PatientID   uuid.UUID               `json:"patientId"`
	RequestedBy uuid.UUID               `json:"requestedBy"` // hospital staff user id
	Purpose     string                  `json:"purpose"`
	Categories  []datacategory.Category `json:"categories"`
	Status      Status                  `json:"status"`
	DecidedAt   *time.Time              `json:"decidedAt,omitempty"`
	CreatedAt   time.Time               `json:"createdAt"`
	UpdatedAt   time.Time               `json:"updatedAt"`
}

// IsGrantedNow tells whether a hospital may currently read patient data
// under this request - approved and not later revoked.
func (r *Request) IsGrantedNow() bool {
	return r.Status == StatusApproved
}
