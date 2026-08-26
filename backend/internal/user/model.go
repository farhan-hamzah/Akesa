package user

import (
	"time"

	"github.com/google/uuid"
)

type Role string

const (
	RolePatient       Role = "PATIENT"
	RoleHospitalStaff Role = "HOSPITAL_STAFF"
	RoleAdmin         Role = "ADMIN"
)

type Status string

const (
	StatusActive   Status = "ACTIVE"
	StatusInactive Status = "INACTIVE"
)

type User struct {
	ID          uuid.UUID `json:"id"`
	ClerkUserID string    `json:"clerkUserId"`
	Role        Role      `json:"role"`
	Status      Status    `json:"status"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}
