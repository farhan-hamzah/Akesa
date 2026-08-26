package audit

import (
	"time"

	"github.com/google/uuid"
)

type AuditLog struct {
	ID           uuid.UUID      `json:"id"`
	UserID       *uuid.UUID     `json:"userId,omitempty"`
	PatientID    *uuid.UUID     `json:"patientId,omitempty"`
	Action       string         `json:"action"`
	ResourceType *string        `json:"resourceType,omitempty"`
	ResourceID   *uuid.UUID     `json:"resourceId,omitempty"`
	Metadata     map[string]any `json:"metadata,omitempty"`
	IPAddress    *string        `json:"ipAddress,omitempty"`
	UserAgent    *string        `json:"userAgent,omitempty"`
	CreatedAt    time.Time      `json:"createdAt"`
}
