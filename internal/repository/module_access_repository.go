package repository

import (
	"context"

	"github.com/google/uuid"

	"shiftmaster-backend/internal/models"
	"shiftmaster-backend/pkg/database"
)

type ModuleAccessRepository interface {
	// Department access
	AddDepartmentAccess(ctx context.Context, moduleName string, departmentID uuid.UUID, grantedBy *uuid.UUID) error
	RemoveDepartmentAccess(ctx context.Context, moduleName string, departmentID uuid.UUID) error
	GetDepartmentsWithAccess(ctx context.Context, moduleName string) ([]uuid.UUID, error)

	// Employee exclusions
	AddEmployeeExclusion(ctx context.Context, moduleName string, employeeID uuid.UUID, excludedBy *uuid.UUID) error
	RemoveEmployeeExclusion(ctx context.Context, moduleName string, employeeID uuid.UUID) error
	GetExcludedEmployees(ctx context.Context, moduleName string, departmentID *uuid.UUID) ([]uuid.UUID, error)

	// Employee specific
	GetEmployeeAllowedModules(ctx context.Context, employeeID uuid.UUID, departmentID *uuid.UUID) ([]string, error)
}

type moduleAccessRepo struct {
	db *database.DB
}

func NewModuleAccessRepository(db *database.DB) ModuleAccessRepository {
	return &moduleAccessRepo{db: db}
}

func (r *moduleAccessRepo) AddDepartmentAccess(ctx context.Context, moduleName string, departmentID uuid.UUID, grantedBy *uuid.UUID) error {
	query := `
		INSERT INTO module_departments (module_name, department_id, granted_by)
		VALUES ($1, $2, $3)
		ON CONFLICT (module_name, department_id) DO NOTHING
	`
	_, err := r.db.Exec(ctx, query, moduleName, departmentID, grantedBy)
	return err
}

func (r *moduleAccessRepo) RemoveDepartmentAccess(ctx context.Context, moduleName string, departmentID uuid.UUID) error {
	query := `DELETE FROM module_departments WHERE module_name = $1 AND department_id = $2`
	_, err := r.db.Exec(ctx, query, moduleName, departmentID)
	return err
}

func (r *moduleAccessRepo) GetDepartmentsWithAccess(ctx context.Context, moduleName string) ([]uuid.UUID, error) {
	query := `SELECT department_id FROM module_departments WHERE module_name = $1`
	rows, err := r.db.Query(ctx, query, moduleName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var deps []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err == nil {
			deps = append(deps, id)
		}
	}
	return deps, nil
}

func (r *moduleAccessRepo) AddEmployeeExclusion(ctx context.Context, moduleName string, employeeID uuid.UUID, excludedBy *uuid.UUID) error {
	query := `
		INSERT INTO module_exclusions (module_name, employee_id, excluded_by)
		VALUES ($1, $2, $3)
		ON CONFLICT (module_name, employee_id) DO NOTHING
	`
	_, err := r.db.Exec(ctx, query, moduleName, employeeID, excludedBy)
	return err
}

func (r *moduleAccessRepo) RemoveEmployeeExclusion(ctx context.Context, moduleName string, employeeID uuid.UUID) error {
	query := `DELETE FROM module_exclusions WHERE module_name = $1 AND employee_id = $2`
	_, err := r.db.Exec(ctx, query, moduleName, employeeID)
	return err
}

func (r *moduleAccessRepo) GetExcludedEmployees(ctx context.Context, moduleName string, departmentID *uuid.UUID) ([]uuid.UUID, error) {
	query := `
		SELECT me.employee_id 
		FROM module_exclusions me
		JOIN employees e ON me.employee_id = e.id
		WHERE me.module_name = $1
	`
	var args []interface{}
	args = append(args, moduleName)

	if departmentID != nil {
		query += ` AND e.department_id = $2`
		args = append(args, *departmentID)
	}

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var emps []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err == nil {
			emps = append(emps, id)
		}
	}
	return emps, nil
}

func (r *moduleAccessRepo) GetEmployeeAllowedModules(ctx context.Context, employeeID uuid.UUID, departmentID *uuid.UUID) ([]string, error) {
	if departmentID == nil {
		return []string{}, nil
	}

	query := `
		SELECT md.module_name
		FROM module_departments md
		LEFT JOIN module_exclusions me ON md.module_name = me.module_name AND me.employee_id = $1
		WHERE md.department_id = $2 AND me.employee_id IS NULL
	`
	rows, err := r.db.Query(ctx, query, employeeID, *departmentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var modules []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err == nil {
			modules = append(modules, name)
		}
	}
	return modules, nil
}
