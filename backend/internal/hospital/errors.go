package hospital

import "errors"

var (
	ErrNotFound           = errors.New("hospital not found")
	ErrEmailTaken         = errors.New("email already registered to another hospital")
	ErrRegistrationTaken  = errors.New("registration number already registered")
	ErrStaffNotFound      = errors.New("hospital staff record not found")
	ErrStaffAlreadyLinked = errors.New("user is already linked to a hospital as staff")
	ErrHospitalNotActive  = errors.New("hospital is not active")
)
