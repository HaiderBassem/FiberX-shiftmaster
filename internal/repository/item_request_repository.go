package repository

import (
	"context"

	"github.com/google/uuid"

	"shiftmaster-backend/internal/models"
	"shiftmaster-backend/pkg/database"
)

type ItemRequestRepository interface {
	// Categories
	CreateCategory(ctx context.Context, cat *models.ItemRequestCategory) error
	GetCategoriesByDepartment(ctx context.Context, departmentID uuid.UUID) ([]models.ItemRequestCategory, error)
	GetCategoryByID(ctx context.Context, id uuid.UUID) (*models.ItemRequestCategory, error)
	UpdateCategory(ctx context.Context, cat *models.ItemRequestCategory) error
	DeleteCategory(ctx context.Context, id uuid.UUID) error

	// Requests
	CreateRequest(ctx context.Context, req *models.ItemRequest) error
	GetRequestsByEmployee(ctx context.Context, employeeID uuid.UUID) ([]models.ItemRequest, error)
	GetPendingRequests(ctx context.Context, departmentID uuid.UUID) ([]models.ItemRequest, error)
	UpdateStatus(ctx context.Context, id uuid.UUID, status string) error
	Cancel(ctx context.Context, id uuid.UUID, employeeID uuid.UUID) error
}

type itemRequestRepo struct {
	db *database.DB
}

func NewItemRequestRepository(db *database.DB) ItemRequestRepository {
	return &itemRequestRepo{db: db}
}

func (r *itemRequestRepo) CreateCategory(ctx context.Context, cat *models.ItemRequestCategory) error {
	return r.db.QueryRow(ctx,
		`INSERT INTO item_request_categories (department_id, name, to_emails, cc_emails)
		 VALUES ($1, $2, $3, $4) RETURNING id, created_at, updated_at`,
		cat.DepartmentID, cat.Name, cat.ToEmails, cat.CCEmails,
	).Scan(&cat.ID, &cat.CreatedAt, &cat.UpdatedAt)
}

func (r *itemRequestRepo) GetCategoriesByDepartment(ctx context.Context, departmentID uuid.UUID) ([]models.ItemRequestCategory, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, department_id, name, to_emails, cc_emails, created_at, updated_at
		 FROM item_request_categories
		 WHERE department_id = $1 ORDER BY name ASC`,
		departmentID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var cats []models.ItemRequestCategory
	for rows.Next() {
		var c models.ItemRequestCategory
		if err := rows.Scan(&c.ID, &c.DepartmentID, &c.Name, &c.ToEmails, &c.CCEmails, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, err
		}
		cats = append(cats, c)
	}
	return cats, rows.Err()
}

func (r *itemRequestRepo) GetCategoryByID(ctx context.Context, id uuid.UUID) (*models.ItemRequestCategory, error) {
	var c models.ItemRequestCategory
	err := r.db.QueryRow(ctx,
		`SELECT id, department_id, name, to_emails, cc_emails, created_at, updated_at
		 FROM item_request_categories WHERE id = $1`,
		id,
	).Scan(&c.ID, &c.DepartmentID, &c.Name, &c.ToEmails, &c.CCEmails, &c.CreatedAt, &c.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &c, nil
}

func (r *itemRequestRepo) UpdateCategory(ctx context.Context, cat *models.ItemRequestCategory) error {
	return r.db.QueryRow(ctx,
		`UPDATE item_request_categories
		 SET name=$1, to_emails=$2, cc_emails=$3, updated_at=CURRENT_TIMESTAMP
		 WHERE id=$4 RETURNING updated_at`,
		cat.Name, cat.ToEmails, cat.CCEmails, cat.ID,
	).Scan(&cat.UpdatedAt)
}

func (r *itemRequestRepo) DeleteCategory(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.Exec(ctx, `DELETE FROM item_request_categories WHERE id=$1`, id)
	return err
}

func (r *itemRequestRepo) CreateRequest(ctx context.Context, req *models.ItemRequest) error {
	return r.db.QueryRow(ctx,
		`INSERT INTO item_requests (employee_id, category_id, description, status)
		 VALUES ($1, $2, $3, $4) RETURNING id, created_at, updated_at`,
		req.EmployeeID, req.CategoryID, req.Description, req.Status,
	).Scan(&req.ID, &req.CreatedAt, &req.UpdatedAt)
}

func (r *itemRequestRepo) GetRequestsByEmployee(ctx context.Context, employeeID uuid.UUID) ([]models.ItemRequest, error) {
	rows, err := r.db.Query(ctx,
		`SELECT r.id, r.employee_id, r.category_id, r.description, r.status, r.created_at, r.updated_at,
		        c.name as category_name
		 FROM item_requests r
		 JOIN item_request_categories c ON c.id = r.category_id
		 WHERE r.employee_id = $1 ORDER BY r.created_at DESC`,
		employeeID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var reqs []models.ItemRequest
	for rows.Next() {
		var r models.ItemRequest
		if err := rows.Scan(&r.ID, &r.EmployeeID, &r.CategoryID, &r.Description, &r.Status, &r.CreatedAt, &r.UpdatedAt, &r.CategoryName); err != nil {
			return nil, err
		}
		reqs = append(reqs, r)
	}
	return reqs, rows.Err()
}

func (r *itemRequestRepo) GetPendingRequests(ctx context.Context, departmentID uuid.UUID) ([]models.ItemRequest, error) {
	rows, err := r.db.Query(ctx, 
		"SELECT r.id, r.employee_id, r.category_id, r.description, r.status, r.created_at, r.updated_at, c.name as category_name, e.first_name || ' ' || e.last_name as employee_name " +
		"FROM item_requests r " +
		"JOIN item_request_categories c ON c.id = r.category_id " +
		"JOIN employees e ON e.id = r.employee_id " +
		"WHERE r.status = 'pending' AND e.department_id = $1 ORDER BY r.created_at ASC",
		departmentID,
	)
	if err != nil { return nil, err }
	defer rows.Close()
	var reqs []models.ItemRequest
	for rows.Next() {
		var r models.ItemRequest
		if err := rows.Scan(&r.ID, &r.EmployeeID, &r.CategoryID, &r.Description, &r.Status, &r.CreatedAt, &r.UpdatedAt, &r.CategoryName, &r.EmployeeName); err != nil { return nil, err }
		reqs = append(reqs, r)
	}
	return reqs, rows.Err()
}

func (r *itemRequestRepo) UpdateStatus(ctx context.Context, id uuid.UUID, status string) error {
	_, err := r.db.Exec(ctx, "UPDATE item_requests SET status = $1, updated_at = CURRENT_TIMESTAMP WHERE id = $2", status, id)
	return err
}

func (r *itemRequestRepo) Cancel(ctx context.Context, id uuid.UUID, employeeID uuid.UUID) error {
	_, err := r.db.Exec(ctx, "UPDATE item_requests SET status = 'cancelled', updated_at = CURRENT_TIMESTAMP WHERE id = $1 AND employee_id = $2 AND status = 'pending'", id, employeeID)
	return err
}

