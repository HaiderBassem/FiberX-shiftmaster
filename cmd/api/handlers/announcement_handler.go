package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"shiftmaster-backend/internal/config"
	"shiftmaster-backend/internal/models"
	"shiftmaster-backend/internal/repository"
	"shiftmaster-backend/internal/service"
	"shiftmaster-backend/internal/upload"
)

type AnnouncementHandler struct {
	announcementRepo repository.AnnouncementRepository
	announcementSvc  service.AnnouncementService
	uploadCfg        config.UploadConfig
}

func NewAnnouncementHandler(ar repository.AnnouncementRepository, svc service.AnnouncementService, uploadCfg config.UploadConfig) *AnnouncementHandler {
	return &AnnouncementHandler{
		announcementRepo: ar,
		announcementSvc:  svc,
		uploadCfg:        uploadCfg,
	}
}

func (h *AnnouncementHandler) GetActive(c *gin.Context) {
	depID := getDepartmentID(c)
	if depID == nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "Department ID is required"})
		return
	}

	announcement, err := h.announcementRepo.GetActiveByDepartment(c.Request.Context(), *depID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "data": announcement})
}

func (h *AnnouncementHandler) GetActiveTicker(c *gin.Context) {
	depID := getDepartmentID(c)
	if depID == nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "Department ID is required"})
		return
	}

	announcement, err := h.announcementRepo.GetActiveTickerByDepartment(c.Request.Context(), *depID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "data": announcement})
}

func (h *AnnouncementHandler) GetAll(c *gin.Context) {
	depID := getDepartmentID(c)
	if depID == nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "Department ID is required"})
		return
	}

	announcements, err := h.announcementRepo.GetAllByDepartment(c.Request.Context(), *depID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
		return
	}

	// Initialize to empty array if null
	if announcements == nil {
		announcements = []models.Announcement{}
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "data": announcements})
}

// storedUploadPath reports whether s is a same-origin path into the
// authenticated upload store — the only shape this application has ever
// written into announcements.images. It is the validation gate for
// "existing_images", which round-trips previously stored URLs through the
// client on edit: anything else (absolute URLs, javascript:, data:) has no
// legitimate way to appear there and is dropped.
func storedUploadPath(s string) bool {
	return strings.HasPrefix(s, "/api/uploads/") || strings.HasPrefix(s, "/uploads/")
}

func (h *AnnouncementHandler) Create(c *gin.Context) {
	// Authorization first: image files must not be written to disk for a
	// requester who is not allowed to post announcements at all. The service
	// re-checks this on create; this early check only prevents the side effect.
	requesterStr, _ := c.Get("employee_id")
	requesterID, _ := uuid.Parse(requesterStr.(string))
	roleValue, _ := c.Get("role")
	role, _ := roleValue.(string)
	if !h.announcementSvc.CanManage(c.Request.Context(), requesterID, role) {
		c.JSON(http.StatusForbidden, gin.H{"success": false, "error": "unauthorized to post announcements"})
		return
	}

	contentType := c.ContentType()

	var req models.Announcement

	if contentType == "application/json" {
		// Legacy JSON support (no images)
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": err.Error()})
			return
		}
	} else {
		// Multipart form data (with images)
		req.Title = c.PostForm("title")
		req.Message = c.PostForm("message")
		req.Priority = c.PostForm("priority")
		if req.Priority == "" {
			req.Priority = "normal"
		}
		req.IsActive = c.PostForm("is_active") == "true"
		req.IsTicker = c.PostForm("is_ticker") == "true"

		// Announcement images go through the same sanitize-and-store pipeline
		// as every other upload: decoded, re-encoded, stored under a
		// server-generated name in the configured base path. This used to write
		// raw client bytes under a hardcoded ./uploads/announcements, which
		// bypassed validation entirely and broke whenever UPLOAD_BASE_PATH
		// pointed anywhere else — the serving handler reads the configured path.
		form, _ := c.MultipartForm()
		if form != nil && form.File["images"] != nil {
			opts := upload.Options{
				MaxBytes:     h.uploadCfg.MaxSizeBytes(),
				AllowedTypes: h.uploadCfg.AllowedTypes,
			}
			dir := filepath.Join(h.uploadCfg.BasePath, "announcements")

			var imageURLs []string
			for _, file := range form.File["images"] {
				img, err := upload.SanitizeImage(file, opts)
				if err != nil {
					respondUploadError(c, err)
					return
				}
				if _, err := upload.Store(dir, img); err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "failed to save image"})
					return
				}
				imageURLs = append(imageURLs, fmt.Sprintf("/api/uploads/announcements/%s", img.Filename))
			}
			req.Images = imageURLs
		}

		// URLs of already-stored images, round-tripped by the editor when an
		// announcement is re-posted with its old pictures. Only paths this
		// server could have issued are accepted.
		if existingImages := c.PostForm("existing_images"); existingImages != "" {
			var urls []string
			if err := json.Unmarshal([]byte(existingImages), &urls); err == nil {
				for _, u := range urls {
					if storedUploadPath(u) {
						req.Images = append(req.Images, u)
					}
				}
			}
		}
	}

	// Ensure images is not nil
	if req.Images == nil {
		req.Images = []string{}
	}

	req.CreatedBy = requesterID

	depID := getDepartmentID(c)
	if depID == nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "Department ID is required"})
		return
	}
	req.DepartmentID = *depID

	if err := h.announcementSvc.CreateAnnouncement(c.Request.Context(), &req, role); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"success": true, "data": req})
}

func (h *AnnouncementHandler) Delete(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "Invalid announcement ID"})
		return
	}

	me, ok := actorID(c)
	if !ok || !h.announcementSvc.CanManage(c.Request.Context(), me, actorRole(c)) {
		c.JSON(http.StatusForbidden, gin.H{"success": false, "error": "unauthorized to manage announcements"})
		return
	}

	depID := getDepartmentID(c)
	if depID == nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "Department ID is required"})
		return
	}

	err = h.announcementRepo.Delete(c.Request.Context(), id, *depID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}

func (h *AnnouncementHandler) SetActive(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "Invalid announcement ID"})
		return
	}

	me, ok := actorID(c)
	if !ok || !h.announcementSvc.CanManage(c.Request.Context(), me, actorRole(c)) {
		c.JSON(http.StatusForbidden, gin.H{"success": false, "error": "unauthorized to manage announcements"})
		return
	}

	depID := getDepartmentID(c)
	if depID == nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "Department ID is required"})
		return
	}

	// If the new announcement is active, we don't deactivate others anymore. They can have multiple active announcements.
	// Oh wait, if they only want ONE active announcement, then we should keep SetInactiveByDepartment here.
	// Let's actually keep SetInactiveByDepartment and add it back to the repo because SetActive uses it!
	err = h.announcementRepo.SetInactiveByDepartment(c.Request.Context(), *depID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
		return
	}

	err = h.announcementRepo.SetActive(c.Request.Context(), id, *depID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}

func (h *AnnouncementHandler) Deactivate(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "Invalid announcement ID"})
		return
	}
	me, ok := actorID(c)
	if !ok || !h.announcementSvc.CanManage(c.Request.Context(), me, actorRole(c)) {
		c.JSON(http.StatusForbidden, gin.H{"success": false, "error": "unauthorized to manage announcements"})
		return
	}

	depID := getDepartmentID(c)
	if depID == nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "Department ID is required"})
		return
	}
	err = h.announcementRepo.SetInactive(c.Request.Context(), id, *depID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}
