package user

import (
	"errors"
	"net/http"

	"github.com/farhan-hamzah/Akesa/backend/internal/auth"
	"github.com/farhan-hamzah/Akesa/backend/internal/response"
)

type Handler struct {	
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{
		service: service,
	}
}

func (h *Handler) Me(
	w http.ResponseWriter,
	r *http.Request,
) {
	clerkUserID, ok := auth.GetClerkUserID(r)

	if !ok {
		response.Error(
			w,
			http.StatusUnauthorized,
			"unauthorized",
		)
		return
	}

	user, err := h.service.GetByClerkUserID(
		r.Context(),
		clerkUserID,
	)

	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			response.Error(
				w,
				http.StatusNotFound,
				"user_not_found",
			)
			return
		}

		response.Error(
			w,
			http.StatusInternalServerError,
			"internal_server_error",
		)
		return
	}

	response.JSON(
		w,
		http.StatusOK,
		user,
	)
}

func (h *Handler) Sync(
	w http.ResponseWriter,
	r *http.Request,
) {
	clerkUserID, ok := auth.GetClerkUserID(r)

	if !ok {
		response.Error(
			w,
			http.StatusUnauthorized,
			"unauthorized",
		)
		return
	}

	existingUser, err := h.service.GetByClerkUserID(
		r.Context(),
		clerkUserID,
	)

	if err == nil {
		response.JSON(
			w,
			http.StatusOK,
			existingUser,
		)
		return
	}

	if !errors.Is(err, ErrUserNotFound) {
		response.Error(
			w,
			http.StatusInternalServerError,
			"internal_server_error",
		)
		return
	}

	newUser, err := h.service.CreateFromClerk(
		r.Context(),
		clerkUserID,
	)

	if err != nil {
		response.Error(
			w,
			http.StatusInternalServerError,
			"failed_to_create_user",
		)
		return
	}

	response.JSON(
		w,
		http.StatusCreated,
		newUser,
	)
}
