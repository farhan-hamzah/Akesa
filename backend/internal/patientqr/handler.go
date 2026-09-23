package patientqr

import (
	"encoding/json"
	"net/http"

	"github.com/farhan-hamzah/Akesa/backend/internal/auth"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{
		service: service,
	}
}

func (h *Handler) Get(
	w http.ResponseWriter,
	r *http.Request,
) {
	authUser, ok := auth.GetAuthUser(r)

	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{
			"error": "unauthorized",
		})
		return
	}

	credential, err := h.service.GetOrCreate(
		r.Context(),
		authUser.ID,
	)

	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "failed to get patient QR",
		})
		return
	}

	response := ToQRResponse(credential)

	writeJSON(w, http.StatusOK, response)
}

func (h *Handler) Rotate(
	w http.ResponseWriter,
	r *http.Request,
) {
	authUser, ok := auth.GetAuthUser(r)

	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{
			"error": "unauthorized",
		})
		return
	}

	credential, err := h.service.Rotate(
		r.Context(),
		authUser.ID,
	)

	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "failed to rotate patient QR",
		})
		return
	}

	response := ToQRResponse(credential)

	writeJSON(w, http.StatusOK, response)
}

func writeJSON(
	w http.ResponseWriter,
	status int,
	data any,
) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	_ = json.NewEncoder(w).Encode(data)
}
