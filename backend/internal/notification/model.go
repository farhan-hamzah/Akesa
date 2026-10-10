package notification

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type Category string

const (
	CategorySecurityAlert     Category = "SECURITY_ALERT"
	CategoryAccessRequest     Category = "ACCESS_REQUEST"
	CategoryHospitalLifecycle Category = "HOSPITAL_LIFECYCLE"
	CategorySystem            Category = "SYSTEM"
)

type Severity string

const (
	SeverityInfo     Severity = "INFO"
	SeverityWarning  Severity = "WARNING"
	SeverityCritical Severity = "CRITICAL"
)

type Notification struct {
	ID        uuid.UUID       `json:"id"`
	UserID    uuid.UUID       `json:"userId"`
	Title     string          `json:"title"`
	Message   string          `json:"message"`
	Category  Category        `json:"category"`
	Severity  Severity        `json:"severity"`
	IsRead    bool            `json:"isRead"`
	ActionURL *string         `json:"actionUrl,omitempty"`
	Metadata  json.RawMessage `json:"metadata"`
	CreatedAt time.Time       `json:"createdAt"`
	ReadAt    *time.Time      `json:"readAt,omitempty"`
}

type DeviceToken struct {
	ID        uuid.UUID `json:"id"`
	UserID    uuid.UUID `json:"userId"`
	Token     string    `json:"token"`
	Platform  string    `json:"platform"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}
