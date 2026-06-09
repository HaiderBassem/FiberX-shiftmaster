package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"shiftmaster-backend/internal/models"
	"shiftmaster-backend/internal/repository"
	"shiftmaster-backend/internal/service"
)

type AnnouncementHandler struct {
	announcementRepo repository.AnnouncementRepository
	announcementSvc  service.AnnouncementService
}

func NewAnnouncementHandler(ar repository.AnnouncementRepository, svc service.AnnouncementService) *AnnouncementHandler {
	return &AnnouncementHandler{
		announcementRepo: ar,
		announcementSvc:  svc,
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

func (h *AnnouncementHandler) Create(c *gin.Context) {
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

		// Handle image uploads
		form, _ := c.MultipartForm()
		if form != nil && form.File["images"] != nil {
			uploadDir := "./uploads/announcements"
			if err := os.MkdirAll(uploadDir, os.ModePerm); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "could not create upload directory"})
				return
			}

			var imageURLs []string
			for _, file := range form.File["images"] {
				// Max 5MB per image
				if file.Size > 5*1024*1024 {
					c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": fmt.Sprintf("Image %s exceeds 5MB limit", file.Filename)})
					return
				}

				fileName := fmt.Sprintf("%s_%d_%s", uuid.New().String()[:8], time.Now().UnixNano(), file.Filename)
				filePath := fmt.Sprintf("%s/%s", uploadDir, fileName)

				if err := c.SaveUploadedFile(file, filePath); err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "failed to save image"})
					return
				}

				publicURL := fmt.Sprintf("/api/uploads/announcements/%s", fileName)
				imageURLs = append(imageURLs, publicURL)
			}
			req.Images = imageURLs
		}

		// Also check for JSON-encoded images field (URLs from existing images)
		if existingImages := c.PostForm("existing_images"); existingImages != "" {
			var urls []string
			if err := json.Unmarshal([]byte(existingImages), &urls); err == nil {
				req.Images = append(req.Images, urls...)
			}
		}
	}

	// Ensure images is not nil
	if req.Images == nil {
		req.Images = []string{}
	}

	// Basic authorization check
	requesterStr, _ := c.Get("employee_id")
	requesterID, _ := uuid.Parse(requesterStr.(string))
	req.CreatedBy = requesterID

	depID := getDepartmentID(c)
	if depID == nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "Department ID is required"})
		return
	}
	req.DepartmentID = *depID

	roleStr, _ := c.Get("role")
	role := roleStr.(string)

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

	depID := getDepartmentID(c)
	if depID == nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "Department ID is required"})
		return
	}

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
