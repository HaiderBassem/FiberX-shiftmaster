package repository

import (
	"context"
	"errors"
	"shiftmaster-backend/internal/models"
	"shiftmaster-backend/pkg/database"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type HelpDocumentRepository struct {
	db *database.DB
}

func NewHelpDocumentRepository(db *database.DB) *HelpDocumentRepository {
	return &HelpDocumentRepository{db: db}
}

// Get accessible documents for a user in a department
func (r *HelpDocumentRepository) GetVisibleDocuments(ctx context.Context, departmentID uuid.UUID, employeeID uuid.UUID, role string) ([]models.HelpDocument, error) {
	var docs []models.HelpDocument
	
	query := `
		SELECT 
			d.id, d.department_id, d.title, d.content, d.created_by, d.created_at, d.updated_at,
			COALESCE(a.access_level, 'hide') as access_level
		FROM help_documents d
		LEFT JOIN help_document_access a ON a.document_id = d.id AND a.employee_id = $2
		WHERE d.department_id = $1
	`
	
	// If not manager/team_leader, only show docs with explicit read/write access
	if role != "manager" && role != "team_leader" && role != "admin" {
		query += ` AND a.access_level IN ('read', 'write') `
	}
	
	query += ` ORDER BY d.created_at DESC`

	rows, err := r.db.Query(ctx, query, departmentID, employeeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var d models.HelpDocument
		var accLevel string
		err := rows.Scan(
			&d.ID, &d.DepartmentID, &d.Title, &d.Content,
			&d.CreatedBy, &d.CreatedAt, &d.UpdatedAt, &accLevel,
		)
		if err != nil {
			return nil, err
		}
		
		// If manager/tl, and no explicit access, default to write
		if (role == "manager" || role == "team_leader" || role == "admin") && accLevel == "hide" {
			accLevel = "write"
		}
		
		d.AccessLevel = &accLevel
		docs = append(docs, d)
	}

	return docs, nil
}

func (r *HelpDocumentRepository) GetDocumentByID(ctx context.Context, id uuid.UUID, employeeID uuid.UUID, role string) (*models.HelpDocument, error) {
	query := `
		SELECT 
			d.id, d.department_id, d.title, d.content, d.created_by, d.created_at, d.updated_at,
			COALESCE(a.access_level, 'hide') as access_level
		FROM help_documents d
		LEFT JOIN help_document_access a ON a.document_id = d.id AND a.employee_id = $2
		WHERE d.id = $1
	`

	var d models.HelpDocument
	var accLevel string
	err := r.db.QueryRow(ctx, query, id, employeeID).Scan(
		&d.ID, &d.DepartmentID, &d.Title, &d.Content,
		&d.CreatedBy, &d.CreatedAt, &d.UpdatedAt, &accLevel,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil // not found
		}
		return nil, err
	}

	if (role == "manager" || role == "team_leader" || role == "admin") && accLevel == "hide" {
		accLevel = "write"
	}

	d.AccessLevel = &accLevel

	// Only return if they have access
	if accLevel != "read" && accLevel != "write" {
		return nil, errors.New("access denied")
	}

	return &d, nil
}

func (r *HelpDocumentRepository) CreateDocument(ctx context.Context, doc *models.HelpDocument) (*models.HelpDocument, error) {
	query := `
		INSERT INTO help_documents (department_id, title, content, created_by)
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

func (r *HelpDocumentRepository) UpdateDocument(ctx context.Context, doc *models.HelpDocument) (*models.HelpDocument, error) {
	query := `
		UPDATE help_documents
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

func (r *HelpDocumentRepository) DeleteDocument(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.Exec(ctx, "DELETE FROM help_documents WHERE id = $1", id)
	return err
}

func (r *HelpDocumentRepository) SetEmployeeAccess(ctx context.Context, documentID, employeeID uuid.UUID, accessLevel string, grantedBy uuid.UUID) error {
	if accessLevel == "hide" {
		// Just remove the record
		_, err := r.db.Exec(ctx, "DELETE FROM help_document_access WHERE document_id = $1 AND employee_id = $2", documentID, employeeID)
		return err
	}

	query := `
		INSERT INTO help_document_access (document_id, employee_id, access_level, granted_by)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (document_id, employee_id) 
		DO UPDATE SET access_level = EXCLUDED.access_level, granted_by = EXCLUDED.granted_by
	`
	_, err := r.db.Exec(ctx, query, documentID, employeeID, accessLevel, grantedBy)
	return err
}

func (r *HelpDocumentRepository) GetDocumentAccessList(ctx context.Context, documentID uuid.UUID) ([]models.HelpDocumentAccess, error) {
	query := `
		SELECT id, document_id, employee_id, access_level, granted_by, created_at
		FROM help_document_access
		WHERE document_id = $1
	`
	rows, err := r.db.Query(ctx, query, documentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []models.HelpDocumentAccess
	for rows.Next() {
		var a models.HelpDocumentAccess
		if err := rows.Scan(&a.ID, &a.DocumentID, &a.EmployeeID, &a.AccessLevel, &a.GrantedBy, &a.CreatedAt); err != nil {
			return nil, err
		}
		list = append(list, a)
	}
	return list, nil
}
