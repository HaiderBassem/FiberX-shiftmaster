import sys

with open("cmd/api/handlers/employee_handler.go", "r") as f:
    content = f.read()

new_func = """
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
"""

if "UpdateFiberxPermission" not in content:
    with open("cmd/api/handlers/employee_handler.go", "w") as f:
        f.write(content.replace("func (h *EmployeeHandler) UpdateHelpPermission(c *gin.Context) {", new_func + "\nfunc (h *EmployeeHandler) UpdateHelpPermission(c *gin.Context) {"))
