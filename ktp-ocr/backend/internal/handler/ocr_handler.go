package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"ktp-ocr/backend/internal/service"
)

type OCRHandler struct {
	ocrService *service.OCRService
}

func NewOCRHandler(ocrService *service.OCRService) *OCRHandler {
	return &OCRHandler{
		ocrService: ocrService,
	}
}

func (h *OCRHandler) HandleKTPOCR(c *gin.Context) {
	fileHeader, err := c.FormFile("image")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Field 'image' (multipart/form-data) tidak ditemukan atau tidak valid.",
		})
		return
	}

	result, err := h.ocrService.ProcessKTP(c.Request.Context(), fileHeader)
	if err != nil {
		if errors.Is(err, service.ErrEmptyFile) || errors.Is(err, service.ErrFileTooLarge) || errors.Is(err, service.ErrInvalidFileType) {
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"error":   err.Error(),
			})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    result,
	})
}

func (h *OCRHandler) HandleHealth(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":  "healthy",
		"service": "ktp-ocr-go-backend",
	})
}
