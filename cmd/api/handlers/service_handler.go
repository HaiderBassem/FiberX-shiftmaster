package handlers

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"shiftmaster-backend/internal/models"
	"shiftmaster-backend/internal/repository"
	"shiftmaster-backend/pkg/database"
)

// ServiceHandler handles FTTH service catalog API requests.
type ServiceHandler struct {
	repo repository.ServiceRepository
	db   *database.DB
}

// NewServiceHandler creates a new ServiceHandler.
func NewServiceHandler(repo repository.ServiceRepository, db *database.DB) *ServiceHandler {
	return &ServiceHandler{repo: repo, db: db}
}

// canManageServices checks if the current user has service management permission.
func (h *ServiceHandler) canManageServices(c *gin.Context) bool {
	roleAny, _ := c.Get("role")
	role, _ := roleAny.(string)

	// Admins always have access
	if role == "admin" {
		return true
	}

	// Team leaders need explicit can_manage_services permission
	if role == "team_leader" {
		empIDStr, ok := c.Get("employee_id")
		if !ok {
			return false
		}
		empID, err := uuid.Parse(empIDStr.(string))
		if err != nil {
			return false
		}
		var canManage bool
		err = h.db.QueryRow(c.Request.Context(),
			`SELECT COALESCE(can_manage_services, false) FROM employees WHERE id = $1`, empID,
		).Scan(&canManage)
		if err != nil {
			return false
		}
		return canManage
	}

	return false
}

// ═══════════════════════════════════════════════════════════
// Category Endpoints
// ═══════════════════════════════════════════════════════════

// ListCategoriesByProvince returns all service categories for a province.
func (h *ServiceHandler) ListCategoriesByProvince(c *gin.Context) {
	provinceIDStr := c.Query("province_id")
	if provinceIDStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "province_id is required"})
		return
	}
	provinceID, err := uuid.Parse(provinceIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "Invalid province ID"})
		return
	}

	cats, err := h.repo.GetCategoriesByProvince(c.Request.Context(), provinceID)
	if err != nil {
		log.Printf("ListCategoriesByProvince error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "Failed to load categories"})
		return
	}
	if cats == nil {
		cats = []models.ServiceCategory{}
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": cats})
}

// CreateCategory creates a new service category.
func (h *ServiceHandler) CreateCategory(c *gin.Context) {
	if !h.canManageServices(c) {
		c.JSON(http.StatusForbidden, gin.H{"success": false, "error": "You don't have permission to manage services"})
		return
	}

	var req struct {
		ProvinceID  uuid.UUID `json:"province_id" binding:"required"`
		Name        string    `json:"name" binding:"required"`
		Description *string   `json:"description"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "Invalid request: " + err.Error()})
		return
	}

	empIDStr, _ := c.Get("employee_id")
	empID, _ := uuid.Parse(empIDStr.(string))

	cat := &models.ServiceCategory{
		ProvinceID:  req.ProvinceID,
		Name:        req.Name,
		Description: req.Description,
		IsActive:    true,
		CreatedBy:   empID,
	}

	if err := h.repo.CreateCategory(c.Request.Context(), cat); err != nil {
		log.Printf("CreateCategory error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "Failed to create category"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"success": true, "data": cat})
}

// UpdateCategory updates an existing service category.
func (h *ServiceHandler) UpdateCategory(c *gin.Context) {
	if !h.canManageServices(c) {
		c.JSON(http.StatusForbidden, gin.H{"success": false, "error": "You don't have permission to manage services"})
		return
	}

	catID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "Invalid category ID"})
		return
	}

	var req struct {
		Name        string  `json:"name" binding:"required"`
		Description *string `json:"description"`
		IsActive    *bool   `json:"is_active"`
		SortOrder   *int    `json:"sort_order"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "Invalid request: " + err.Error()})
		return
	}

	existing, err := h.repo.GetCategoryByID(c.Request.Context(), catID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "error": "Category not found"})
		return
	}

	existing.Name = req.Name
	existing.Description = req.Description
	if req.IsActive != nil {
		existing.IsActive = *req.IsActive
	}
	if req.SortOrder != nil {
		existing.SortOrder = *req.SortOrder
	}

	if err := h.repo.UpdateCategory(c.Request.Context(), existing); err != nil {
		log.Printf("UpdateCategory error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "Failed to update category"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "data": existing})
}

// DeleteCategory deletes a service category and all its plans.
func (h *ServiceHandler) DeleteCategory(c *gin.Context) {
	if !h.canManageServices(c) {
		c.JSON(http.StatusForbidden, gin.H{"success": false, "error": "You don't have permission to manage services"})
		return
	}

	catID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "Invalid category ID"})
		return
	}

	existing, err := h.repo.GetCategoryByID(c.Request.Context(), catID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "error": "Category not found"})
		return
	}

	if err := h.repo.DeleteCategory(c.Request.Context(), catID); err != nil {
		log.Printf("DeleteCategory error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "Failed to delete category"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}

// ═══════════════════════════════════════════════════════════
// Plan Endpoints
// ═══════════════════════════════════════════════════════════

// ListPlans returns all plans for a category.
func (h *ServiceHandler) ListPlans(c *gin.Context) {
	catID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "Invalid category ID"})
		return
	}

	plans, err := h.repo.GetPlansByCategory(c.Request.Context(), catID)
	if err != nil {
		log.Printf("ListPlans error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "Failed to load plans"})
		return
	}
	if plans == nil {
		plans = []models.ServicePlan{}
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "data": plans})
}

// GetPlan returns a single plan by ID.
func (h *ServiceHandler) GetPlan(c *gin.Context) {
	planID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "Invalid plan ID"})
		return
	}

	plan, err := h.repo.GetPlanByID(c.Request.Context(), planID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "error": "Plan not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "data": plan})
}

// CreatePlan creates a new FTTH service plan inside a category.
func (h *ServiceHandler) CreatePlan(c *gin.Context) {
	if !h.canManageServices(c) {
		c.JSON(http.StatusForbidden, gin.H{"success": false, "error": "You don't have permission to manage services"})
		return
	}

	catID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "Invalid category ID"})
		return
	}

	var req struct {
		Name            string  `json:"name" binding:"required"`
		Price           float64 `json:"price" binding:"required"`
		DurationDays    int     `json:"duration_days" binding:"required"`
		SpeedDownload   *string `json:"speed_download"`
		SpeedUpload     *string `json:"speed_upload"`
		DataCap         *string `json:"data_cap"`
		ConnectionType  string  `json:"connection_type"`
		InstallationFee float64 `json:"installation_fee"`
		RouterIncluded  bool    `json:"router_included"`
		IPType          string  `json:"ip_type"`
		Description     *string `json:"description"`
		CabinetNotes    *string `json:"cabinet_notes"`
		Features        *string `json:"features"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "Invalid request: " + err.Error()})
		return
	}

	empIDStr, _ := c.Get("employee_id")
	empID, _ := uuid.Parse(empIDStr.(string))

	connType := req.ConnectionType
	if connType == "" {
		connType = "FTTH"
	}
	ipType := req.IPType
	if ipType == "" {
		ipType = "Dynamic"
	}

	plan := &models.ServicePlan{
		CategoryID:      catID,
		Name:            req.Name,
		Price:           req.Price,
		DurationDays:    req.DurationDays,
		SpeedDownload:   req.SpeedDownload,
		SpeedUpload:     req.SpeedUpload,
		DataCap:         req.DataCap,
		ConnectionType:  connType,
		InstallationFee: req.InstallationFee,
		RouterIncluded:  req.RouterIncluded,
		IPType:          ipType,
		Description:     req.Description,
		CabinetNotes:    req.CabinetNotes,
		Features:        req.Features,
		IsActive:        true,
		CreatedBy:       empID,
	}

	if err := h.repo.CreatePlan(c.Request.Context(), plan); err != nil {
		log.Printf("CreatePlan error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "Failed to create plan"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"success": true, "data": plan})
}

// UpdatePlan updates an existing service plan.
func (h *ServiceHandler) UpdatePlan(c *gin.Context) {
	if !h.canManageServices(c) {
		c.JSON(http.StatusForbidden, gin.H{"success": false, "error": "You don't have permission to manage services"})
		return
	}

	planID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "Invalid plan ID"})
		return
	}

	var req struct {
		Name            string  `json:"name" binding:"required"`
		Price           float64 `json:"price" binding:"required"`
		DurationDays    int     `json:"duration_days" binding:"required"`
		SpeedDownload   *string `json:"speed_download"`
		SpeedUpload     *string `json:"speed_upload"`
		DataCap         *string `json:"data_cap"`
		ConnectionType  string  `json:"connection_type"`
		InstallationFee float64 `json:"installation_fee"`
		RouterIncluded  bool    `json:"router_included"`
		IPType          string  `json:"ip_type"`
		Description     *string `json:"description"`
		CabinetNotes    *string `json:"cabinet_notes"`
		Features        *string `json:"features"`
		IsActive        *bool   `json:"is_active"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "Invalid request: " + err.Error()})
		return
	}

	existing, err := h.repo.GetPlanByID(c.Request.Context(), planID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "error": "Plan not found"})
		return
	}

	connType := req.ConnectionType
	if connType == "" {
		connType = "FTTH"
	}
	ipType := req.IPType
	if ipType == "" {
		ipType = "Dynamic"
	}

	existing.Name = req.Name
	existing.Price = req.Price
	existing.DurationDays = req.DurationDays
	existing.SpeedDownload = req.SpeedDownload
	existing.SpeedUpload = req.SpeedUpload
	existing.DataCap = req.DataCap
	existing.ConnectionType = connType
	existing.InstallationFee = req.InstallationFee
	existing.RouterIncluded = req.RouterIncluded
	existing.IPType = ipType
	existing.Description = req.Description
	existing.CabinetNotes = req.CabinetNotes
	existing.Features = req.Features
	if req.IsActive != nil {
		existing.IsActive = *req.IsActive
	}

	if err := h.repo.UpdatePlan(c.Request.Context(), existing); err != nil {
		log.Printf("UpdatePlan error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "Failed to update plan"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "data": existing})
}

// DeletePlan deletes a service plan.
func (h *ServiceHandler) DeletePlan(c *gin.Context) {
	if !h.canManageServices(c) {
		c.JSON(http.StatusForbidden, gin.H{"success": false, "error": "You don't have permission to manage services"})
		return
	}

	planID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "Invalid plan ID"})
		return
	}

	if err := h.repo.DeletePlan(c.Request.Context(), planID); err != nil {
		log.Printf("DeletePlan error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "Failed to delete plan"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}
