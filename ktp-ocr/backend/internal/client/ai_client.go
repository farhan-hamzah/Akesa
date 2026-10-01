package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"time"
)

type KTPData struct {
	NIK              *string `json:"nik"`
	Nama             *string `json:"nama"`
	TempatLahir      *string `json:"tempat_lahir"`
	TanggalLahir     *string `json:"tanggal_lahir"`
	JenisKelamin     *string `json:"jenis_kelamin"`
	Alamat           *string `json:"alamat"`
	Agama            *string `json:"agama"`
	StatusPerkawinan *string `json:"status_perkawinan"`
	Pekerjaan        *string `json:"pekerjaan"`
	Kewarganegaraan  *string `json:"kewarganegaraan"`
}

type AIResponse struct {
	Success bool     `json:"success"`
	Data    *KTPData `json:"data,omitempty"`
	Error   string   `json:"error,omitempty"`
}

type AIClient struct {
	baseURL    string
	httpClient *http.Client
}

func NewAIClient(baseURL string, timeout time.Duration) *AIClient {
	return &AIClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}
}

func (c *AIClient) ExtractKTP(ctx context.Context, filename string, fileReader io.Reader) (*KTPData, error) {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	part, err := writer.CreateFormFile("image", filename)
	if err != nil {
		return nil, fmt.Errorf("gagal membuat form file multipart: %w", err)
	}

	if _, err := io.Copy(part, fileReader); err != nil {
		return nil, fmt.Errorf("gagal menyalin isi file ke multipart: %w", err)
	}

	if err := writer.Close(); err != nil {
		return nil, fmt.Errorf("gagal menutup multipart writer: %w", err)
	}

	targetURL := fmt.Sprintf("%s/ocr", c.baseURL)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, targetURL, body)
	if err != nil {
		return nil, fmt.Errorf("gagal membuat http request ke AI service: %w", err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("koneksi ke AI Service (%s) gagal: %w", targetURL, err)
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("gagal membaca response dari AI service: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		var errResp AIResponse
		if jsonErr := json.Unmarshal(respBytes, &errResp); jsonErr == nil && errResp.Error != "" {
			return nil, fmt.Errorf("AI service error (%d): %s", resp.StatusCode, errResp.Error)
		}
		return nil, fmt.Errorf("AI service mengembalikan status HTTP %d: %s", resp.StatusCode, string(respBytes))
	}

	var aiResp AIResponse
	if err := json.Unmarshal(respBytes, &aiResp); err != nil {
		return nil, fmt.Errorf("gagal parse JSON response dari AI service: %w", err)
	}

	if !aiResp.Success {
		return nil, fmt.Errorf("AI service gagal melakukan ekstraksi: %s", aiResp.Error)
	}

	return aiResp.Data, nil
}
