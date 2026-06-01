package repository

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"shiftmaster-backend/internal/models"
	"shiftmaster-backend/pkg/database"
)

// DepartmentRepository defines the interface for department data access.
type DepartmentRepository interface {
	GetByID(ctx context.Context, id uuid.UUID) (*models.Department, error)
	GetByCode(ctx context.Context, code string) (*models.Department, error)
	GetAll(ctx context.Context) ([]models.Department, error)
	Create(ctx context.Context, dept *models.Department) error
	Update(ctx context.Context, dept *models.Department) error
	Delete(ctx context.Context, id uuid.UUID) error

	// Multi-manager operations
	AddManager(ctx context.Context, departmentID, managerID uuid.UUID) error
	RemoveManager(ctx context.Context, departmentID, managerID uuid.UUID) error
	SetManagers(ctx context.Context, departmentID uuid.UUID, managerIDs []uuid.UUID) error
	GetManagers(ctx context.Context, departmentID uuid.UUID) ([]uuid.UUID, error)

	// GetByManagerID returns all departments that a given manager is assigned to.
	GetByManagerID(ctx context.Context, managerID uuid.UUID) ([]models.Department, error)
}

type departmentRepo struct {
	db *database.DB
}

func NewDepartmentRepository(db *database.DB) DepartmentRepository {
	return &departmentRepo{db: db}
}

// loadManagerIDs fetches manager UUIDs for a department from the junction table.
func (r *departmentRepo) loadManagerIDs(ctx context.Context, departmentID uuid.UUID) ([]uuid.UUID, error) {
	rows, err := r.db.Query(ctx,
		`SELECT manager_id FROM department_managers WHERE department_id = $1 ORDER BY assigned_at`,
		departmentID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ids []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	if ids == nil {
		ids = []uuid.UUID{}
	}
	return ids, rows.Err()
}

func (r *departmentRepo) GetByID(ctx context.Context, id uuid.UUID) (*models.Department, error) {
	var dept models.Department
	err := r.db.QueryRow(ctx,
		`SELECT id, department_code, name, description, created_at, updated_at
		 FROM departments WHERE id = $1`, id,
	).Scan(&dept.ID, &dept.DepartmentCode, &dept.Name, &dept.Description,
		&dept.CreatedAt, &dept.UpdatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("department not found: %w", err)
		}
		return nil, fmt.Errorf("get department by id: %w", err)
	}

	dept.ManagerIDs, err = r.loadManagerIDs(ctx, dept.ID)
	if err != nil {
		return nil, fmt.Errorf("load manager ids: %w", err)
	}
	return &dept, nil
}

func (r *departmentRepo) GetByCode(ctx context.Context, code string) (*models.Department, error) {
	var dept models.Department
	err := r.db.QueryRow(ctx,
		`SELECT id, department_code, name, description, created_at, updated_at
		 FROM departments WHERE department_code = $1`, code,
	).Scan(&dept.ID, &dept.DepartmentCode, &dept.Name, &dept.Description,
		&dept.CreatedAt, &dept.UpdatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("department not found: %w", err)
		}
		return nil, fmt.Errorf("get department by code: %w", err)
	}

	dept.ManagerIDs, err = r.loadManagerIDs(ctx, dept.ID)
	if err != nil {
		return nil, fmt.Errorf("load manager ids: %w", err)
	}
	return &dept, nil
}

func (r *departmentRepo) GetAll(ctx context.Context) ([]models.Department, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, department_code, name, description, created_at, updated_at
		 FROM departments ORDER BY name`)
	if err != nil {
		return nil, fmt.Errorf("get all departments: %w", err)
	}
	defer rows.Close()

	var departments []models.Department
	for rows.Next() {
		var dept models.Department
		if err := rows.Scan(&dept.ID, &dept.DepartmentCode, &dept.Name, &dept.Description,
			&dept.CreatedAt, &dept.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan department: %w", err)
		}
		departments = append(departments, dept)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	// Load manager IDs for each department
	for i := range departments {
		departments[i].ManagerIDs, err = r.loadManagerIDs(ctx, departments[i].ID)
		if err != nil {
			return nil, fmt.Errorf("load manager ids for %s: %w", departments[i].ID, err)
		}
	}
	return departments, nil
}

func (r *departmentRepo) Create(ctx context.Context, dept *models.Department) error {
	err := r.db.QueryRow(ctx,
		`INSERT INTO departments (department_code, name, description)
		 VALUES ($1,$2,$3) RETURNING id, created_at, updated_at`,
		dept.DepartmentCode, dept.Name, dept.Description,
	).Scan(&dept.ID, &dept.CreatedAt, &dept.UpdatedAt)
	if err != nil {
		return err
	}

	// Assign managers if provided
	for _, mID := range dept.ManagerIDs {
		if err := r.AddManager(ctx, dept.ID, mID); err != nil {
			return fmt.Errorf("add manager %s: %w", mID, err)
		}
	}
	return nil
}

func (r *departmentRepo) Update(ctx context.Context, dept *models.Department) error {
	_, err := r.db.Exec(ctx,
		`UPDATE departments SET name=$1, description=$2, updated_at=CURRENT_TIMESTAMP
		 WHERE id=$3`,
		dept.Name, dept.Description, dept.ID)
	if err != nil {
		return err
	}

	// If ManagerIDs was explicitly set (even empty), sync the junction table
	if dept.ManagerIDs != nil {
		if err := r.SetManagers(ctx, dept.ID, dept.ManagerIDs); err != nil {
			return fmt.Errorf("set managers: %w", err)
		}
	}
	return nil
}

func (r *departmentRepo) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.Exec(ctx, `DELETE FROM departments WHERE id=$1`, id)
	return err
}

// AddManager links a manager to a department (idempotent).
func (r *departmentRepo) AddManager(ctx context.Context, departmentID, managerID uuid.UUID) error {
	_, err := r.db.Exec(ctx,
		`INSERT INTO department_managers (department_id, manager_id)
		 VALUES ($1, $2) ON CONFLICT DO NOTHING`,
		departmentID, managerID)
	return err
}

// RemoveManager unlinks a manager from a department.
func (r *departmentRepo) RemoveManager(ctx context.Context, departmentID, managerID uuid.UUID) error {
	_, err := r.db.Exec(ctx,
		`DELETE FROM department_managers WHERE department_id=$1 AND manager_id=$2`,
		departmentID, managerID)
	return err
}

// SetManagers replaces all managers for a department with the provided list.
func (r *departmentRepo) SetManagers(ctx context.Context, departmentID uuid.UUID, managerIDs []uuid.UUID) error {
	return r.db.ExecTx(ctx, func(ctx context.Context, tx pgx.Tx) error {
		if _, err := tx.Exec(ctx,
			`DELETE FROM department_managers WHERE department_id=$1`, departmentID); err != nil {
			return err
		}
		for _, mID := range managerIDs {
			if _, err := tx.Exec(ctx,
				`INSERT INTO department_managers (department_id, manager_id) VALUES ($1,$2)`,
				departmentID, mID); err != nil {
				return err
			}
		}
		return nil
	})
}

// GetManagers returns the manager IDs for a department.
func (r *departmentRepo) GetManagers(ctx context.Context, departmentID uuid.UUID) ([]uuid.UUID, error) {
	return r.loadManagerIDs(ctx, departmentID)
}

// GetByManagerID returns all departments that the given manager is assigned to
// via the department_managers junction table.
func (r *departmentRepo) GetByManagerID(ctx context.Context, managerID uuid.UUID) ([]models.Department, error) {
	rows, err := r.db.Query(ctx,
		`SELECT d.id, d.department_code, d.name, d.description, d.created_at, d.updated_at
		 FROM departments d
		 INNER JOIN department_managers dm ON dm.department_id = d.id
		 WHERE dm.manager_id = $1
		 ORDER BY d.name`,
		managerID,
	)
	if err != nil {
		return nil, fmt.Errorf("get departments by manager: %w", err)
	}
	defer rows.Close()

	var departments []models.Department
	for rows.Next() {
		var dept models.Department
		if err := rows.Scan(&dept.ID, &dept.DepartmentCode, &dept.Name, &dept.Description,
			&dept.CreatedAt, &dept.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan department: %w", err)
		}
		departments = append(departments, dept)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	// Load manager IDs for each department
	var loadErr error
	for i := range departments {
		departments[i].ManagerIDs, loadErr = r.loadManagerIDs(ctx, departments[i].ID)
		if loadErr != nil {
			return nil, fmt.Errorf("load manager ids for %s: %w", departments[i].ID, loadErr)
		}
	}
	if departments == nil {
		departments = []models.Department{}
	}
	return departments, nil
}
