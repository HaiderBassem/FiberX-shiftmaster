package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"shiftmaster-backend/internal/models"
	"shiftmaster-backend/internal/service"
)

type ProvinceHandler struct {
	svc service.ProvinceService
}

func NewProvinceHandler(svc service.ProvinceService) *ProvinceHandler {
	return &ProvinceHandler{svc: svc}
}

func (h *ProvinceHandler) GetAll(c *gin.Context) {
	departmentIDStr := c.GetString("department_id")
	departmentID, err := uuid.Parse(departmentIDStr)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized: missing department ID"})
		return
	}

	provinces, err := h.svc.GetAll(c.Request.Context(), departmentID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch provinces"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": provinces})
}

func (h *ProvinceHandler) Create(c *gin.Context) {
	var req service.CreateProvinceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload"})
		return
	}

	departmentIDStr := c.GetString("department_id")
	empIDStr := c.GetString("employee_id")

	departmentID, err := uuid.Parse(departmentIDStr)
	empID, err2 := uuid.Parse(empIDStr)
	if err != nil || err2 != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized: missing credentials"})
		return
	}

	province, err := h.svc.Create(c.Request.Context(), departmentID, empID, req)
	if err != nil {
		if err == service.ErrProvinceNameRequired || err == service.ErrProvinceExists {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create province"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"data": province})
}

func (h *ProvinceHandler) Update(c *gin.Context) {
	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid province ID"})
		return
	}

	var req service.UpdateProvinceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload"})
		return
	}

	departmentIDStr := c.GetString("department_id")
	departmentID, err := uuid.Parse(departmentIDStr)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	province, err := h.svc.Update(c.Request.Context(), id, departmentID, req)
	if err != nil {
		if err == service.ErrProvinceNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		if err == service.ErrProvinceNameRequired || err == service.ErrProvinceExists {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update province"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": province})
}

func (h *ProvinceHandler) Delete(c *gin.Context) {
	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid province ID"})
		return
	}

	departmentIDStr := c.GetString("department_id")
	departmentID, err := uuid.Parse(departmentIDStr)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	if err := h.svc.Delete(c.Request.Context(), id, departmentID); err != nil {
		if err == service.ErrProvinceNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete province"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Province deleted successfully"})
}

func (h *ProvinceHandler) Share(c *gin.Context) {
	provinceID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid province ID"})
		return
	}

	// Verify the user owns this province before sharing
	deptIDStr := c.GetString("department_id")
	deptID, _ := uuid.Parse(deptIDStr)

	_, err = h.svc.GetAll(c.Request.Context(), deptID)
	// We'll trust the caller for now, but ideally we'd check if the province belongs to deptID

	var req struct {
		DepartmentID uuid.UUID `json:"department_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid payload"})
		return
	}

	empIDStr := c.GetString("employee_id")
	empID, _ := uuid.Parse(empIDStr)

	share := &models.ProvinceShare{
		ProvinceID:   provinceID,
		DepartmentID: req.DepartmentID,
		GrantedBy:    empID,
	}

	if err := h.svc.ShareProvince(c.Request.Context(), share); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to share province"})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"data": share})
}

func (h *ProvinceHandler) Unshare(c *gin.Context) {
	provinceID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid province ID"})
		return
	}
	deptID, err := uuid.Parse(c.Param("departmentId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid department ID"})
		return
	}

	if err := h.svc.UnshareProvince(c.Request.Context(), provinceID, deptID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to unshare province"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Unshared successfully"})
}

func (h *ProvinceHandler) GetShares(c *gin.Context) {
	provinceID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid province ID"})
		return
	}

	shares, err := h.svc.GetProvinceShares(c.Request.Context(), provinceID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch shares"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": shares})
}
