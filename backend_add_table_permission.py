import re

# 1. Update employee_repository.go
with open('internal/repository/employee_repository.go', 'r') as f:
    repo = f.read()

if "UpdateTablePermission(ctx context.Context, id uuid.UUID, canCreate bool) error" not in repo:
    repo = repo.replace(
        'UpdateHelpPermission(ctx context.Context, id uuid.UUID, canManage bool) error',
        'UpdateHelpPermission(ctx context.Context, id uuid.UUID, canManage bool) error\n\tUpdateTablePermission(ctx context.Context, id uuid.UUID, canCreate bool) error'
    )

repo_func = """
func (r *employeeRepo) UpdateTablePermission(ctx context.Context, id uuid.UUID, canCreate bool) error {
\tquery := `UPDATE employees SET can_create_tables = $2, updated_at = CURRENT_TIMESTAMP WHERE id = $1`
\t_, err := r.db.Exec(ctx, query, id, canCreate)
\treturn err
}
"""
if "func (r *employeeRepo) UpdateTablePermission" not in repo:
    repo += "\n" + repo_func

with open('internal/repository/employee_repository.go', 'w') as f:
    f.write(repo)

# 2. Update employee_service.go
with open('internal/service/employee_service.go', 'r') as f:
    svc = f.read()

svc_func = """
func (s *EmployeeService) UpdateTablePermission(ctx context.Context, id uuid.UUID, canCreate bool) error {
\treturn s.employeeRepo.UpdateTablePermission(ctx, id, canCreate)
}
"""
if "func (s *EmployeeService) UpdateTablePermission" not in svc:
    svc += "\n" + svc_func

with open('internal/service/employee_service.go', 'w') as f:
    f.write(svc)

# 3. Update employee_handler.go
with open('cmd/api/handlers/employee_handler.go', 'r') as f:
    handler = f.read()

handler_func = """
type updateTablePermissionRequest struct {
\tCanCreateTables bool `json:"can_create_tables"`
}

func (h *EmployeeHandler) UpdateTablePermission(c *gin.Context) {
\tid, err := uuid.Parse(c.Param("id"))
\tif err != nil {
\t\tc.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "invalid employee ID"})
\t\treturn
\t}

\tvar req updateTablePermissionRequest
\tif err := c.ShouldBindJSON(&req); err != nil {
\t\tc.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "invalid request: " + err.Error()})
\t\treturn
\t}

\troleAny, _ := c.Get("role")
\trole, _ := roleAny.(string)

\tif role != "admin" && role != "manager" {
\t\tc.JSON(http.StatusForbidden, gin.H{"success": false, "error": "forbidden"})
\t\treturn
\t}

\tif role == "manager" {
\t\trequesterStr, _ := c.Get("employee_id")
\t\trequesterID, _ := uuid.Parse(requesterStr.(string))
\t\tme, _ := h.employeeService.GetByID(c.Request.Context(), requesterID)
\t\ttarget, _ := h.employeeService.GetByID(c.Request.Context(), id)
\t\tif me.DepartmentID == nil || target.DepartmentID == nil || *me.DepartmentID != *target.DepartmentID {
\t\t\tc.JSON(http.StatusForbidden, gin.H{"success": false, "error": "forbidden"})
\t\t\treturn
\t\t}
\t}

\tif err := h.employeeService.UpdateTablePermission(c.Request.Context(), id, req.CanCreateTables); err != nil {
\t\tc.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
\t\treturn
\t}

\tc.JSON(http.StatusOK, gin.H{"success": true})
}
"""
if "func (h *EmployeeHandler) UpdateTablePermission" not in handler:
    handler += "\n" + handler_func

with open('cmd/api/handlers/employee_handler.go', 'w') as f:
    f.write(handler)

# 4. Update router.go
with open('cmd/api/router.go', 'r') as f:
    router = f.read()

if "employees.PUT(\"/:id/table-permission" not in router:
    router = router.replace(
        'employees.PUT("/:id/help-permission", empH.UpdateHelpPermission)',
        'employees.PUT("/:id/help-permission", empH.UpdateHelpPermission)\n\t\t\temployees.PUT("/:id/table-permission", empH.UpdateTablePermission)'
    )

with open('cmd/api/router.go', 'w') as f:
    f.write(router)
