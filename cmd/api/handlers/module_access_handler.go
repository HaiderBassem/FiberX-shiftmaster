package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"shiftmaster-backend/internal/service"
)

type ModuleAccessHandler struct {
	svc *service.ModuleAccessService
}

func NewModuleAccessHandler(svc *service.ModuleAccessService) *ModuleAccessHandler {
	return &ModuleAccessHandler{svc: svc}
}

// ---- Link CRUD (Admin Only) ----
func (h *ModuleAccessHandler) CreateLink(c *gin.Context) {
	var req struct {
		Title    string `json:"title" binding:"required"`
		URL      string `json:"url" binding:"required"`
		IconName string `json:"icon_name"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": err.Error()})
		return
	}

	empIDStr, _ := c.Get("employee_id")
	empID, _ := uuid.Parse(empIDStr.(string))

	link, err := h.svc.CreateLink(c.Request.Context(), req.Title, req.URL, req.IconName, &empID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "data": link})
}

func (h *ModuleAccessHandler) UpdateLink(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "invalid link id"})
		return
	}

	var req struct {
		Title    string `json:"title" binding:"required"`
		URL      string `json:"url" binding:"required"`
		IconName string `json:"icon_name"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": err.Error()})
		return
	}

	if err := h.svc.UpdateLink(c.Request.Context(), id, req.Title, req.URL, req.IconName); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}

func (h *ModuleAccessHandler) DeleteLink(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "invalid link id"})
		return
	}

	if err := h.svc.DeleteLink(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}

func (h *ModuleAccessHandler) GetAllLinks(c *gin.Context) {
	links, err := h.svc.GetAllLinks(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": links})
}

// ---- Access Management ----

func (h *ModuleAccessHandler) GetMyModules(c *gin.Context) {
	empIDStr, _ := c.Get("employee_id")
	empID, _ := uuid.Parse(empIDStr.(string))

	modules, err := h.svc.GetMyModules(c.Request.Context(), empID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "data": modules})
}

func (h *ModuleAccessHandler) GetAccess(c *gin.Context) {
	linkID, err := uuid.Parse(c.Param("link_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "invalid link id"})
		return
	}
	
	depID := getDepartmentID(c) 

	resp, err := h.svc.GetLinkAccess(c.Request.Context(), linkID, depID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "data": resp})
}

func (h *ModuleAccessHandler) SetDepartmentAccess(c *gin.Context) {
	linkID, err := uuid.Parse(c.Param("link_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "invalid link id"})
		return
	}

	var req struct {
		DepartmentID uuid.UUID `json:"department_id"`
		Grant        bool      `json:"grant"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": err.Error()})
		return
	}

	empIDStr, _ := c.Get("employee_id")
	empID, _ := uuid.Parse(empIDStr.(string))

	if err := h.svc.SetDepartmentAccess(c.Request.Context(), linkID, req.DepartmentID, req.Grant, &empID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}

func (h *ModuleAccessHandler) SetEmployeeExclusion(c *gin.Context) {
	linkID, err := uuid.Parse(c.Param("link_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "invalid link id"})
		return
	}

	var req struct {
		EmployeeID uuid.UUID `json:"employee_id"`
		Exclude    bool      `json:"exclude"` // true means they CANNOT see it
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": err.Error()})
		return
	}

	empIDStr, _ := c.Get("employee_id")
	empID, _ := uuid.Parse(empIDStr.(string))
	
	if err := h.svc.SetEmployeeExclusion(c.Request.Context(), linkID, req.EmployeeID, req.Exclude, &empID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}
