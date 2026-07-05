package handlers

import (
	"fmt"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"time"

	"shiftmaster-backend/internal/models"
	"shiftmaster-backend/internal/repository"
	"shiftmaster-backend/internal/service"
)

// EmployeeHandler handles employee CRUD endpoints.
type EmployeeHandler struct {
	employeeService  *service.EmployeeService
	leaveBalanceRepo repository.LeaveBalanceRepository
	taskRepo         repository.TaskRepository
	leaveRepo        repository.LeaveRepository
	deptRepo         repository.DepartmentRepository
}

func NewEmployeeHandler(
	empSvc *service.EmployeeService,
	leaveBalanceRepo repository.LeaveBalanceRepository,
	taskRepo repository.TaskRepository,
	leaveRepo repository.LeaveRepository,
	deptRepo repository.DepartmentRepository,
) *EmployeeHandler {
	return &EmployeeHandler{
		employeeService:  empSvc,
		leaveBalanceRepo: leaveBalanceRepo,
		taskRepo:         taskRepo,
		leaveRepo:        leaveRepo,
		deptRepo:         deptRepo,
	}
}

// List returns employees with optional filters.
func (h *EmployeeHandler) List(c *gin.Context) {
	ctx := c.Request.Context()
	var employees []models.Employee
	var err error

	roleAny, _ := c.Get("role")
	role, _ := roleAny.(string)

	// Scope: strictly isolate non-admins to their own department.
	if role != "admin" {
		targetDeptID := getDepartmentID(c)
		var deptEmployees []models.Employee

		if targetDeptID == nil {
			// If no target department is specified, just fetch all active employees (or all employees)
			// Team Leaders might not have a department set in the UI context
			deptEmployees, err = h.employeeService.GetActive(ctx)
		} else {
			deptEmployees, err = h.employeeService.GetByDepartment(ctx, *targetDeptID)
		}

		if err == nil {
			// Role-based visibility filtering:
			filtered := make([]models.Employee, 0, len(deptEmployees))
			for _, e := range deptEmployees {
				if role == "manager" && (e.Role == "employee" || e.Role == "team_leader" || e.Role == "manager") {
					filtered = append(filtered, e)
				} else if role == "team_leader" && (e.Role == "employee" || e.Role == "team_leader") {
					filtered = append(filtered, e)
				} else if role == "employee" {
					// Employees can see other employees/TLs (e.g., for swaps)
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

		// Filter by contextual department if one is selected by an Admin explicitly from header.
		// We only filter if the header is present, to avoid filtering out employees with no department if Admin has a default department.
		if c.GetHeader("X-Department-ID") != "" {
			targetDeptID := getDepartmentID(c)
			if targetDeptID != nil {
				filtered := make([]models.Employee, 0)
				for _, e := range employees {
					if e.DepartmentID != nil && *e.DepartmentID == *targetDeptID {
						filtered = append(filtered, e)
					}
				}
				employees = filtered
			}
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


			scopeDeptID := getDepartmentID(c)

			if scopeDeptID == nil || emp.DepartmentID == nil || *scopeDeptID != *emp.DepartmentID {
				c.JSON(http.StatusForbidden, gin.H{"success": false, "error": "forbidden"})
				return
			}
			// Managers can view employees + team_leaders; team_leaders can only view employees.
			if role == "manager" && emp.Role != "employee" && emp.Role != "team_leader" {
				c.JSON(http.StatusForbidden, gin.H{"success": false, "error": "forbidden"})
				return
			}
			if role == "team_leader" && emp.Role != "employee" && emp.Role != "team_leader" {
				c.JSON(http.StatusForbidden, gin.H{"success": false, "error": "forbidden"})
				return
			}
			// employees can view anyone in their own department (e.g., for swaps)
		}
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "data": emp})
}

type createEmployeeRequest struct {
	EmployeeCode         string     `json:"employee_code"`
	FirstName            string     `json:"first_name" binding:"required"`
	LastName             string     `json:"last_name" binding:"required"`
	Gender               string     `json:"gender" binding:"required"`
	Phone                *string    `json:"phone"`
	Email                string     `json:"email" binding:"required,email"`
	Password             string     `json:"password" binding:"required,min=8"`
	HireDate             string     `json:"hire_date" binding:"required"`
	Role                 string     `json:"role" binding:"required"`
	DepartmentID         *uuid.UUID `json:"department_id"`
	Position             *string    `json:"position"`
	DefaultShiftID       *uuid.UUID `json:"default_shift_id"`
	WeeklyOffDays        int        `json:"weekly_off_days"`
	CanCoverNightShift   bool       `json:"can_cover_night_shift"`
	CanManageHelpDocs    bool       `json:"can_manage_help_docs"`
	CanPostAnnouncements bool       `json:"can_post_announcements"`
	CanManageFiberxData  bool       `json:"can_manage_fiberx_data"`
	Status               string     `json:"status"`
	ProfileImage         *string    `json:"profile_image"`
	SecondaryPhone       *string    `json:"secondary_phone"`
	SecondaryEmail       *string    `json:"secondary_email"`
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
		if creatorRoleStr == "manager" {
			if req.DepartmentID == nil {
				if creator.DepartmentID != nil {
					req.DepartmentID = creator.DepartmentID
				} else {
					c.JSON(http.StatusForbidden, gin.H{"success": false, "error": "managers must specify a department to create an employee in"})
					return
				}
			}
			
			managedDepts, err := h.deptRepo.GetByManagerID(c.Request.Context(), creatorID)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
				return
			}
			isManaging := false
			for _, d := range managedDepts {
				if d.ID == *req.DepartmentID {
					isManaging = true
					break
				}
			}
			if !isManaging && (creator.DepartmentID == nil || *creator.DepartmentID != *req.DepartmentID) {
				c.JSON(http.StatusForbidden, gin.H{"success": false, "error": "cannot create employees in a department you don't manage"})
				return
			}
		} else {
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
	}

	hireDate, err := parseTime(req.HireDate)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "invalid hire_date format"})
		return
	}

	createdByStr, _ := c.Get("employee_id")
	createdBy, _ := uuid.Parse(createdByStr.(string))

	emp := &models.Employee{
		EmployeeCode:         req.EmployeeCode,
		FirstName:            req.FirstName,
		LastName:             req.LastName,
		Gender:               req.Gender,
		Phone:                req.Phone,
		Email:                req.Email,
		HireDate:             hireDate,
		Role:                 req.Role,
		DepartmentID:         req.DepartmentID,
		Position:             req.Position,
		DefaultShiftID:       req.DefaultShiftID,
		WeeklyOffDays:        req.WeeklyOffDays,
		CanCoverNightShift:   req.CanCoverNightShift,
		CanManageHelpDocs:    req.CanManageHelpDocs,
		CanPostAnnouncements: req.CanPostAnnouncements,
		CanManageFiberxData:  req.CanManageFiberxData,
		Status:               req.Status,
		ProfileImage:         req.ProfileImage,
		SecondaryPhone:       req.SecondaryPhone,
		SecondaryEmail:       req.SecondaryEmail,
		CreatedBy:            &createdBy,
	}

	if err := h.employeeService.CreateEmployee(c.Request.Context(), emp, req.Password); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"success": true, "data": emp})
}

type updateEmployeeRequest struct {
	FirstName            string     `json:"first_name"`
	LastName             string     `json:"last_name"`
	Gender               string     `json:"gender"`
	Phone                *string    `json:"phone"`
	Email                string     `json:"email"`
	Role                 string     `json:"role"`
	DepartmentID         *uuid.UUID `json:"department_id"`
	Position             *string    `json:"position"`
	DefaultShiftID       *uuid.UUID `json:"default_shift_id"`
	WeeklyOffDays        int        `json:"weekly_off_days"`
	CanCoverNightShift   bool       `json:"can_cover_night_shift"`
	CanManageHelpDocs    *bool      `json:"can_manage_help_docs"`
	CanPostAnnouncements *bool      `json:"can_post_announcements"`
	CanManageFiberxData  *bool      `json:"can_manage_fiberx_data"`
	Status               string     `json:"status"`
	ProfileImage         *string    `json:"profile_image"`
	SecondaryPhone       *string    `json:"secondary_phone"`
	SecondaryEmail       *string    `json:"secondary_email"`
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

		if role == "manager" {
			managedDepts, _ := h.deptRepo.GetByManagerID(c.Request.Context(), requesterID)
			isManagingTargetDept := false
			if target.DepartmentID != nil {
				for _, d := range managedDepts {
					if d.ID == *target.DepartmentID {
						isManagingTargetDept = true
						break
					}
				}
			}
			
			if !isManagingTargetDept && (me.DepartmentID == nil || target.DepartmentID == nil || *me.DepartmentID != *target.DepartmentID) {
				c.JSON(http.StatusForbidden, gin.H{"success": false, "error": "forbidden: you do not manage this employee's department"})
				return
			}
			
			if target.Role == "admin" || target.Role == "manager" {
				c.JSON(http.StatusForbidden, gin.H{"success": false, "error": "managers cannot edit admin or manager accounts"})
				return
			}
			
			if req.Role != "" && req.Role != "employee" && req.Role != "team_leader" {
				c.JSON(http.StatusForbidden, gin.H{"success": false, "error": "can only set role to employee or team_leader"})
				return
			}

			if req.DepartmentID != nil && target.DepartmentID != nil && *req.DepartmentID != *target.DepartmentID {
				isManagingNewDept := false
				for _, d := range managedDepts {
					if d.ID == *req.DepartmentID {
						isManagingNewDept = true
						break
					}
				}
				if !isManagingNewDept && (me.DepartmentID == nil || *req.DepartmentID != *me.DepartmentID) {
					c.JSON(http.StatusForbidden, gin.H{"success": false, "error": "cannot move employee to a department you don't manage"})
					return
				}
			}
		} else {
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
	}

	current, fetchErr := h.employeeService.GetByID(c.Request.Context(), id)
	if fetchErr != nil {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "error": fetchErr.Error()})
		return
	}

	// If status or other enum fields are empty, fall back to current values.
	if req.Status == "" || req.Role == "" || req.Gender == "" {
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
		// Preserve permissions and preferences since they are managed by separate endpoints
		// or omitted from the update payload
		CanManageHelpDocs:  func() bool { if req.CanManageHelpDocs != nil { return *req.CanManageHelpDocs }; return current.CanManageHelpDocs }(),
		CanPostAnnouncements: func() bool { if req.CanPostAnnouncements != nil { return *req.CanPostAnnouncements }; return current.CanPostAnnouncements }(),
		CanManageFiberxData:  func() bool { if req.CanManageFiberxData != nil { return *req.CanManageFiberxData }; return current.CanManageFiberxData }(),
		CanCreateTables:    current.CanCreateTables,
		UIPreferences:      current.UIPreferences,
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

	roleAny, _ := c.Get("role")
	role, _ := roleAny.(string)
	
	if role == "team_leader" || role == "manager" {
		requesterStr, _ := c.Get("employee_id")
		requesterID, _ := uuid.Parse(requesterStr.(string))

		target, targetErr := h.employeeService.GetByID(c.Request.Context(), id)
		if targetErr != nil {
			c.JSON(http.StatusNotFound, gin.H{"success": false, "error": "employee not found"})
			return
		}

		if role == "manager" {
			managedDepts, _ := h.deptRepo.GetByManagerID(c.Request.Context(), requesterID)
			isManagingTarget := false
			if target.DepartmentID != nil {
				for _, d := range managedDepts {
					if d.ID == *target.DepartmentID {
						isManagingTarget = true
						break
					}
				}
			}
			
			me, _ := h.employeeService.GetByID(c.Request.Context(), requesterID)
			if !isManagingTarget && (me.DepartmentID == nil || target.DepartmentID == nil || *me.DepartmentID != *target.DepartmentID) {
				c.JSON(http.StatusForbidden, gin.H{"success": false, "error": "forbidden: you do not manage this employee's department"})
				return
			}
			if target.Role == "admin" || target.Role == "manager" {
				c.JSON(http.StatusForbidden, gin.H{"success": false, "error": "managers cannot update status for admin or manager accounts"})
				return
			}
		} else {
			me, _ := h.employeeService.GetByID(c.Request.Context(), requesterID)
			if me.DepartmentID == nil || target.DepartmentID == nil || *me.DepartmentID != *target.DepartmentID {
				c.JSON(http.StatusForbidden, gin.H{"success": false, "error": "forbidden: employee is not in your department"})
				return
			}
			if target.Role != "employee" {
				c.JSON(http.StatusForbidden, gin.H{"success": false, "error": "team leaders can only update status for employees"})
				return
			}
		}
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

	roleAny, _ := c.Get("role")
	role, _ := roleAny.(string)

	if role == "team_leader" || role == "manager" {
		requesterStr, _ := c.Get("employee_id")
		requesterID, _ := uuid.Parse(requesterStr.(string))

		target, targetErr := h.employeeService.GetByID(c.Request.Context(), id)
		if targetErr != nil {
			c.JSON(http.StatusNotFound, gin.H{"success": false, "error": "employee not found"})
			return
		}

		if role == "manager" {
			managedDepts, _ := h.deptRepo.GetByManagerID(c.Request.Context(), requesterID)
			isManagingTarget := false
			if target.DepartmentID != nil {
				for _, d := range managedDepts {
					if d.ID == *target.DepartmentID {
						isManagingTarget = true
						break
					}
				}
			}
			
			me, _ := h.employeeService.GetByID(c.Request.Context(), requesterID)
			if !isManagingTarget && (me.DepartmentID == nil || target.DepartmentID == nil || *me.DepartmentID != *target.DepartmentID) {
				c.JSON(http.StatusForbidden, gin.H{"success": false, "error": "forbidden: you do not manage this employee's department"})
				return
			}
			if target.Role == "admin" || target.Role == "manager" {
				c.JSON(http.StatusForbidden, gin.H{"success": false, "error": "managers cannot delete admin or manager accounts"})
				return
			}
		} else {
			me, _ := h.employeeService.GetByID(c.Request.Context(), requesterID)
			if me.DepartmentID == nil || target.DepartmentID == nil || *me.DepartmentID != *target.DepartmentID {
				c.JSON(http.StatusForbidden, gin.H{"success": false, "error": "forbidden: employee is not in your department"})
				return
			}
			if target.Role != "employee" {
				c.JSON(http.StatusForbidden, gin.H{"success": false, "error": "team leaders can only delete employees"})
				return
			}
		}
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


type updateHelpPermissionRequest struct {
	CanManageHelpDocs bool `json:"can_manage_help_docs"`
}


type updateFiberxPermissionRequest struct {
	CanManageFiberxData bool `json:"can_manage_fiberx_data"`
}

func (h *EmployeeHandler) UpdateFiberxPermission(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "invalid employee ID"})
		return
	}

	var req updateFiberxPermissionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "invalid request: " + err.Error()})
		return
	}

	roleAny, _ := c.Get("role")
	role, _ := roleAny.(string)

	// Only admins, managers and team_leaders can do this
	if role != "admin" && role != "manager" && role != "team_leader" {
		c.JSON(http.StatusForbidden, gin.H{"success": false, "error": "forbidden"})
		return
	}

	if role == "manager" || role == "team_leader" {
		requesterStr, _ := c.Get("employee_id")
		requesterID, _ := uuid.Parse(requesterStr.(string))
		me, _ := h.employeeService.GetByID(c.Request.Context(), requesterID)
		target, _ := h.employeeService.GetByID(c.Request.Context(), id)
		if me.DepartmentID == nil || target.DepartmentID == nil || *me.DepartmentID != *target.DepartmentID {
			c.JSON(http.StatusForbidden, gin.H{"success": false, "error": "forbidden"})
			return
		}
	}

	if err := h.employeeService.UpdateFiberxPermission(c.Request.Context(), id, req.CanManageFiberxData); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}

func (h *EmployeeHandler) UpdateHelpPermission(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "invalid employee ID"})
		return
	}

	var req updateHelpPermissionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "invalid request: " + err.Error()})
		return
	}

	roleAny, _ := c.Get("role")
	role, _ := roleAny.(string)

	// Only admins and managers can do this
	if role != "admin" && role != "manager" {
		c.JSON(http.StatusForbidden, gin.H{"success": false, "error": "forbidden"})
		return
	}

	if role == "manager" {
		requesterStr, _ := c.Get("employee_id")
		requesterID, _ := uuid.Parse(requesterStr.(string))
		me, _ := h.employeeService.GetByID(c.Request.Context(), requesterID)
		target, _ := h.employeeService.GetByID(c.Request.Context(), id)
		if me.DepartmentID == nil || target.DepartmentID == nil || *me.DepartmentID != *target.DepartmentID {
			c.JSON(http.StatusForbidden, gin.H{"success": false, "error": "forbidden"})
			return
		}
	}

	if err := h.employeeService.UpdateHelpPermission(c.Request.Context(), id, req.CanManageHelpDocs); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}

type updateAnnouncementPermissionRequest struct {
	CanPostAnnouncements bool `json:"can_post_announcements"`
}

func (h *EmployeeHandler) UpdateAnnouncementPermission(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "invalid employee ID"})
		return
	}

	var req updateAnnouncementPermissionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "invalid request: " + err.Error()})
		return
	}

	roleAny, _ := c.Get("role")
	role, _ := roleAny.(string)

	// Only admins and managers can do this
	if role != "admin" && role != "manager" {
		c.JSON(http.StatusForbidden, gin.H{"success": false, "error": "forbidden"})
		return
	}

	if role == "manager" {
		requesterStr, _ := c.Get("employee_id")
		requesterID, _ := uuid.Parse(requesterStr.(string))
		me, _ := h.employeeService.GetByID(c.Request.Context(), requesterID)
		target, _ := h.employeeService.GetByID(c.Request.Context(), id)
		if me.DepartmentID == nil || target.DepartmentID == nil || *me.DepartmentID != *target.DepartmentID {
			c.JSON(http.StatusForbidden, gin.H{"success": false, "error": "forbidden"})
			return
		}
	}

	if err := h.employeeService.UpdateAnnouncementPermission(c.Request.Context(), id, req.CanPostAnnouncements); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}


type updateTablePermissionRequest struct {
	CanCreateTables bool `json:"can_create_tables"`
}

func (h *EmployeeHandler) UpdateTablePermission(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "invalid employee ID"})
		return
	}

	var req updateTablePermissionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "invalid request: " + err.Error()})
		return
	}

	roleAny, _ := c.Get("role")
	role, _ := roleAny.(string)

	if role != "admin" && role != "manager" {
		c.JSON(http.StatusForbidden, gin.H{"success": false, "error": "forbidden"})
		return
	}

	if role == "manager" {
		requesterStr, _ := c.Get("employee_id")
		requesterID, _ := uuid.Parse(requesterStr.(string))
		me, _ := h.employeeService.GetByID(c.Request.Context(), requesterID)
		target, _ := h.employeeService.GetByID(c.Request.Context(), id)
		if me.DepartmentID == nil || target.DepartmentID == nil || *me.DepartmentID != *target.DepartmentID {
			c.JSON(http.StatusForbidden, gin.H{"success": false, "error": "forbidden"})
			return
		}
	}

	if err := h.employeeService.UpdateTablePermission(c.Request.Context(), id, req.CanCreateTables); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}

type updatePreferencesRequest struct {
	UIPreferences map[string]interface{} `json:"ui_preferences"`
}

func (h *EmployeeHandler) UpdatePreferences(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "invalid employee ID"})
		return
	}

	var req updatePreferencesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "invalid request: " + err.Error()})
		return
	}

	roleAny, _ := c.Get("role")
	role, _ := roleAny.(string)
	requesterStr, _ := c.Get("employee_id")
	requesterID, _ := uuid.Parse(requesterStr.(string))

	if role != "admin" && requesterID != id {
		c.JSON(http.StatusForbidden, gin.H{"success": false, "error": "you can only update your own preferences"})
		return
	}

	if err := h.employeeService.UpdatePreferences(c.Request.Context(), id, req.UIPreferences); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}

func parseTimeStr(t string) (time.Time, error) {
	if len(t) > 5 {
		t = t[:5]
	}
	return time.Parse("15:04", t)
}

// GetProfileStats returns statistics and leave balances for the authenticated employee
func (h *EmployeeHandler) GetProfileStats(c *gin.Context) {
	ctx := c.Request.Context()
	requesterStr, _ := c.Get("employee_id")
	empID, err := uuid.Parse(requesterStr.(string))
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"success": false, "error": "invalid user"})
		return
	}

	year := time.Now().Year()

	// 1. Get Leave Balances
	balances, err := h.leaveBalanceRepo.GetByEmployeeAndYear(ctx, empID, year)
	if err != nil {
		balances = []models.EmployeeLeaveBalance{}
	}

	// 2. Get Task Stats
	// Count completed tasks vs active tasks assigned to this employee
	tasks, err := h.taskRepo.GetTasksByAssignee(ctx, empID)
	completedTasks := 0
	activeTasks := 0
	if err == nil {
		for _, t := range tasks {
			if t.Status == "completed" {
				completedTasks++
			} else if t.Status != "cancelled" {
				activeTasks++
			}
		}
	}

	// 3. Get total approved leaves count and calculate pending amounts
	totalLeavesTaken := 0
	totalHourlyLeavesTaken := 0
	leaves, err := h.leaveRepo.GetByEmployee(ctx, empID)
	if err == nil {
		for _, l := range leaves {
			if l.Status == "approved" || l.Status == "approved_by_manager" {
				if l.StartTime != nil && l.EndTime != nil {
					totalHourlyLeavesTaken++
				} else {
					totalLeavesTaken++
				}
			} else if l.Status == "pending" || l.Status == "approved_by_team_leader" {
				// Calculate pending amount and add to the corresponding balance
				for i, b := range balances {
					if b.LeaveTypeID == l.LeaveTypeID {
						y := l.StartDate.Year()
						m := 0
						if b.ResetCycle == "monthly" {
							m = int(l.StartDate.Month())
						}
						if y == b.Year && m == b.Month {
							if b.Unit == "hours" && l.StartTime != nil && l.EndTime != nil {
								st, _ := parseTimeStr(*l.StartTime)
								en, _ := parseTimeStr(*l.EndTime)
								balances[i].PendingAmount += en.Sub(st).Hours()
							} else if b.Unit != "hours" {
								for d := l.StartDate.UTC().Truncate(24 * time.Hour); !d.After(l.EndDate.UTC().Truncate(24 * time.Hour)); d = d.AddDate(0, 0, 1) {
									balances[i].PendingAmount += 1.0
								}
							}
						}
					}
				}
			}
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"leave_balances":            balances,
			"completed_tasks":           completedTasks,
			"active_tasks":              activeTasks,
			"total_leaves_taken":        totalLeavesTaken,
			"total_hourly_leaves_taken": totalHourlyLeavesTaken,
			"worked_hours":              0, // Could calculate from schedule/check-in times if implemented
		},
	})
}

// UploadProfilePicture handles uploading and updating an employee's profile picture.
func (h *EmployeeHandler) UploadProfilePicture(c *gin.Context) {
	ctx := c.Request.Context()
	requesterStr, _ := c.Get("employee_id")
	requesterID, _ := uuid.Parse(requesterStr.(string))

	file, err := c.FormFile("profile_picture")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "file missing"})
		return
	}

	// Make uploads directory if it doesn't exist
	uploadDir := "./uploads/profiles"
	if err := os.MkdirAll(uploadDir, os.ModePerm); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "could not create upload directory"})
		return
	}

	// Generate a unique file name
	fileName := fmt.Sprintf("%s_%d_%s", requesterID.String(), time.Now().Unix(), file.Filename)
	filePath := fmt.Sprintf("%s/%s", uploadDir, fileName)

	if err := c.SaveUploadedFile(file, filePath); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "failed to save file"})
		return
	}

	// Update the profile image path in the database. 
	// We'll store it as relative URL including /api prefix to be served by the static route.
	publicURL := fmt.Sprintf("/api/uploads/profiles/%s", fileName)
	if err := h.employeeService.UpdateProfileImage(ctx, requesterID, publicURL); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"profile_image": publicURL,
		},
	})
}
