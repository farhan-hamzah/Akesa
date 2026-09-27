package identity

type VerificationResponse struct {
	ID            string  `json:"id"`
	Status        string  `json:"status"`
	DocumentType  string  `json:"documentType"`
	FailureReason *string `json:"failureReason,omitempty"`
	VerifiedAt    *string `json:"verifiedAt,omitempty"`
	ExpiresAt     *string `json:"expiresAt,omitempty"`
}
