package notification

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/farhan-hamzah/Akesa/backend/internal/auth"
	"github.com/google/uuid"
)

type mockStore struct {
	notifications map[uuid.UUID]*Notification
	deviceTokens  map[string]*DeviceToken // key: token
	usersByRole   map[string][]uuid.UUID
}

func newMockStore() *mockStore {
	return &mockStore{
		notifications: make(map[uuid.UUID]*Notification),
		deviceTokens:  make(map[string]*DeviceToken),
		usersByRole:   make(map[string][]uuid.UUID),
	}
}

func (m *mockStore) Create(ctx context.Context, n *Notification) error {
	m.notifications[n.ID] = n
	return nil
}

func (m *mockStore) ListByUserID(ctx context.Context, userID uuid.UUID, unreadOnly bool, limit, offset int) ([]*Notification, int, error) {
	var list []*Notification
	for _, n := range m.notifications {
		if n.UserID == userID {
			if unreadOnly && n.IsRead {
				continue
			}
			list = append(list, n)
		}
	}
	total := len(list)
	return list, total, nil
}

func (m *mockStore) CountUnread(ctx context.Context, userID uuid.UUID) (int, error) {
	count := 0
	for _, n := range m.notifications {
		if n.UserID == userID && !n.IsRead {
			count++
		}
	}
	return count, nil
}

func (m *mockStore) MarkAsRead(ctx context.Context, id, userID uuid.UUID) (*Notification, error) {
	n, ok := m.notifications[id]
	if !ok || n.UserID != userID {
		return nil, ErrNotificationNotFound
	}
	now := time.Now().UTC()
	n.IsRead = true
	n.ReadAt = &now
	return n, nil
}

func (m *mockStore) MarkAllAsRead(ctx context.Context, userID uuid.UUID) error {
	now := time.Now().UTC()
	for _, n := range m.notifications {
		if n.UserID == userID && !n.IsRead {
			n.IsRead = true
			n.ReadAt = &now
		}
	}
	return nil
}

func (m *mockStore) UpsertDeviceToken(ctx context.Context, dt *DeviceToken) error {
	m.deviceTokens[dt.Token] = dt
	return nil
}

func (m *mockStore) DeleteDeviceToken(ctx context.Context, userID uuid.UUID, token string) error {
	if dt, ok := m.deviceTokens[token]; ok && dt.UserID == userID {
		delete(m.deviceTokens, token)
	}
	return nil
}

func (m *mockStore) GetDeviceTokensByUserID(ctx context.Context, userID uuid.UUID) ([]string, error) {
	var tokens []string
	for _, dt := range m.deviceTokens {
		if dt.UserID == userID {
			tokens = append(tokens, dt.Token)
		}
	}
	return tokens, nil
}

func (m *mockStore) GetUsersByRole(ctx context.Context, role string) ([]uuid.UUID, error) {
	return m.usersByRole[role], nil
}

type mockPushClient struct {
	sentCount int
	lastTitle string
	lastBody  string
	lastData  map[string]string
}

func (p *mockPushClient) SendPush(ctx context.Context, tokens []string, title, message string, data map[string]string) error {
	p.sentCount += len(tokens)
	p.lastTitle = title
	p.lastBody = message
	p.lastData = data
	return nil
}

func TestService_NotifyUserAndPush(t *testing.T) {
	store := newMockStore()
	push := &mockPushClient{}
	svc := NewService(store, push)

	userID := uuid.New()
	_ = svc.RegisterDeviceToken(context.Background(), userID, "test-fcm-token-123", "android")

	err := svc.NotifyUser(context.Background(), userID, NotificationInput{
		Title:    "Peringatan Integritas Data",
		Message:  "Hash mismatch detected",
		Category: CategorySecurityAlert,
		Severity: SeverityCritical,
		Metadata: map[string]any{"requestId": "1234"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if push.sentCount != 1 {
		t.Fatalf("expected 1 push notification sent, got %d", push.sentCount)
	}
	if push.lastTitle != "Peringatan Integritas Data" {
		t.Fatalf("expected title to match, got %q", push.lastTitle)
	}
	if push.lastData["category"] != string(CategorySecurityAlert) {
		t.Fatalf("expected category data, got %q", push.lastData["category"])
	}

	unread, err := svc.GetUnreadCount(context.Background(), userID)
	if err != nil || unread != 1 {
		t.Fatalf("expected unread count 1, got %d (err: %v)", unread, err)
	}
}

func TestService_NotifyRole(t *testing.T) {
	store := newMockStore()
	push := &mockPushClient{}
	svc := NewService(store, push)

	admin1 := uuid.New()
	admin2 := uuid.New()
	store.usersByRole["ADMIN"] = []uuid.UUID{admin1, admin2}

	err := svc.NotifyRole(context.Background(), "ADMIN", NotificationInput{
		Title:    "Admin Alert",
		Message:  "Security alert broadcast",
		Category: CategorySecurityAlert,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	unread1, _ := svc.GetUnreadCount(context.Background(), admin1)
	unread2, _ := svc.GetUnreadCount(context.Background(), admin2)
	if unread1 != 1 || unread2 != 1 {
		t.Fatalf("expected both admins to receive notification, got %d and %d", unread1, unread2)
	}
}

func TestHandler_RegisterAndUnregisterDeviceToken(t *testing.T) {
	store := newMockStore()
	svc := NewService(store, nil)
	handler := NewHandler(svc)

	userID := uuid.New()
	authUser := &auth.AuthUser{ID: userID, Role: "PATIENT", Status: "ACTIVE"}

	// 1. Register Token
	body, _ := json.Marshal(RegisterDeviceTokenRequest{
		Token:    "fcm-sample-token",
		Platform: "android",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/notifications/device-token", bytes.NewReader(body))
	req = req.WithContext(auth.WithAuthUser(req.Context(), authUser))
	rr := httptest.NewRecorder()

	handler.RegisterDeviceToken(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d (body: %s)", rr.Code, rr.Body.String())
	}

	tokens, _ := store.GetDeviceTokensByUserID(context.Background(), userID)
	if len(tokens) != 1 || tokens[0] != "fcm-sample-token" {
		t.Fatalf("expected token registered, got %v", tokens)
	}

	// 2. Unregister Token
	unregBody, _ := json.Marshal(UnregisterDeviceTokenRequest{Token: "fcm-sample-token"})
	req2 := httptest.NewRequest(http.MethodDelete, "/api/v1/notifications/device-token", bytes.NewReader(unregBody))
	req2 = req2.WithContext(auth.WithAuthUser(req2.Context(), authUser))
	rr2 := httptest.NewRecorder()

	handler.UnregisterDeviceToken(rr2, req2)
	if rr2.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rr2.Code)
	}

	tokensAfter, _ := store.GetDeviceTokensByUserID(context.Background(), userID)
	if len(tokensAfter) != 0 {
		t.Fatalf("expected tokens to be empty after unregister, got %v", tokensAfter)
	}
}

func TestHandler_ListAndMarkAsRead(t *testing.T) {
	store := newMockStore()
	svc := NewService(store, nil)
	handler := NewHandler(svc)

	userID := uuid.New()
	authUser := &auth.AuthUser{ID: userID, Role: "PATIENT", Status: "ACTIVE"}

	// Create test notification
	notifID := uuid.New()
	store.notifications[notifID] = &Notification{
		ID:        notifID,
		UserID:    userID,
		Title:     "Test Notif",
		Message:   "Test Body",
		Category:  CategorySystem,
		Severity:  SeverityInfo,
		IsRead:    false,
		CreatedAt: time.Now().UTC(),
	}

	// 1. Check Unread Count
	reqCount := httptest.NewRequest(http.MethodGet, "/api/v1/notifications/unread-count", nil)
	reqCount = reqCount.WithContext(auth.WithAuthUser(reqCount.Context(), authUser))
	rrCount := httptest.NewRecorder()
	handler.GetUnreadCount(rrCount, reqCount)
	if rrCount.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rrCount.Code)
	}

	var countResp UnreadCountResponse
	_ = json.Unmarshal(rrCount.Body.Bytes(), &countResp)
	if countResp.UnreadCount != 1 {
		t.Fatalf("expected unread count 1, got %d", countResp.UnreadCount)
	}

	// 2. Mark as read
	reqRead := httptest.NewRequest(http.MethodPatch, "/api/v1/notifications/"+notifID.String()+"/read", nil)
	reqRead.SetPathValue("id", notifID.String())
	reqRead = reqRead.WithContext(auth.WithAuthUser(reqRead.Context(), authUser))
	rrRead := httptest.NewRecorder()
	handler.MarkAsRead(rrRead, reqRead)
	if rrRead.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (body: %s)", rrRead.Code, rrRead.Body.String())
	}

	// 3. Mark all as read
	reqReadAll := httptest.NewRequest(http.MethodPost, "/api/v1/notifications/read-all", nil)
	reqReadAll = reqReadAll.WithContext(auth.WithAuthUser(reqReadAll.Context(), authUser))
	rrReadAll := httptest.NewRecorder()
	handler.MarkAllAsRead(rrReadAll, reqReadAll)
	if rrReadAll.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rrReadAll.Code)
	}
}
