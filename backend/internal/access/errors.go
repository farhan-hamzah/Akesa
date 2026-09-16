package access

import "errors"

var (
	ErrRequestNotFound = errors.New("access request not found")
	ErrNotOwner        = errors.New("this access request does not belong to you")
	ErrNotPending      = errors.New("access request is not pending")
	ErrNotApproved     = errors.New("access request is not approved")
	ErrPatientNotFound = errors.New("patient not found for the given patient code")
	ErrEmptyPurpose    = errors.New("purpose is required")
	ErrDataTampered    = errors.New("patient profile data integrity violation - hash mismatch")
)
