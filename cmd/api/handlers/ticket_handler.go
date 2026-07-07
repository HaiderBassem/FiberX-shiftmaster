package handlers

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"shiftmaster-backend/internal/models"
	"shiftmaster-backend/internal/repository"
)

type TicketHandler struct {
	repo repository.TicketRepository
}

func NewTicketHandler(repo repository.TicketRepository) *TicketHandler {
	return &TicketHandler{repo: repo}
}

// CreateTicket creates a new cross-department ticket
func (h *TicketHandler) CreateTicket(c *gin.Context) {
	var req struct {
		TargetDepartmentID uuid.UUID `json:"target_department_id" binding:"required"`
		Title              string    `json:"title" binding:"required"`
		Description        string    `json:"description" binding:"required"`
		Attachments        *string   `json:"attachments"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "Invalid request payload: " + err.Error()})
		return
	}

	empIDStr, ok := c.Get("employee_id")
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"success": false, "error": "Missing employee context"})
		return
	}
	empID, err := uuid.Parse(empIDStr.(string))
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"success": false, "error": "Invalid employee ID"})
		return
	}

	deptIDStr, ok := c.Get("department_id")
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"success": false, "error": "Missing department context"})
		return
	}
	deptID, err := uuid.Parse(deptIDStr.(string))
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"success": false, "error": "Invalid department ID"})
		return
	}

	if req.TargetDepartmentID == deptID {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "Cannot create ticket to your own department"})
		return
	}

	ticket := &models.Ticket{
		SourceDepartmentID: deptID,
		TargetDepartmentID: req.TargetDepartmentID,
		CreatorID:          empID,
		Title:              req.Title,
		Description:        req.Description,
		Attachments:        req.Attachments,
	}

	if err := h.repo.Create(c.Request.Context(), ticket); err != nil {
		log.Printf("CreateTicket error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "Failed to create ticket"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"success": true, "data": ticket})
}

// GetTickets lists tickets relevant to the user's department
func (h *TicketHandler) GetTickets(c *gin.Context) {
	deptIDStr, ok := c.Get("department_id")
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"success": false, "error": "Missing department context"})
		return
	}
	deptID, err := uuid.Parse(deptIDStr.(string))
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"success": false, "error": "Invalid department ID"})
		return
	}

	tickets, err := h.repo.GetTicketsForDepartment(c.Request.Context(), deptID)
	if err != nil {
		log.Printf("GetTicketsForDepartment error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "Failed to fetch tickets"})
		return
	}
	if tickets == nil {
		tickets = []models.Ticket{}
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "data": tickets})
}

// CloseTicket closes a ticket
func (h *TicketHandler) CloseTicket(c *gin.Context) {
	ticketID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "Invalid ticket ID"})
		return
	}

	empIDStr, ok := c.Get("employee_id")
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"success": false, "error": "Missing employee context"})
		return
	}
	empID, err := uuid.Parse(empIDStr.(string))
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"success": false, "error": "Invalid employee ID"})
		return
	}

	deptIDStr, ok := c.Get("department_id")
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"success": false, "error": "Missing department context"})
		return
	}
	deptID, err := uuid.Parse(deptIDStr.(string))
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"success": false, "error": "Invalid department ID"})
		return
	}

	ticket, err := h.repo.GetTicketByID(c.Request.Context(), ticketID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "error": "Ticket not found"})
		return
	}

	// Verify department has access
	if ticket.SourceDepartmentID != deptID && ticket.TargetDepartmentID != deptID {
		c.JSON(http.StatusForbidden, gin.H{"success": false, "error": "You don't have access to this ticket"})
		return
	}

	if ticket.Status == "closed" {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "Ticket is already closed"})
		return
	}

	if err := h.repo.UpdateStatus(c.Request.Context(), ticketID, "closed", &empID); err != nil {
		log.Printf("CloseTicket error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "Failed to close ticket"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}

// AddComment adds a comment to a ticket
func (h *TicketHandler) AddComment(c *gin.Context) {
	ticketID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "Invalid ticket ID"})
		return
	}

	var req struct {
		Comment     string  `json:"comment" binding:"required"`
		Attachments *string `json:"attachments"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "Invalid request payload"})
		return
	}

	empIDStr, ok := c.Get("employee_id")
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"success": false, "error": "Missing employee context"})
		return
	}
	empID, err := uuid.Parse(empIDStr.(string))
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"success": false, "error": "Invalid employee ID"})
		return
	}

	deptIDStr, ok := c.Get("department_id")
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"success": false, "error": "Missing department context"})
		return
	}
	deptID, err := uuid.Parse(deptIDStr.(string))
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"success": false, "error": "Invalid department ID"})
		return
	}

	ticket, err := h.repo.GetTicketByID(c.Request.Context(), ticketID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "error": "Ticket not found"})
		return
	}

	// Verify department has access
	if ticket.SourceDepartmentID != deptID && ticket.TargetDepartmentID != deptID {
		c.JSON(http.StatusForbidden, gin.H{"success": false, "error": "You don't have access to this ticket"})
		return
	}

	if ticket.Status == "closed" {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "Cannot comment on a closed ticket"})
		return
	}

	comment := &models.TicketComment{
		TicketID:    ticketID,
		EmployeeID:  empID,
		Comment:     req.Comment,
		Attachments: req.Attachments,
	}

	if err := h.repo.AddComment(c.Request.Context(), comment); err != nil {
		log.Printf("AddComment error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "Failed to add comment"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"success": true, "data": comment})
}
