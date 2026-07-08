package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"shiftmaster-backend/internal/service"
)

type ProvinceHandler struct {
	svc service.ProvinceService
}

func NewProvinceHandler(svc service.ProvinceService) *ProvinceHandler {
	return &ProvinceHandler{svc: svc}
}

func (h *ProvinceHandler) GetAll(c *gin.Context) {
	provinces, err := h.svc.GetAll(c.Request.Context())
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

	province, err := h.svc.Create(c.Request.Context(), req)
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

	province, err := h.svc.Update(c.Request.Context(), id, req)
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

	if err := h.svc.Delete(c.Request.Context(), id); err != nil {
		if err == service.ErrProvinceNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete province"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Province deleted successfully"})
}
