package identity

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/farhan-hamzah/Akesa/backend/internal/auth"
	"github.com/farhan-hamzah/Akesa/backend/internal/patient"
	"github.com/farhan-hamzah/Akesa/backend/internal/response"
	"github.com/google/uuid"
)

const maxDocumentSize = 5 * 1024 * 1024

type Handler struct {
	service *Service
}

type RejectVerificationRequest struct {
	Reason string `json:"reason"`
}

func NewHandler(service *Service) *Handler {
	return &Handler{
		service: service,
	}
}

func (h *Handler) CreateVerification(
	w http.ResponseWriter,
	r *http.Request,
) {
	authUser, ok := auth.GetAuthUser(r)
	if !ok {
		response.Error(
			w,
			http.StatusUnauthorized,
			"unauthorized",
		)
		return
	}

	verification, err := h.service.CreateVerification(
		r.Context(),
		authUser.ID,
	)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}

	response.JSON(
		w,
		http.StatusCreated,
		toVerificationResponse(verification),
	)
}

func (h *Handler) GetVerification(
	w http.ResponseWriter,
	r *http.Request,
) {
	authUser, ok := auth.GetAuthUser(r)
	if !ok {
		response.Error(
			w,
			http.StatusUnauthorized,
			"unauthorized",
		)
		return
	}

	id := r.PathValue("id")

	verificationID, err := uuid.Parse(id)
	if err != nil {
		response.Error(
			w,
			http.StatusBadRequest,
			"invalid_verification_id",
		)
		return
	}

	verification, err := h.service.GetVerification(
		r.Context(),
		authUser.ID,
		verificationID,
	)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}

	response.JSON(
		w,
		http.StatusOK,
		toVerificationResponse(verification),
	)
}

func (h *Handler) writeServiceError(
	w http.ResponseWriter,
	err error,
) {
	switch {
	case errors.Is(err, patient.ErrProfileNotFound):
		response.Error(
			w,
			http.StatusNotFound,
			"profile_not_found",
		)

	case errors.Is(err, ErrVerificationNotFound):
		response.Error(
			w,
			http.StatusNotFound,
			"verification_not_found",
		)

	case errors.Is(err, ErrVerificationExpired):
		response.Error(
			w,
			http.StatusGone,
			"verification_expired",
		)

	case errors.Is(err, ErrTooManyAttempts):
		response.Error(
			w,
			http.StatusTooManyRequests,
			"too_many_verification_attempts",
		)

	case errors.Is(err, ErrVerificationCompleted):
		response.Error(
			w,
			http.StatusConflict,
			"verification_already_completed",
		)

	case errors.Is(err, ErrVerificationNotInReview):
		response.Error(
			w,
			http.StatusConflict,
			"verification_not_in_review",
		)

	case errors.Is(err, ErrInvalidDocumentType):
		response.Error(
			w,
			http.StatusUnsupportedMediaType,
			"unsupported_document_type",
		)

	case errors.Is(err, ErrDocumentTooLarge):
		response.Error(
			w,
			http.StatusRequestEntityTooLarge,
			"document_too_large",
		)

	default:
		response.Error(
			w,
			http.StatusInternalServerError,
			"internal_server_error",
		)
	}
}

func toVerificationResponse(
	verification *Verification,
) VerificationResponse {
	result := VerificationResponse{
		ID:           verification.ID.String(),
		Status:       string(verification.Status),
		DocumentType: string(verification.DocumentType),
	}

	if verification.FailureReason != nil {
		result.FailureReason = verification.FailureReason
	}

	if verification.VerifiedAt != nil {
		value := verification.VerifiedAt.Format(
			"2006-01-02T15:04:05Z07:00",
		)
		result.VerifiedAt = &value
	}

	if verification.ExpiresAt != nil {
		value := verification.ExpiresAt.Format(
			"2006-01-02T15:04:05Z07:00",
		)
		result.ExpiresAt = &value
	}

	return result
}

func (h *Handler) UploadDocument(
	w http.ResponseWriter,
	r *http.Request,
) {
	authUser, ok := auth.GetAuthUser(r)
	if !ok {
		response.Error(
			w,
			http.StatusUnauthorized,
			"unauthorized",
		)
		return
	}

	id := r.PathValue("id")

	verificationID, err := uuid.Parse(id)
	if err != nil {
		response.Error(
			w,
			http.StatusBadRequest,
			"invalid_verification_id",
		)
		return
	}

	r.Body = http.MaxBytesReader(
		w,
		r.Body,
		maxDocumentSize,
	)

	file, header, err := r.FormFile("document")
	if err != nil {
		response.Error(
			w,
			http.StatusBadRequest,
			"document_file_required",
		)
		return
	}
	defer file.Close()

	if header.Size > maxDocumentSize {
		response.Error(
			w,
			http.StatusRequestEntityTooLarge,
			"document_too_large",
		)
		return
	}

	buffer := make([]byte, 512)

	n, err := file.Read(buffer)
	if err != nil && err != io.EOF {
		response.Error(
			w,
			http.StatusBadRequest,
			"failed_to_read_document",
		)
		return
	}

	if n == 0 {
		response.Error(
			w,
			http.StatusBadRequest,
			"empty_document",
		)
		return
	}

	contentType := http.DetectContentType(buffer[:n])

	var extension string

	switch contentType {
	case "image/jpeg":
		extension = ".jpg"

	case "image/png":
		extension = ".png"

	default:
		response.Error(
			w,
			http.StatusUnsupportedMediaType,
			"unsupported_document_type",
		)
		return
	}

	seeker, ok := file.(io.Seeker)
	if !ok {
		response.Error(
			w,
			http.StatusInternalServerError,
			"document_not_seekable",
		)
		return
	}

	if _, err := seeker.Seek(0, io.SeekStart); err != nil {
		response.Error(
			w,
			http.StatusInternalServerError,
			"failed_to_reset_document",
		)
		return
	}

	verification, err := h.service.UploadDocument(
		r.Context(),
		authUser.ID,
		verificationID,
		file,
		extension,
	)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}

	response.JSON(
		w,
		http.StatusOK,
		toVerificationResponse(verification),
	)
}

func (h *Handler) ApproveVerification(
	w http.ResponseWriter,
	r *http.Request,
) {
	id := r.PathValue("id")

	verificationID, err := uuid.Parse(id)
	if err != nil {
		response.Error(
			w,
			http.StatusBadRequest,
			"invalid_verification_id",
		)
		return
	}

	verification, err := h.service.ApproveVerification(
		r.Context(),
		verificationID,
	)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}

	response.JSON(
		w,
		http.StatusOK,
		toVerificationResponse(verification),
	)
}

func (h *Handler) RejectVerification(
	w http.ResponseWriter,
	r *http.Request,
) {
	id := r.PathValue("id")

	verificationID, err := uuid.Parse(id)
	if err != nil {
		response.Error(
			w,
			http.StatusBadRequest,
			"invalid_verification_id",
		)
		return
	}

	var req RejectVerificationRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(
			w,
			http.StatusBadRequest,
			"invalid_request_body",
		)
		return
	}

	if strings.TrimSpace(req.Reason) == "" {
		response.Error(
			w,
			http.StatusBadRequest,
			"rejection_reason_required",
		)
		return
	}

	verification, err := h.service.RejectVerification(
		r.Context(),
		verificationID,
		strings.TrimSpace(req.Reason),
	)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}

	response.JSON(
		w,
		http.StatusOK,
		toVerificationResponse(verification),
	)
}

func (h *Handler) GetLatestVerification(
	w http.ResponseWriter,
	r *http.Request,
) {
	authUser, ok := auth.GetAuthUser(r)
	if !ok {
		response.Error(
			w,
			http.StatusUnauthorized,
			"unauthorized",
		)
		return
	}

	verification, err := h.service.GetLatestVerification(
		r.Context(),
		authUser.ID,
	)

	if err != nil {
		h.writeServiceError(w, err)
		return
	}

	response.JSON(
		w,
		http.StatusOK,
		toVerificationResponse(verification),
	)
}
