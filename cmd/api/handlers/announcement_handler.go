package handlers

import (
	"net/http"

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

// Helper to extract department_id from header/token (similar to others)
func (h *AnnouncementHandler) getDepartmentID(c *gin.Context) *uuid.UUID {
	headerDeptID := c.GetHeader("X-Department-ID")
	if headerDeptID != "" {
		if id, err := uuid.Parse(headerDeptID); err == nil {
			return &id
		}
	}
	tokenDeptID, exists := c.Get("department_id")
	if exists {
		if deptIDStr, ok := tokenDeptID.(string); ok && deptIDStr != "" {
			if id, err := uuid.Parse(deptIDStr); err == nil {
				return &id
			}
		}
	}
	return nil
}

func (h *AnnouncementHandler) GetActive(c *gin.Context) {
	depID := h.getDepartmentID(c)
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
	depID := h.getDepartmentID(c)
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
	var req models.Announcement
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": err.Error()})
		return
	}

	// Basic authorization check: must be admin/manager or have can_post_announcements
	// Since we are enforcing it primarily on UI, let's just make sure they are valid.
	requesterStr, _ := c.Get("employee_id")
	requesterID, _ := uuid.Parse(requesterStr.(string))
	req.CreatedBy = requesterID

	depID := h.getDepartmentID(c)
	if depID == nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "Department ID is required"})
		return
	}
	req.DepartmentID = *depID

	if err := h.announcementSvc.CreateAnnouncement(c.Request.Context(), &req); err != nil {
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

	depID := h.getDepartmentID(c)
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

	depID := h.getDepartmentID(c)
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
