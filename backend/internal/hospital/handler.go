package hospital

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/farhan-hamzah/Akesa/backend/internal/response"
	"github.com/google/uuid"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// Create - POST /api/v1/admin/hospitals (role: ADMIN)
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var req HospitalRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid_json_body")
		return
	}

	input, err := req.Validate()
	if err != nil {
		response.Error(w, http.StatusUnprocessableEntity, err.Error())
		return
	}

	hospitalRecord, err := h.service.Create(r.Context(), input)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}

	response.JSON(w, http.StatusCreated, hospitalRecord)
}

// List - GET /api/v1/admin/hospitals (role: ADMIN)
// Also used to power the patient-facing "daftar rumah sakit" list, but
// that route should filter to ACTIVE only - see server routing.
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	hospitals, err := h.service.List(r.Context())
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "internal_server_error")
		return
	}

	response.JSON(w, http.StatusOK, hospitals)
}

// ListActive - GET /api/v1/hospitals (role: PATIENT, HOSPITAL_STAFF) -
// public-ish read of only active hospitals, so a patient can pick where
// to register.
func (h *Handler) ListActive(w http.ResponseWriter, r *http.Request) {
	hospitals, err := h.service.List(r.Context())
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "internal_server_error")
		return
	}

	active := make([]*Hospital, 0, len(hospitals))
	for _, hospitalRecord := range hospitals {
		if hospitalRecord.Status == StatusActive {
			active = append(active, hospitalRecord)
		}
	}

	response.JSON(w, http.StatusOK, active)
}

// Verify - PATCH /api/v1/admin/hospitals/{id}/verify (role: ADMIN)
func (h *Handler) Verify(w http.ResponseWriter, r *http.Request) {
	h.transition(w, r, h.service.Verify)
}

// Activate - PATCH /api/v1/admin/hospitals/{id}/activate (role: ADMIN)
func (h *Handler) Activate(w http.ResponseWriter, r *http.Request) {
	h.transition(w, r, h.service.Activate)
}

// Deactivate - PATCH /api/v1/admin/hospitals/{id}/deactivate (role: ADMIN)
func (h *Handler) Deactivate(w http.ResponseWriter, r *http.Request) {
	h.transition(w, r, h.service.Deactivate)
}

// transition is the shared body for the three status-change endpoints
// above, which all take a hospital id from the path and return the
// updated record.
func (h *Handler) transition(
	w http.ResponseWriter,
	r *http.Request,
	fn func(ctx context.Context, id uuid.UUID) (*Hospital, error),
) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		response.Error(w, http.StatusBadRequest, "invalid_hospital_id")
		return
	}

	hospitalRecord, err := fn(r.Context(), id)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, hospitalRecord)
}

// AddStaff - POST /api/v1/admin/hospitals/{id}/staff (role: ADMIN)
func (h *Handler) AddStaff(w http.ResponseWriter, r *http.Request) {
	hospitalID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		response.Error(w, http.StatusBadRequest, "invalid_hospital_id")
		return
	}

	var req StaffRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid_json_body")
		return
	}

	userID, err := uuid.Parse(req.UserID)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "invalid_user_id")
		return
	}

	staff, err := h.service.AddStaff(r.Context(), hospitalID, userID, req.FullName, req.Position)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}

	response.JSON(w, http.StatusCreated, staff)
}

func (h *Handler) writeServiceError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrNotFound):
		response.Error(w, http.StatusNotFound, "hospital_not_found")
	case errors.Is(err, ErrEmailTaken):
		response.Error(w, http.StatusConflict, "email_already_registered")
	case errors.Is(err, ErrRegistrationTaken):
		response.Error(w, http.StatusConflict, "registration_number_already_registered")
	case errors.Is(err, ErrStaffAlreadyLinked):
		response.Error(w, http.StatusConflict, "user_already_staff")
	case errors.Is(err, ErrHospitalNotActive):
		response.Error(w, http.StatusUnprocessableEntity, "hospital_not_active")
	default:
		response.Error(w, http.StatusInternalServerError, "internal_server_error")
	}
}
