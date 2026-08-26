package patient

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/farhan-hamzah/Akesa/backend/internal/auth"
	"github.com/farhan-hamzah/Akesa/backend/internal/response"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// GetMyProfile - GET /api/v1/patient/profile (role: PATIENT)
func (h *Handler) GetMyProfile(w http.ResponseWriter, r *http.Request) {
	authUser, ok := auth.GetAuthUser(r)
	if !ok {
		response.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	profile, err := h.service.GetByUserID(r.Context(), authUser.ID)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, profile)
}

// CreateProfile - POST /api/v1/patient/profile (role: PATIENT)
func (h *Handler) CreateProfile(w http.ResponseWriter, r *http.Request) {
	authUser, ok := auth.GetAuthUser(r)
	if !ok {
		response.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req ProfileRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid_json_body")
		return
	}

	input, err := req.Validate()
	if err != nil {
		response.Error(w, http.StatusUnprocessableEntity, err.Error())
		return
	}

	profile, err := h.service.Create(r.Context(), authUser.ID, input)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}

	response.JSON(w, http.StatusCreated, profile)
}

// UpdateProfile - PUT /api/v1/patient/profile (role: PATIENT)
func (h *Handler) UpdateProfile(w http.ResponseWriter, r *http.Request) {
	authUser, ok := auth.GetAuthUser(r)
	if !ok {
		response.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req ProfileRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid_json_body")
		return
	}

	input, err := req.Validate()
	if err != nil {
		response.Error(w, http.StatusUnprocessableEntity, err.Error())
		return
	}

	profile, err := h.service.Update(r.Context(), authUser.ID, input)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, profile)
}

func (h *Handler) writeServiceError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrProfileNotFound):
		response.Error(w, http.StatusNotFound, "profile_not_found")
	case errors.Is(err, ErrProfileExists):
		response.Error(w, http.StatusConflict, "profile_already_exists")
	case errors.Is(err, ErrNIKAlreadyRegistered):
		response.Error(w, http.StatusConflict, "nik_already_registered")
	default:
		response.Error(w, http.StatusInternalServerError, "internal_server_error")
	}
}
