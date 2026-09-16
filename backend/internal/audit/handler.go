package audit

import (
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

// MyProfileHistory - GET /api/v1/patient/history (role: PATIENT)
// "riwayat penggunaan data" - every profile update, consent decision, and
// data access tied to this patient, in order.
func (h *Handler) MyProfileHistory(w http.ResponseWriter, r *http.Request) {
	authUser, ok := auth.GetAuthUser(r)
	if !ok {
		response.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	logs, err := h.service.History(r.Context(), "PATIENT_PROFILE", authUser.ID)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "internal_server_error")
		return
	}

	response.JSON(w, http.StatusOK, logs)
}

// VerifyChain - GET /api/v1/audit/verify-chain
// Cryptographically checks all blocks in the audit hash chain.
func (h *Handler) VerifyChain(w http.ResponseWriter, r *http.Request) {
	valid, totalBlocks, err := h.service.VerifyChainIntegrity(r.Context())
	if err != nil {
		response.JSON(w, http.StatusOK, map[string]any{
			"valid":       false,
			"totalBlocks": totalBlocks,
			"status":      "TAMPERED_OR_BROKEN",
			"error":       err.Error(),
		})
		return
	}

	response.JSON(w, http.StatusOK, map[string]any{
		"valid":       valid,
		"totalBlocks": totalBlocks,
		"status":      "VERIFIED_INTACT",
		"message":     "All SHA-256 block hashes verified and intact",
	})
}
