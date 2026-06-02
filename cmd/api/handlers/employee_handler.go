package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"shiftmaster-backend/internal/models"
	"shiftmaster-backend/internal/service"
)

// EmployeeHandler handles employee CRUD endpoints.
type EmployeeHandler struct {
	employeeService *service.EmployeeService
}

func NewEmployeeHandler(empSvc *service.EmployeeService) *EmployeeHandler {
	return &EmployeeHandler{employeeService: empSvc}
}

// List returns employees with optional filters.
func (h *EmployeeHandler) List(c *gin.Context) {
	ctx := c.Request.Context()
	var employees []models.Employee
	var err error

	roleAny, _ := c.Get("role")
	role, _ := roleAny.(string)

	requesterStr, _ := c.Get("employee_id")
	requesterID, _ := uuid.Parse(requesterStr.(string))

	// Scope: strictly isolate non-admins to their own department.
	if role != "admin" {
		me, meErr := h.employeeService.GetByID(ctx, requesterID)
		if meErr != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": meErr.Error()})
			return
		}

		// For managers, honour the X-Department-ID header so they can switch
		// between departments they manage. For other roles, always use their own
		// department_id from the JWT.
		var targetDeptID *uuid.UUID
		if role == "manager" {
			targetDeptID = getDepartmentID(c)
		}
		if targetDeptID == nil {
			targetDeptID = me.DepartmentID
		}

		if targetDeptID == nil {
			employees = []models.Employee{}
			c.JSON(http.StatusOK, gin.H{"success": true, "data": employees, "meta": gin.H{"count": 0}})
			return
		}

		// Employees can only ever see people in their own department.
		deptEmployees, err := h.employeeService.GetByDepartment(ctx, *targetDeptID)
		if err == nil {
			// Role-based visibility filtering within the department:
			filtered := make([]models.Employee, 0, len(deptEmployees))
			for _, e := range deptEmployees {
				if role == "manager" && (e.Role == "employee" || e.Role == "team_leader") {
					filtered = append(filtered, e)
				} else if role == "team_leader" && e.Role == "employee" {
					filtered = append(filtered, e)
				} else if role == "employee" {
					// Employees can see other employees/TLs in their department (e.g., for swaps)
					filtered = append(filtered, e)
				}
			}
			employees = filtered
		}
	} else {
		// Admin keeps existing unrestricted filter behavior.
		switch {
		case c.Query("active") == "true":
			employees, err = h.employeeService.GetActive(ctx)
		case c.Query("department_id") != "":
			deptID, parseErr := uuid.Parse(c.Query("department_id"))
			if parseErr != nil {
				c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "invalid department_id"})
				return
			}
			employees, err = h.employeeService.GetByDepartment(ctx, deptID)
		case c.Query("role") != "":
			employees, err = h.employeeService.GetByRole(ctx, c.Query("role"))
		case c.Query("shift_id") != "":
			shiftID, parseErr := uuid.Parse(c.Query("shift_id"))
			if parseErr != nil {
				c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "invalid shift_id"})
				return
			}
			employees, err = h.employeeService.GetByShiftID(ctx, shiftID)
		default:
			employees, err = h.employeeService.GetAll(ctx)
		}
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
		return
	}
	if employees == nil {
		employees = []models.Employee{}
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "data": employees, "meta": gin.H{"count": len(employees)}})
}

// GetByID returns a single employee.
func (h *EmployeeHandler) GetByID(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "invalid employee ID"})
		return
	}

	emp, err := h.employeeService.GetByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "error": err.Error()})
		return
	}

	// Scope enforcement: strictly isolate non-admins to their own department.
	roleAny, _ := c.Get("role")
	role, _ := roleAny.(string)
	if role != "admin" {
		requesterStr, _ := c.Get("employee_id")
		requesterID, _ := uuid.Parse(requesterStr.(string))

		// allow an employee to always view their own profile
		if requesterID != id {
			me, meErr := h.employeeService.GetByID(c.Request.Context(), requesterID)
			if meErr != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": meErr.Error()})
				return
			}

			// For managers use the selected department (X-Department-ID header if present,
			// otherwise fall back to the manager's own department_id).
			var scopeDeptID *uuid.UUID
			if role == "manager" {
				scopeDeptID = getDepartmentID(c)
			}
			if scopeDeptID == nil {
				scopeDeptID = me.DepartmentID
			}

			if scopeDeptID == nil || emp.DepartmentID == nil || *scopeDeptID != *emp.DepartmentID {
				c.JSON(http.StatusForbidden, gin.H{"success": false, "error": "forbidden"})
				return
			}
			// Managers can view employees + team_leaders; team_leaders can only view employees.
			if role == "manager" && emp.Role != "employee" && emp.Role != "team_leader" {
				c.JSON(http.StatusForbidden, gin.H{"success": false, "error": "forbidden"})
				return
			}
			if role == "team_leader" && emp.Role != "employee" {
				c.JSON(http.StatusForbidden, gin.H{"success": false, "error": "forbidden"})
				return
			}
			// employees can view anyone in their own department (e.g., for swaps)
		}
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "data": emp})
}

type createEmployeeRequest struct {
	EmployeeCode       string     `json:"employee_code"`
	FirstName          string     `json:"first_name" binding:"required"`
	LastName           string     `json:"last_name" binding:"required"`
	Gender             string     `json:"gender" binding:"required"`
	Phone              *string    `json:"phone"`
	Email              string     `json:"email" binding:"required,email"`
	Password           string     `json:"password" binding:"required,min=8"`
	HireDate           string     `json:"hire_date" binding:"required"`
	Role               string     `json:"role" binding:"required"`
	DepartmentID       *uuid.UUID `json:"department_id"`
	Position           *string    `json:"position"`
	DefaultShiftID     *uuid.UUID `json:"default_shift_id"`
	WeeklyOffDays      int        `json:"weekly_off_days"`
	CanCoverNightShift bool       `json:"can_cover_night_shift"`
	CanManageHelpDocs  bool       `json:"can_manage_help_docs"`
	Status             string     `json:"status"`
	ProfileImage       *string    `json:"profile_image"`
	SecondaryPhone     *string    `json:"secondary_phone"`
	SecondaryEmail     *string    `json:"secondary_email"`
}

// Create creates a new employee.
// Role-based restrictions:
//   - team_leader → can only create role=employee
//   - manager     → can create role=employee or role=team_leader
//   - admin       → can create any role
func (h *EmployeeHandler) Create(c *gin.Context) {
	var req createEmployeeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "invalid request: " + err.Error()})
		return
	}

	// Enforce role-based creation hierarchy
	creatorRole, _ := c.Get("role")
	creatorRoleStr, _ := creatorRole.(string)

	switch creatorRoleStr {
	case "team_leader":
		if req.Role != "employee" {
			c.JSON(http.StatusForbidden, gin.H{"success": false, "error": "team leaders can only create employee accounts"})
			return
		}
	case "manager":
		if req.Role != "employee" && req.Role != "team_leader" {
			c.JSON(http.StatusForbidden, gin.H{"success": false, "error": "managers can only create employee and team_leader accounts"})
			return
		}
	case "admin":
		// admin can create any role
	default:
		c.JSON(http.StatusForbidden, gin.H{"success": false, "error": "insufficient permissions to create accounts"})
		return
	}

	// Department scope: team leaders/managers can only create in their own department.
	if creatorRoleStr == "team_leader" || creatorRoleStr == "manager" {
		creatorIDStr, _ := c.Get("employee_id")
		creatorID, _ := uuid.Parse(creatorIDStr.(string))
		creator, getErr := h.employeeService.GetByID(c.Request.Context(), creatorID)
		if getErr != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": getErr.Error()})
			return
		}
		if creator.DepartmentID == nil {
			c.JSON(http.StatusForbidden, gin.H{"success": false, "error": "you are not assigned to a department"})
			return
		}
		if req.DepartmentID != nil && *req.DepartmentID != *creator.DepartmentID {
			c.JSON(http.StatusForbidden, gin.H{"success": false, "error": "cannot create employees in another department"})
			return
		}
		req.DepartmentID = creator.DepartmentID
	}

	hireDate, err := parseTime(req.HireDate)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "invalid hire_date format"})
		return
	}

	createdByStr, _ := c.Get("employee_id")
	createdBy, _ := uuid.Parse(createdByStr.(string))

	emp := &models.Employee{
		EmployeeCode:       req.EmployeeCode,
		FirstName:          req.FirstName,
		LastName:           req.LastName,
		Gender:             req.Gender,
		Phone:              req.Phone,
		Email:              req.Email,
		HireDate:           hireDate,
		Role:               req.Role,
		DepartmentID:       req.DepartmentID,
		Position:           req.Position,
		DefaultShiftID:     req.DefaultShiftID,
		WeeklyOffDays:      req.WeeklyOffDays,
		CanCoverNightShift: req.CanCoverNightShift,
		CanManageHelpDocs:  req.CanManageHelpDocs,
		Status:             req.Status,
		ProfileImage:       req.ProfileImage,
		SecondaryPhone:     req.SecondaryPhone,
		SecondaryEmail:     req.SecondaryEmail,
		CreatedBy:          &createdBy,
	}

	if err := h.employeeService.CreateEmployee(c.Request.Context(), emp, req.Password); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"success": true, "data": emp})
}

type updateEmployeeRequest struct {
	FirstName          string     `json:"first_name"`
	LastName           string     `json:"last_name"`
	Gender             string     `json:"gender"`
	Phone              *string    `json:"phone"`
	Email              string     `json:"email"`
	Role               string     `json:"role"`
	DepartmentID       *uuid.UUID `json:"department_id"`
	Position           *string    `json:"position"`
	DefaultShiftID     *uuid.UUID `json:"default_shift_id"`
	WeeklyOffDays      int        `json:"weekly_off_days"`
	CanCoverNightShift bool       `json:"can_cover_night_shift"`
	CanManageHelpDocs  bool       `json:"can_manage_help_docs"`
	Status             string     `json:"status"`
	ProfileImage       *string    `json:"profile_image"`
	SecondaryPhone     *string    `json:"secondary_phone"`
	SecondaryEmail     *string    `json:"secondary_email"`
}

// Update updates an employee.
func (h *EmployeeHandler) Update(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "invalid employee ID"})
		return
	}

	var req updateEmployeeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "invalid request: " + err.Error()})
		return
	}

	roleAny, _ := c.Get("role")
	role, _ := roleAny.(string)

	// Scope enforcement for team_leader/manager.
	if role == "team_leader" || role == "manager" {
		requesterStr, _ := c.Get("employee_id")
		requesterID, _ := uuid.Parse(requesterStr.(string))

		me, meErr := h.employeeService.GetByID(c.Request.Context(), requesterID)
		if meErr != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": meErr.Error()})
			return
		}
		target, targetErr := h.employeeService.GetByID(c.Request.Context(), id)
		if targetErr != nil {
			c.JSON(http.StatusNotFound, gin.H{"success": false, "error": targetErr.Error()})
			return
		}

		if me.DepartmentID == nil || target.DepartmentID == nil || *me.DepartmentID != *target.DepartmentID {
			c.JSON(http.StatusForbidden, gin.H{"success": false, "error": "forbidden"})
			return
		}
		if target.Role != "employee" {
			c.JSON(http.StatusForbidden, gin.H{"success": false, "error": "can only edit employees in your department"})
			return
		}
		if req.Role != "" && req.Role != "employee" {
			c.JSON(http.StatusForbidden, gin.H{"success": false, "error": "can only set role=employee"})
			return
		}
		if req.DepartmentID != nil && me.DepartmentID != nil && *req.DepartmentID != *me.DepartmentID {
			c.JSON(http.StatusForbidden, gin.H{"success": false, "error": "cannot move employee to another department"})
			return
		}
		req.Role = "employee"
		req.DepartmentID = me.DepartmentID
	}

	// If status or other enum fields are empty, fall back to current values.
	if req.Status == "" || req.Role == "" || req.Gender == "" {
		current, fetchErr := h.employeeService.GetByID(c.Request.Context(), id)
		if fetchErr != nil {
			c.JSON(http.StatusNotFound, gin.H{"success": false, "error": fetchErr.Error()})
			return
		}
		if req.Status == "" {
			req.Status = current.Status
		}
		if req.Role == "" {
			req.Role = current.Role
		}
		if req.Gender == "" {
			req.Gender = current.Gender
		}
	}

	emp := &models.Employee{
		ID:                 id,
		FirstName:          req.FirstName,
		LastName:           req.LastName,
		Gender:             req.Gender,
		Phone:              req.Phone,
		Email:              req.Email,
		Role:               req.Role,
		DepartmentID:       req.DepartmentID,
		Position:           req.Position,
		DefaultShiftID:     req.DefaultShiftID,
		WeeklyOffDays:      req.WeeklyOffDays,
		CanCoverNightShift: req.CanCoverNightShift,
		CanManageHelpDocs:  req.CanManageHelpDocs,
		Status:             req.Status,
		ProfileImage:       req.ProfileImage,
		SecondaryPhone:     req.SecondaryPhone,
		SecondaryEmail:     req.SecondaryEmail,
	}

	if err := h.employeeService.UpdateEmployee(c.Request.Context(), emp); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "data": emp})
}

type updateStatusRequest struct {
	Status string `json:"status" binding:"required"`
}

// UpdateStatus updates an employee's status.
func (h *EmployeeHandler) UpdateStatus(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "invalid employee ID"})
		return
	}

	var req updateStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "invalid request: " + err.Error()})
		return
	}

	if err := h.employeeService.UpdateStatus(c.Request.Context(), id, req.Status); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "data": gin.H{"message": "status updated"}})
}

// Delete deletes an employee.
func (h *EmployeeHandler) Delete(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "invalid employee ID"})
		return
	}

	if err := h.employeeService.DeleteEmployee(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "data": gin.H{"message": "employee deleted"}})
}

type updatePasswordRequest struct {
	OldPassword string `json:"old_password"`
	NewPassword string `json:"new_password" binding:"required,min=8"`
}

// UpdatePassword updates an employee's password.
func (h *EmployeeHandler) UpdatePassword(c *gin.Context) {
	targetID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "invalid employee ID"})
		return
	}

	var req updatePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "invalid request: " + err.Error()})
		return
	}

	roleAny, _ := c.Get("role")
	role, _ := roleAny.(string)
	isAdmin := role == "admin"

	requesterStr, _ := c.Get("employee_id")
	requesterID, _ := uuid.Parse(requesterStr.(string))

	// If not admin, you can only change your own password
	if !isAdmin && targetID != requesterID {
		c.JSON(http.StatusForbidden, gin.H{"success": false, "error": "you can only change your own password"})
		return
	}

	if err := h.employeeService.ChangePassword(c.Request.Context(), targetID, req.OldPassword, req.NewPassword, isAdmin); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "data": gin.H{"message": "password updated successfully"}})
}
