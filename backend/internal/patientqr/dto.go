package patientqr

type QRResponse struct {
	DisplayCode string `json:"displayCode"`
	QRPayload   string `json:"qrPayload"`
	IsActive    bool   `json:"isActive"`
}

func ToQRResponse(credential *Credential) QRResponse {
	return QRResponse{
		DisplayCode: credential.DisplayCode,
		QRPayload:   "akesa://patient/qr?token=" + credential.Token,
		IsActive:    credential.IsActive,
	}
}
