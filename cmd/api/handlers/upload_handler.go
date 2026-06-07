package handlers

import (
	"fmt"
	"mime/multipart"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// UploadHandler handles generic file uploads (e.g., for rich text editors)
type UploadHandler struct{}

func NewUploadHandler() *UploadHandler {
	return &UploadHandler{}
}

func (h *UploadHandler) UploadImage(c *gin.Context) {
	form, err := c.MultipartForm()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "invalid form"})
		return
	}

	var file *multipart.FileHeader
	for _, files := range form.File {
		if len(files) > 0 {
			file = files[0]
			break
		}
	}

	if file == nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "no file uploaded"})
		return
	}

	uploadDir := "./uploads/images"
	if err := os.MkdirAll(uploadDir, os.ModePerm); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "could not create upload directory"})
		return
	}

	// Generate a unique file name
	fileName := fmt.Sprintf("%s_%d_%s", uuid.New().String()[:8], time.Now().Unix(), file.Filename)
	filePath := fmt.Sprintf("%s/%s", uploadDir, fileName)

	if err := c.SaveUploadedFile(file, filePath); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "failed to save file"})
		return
	}

	publicURL := fmt.Sprintf("/api/uploads/images/%s", fileName)

	// Jodit expects a specific response format, but we can configure Jodit to read standard ones.
	// For general usage and standard Jodit response:
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"url": publicURL,
		},
		// Jodit specific format fallback
		"file": publicURL,
	})
}
