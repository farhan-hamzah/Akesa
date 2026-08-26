package patient

import (
	"time"

	"github.com/google/uuid"
)

type Gender string

const (
	GenderMale   Gender = "MALE"
	GenderFemale Gender = "FEMALE"
)

type Patient struct {
	ID          uuid.UUID  `json:"id"`
	UserID      uuid.UUID  `json:"userId"`
	NIKHash     string     `json:"nikHash"`
	FullName    string     `json:"fullName"`
	DateOfBirth *time.Time `json:"dateOfBirth,omitempty"`
	Gender      *Gender    `json:"gender,omitempty"`
	Phone       *string    `json:"phone,omitempty"`
	Address     *string    `json:"address,omitempty"`
	CreatedAt   time.Time  `json:"createdAt"`
	UpdatedAt   time.Time  `json:"updatedAt"`
}
