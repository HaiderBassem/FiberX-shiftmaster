package repository

import (
	"context"
	"log"

	"github.com/google/uuid"

	"shiftmaster-backend/internal/models"
	"shiftmaster-backend/pkg/database"
)

type ModuleAccessRepository interface {
	// Link CRUD
	CreateLink(ctx context.Context, link *models.ExternalLink) error
	UpdateLink(ctx context.Context, link *models.ExternalLink) error
	DeleteLink(ctx context.Context, id uuid.UUID) error
	GetAllLinks(ctx context.Context) ([]models.ExternalLink, error)
	GetLinkByID(ctx context.Context, id uuid.UUID) (*models.ExternalLink, error)

	// Department access
	AddDepartmentAccess(ctx context.Context, linkID uuid.UUID, departmentID uuid.UUID, grantedBy *uuid.UUID) error
	RemoveDepartmentAccess(ctx context.Context, linkID uuid.UUID, departmentID uuid.UUID) error
	GetDepartmentsWithAccess(ctx context.Context, linkID uuid.UUID) ([]uuid.UUID, error)

	// Employee exclusions
	AddEmployeeExclusion(ctx context.Context, linkID uuid.UUID, employeeID uuid.UUID, excludedBy *uuid.UUID) error
	RemoveEmployeeExclusion(ctx context.Context, linkID uuid.UUID, employeeID uuid.UUID) error
	GetExcludedEmployees(ctx context.Context, linkID uuid.UUID, departmentID *uuid.UUID) ([]uuid.UUID, error)

	// Employee specific
	GetEmployeeAllowedLinks(ctx context.Context, employeeID uuid.UUID, departmentID *uuid.UUID) ([]models.ExternalLink, error)
}

type moduleAccessRepo struct {
	db *database.DB
}

func NewModuleAccessRepository(db *database.DB) ModuleAccessRepository {
	repo := &moduleAccessRepo{db: db}
	repo.autoMigrate()
	return repo
}

func (r *moduleAccessRepo) autoMigrate() {
	query := `
	CREATE TABLE IF NOT EXISTS external_links (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		title VARCHAR(255) NOT NULL,
		url VARCHAR(2048) NOT NULL,
		icon_name VARCHAR(100) DEFAULT 'link',
		created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
		created_by UUID REFERENCES employees(id) ON DELETE SET NULL
	);
	CREATE TABLE IF NOT EXISTS link_departments (
		link_id UUID NOT NULL REFERENCES external_links(id) ON DELETE CASCADE,
		department_id UUID NOT NULL REFERENCES departments(id) ON DELETE CASCADE,
		granted_by UUID REFERENCES employees(id) ON DELETE SET NULL,
		created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
		PRIMARY KEY (link_id, department_id)
	);
	CREATE TABLE IF NOT EXISTS link_exclusions (
		link_id UUID NOT NULL REFERENCES external_links(id) ON DELETE CASCADE,
		employee_id UUID NOT NULL REFERENCES employees(id) ON DELETE CASCADE,
		excluded_by UUID REFERENCES employees(id) ON DELETE SET NULL,
		created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
		PRIMARY KEY (link_id, employee_id)
	);
	`
	_, err := r.db.Exec(context.Background(), query)
	if err != nil {
		log.Printf("Failed to auto-migrate external_links tables: %v", err)
	}
}

func (r *moduleAccessRepo) CreateLink(ctx context.Context, link *models.ExternalLink) error {
	query := `
		INSERT INTO external_links (id, title, url, icon_name, created_by)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING created_at
	`
	if link.ID == uuid.Nil {
		link.ID = uuid.New()
	}
	return r.db.QueryRow(ctx, query, link.ID, link.Title, link.URL, link.IconName, link.CreatedBy).Scan(&link.CreatedAt)
}

func (r *moduleAccessRepo) UpdateLink(ctx context.Context, link *models.ExternalLink) error {
	query := `
		UPDATE external_links
		SET title = $1, url = $2, icon_name = $3
		WHERE id = $4
	`
	_, err := r.db.Exec(ctx, query, link.Title, link.URL, link.IconName, link.ID)
	return err
}

func (r *moduleAccessRepo) DeleteLink(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM external_links WHERE id = $1`
	_, err := r.db.Exec(ctx, query, id)
	return err
}

func (r *moduleAccessRepo) GetAllLinks(ctx context.Context) ([]models.ExternalLink, error) {
	query := `SELECT id, title, url, icon_name, created_at, created_by FROM external_links ORDER BY created_at ASC`
	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var links []models.ExternalLink
	for rows.Next() {
		var l models.ExternalLink
		if err := rows.Scan(&l.ID, &l.Title, &l.URL, &l.IconName, &l.CreatedAt, &l.CreatedBy); err == nil {
			links = append(links, l)
		}
	}
	return links, nil
}

func (r *moduleAccessRepo) GetLinkByID(ctx context.Context, id uuid.UUID) (*models.ExternalLink, error) {
	query := `SELECT id, title, url, icon_name, created_at, created_by FROM external_links WHERE id = $1`
	var l models.ExternalLink
	err := r.db.QueryRow(ctx, query, id).Scan(&l.ID, &l.Title, &l.URL, &l.IconName, &l.CreatedAt, &l.CreatedBy)
	if err != nil {
		return nil, err
	}
	return &l, nil
}

func (r *moduleAccessRepo) AddDepartmentAccess(ctx context.Context, linkID uuid.UUID, departmentID uuid.UUID, grantedBy *uuid.UUID) error {
	query := `
		INSERT INTO link_departments (link_id, department_id, granted_by)
		VALUES ($1, $2, $3)
		ON CONFLICT (link_id, department_id) DO NOTHING
	`
	_, err := r.db.Exec(ctx, query, linkID, departmentID, grantedBy)
	return err
}

func (r *moduleAccessRepo) RemoveDepartmentAccess(ctx context.Context, linkID uuid.UUID, departmentID uuid.UUID) error {
	query := `DELETE FROM link_departments WHERE link_id = $1 AND department_id = $2`
	_, err := r.db.Exec(ctx, query, linkID, departmentID)
	return err
}

func (r *moduleAccessRepo) GetDepartmentsWithAccess(ctx context.Context, linkID uuid.UUID) ([]uuid.UUID, error) {
	query := `SELECT department_id FROM link_departments WHERE link_id = $1`
	rows, err := r.db.Query(ctx, query, linkID)
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

func (r *moduleAccessRepo) AddEmployeeExclusion(ctx context.Context, linkID uuid.UUID, employeeID uuid.UUID, excludedBy *uuid.UUID) error {
	query := `
		INSERT INTO link_exclusions (link_id, employee_id, excluded_by)
		VALUES ($1, $2, $3)
		ON CONFLICT (link_id, employee_id) DO NOTHING
	`
	_, err := r.db.Exec(ctx, query, linkID, employeeID, excludedBy)
	return err
}

func (r *moduleAccessRepo) RemoveEmployeeExclusion(ctx context.Context, linkID uuid.UUID, employeeID uuid.UUID) error {
	query := `DELETE FROM link_exclusions WHERE link_id = $1 AND employee_id = $2`
	_, err := r.db.Exec(ctx, query, linkID, employeeID)
	return err
}

func (r *moduleAccessRepo) GetExcludedEmployees(ctx context.Context, linkID uuid.UUID, departmentID *uuid.UUID) ([]uuid.UUID, error) {
	query := `
		SELECT me.employee_id 
		FROM link_exclusions me
		JOIN employees e ON me.employee_id = e.id
		WHERE me.link_id = $1
	`
	var args []interface{}
	args = append(args, linkID)

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

func (r *moduleAccessRepo) GetEmployeeAllowedLinks(ctx context.Context, employeeID uuid.UUID, departmentID *uuid.UUID) ([]models.ExternalLink, error) {
	if departmentID == nil {
		return []models.ExternalLink{}, nil
	}

	query := `
		SELECT DISTINCT el.id, el.title, el.url, el.icon_name, el.created_at, el.created_by
		FROM external_links el
		JOIN link_departments md ON el.id = md.link_id
		LEFT JOIN link_exclusions me ON el.id = me.link_id AND me.employee_id = $1
		WHERE md.department_id = $2 AND me.employee_id IS NULL
		ORDER BY el.created_at ASC
	`
	rows, err := r.db.Query(ctx, query, employeeID, *departmentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var links []models.ExternalLink
	for rows.Next() {
		var l models.ExternalLink
		if err := rows.Scan(&l.ID, &l.Title, &l.URL, &l.IconName, &l.CreatedAt, &l.CreatedBy); err == nil {
			links = append(links, l)
		}
	}
	return links, nil
}
