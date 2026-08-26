package user

import (
	"errors"
	"net/http"

	"github.com/farhan-hamzah/Akesa/backend/internal/auth"
	"github.com/farhan-hamzah/Akesa/backend/internal/response"
	"github.com/jackc/pgx/v5"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{
		service: service,
	}
}

// Me - GET /api/v1/me (auth required, no role check: any synced user)
func (h *Handler) Me(w http.ResponseWriter, r *http.Request) {
	clerkUserID, ok := auth.GetClerkUserID(r)
	if !ok {
		response.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	currentUser, err := h.service.GetByClerkUserID(r.Context(), clerkUserID)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			response.Error(w, http.StatusNotFound, "user_not_found")
			return
		}

		response.Error(w, http.StatusInternalServerError, "internal_server_error")
		return
	}

	response.JSON(w, http.StatusOK, currentUser)
}

// Sync - POST /api/v1/users/sync (auth required) - called once right
// after a successful Clerk sign-in/sign-up to create our own user row.
// Idempotent: if the row already exists, it is simply returned.
func (h *Handler) Sync(w http.ResponseWriter, r *http.Request) {
	clerkUserID, ok := auth.GetClerkUserID(r)
	if !ok {
		response.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	existingUser, err := h.service.GetByClerkUserID(r.Context(), clerkUserID)
	if err == nil {
		response.JSON(w, http.StatusOK, existingUser)
		return
	}

	if !errors.Is(err, ErrUserNotFound) && !errors.Is(err, pgx.ErrNoRows) {
		response.Error(w, http.StatusInternalServerError, "internal_server_error")
		return
	}

	newUser, err := h.service.CreateFromClerk(r.Context(), clerkUserID)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "failed_to_create_user")
		return
	}

	response.JSON(w, http.StatusCreated, newUser)
}
