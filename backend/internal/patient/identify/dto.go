package identity

type VerificationResponse struct {
	ID              *string `json:"id"`
	Status          *string `json:"status"`
	DocumentType    *string `json:"documentType"`
	DocumentStatus  *string `json:"documentStatus,omitempty"`
	LivenessStatus  *string `json:"livenessStatus,omitempty"`
	FaceMatchStatus *string `json:"faceMatchStatus,omitempty"`
	FailureReason   *string `json:"failureReason,omitempty"`
	VerifiedAt      *string `json:"verifiedAt,omitempty"`
	ExpiresAt       *string `json:"expiresAt,omitempty"`

	DocumentUploaded bool `json:"documentUploaded"`
	SelfieUploaded   bool `json:"selfieUploaded"`
}

type AdminIdentityQueueItem struct {
	ID              string               `json:"id"`
	PatientID       string               `json:"patientId"`
	Status          string               `json:"status"`
	DocumentType    string               `json:"documentType"`
	DocumentStatus  *string              `json:"documentStatus,omitempty"`
	LivenessStatus  *string              `json:"livenessStatus,omitempty"`
	FaceMatchStatus *string              `json:"faceMatchStatus,omitempty"`
	CreatedAt       string               `json:"createdAt"`
	Patient         AdminPatientIdentity `json:"patient"`
}

type AdminPatientIdentity struct {
	PatientCode string  `json:"patientCode"`
	FullName    *string `json:"fullName"`
	MaskedNIK   string  `json:"nik"`
	DateOfBirth *string `json:"dateOfBirth,omitempty"`
	Gender      *string `json:"gender,omitempty"`
}
