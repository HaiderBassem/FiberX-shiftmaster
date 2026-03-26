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
}

type departmentRepo struct {
	db *database.DB
}

func NewDepartmentRepository(db *database.DB) DepartmentRepository {
	return &departmentRepo{db: db}
}

func (r *departmentRepo) GetByID(ctx context.Context, id uuid.UUID) (*models.Department, error) {
	var dept models.Department
	err := r.db.QueryRow(ctx,
		`SELECT id, department_code, name, description, manager_id, created_at, updated_at
		 FROM departments WHERE id = $1`, id,
	).Scan(&dept.ID, &dept.DepartmentCode, &dept.Name, &dept.Description, &dept.ManagerID,
		&dept.CreatedAt, &dept.UpdatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("department not found: %w", err)
		}
		return nil, fmt.Errorf("get department by id: %w", err)
	}
	return &dept, nil
}

func (r *departmentRepo) GetByCode(ctx context.Context, code string) (*models.Department, error) {
	var dept models.Department
	err := r.db.QueryRow(ctx,
		`SELECT id, department_code, name, description, manager_id, created_at, updated_at
		 FROM departments WHERE department_code = $1`, code,
	).Scan(&dept.ID, &dept.DepartmentCode, &dept.Name, &dept.Description, &dept.ManagerID,
		&dept.CreatedAt, &dept.UpdatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("department not found: %w", err)
		}
		return nil, fmt.Errorf("get department by code: %w", err)
	}
	return &dept, nil
}

func (r *departmentRepo) GetAll(ctx context.Context) ([]models.Department, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, department_code, name, description, manager_id, created_at, updated_at
		 FROM departments ORDER BY name`)
	if err != nil {
		return nil, fmt.Errorf("get all departments: %w", err)
	}
	defer rows.Close()

	var departments []models.Department
	for rows.Next() {
		var dept models.Department
		if err := rows.Scan(&dept.ID, &dept.DepartmentCode, &dept.Name, &dept.Description,
			&dept.ManagerID, &dept.CreatedAt, &dept.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan department: %w", err)
		}
		departments = append(departments, dept)
	}
	return departments, rows.Err()
}

func (r *departmentRepo) Create(ctx context.Context, dept *models.Department) error {
	return r.db.QueryRow(ctx,
		`INSERT INTO departments (department_code, name, description, manager_id)
		 VALUES ($1,$2,$3,$4) RETURNING id, created_at, updated_at`,
		dept.DepartmentCode, dept.Name, dept.Description, dept.ManagerID,
	).Scan(&dept.ID, &dept.CreatedAt, &dept.UpdatedAt)
}

func (r *departmentRepo) Update(ctx context.Context, dept *models.Department) error {
	_, err := r.db.Exec(ctx,
		`UPDATE departments SET name=$1, description=$2, manager_id=$3, updated_at=CURRENT_TIMESTAMP
		 WHERE id=$4`,
		dept.Name, dept.Description, dept.ManagerID, dept.ID)
	return err
}

func (r *departmentRepo) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.Exec(ctx, `DELETE FROM departments WHERE id=$1`, id)
	return err
}
