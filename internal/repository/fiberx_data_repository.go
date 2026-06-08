package repository

import (
	"context"
	"errors"
	"shiftmaster-backend/internal/models"
	"shiftmaster-backend/pkg/database"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type FiberxDataRepository struct {
	db *database.DB
}

func NewFiberxDataRepository(db *database.DB) *FiberxDataRepository {
	return &FiberxDataRepository{db: db}
}

// Get accessible documents for an employee
func (r *FiberxDataRepository) GetVisibleDocuments(ctx context.Context, departmentID uuid.UUID, employeeID uuid.UUID, role string, canManageFiberxData bool) ([]models.FiberxDataResponse, error) {
	var docs []models.FiberxDataResponse
	
	query := `
		SELECT 
			d.id, d.department_id, d.title, d.content, d.created_by, d.created_at, d.updated_at,
			creator.first_name || ' ' || creator.last_name as creator_name,
			dep.name as department_name,
			CASE WHEN d.department_id = $1 THEN false ELSE true END as is_shared,
			COALESCE(
				ea.access_level, 
				CASE WHEN d.department_id = $1 THEN 'read' ELSE ds.access_level END
			) as effective_access
		FROM fiberx_data d
		LEFT JOIN employees creator ON creator.id = d.created_by
		LEFT JOIN departments dep ON dep.id = d.department_id
		LEFT JOIN fiberx_data_department_shares ds ON ds.data_id = d.id AND ds.department_id = $1
		LEFT JOIN fiberx_data_employee_access ea ON ea.data_id = d.id AND ea.employee_id = $2
		WHERE 
			(d.department_id = $1 OR ds.id IS NOT NULL)
	`
	
	// If not manager/admin and doesn't have explicit manage permission, hide documents marked as 'hide'
	if role != "manager" && role != "admin" && !canManageFiberxData {
		query += ` AND COALESCE(ea.access_level, 'read') != 'hide' `
	}
	
	query += ` ORDER BY d.created_at DESC`

	rows, err := r.db.Query(ctx, query, departmentID, employeeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var d models.FiberxDataResponse
		var accLevel string
		err := rows.Scan(
			&d.ID, &d.DepartmentID, &d.Title, &d.Content,
			&d.CreatedBy, &d.CreatedAt, &d.UpdatedAt,
			&d.CreatorName, &d.DepartmentName, &d.IsShared, &accLevel,
		)
		if err != nil {
			return nil, err
		}
		
		// If manager or has can_manage_fiberx_data and it belongs to their department, grant 'write' access globally
		if (role == "manager" || role == "admin" || canManageFiberxData) && !d.IsShared {
			accLevel = "write"
		}
		
		d.AccessLevel = accLevel
		docs = append(docs, d)
	}

	return docs, nil
}

func (r *FiberxDataRepository) GetDocumentByID(ctx context.Context, id uuid.UUID, departmentID uuid.UUID, employeeID uuid.UUID, role string, canManageFiberxData bool) (*models.FiberxDataResponse, error) {
	query := `
		SELECT 
			d.id, d.department_id, d.title, d.content, d.created_by, d.created_at, d.updated_at,
			creator.first_name || ' ' || creator.last_name as creator_name,
			dep.name as department_name,
			CASE WHEN d.department_id = $1 THEN false ELSE true END as is_shared,
			COALESCE(
				ea.access_level, 
				CASE WHEN d.department_id = $1 THEN 'read' ELSE ds.access_level END
			) as effective_access
		FROM fiberx_data d
		LEFT JOIN employees creator ON creator.id = d.created_by
		LEFT JOIN departments dep ON dep.id = d.department_id
		LEFT JOIN fiberx_data_department_shares ds ON ds.data_id = d.id AND ds.department_id = $1
		LEFT JOIN fiberx_data_employee_access ea ON ea.data_id = d.id AND ea.employee_id = $2
		WHERE d.id = $3 AND (d.department_id = $1 OR ds.id IS NOT NULL)
	`

	var d models.FiberxDataResponse
	var accLevel string
	err := r.db.QueryRow(ctx, query, departmentID, employeeID, id).Scan(
		&d.ID, &d.DepartmentID, &d.Title, &d.Content,
		&d.CreatedBy, &d.CreatedAt, &d.UpdatedAt,
		&d.CreatorName, &d.DepartmentName, &d.IsShared, &accLevel,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil // not found
		}
		return nil, err
	}

	if (role == "manager" || role == "admin" || canManageFiberxData) && !d.IsShared {
		accLevel = "write"
	}

	d.AccessLevel = accLevel

	if accLevel == "hide" {
		return nil, errors.New("access denied")
	}

	return &d, nil
}

func (r *FiberxDataRepository) CreateDocument(ctx context.Context, doc *models.FiberxData) (*models.FiberxData, error) {
	query := `
		INSERT INTO fiberx_data (department_id, title, content, created_by)
		VALUES ($1, $2, $3, $4)
		RETURNING id, created_at, updated_at
	`
	err := r.db.QueryRow(ctx, query, doc.DepartmentID, doc.Title, doc.Content, doc.CreatedBy).
		Scan(&doc.ID, &doc.CreatedAt, &doc.UpdatedAt)
		
	if err != nil {
		return nil, err
	}
	return doc, nil
}

func (r *FiberxDataRepository) UpdateDocument(ctx context.Context, doc *models.FiberxData) (*models.FiberxData, error) {
	query := `
		UPDATE fiberx_data
		SET title = $1, content = $2, updated_at = CURRENT_TIMESTAMP
		WHERE id = $3
		RETURNING updated_at
	`
	err := r.db.QueryRow(ctx, query, doc.Title, doc.Content, doc.ID).Scan(&doc.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return doc, nil
}

func (r *FiberxDataRepository) DeleteDocument(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.Exec(ctx, "DELETE FROM fiberx_data WHERE id = $1", id)
	return err
}

func (r *FiberxDataRepository) SetEmployeeAccess(ctx context.Context, documentID, employeeID uuid.UUID, accessLevel string, grantedBy uuid.UUID) error {
	query := `
		INSERT INTO fiberx_data_employee_access (data_id, employee_id, access_level, granted_by)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (data_id, employee_id) 
		DO UPDATE SET access_level = EXCLUDED.access_level, granted_by = EXCLUDED.granted_by
	`
	_, err := r.db.Exec(ctx, query, documentID, employeeID, accessLevel, grantedBy)
	return err
}

func (r *FiberxDataRepository) GetEmployeeAccessList(ctx context.Context, documentID uuid.UUID) ([]models.FiberxDataEmployeeAccess, error) {
	query := `
		SELECT ea.id, ea.data_id, ea.employee_id, ea.access_level, ea.granted_by, ea.created_at, e.first_name || ' ' || e.last_name as employee_name
		FROM fiberx_data_employee_access ea
		JOIN employees e ON e.id = ea.employee_id
		WHERE ea.data_id = $1
	`
	rows, err := r.db.Query(ctx, query, documentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []models.FiberxDataEmployeeAccess
	for rows.Next() {
		var a models.FiberxDataEmployeeAccess
		if err := rows.Scan(&a.ID, &a.DataID, &a.EmployeeID, &a.AccessLevel, &a.GrantedBy, &a.CreatedAt, &a.EmployeeName); err != nil {
			return nil, err
		}
		list = append(list, a)
	}
	return list, nil
}

func (r *FiberxDataRepository) SetDepartmentShare(ctx context.Context, documentID, departmentID uuid.UUID, accessLevel string, grantedBy uuid.UUID) error {
	if accessLevel == "none" {
		_, err := r.db.Exec(ctx, "DELETE FROM fiberx_data_department_shares WHERE data_id = $1 AND department_id = $2", documentID, departmentID)
		return err
	}

	query := `
		INSERT INTO fiberx_data_department_shares (data_id, department_id, access_level, granted_by)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (data_id, department_id) 
		DO UPDATE SET access_level = EXCLUDED.access_level, granted_by = EXCLUDED.granted_by
	`
	_, err := r.db.Exec(ctx, query, documentID, departmentID, accessLevel, grantedBy)
	return err
}

func (r *FiberxDataRepository) GetDepartmentShares(ctx context.Context, documentID uuid.UUID) ([]models.FiberxDataDepartmentShare, error) {
	query := `
		SELECT ds.id, ds.data_id, ds.department_id, ds.access_level, ds.granted_by, ds.created_at, d.name as department_name
		FROM fiberx_data_department_shares ds
		JOIN departments d ON d.id = ds.department_id
		WHERE ds.data_id = $1
	`
	rows, err := r.db.Query(ctx, query, documentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []models.FiberxDataDepartmentShare
	for rows.Next() {
		var s models.FiberxDataDepartmentShare
		if err := rows.Scan(&s.ID, &s.DataID, &s.DepartmentID, &s.AccessLevel, &s.GrantedBy, &s.CreatedAt, &s.DepartmentName); err != nil {
			return nil, err
		}
		list = append(list, s)
	}
	return list, nil
}
