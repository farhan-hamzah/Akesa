package patient

import "errors"

var (
	ErrProfileNotFound      = errors.New("patient profile not found")
	ErrProfileExists        = errors.New("patient profile already exists for this user")
	ErrInvalidInput         = errors.New("invalid patient profile input")
	ErrPatientCodeTaken     = errors.New("patient code already in use")
	ErrNIKAlreadyRegistered = errors.New("NIK is already registered to another profile")
)
