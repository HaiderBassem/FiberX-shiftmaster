import re

# 1. Update employee_repository.go
with open('internal/repository/employee_repository.go', 'r') as f:
    repo = f.read()

if "UpdateHelpPermission(ctx context.Context, id uuid.UUID, canManage bool) error" not in repo:
    repo = repo.replace(
        'ForceDelete(ctx context.Context, id uuid.UUID) error',
        'ForceDelete(ctx context.Context, id uuid.UUID) error\n\tUpdateHelpPermission(ctx context.Context, id uuid.UUID, canManage bool) error'
    )

repo_func = """
func (r *employeeRepo) UpdateHelpPermission(ctx context.Context, id uuid.UUID, canManage bool) error {
\tquery := `UPDATE employees SET can_manage_help_docs = $2, updated_at = CURRENT_TIMESTAMP WHERE id = $1`
\t_, err := r.db.Exec(ctx, query, id, canManage)
\treturn err
}
"""
repo = re.sub(r'func \(r \*EmployeeRepository\) UpdateHelpPermission.*?\n}', '', repo, flags=re.DOTALL)
if "func (r *employeeRepo) UpdateHelpPermission" not in repo:
    repo += "\n" + repo_func
    
with open('internal/repository/employee_repository.go', 'w') as f:
    f.write(repo)

# 2. router.go
with open('cmd/api/router.go', 'r') as f:
    router = f.read()

if "employees.PUT(\"/:id/help-permission" not in router:
    router = router.replace(
        'employees.PUT("/:id/password", empH.UpdatePassword)',
        'employees.PUT("/:id/password", empH.UpdatePassword)\n\t\t\temployees.PUT("/:id/help-permission", empH.UpdateHelpPermission)'
    )
    with open('cmd/api/router.go', 'w') as f:
        f.write(router)
