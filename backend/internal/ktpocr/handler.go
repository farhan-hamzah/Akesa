package ktpocr

import (
	"bytes"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	maxUploadSize     = 15 << 20 // 15 MiB
	maxRequestSize    = maxUploadSize + (1 << 20)
	maxResponseSize   = 1 << 20 // 1 MiB
	defaultServiceURL = "http://localhost:9090"
)

type Handler struct {
	serviceURL string
	client     *http.Client
}

func NewHandler() *Handler {
	serviceURL := strings.TrimRight(
		strings.TrimSpace(os.Getenv("KTP_OCR_SERVICE_URL")),
		"/",
	)

	// Default ini untuk pengembangan lokal.
	// Pada deployment, atur KTP_OCR_SERVICE_URL secara eksplisit.
	if serviceURL == "" {
		serviceURL = defaultServiceURL
	}

	return &Handler{
		serviceURL: serviceURL,
		client: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
}

func (h *Handler) Upload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxRequestSize)

	if err := r.ParseMultipartForm(maxRequestSize); err != nil {
		http.Error(w, "invalid multipart upload or file too large",
			http.StatusBadRequest)
		return
	}

	if r.MultipartForm != nil {
		defer r.MultipartForm.RemoveAll()
	}

	file, header, err := r.FormFile("image")
	if err != nil {
		http.Error(w, "image file is required", http.StatusBadRequest)
		return
	}
	defer file.Close()

	if header.Size <= 0 || header.Size > maxUploadSize {
		http.Error(w, "image must be between 1 byte and 15 MiB",
			http.StatusBadRequest)
		return
	}

	ext := strings.ToLower(filepath.Ext(header.Filename))
	switch ext {
	case ".jpg", ".jpeg", ".png":
		// Extension diizinkan.
	default:
		http.Error(w, "only JPG, JPEG, and PNG images are allowed",
			http.StatusBadRequest)
		return
	}

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)

	part, err := writer.CreateFormFile("image", filepath.Base(header.Filename))
	if err != nil {
		http.Error(w, "failed to prepare image", http.StatusInternalServerError)
		return
	}

	if _, err := io.Copy(part, file); err != nil {
		http.Error(w, "failed to read image", http.StatusBadRequest)
		return
	}

	if err := writer.Close(); err != nil {
		http.Error(w, "failed to prepare image", http.StatusInternalServerError)
		return
	}

	targetURL := h.serviceURL + "/api/v1/ktp/ocr"
	req, err := http.NewRequestWithContext(
		r.Context(),
		http.MethodPost,
		targetURL,
		&body,
	)
	if err != nil {
		http.Error(w, "failed to create OCR request",
			http.StatusInternalServerError)
		return
	}

	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("Accept", "application/json")

	resp, err := h.client.Do(req)
	if err != nil {
		http.Error(w, "OCR service is unavailable",
			http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	responseBody, err := io.ReadAll(
		io.LimitReader(resp.Body, maxResponseSize+1),
	)
	if err != nil || len(responseBody) > maxResponseSize {
		http.Error(w, "invalid or oversized OCR response",
			http.StatusBadGateway)
		return
	}

	contentType := resp.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "application/json"
	}

	w.Header().Set("Content-Type", contentType)
	w.WriteHeader(resp.StatusCode)

	_, _ = w.Write(responseBody)
}
