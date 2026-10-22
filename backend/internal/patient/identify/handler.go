package identity

import (
	"encoding/json"
	"errors"
	"fmt"
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

	case errors.Is(err, ErrDocumentNotUploaded):
		response.Error(
			w,
			http.StatusBadRequest,
			"document_not_uploaded",
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
	documentUploaded := verification.DocumentStorageKey != nil &&
		strings.TrimSpace(*verification.DocumentStorageKey) != ""

	selfieUploaded := verification.SelfieStorageKey != nil &&
		strings.TrimSpace(*verification.SelfieStorageKey) != ""

	result := VerificationResponse{
		ID:               stringPtr(verification.ID.String()),
		Status:           stringPtr(string(verification.Status)),
		DocumentType:     stringPtr(string(verification.DocumentType)),
		DocumentUploaded: documentUploaded,
		SelfieUploaded:   selfieUploaded,
		DocumentStatus:   verification.DocumentStatus,
		LivenessStatus:   verification.LivenessStatus,
		FaceMatchStatus:  verification.FaceMatchStatus,
		FailureReason:    verification.FailureReason,
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

func stringPtr(value string) *string {
	return &value
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

func (h *Handler) UploadSelfie(
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

	verificationID, err := uuid.Parse(
		r.PathValue("id"),
	)
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

	file, header, err := r.FormFile("selfie")
	if err != nil {
		response.Error(
			w,
			http.StatusBadRequest,
			"selfie_file_required",
		)
		return
	}
	defer file.Close()

	if header.Size > maxDocumentSize {
		response.Error(
			w,
			http.StatusRequestEntityTooLarge,
			"selfie_too_large",
		)
		return
	}

	buffer := make([]byte, 512)
	n, err := file.Read(buffer)
	if err != nil && err != io.EOF {
		response.Error(
			w,
			http.StatusBadRequest,
			"failed_to_read_selfie",
		)
		return
	}

	if n == 0 {
		response.Error(
			w,
			http.StatusBadRequest,
			"empty_selfie",
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
			"unsupported_selfie_type",
		)
		return
	}

	seeker, ok := file.(io.Seeker)
	if !ok {
		response.Error(
			w,
			http.StatusInternalServerError,
			"selfie_not_seekable",
		)
		return
	}

	if _, err := seeker.Seek(0, io.SeekStart); err != nil {
		response.Error(
			w,
			http.StatusInternalServerError,
			"failed_to_reset_selfie",
		)
		return
	}

	verification, err := h.service.UploadSelfie(
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

	// 8. Kembalikan status verifikasi terbaru.
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
	authUser, ok := auth.GetAuthUser(r)
	if !ok {
		response.Error(
			w,
			http.StatusUnauthorized,
			"unauthorized",
		)
		return
	}

	verificationID, err := uuid.Parse(r.PathValue("id"))
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

func (h *Handler) RejectVerification(
	w http.ResponseWriter,
	r *http.Request,
) {
	// Ambil identitas admin dari middleware autentikasi.
	authUser, ok := auth.GetAuthUser(r)
	if !ok {
		response.Error(
			w,
			http.StatusUnauthorized,
			"unauthorized",
		)
		return
	}

	// Parse ID verifikasi dari URL.
	verificationID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		response.Error(
			w,
			http.StatusBadRequest,
			"invalid_verification_id",
		)
		return
	}

	// Baca alasan penolakan dari request body.
	var req RejectVerificationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(
			w,
			http.StatusBadRequest,
			"invalid_request_body",
		)
		return
	}

	reason := strings.TrimSpace(req.Reason)
	if reason == "" {
		response.Error(
			w,
			http.StatusBadRequest,
			"rejection_reason_required",
		)
		return
	}

	verification, err := h.service.RejectVerification(
		r.Context(),
		authUser.ID,
		verificationID,
		reason,
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

func (h *Handler) ListPendingReview(w http.ResponseWriter, r *http.Request) {
	items, err := h.service.ListPendingReview(r.Context())
	if err != nil {
		h.writeServiceError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, items)
}

func (h *Handler) GetReviewDocument(
	w http.ResponseWriter,
	r *http.Request,
) {
	verificationID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		response.Error(
			w,
			http.StatusBadRequest,
			"invalid_verification_id",
		)
		return
	}

	documentType := strings.ToLower(r.PathValue("type"))

	if documentType != "ktp" && documentType != "selfie" {
		response.Error(
			w,
			http.StatusBadRequest,
			"invalid_document_type",
		)
		return
	}

	data, contentType, err := h.service.GetReviewDocument(
		r.Context(),
		verificationID,
		documentType,
	)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}

	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Content-Length", fmt.Sprintf("%d", len(data)))
	w.Header().Set("Cache-Control", "no-store, private")
	w.Header().Set("Pragma", "no-cache")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Content-Disposition", "inline")

	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(data)
}
