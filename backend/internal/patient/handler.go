package patient

import (
	"net/http"

	"github.com/farhan-hamzah/Akesa/backend/internal/auth"
	"github.com/farhan-hamzah/Akesa/backend/internal/response"
	"github.com/farhan-hamzah/Akesa/backend/internal/user"
)

type Handler struct {
	service     *Service
	userService *user.Service
}

func NewHandler(service *Service, userService *user.Service) *Handler {
	return &Handler{
		service:     service,
		userService: userService,
	}
}

func (h *Handler) Me(w http.ResponseWriter, r *http.Request) {
	clerkUserID, ok := auth.GetClerkUserID(r)

	if !ok {
		response.Error(
			w,
			http.StatusUnauthorized,
			"unauthorized",
		)
		return
	}

	currentUser, err := h.userService.GetByClerkUserID(
		r.Context(),
		clerkUserID,
	)

	if err != nil {
		response.Error(
			w,
			http.StatusNotFound,
			"user_not_found",
		)
		return
	}

	patient, err := h.service.GetByUserID(
		r.Context(),
		currentUser.ID,
	)

	if err != nil {
		response.Error(
			w,
			http.StatusNotFound,
			"patient_not_found",
		)
		return
	}

	response.JSON(
		w,
		http.StatusOK,
		patient,
	)
}
