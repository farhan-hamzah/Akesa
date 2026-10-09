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
