package notification

type RegisterDeviceTokenRequest struct {
	Token    string `json:"token"`
	Platform string `json:"platform"`
}

type UnregisterDeviceTokenRequest struct {
	Token string `json:"token"`
}

type NotificationListResponse struct {
	Notifications []*Notification `json:"notifications"`
	Total         int             `json:"total"`
	UnreadCount   int             `json:"unreadCount"`
}

type UnreadCountResponse struct {
	UnreadCount int `json:"unreadCount"`
}

type NotificationInput struct {
	Title     string
	Message   string
	Category  Category
	Severity  Severity
	ActionURL string
	Metadata  map[string]any
}
