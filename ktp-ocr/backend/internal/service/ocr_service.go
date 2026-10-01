package service

import (
	"context"
	"errors"
	"fmt"
	"mime/multipart"
	"path/filepath"
	"strings"

	"ktp-ocr/backend/internal/client"
)

const (
	MaxUploadSize = 15 * 1024 * 1024 // 15 MB
)

var (
	ErrFileTooLarge    = errors.New("ukuran file melebihi batas maksimum 15MB")
	ErrInvalidFileType = errors.New("format file tidak didukung: hanya gambar JPG, JPEG, dan PNG yang diperbolehkan")
	ErrEmptyFile       = errors.New("file yang diupload kosong")
)

type OCRService struct {
	aiClient *client.AIClient
}

func NewOCRService(aiClient *client.AIClient) *OCRService {
	return &OCRService{
		aiClient: aiClient,
	}
}

func (s *OCRService) ProcessKTP(ctx context.Context, fileHeader *multipart.FileHeader) (*client.KTPData, error) {
	if fileHeader.Size == 0 {
		return nil, ErrEmptyFile
	}

	if fileHeader.Size > MaxUploadSize {
		return nil, ErrFileTooLarge
	}

	ext := strings.ToLower(filepath.Ext(fileHeader.Filename))
	if ext != ".jpg" && ext != ".jpeg" && ext != ".png" {
		return nil, ErrInvalidFileType
	}

	file, err := fileHeader.Open()
	if err != nil {
		return nil, fmt.Errorf("gagal membuka file upload: %w", err)
	}
	defer file.Close()

	// Forward in-memory stream to Python AI Service
	data, err := s.aiClient.ExtractKTP(ctx, fileHeader.Filename, file)
	if err != nil {
		return nil, err
	}

	return data, nil
}
