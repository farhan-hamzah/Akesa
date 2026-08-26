package patient

import "errors"

var (
	ErrPatientNotFound = errors.New("patient not found")
	ErrPatientExists   = errors.New("patient already exists")
)
