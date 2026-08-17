package handlers

import (
	"errors"
	"fmt"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"

	"github.com/gin-gonic/gin"

	"shiftmaster-backend/internal/config"
	"shiftmaster-backend/internal/upload"
)

// UploadHandler handles generic file uploads (e.g., for rich text editors) and
// serves previously uploaded files back.
type UploadHandler struct {
	cfg config.UploadConfig
}

func NewUploadHandler(cfg config.UploadConfig) *UploadHandler {
	return &UploadHandler{cfg: cfg}
}

// imagesDir is the subdirectory rich-text and announcement images are stored in.
func (h *UploadHandler) imagesDir() string {
	return filepath.Join(h.cfg.BasePath, "images")
}

// uploadOptions builds validation options from configuration.
func (h *UploadHandler) uploadOptions() upload.Options {
	return upload.Options{
		MaxBytes:     h.cfg.MaxSizeBytes(),
		AllowedTypes: h.cfg.AllowedTypes,
	}
}

// respondUploadError maps a validation failure onto a status code and a message
// that tells the user how to fix it without describing server internals.
func respondUploadError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, upload.ErrEmptyFile):
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "the file is empty"})
	case errors.Is(err, upload.ErrTooLarge):
		c.JSON(http.StatusRequestEntityTooLarge, gin.H{"success": false, "error": err.Error()})
	case errors.Is(err, upload.ErrTooManyPixels):
		c.JSON(http.StatusRequestEntityTooLarge, gin.H{"success": false, "error": "the image dimensions are too large"})
	case errors.Is(err, upload.ErrFormatBlocked):
		c.JSON(http.StatusUnsupportedMediaType, gin.H{"success": false, "error": err.Error()})
	case errors.Is(err, upload.ErrNotAnImage):
		c.JSON(http.StatusUnsupportedMediaType, gin.H{"success": false, "error": "the file is not a readable JPEG, PNG or GIF image"})
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "failed to save file"})
	}
}

// firstUploadedFile returns the first file part in a multipart form, regardless of
// the field name. Jodit and the announcement editor use different field names.
func firstUploadedFile(form *multipart.Form) *multipart.FileHeader {
	for _, files := range form.File {
		if len(files) > 0 {
			return files[0]
		}
	}
	return nil
}

// UploadImage accepts an image, validates and re-encodes it, and stores it under a
// server-generated name.
//
// Every uploaded file is decoded and rebuilt (see internal/upload), so content that
// merely claims to be an image — HTML with a .png extension, an SVG, a polyglot
// carrying a script payload — is rejected or stripped rather than being written to
// a path the browser will later fetch from this application's origin.
func (h *UploadHandler) UploadImage(c *gin.Context) {
	// Bound the amount buffered in memory before the body is parsed. Anything
	// larger spills to a temp file, and the size limit below still applies.
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, h.cfg.MaxSizeBytes()+maxMultipartOverhead)

	form, err := c.MultipartForm()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "invalid form or file too large"})
		return
	}

	file := firstUploadedFile(form)
	if file == nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "no file uploaded"})
		return
	}

	img, err := upload.SanitizeImage(file, h.uploadOptions())
	if err != nil {
		respondUploadError(c, err)
		return
	}

	if _, err := upload.Store(h.imagesDir(), img); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "failed to save file"})
		return
	}

	publicURL := fmt.Sprintf("/api/uploads/images/%s", img.Filename)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"url": publicURL,
		},
		// Jodit specific format fallback
		"file": publicURL,
	})
}

// maxMultipartOverhead allows for multipart boundaries and headers on top of the
// file itself, so a legitimate upload at exactly the size limit is not rejected by
// the body reader before the clearer per-file check runs.
const maxMultipartOverhead = 1 << 20 // 1MB

// ServeUpload serves a previously stored upload.
//
// It deliberately replaces gin's static file server. That helper derives the
// Content-Type from the file extension, which means a file on disk named .html is
// returned as text/html and executes in this origin. Here the type comes from a
// fixed table keyed on a validated extension, so an upload can only ever be served
// as one of the image types the validator produces — anything else is a 404.
func (h *UploadHandler) ServeUpload(c *gin.Context) {
	// Gin's wildcard parameter arrives with a leading slash.
	requested := c.Param("filepath")
	subdir, name := splitUploadPath(requested)
	if subdir == "" || name == "" {
		c.AbortWithStatus(http.StatusNotFound)
		return
	}

	contentType, ok := upload.ContentTypeForExt(filepath.Ext(name))
	if !ok {
		c.AbortWithStatus(http.StatusNotFound)
		return
	}

	fullPath, err := upload.ResolveWithin(filepath.Join(h.cfg.BasePath, subdir), name)
	if err != nil {
		c.AbortWithStatus(http.StatusNotFound)
		return
	}

	info, err := os.Stat(fullPath)
	if err != nil || info.IsDir() {
		c.AbortWithStatus(http.StatusNotFound)
		return
	}

	// nosniff stops a browser from second-guessing the declared type; the sandbox
	// CSP neutralises any active content that might still be reached if the type
	// were ever wrong; inline disposition keeps <img> rendering working.
	c.Header("Content-Type", contentType)
	c.Header("X-Content-Type-Options", "nosniff")
	c.Header("Content-Security-Policy", "default-src 'none'; style-src 'unsafe-inline'; sandbox")
	c.Header("Content-Disposition", "inline")
	c.Header("Cross-Origin-Resource-Policy", "same-origin")
	c.Header("Cache-Control", "private, max-age=300")

	c.File(fullPath)
}

// splitUploadPath splits "/images/abc.png" into ("images", "abc.png"). It accepts
// exactly one directory level; anything deeper or shallower is rejected by the
// caller, which keeps the served surface to the known subdirectories.
func splitUploadPath(p string) (subdir string, name string) {
	cleaned := filepath.Clean("/" + p)
	dir, file := filepath.Split(cleaned)

	// dir is "/images/" for a valid request.
	trimmed := filepath.Clean(dir)
	if trimmed == "/" || trimmed == "." {
		return "", ""
	}
	sub := trimmed[1:]
	if sub == "" || filepath.Base(sub) != sub {
		// More than one directory level.
		return "", ""
	}

	switch sub {
	case "images", "profiles", "documents", "assignments":
		return sub, file
	default:
		return "", ""
	}
}
