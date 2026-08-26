package hospital

import (
	"time"

	"github.com/google/uuid"
)

type Status string

const (
	StatusPending  Status = "PENDING"  // registered, not yet verified by admin
	StatusVerified Status = "VERIFIED" // verified but not yet active
	StatusActive   Status = "ACTIVE"
	StatusInactive Status = "INACTIVE"
)

type Hospital struct {
	ID                 uuid.UUID `json:"id"`
	Name               string    `json:"name"`
	Address            string    `json:"address"`
	Phone              string    `json:"phone"`
	Email              string    `json:"email"`
	RegistrationNumber string    `json:"registrationNumber"`
	Status             Status    `json:"status"`
	CreatedAt          time.Time `json:"createdAt"`
	UpdatedAt          time.Time `json:"updatedAt"`
}

// Staff links a `user` (role HOSPITAL_STAFF) account to a specific
// hospital, so a staff member's access requests can be attributed to
// their hospital.
type Staff struct {
	ID         uuid.UUID `json:"id"`
	UserID     uuid.UUID `json:"userId"`
	HospitalID uuid.UUID `json:"hospitalId"`
	FullName   string    `json:"fullName"`
	Position   string    `json:"position"`
	CreatedAt  time.Time `json:"createdAt"`
	UpdatedAt  time.Time `json:"updatedAt"`
}
