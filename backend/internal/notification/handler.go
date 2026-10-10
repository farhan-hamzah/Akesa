package notification

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/farhan-hamzah/Akesa/backend/internal/auth"
	"github.com/farhan-hamzah/Akesa/backend/internal/response"
	"github.com/google/uuid"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) RegisterDeviceToken(w http.ResponseWriter, r *http.Request) {
	u, ok := auth.GetAuthUser(r)
	if !ok {
		response.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req RegisterDeviceTokenRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid_json_body")
		return
	}

	if req.Token == "" {
		response.Error(w, http.StatusBadRequest, "token_required")
		return
	}

	if err := h.service.RegisterDeviceToken(r.Context(), u.ID, req.Token, req.Platform); err != nil {
		response.Error(w, http.StatusInternalServerError, "failed_to_register_device_token")
		return
	}

	response.JSON(w, http.StatusOK, map[string]string{
		"status":  "success",
		"message": "device_token_registered",
	})
}

func (h *Handler) UnregisterDeviceToken(w http.ResponseWriter, r *http.Request) {
	u, ok := auth.GetAuthUser(r)
	if !ok {
		response.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req UnregisterDeviceTokenRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid_json_body")
		return
	}

	if req.Token == "" {
		response.Error(w, http.StatusBadRequest, "token_required")
		return
	}

	if err := h.service.UnregisterDeviceToken(r.Context(), u.ID, req.Token); err != nil {
		response.Error(w, http.StatusInternalServerError, "failed_to_unregister_device_token")
		return
	}

	response.JSON(w, http.StatusOK, map[string]string{
		"status":  "success",
		"message": "device_token_unregistered",
	})
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	u, ok := auth.GetAuthUser(r)
	if !ok {
		response.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	query := r.URL.Query()
	unreadOnly := query.Get("unread_only") == "true"

	limit := 20
	if l := query.Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 {
			limit = parsed
		}
	}

	offset := 0
	if o := query.Get("offset"); o != "" {
		if parsed, err := strconv.Atoi(o); err == nil && parsed >= 0 {
			offset = parsed
		}
	}

	resp, err := h.service.ListNotifications(r.Context(), u.ID, unreadOnly, limit, offset)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "failed_to_fetch_notifications")
		return
	}

	response.JSON(w, http.StatusOK, resp)
}

func (h *Handler) GetUnreadCount(w http.ResponseWriter, r *http.Request) {
	u, ok := auth.GetAuthUser(r)
	if !ok {
		response.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	count, err := h.service.GetUnreadCount(r.Context(), u.ID)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "failed_to_fetch_unread_count")
		return
	}

	response.JSON(w, http.StatusOK, UnreadCountResponse{UnreadCount: count})
}

func (h *Handler) MarkAsRead(w http.ResponseWriter, r *http.Request) {
	u, ok := auth.GetAuthUser(r)
	if !ok {
		response.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	idStr := r.PathValue("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "invalid_notification_id")
		return
	}

	n, err := h.service.MarkAsRead(r.Context(), id, u.ID)
	if err != nil {
		if errors.Is(err, ErrNotificationNotFound) {
			response.Error(w, http.StatusNotFound, "notification_not_found")
			return
		}
		response.Error(w, http.StatusInternalServerError, "failed_to_mark_notification_as_read")
		return
	}

	response.JSON(w, http.StatusOK, n)
}

func (h *Handler) MarkAllAsRead(w http.ResponseWriter, r *http.Request) {
	u, ok := auth.GetAuthUser(r)
	if !ok {
		response.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	if err := h.service.MarkAllAsRead(r.Context(), u.ID); err != nil {
		response.Error(w, http.StatusInternalServerError, "failed_to_mark_all_notifications_as_read")
		return
	}

	response.JSON(w, http.StatusOK, map[string]string{
		"status":  "success",
		"message": "all_marked_as_read",
	})
}
