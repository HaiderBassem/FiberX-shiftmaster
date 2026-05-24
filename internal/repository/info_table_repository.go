package repository

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"shiftmaster-backend/internal/models"
	"shiftmaster-backend/pkg/database"
)

type InfoTableRepository struct {
	db *database.DB
}

func NewInfoTableRepository(db *database.DB) *InfoTableRepository {
	return &InfoTableRepository{db: db}
}

// CreateTable creates a new dynamic table.
func (r *InfoTableRepository) CreateTable(ctx context.Context, table *models.InfoTable) (*models.InfoTable, error) {
	query := `
		INSERT INTO info_tables (name, description, columns, department_id, created_by)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, name, description, columns, department_id, created_by, created_at, updated_at
	`
	columnsJSON, err := json.Marshal(table.Columns)
	if err != nil {
		return nil, err
	}

	row := r.db.QueryRow(ctx, query, table.Name, table.Description, columnsJSON, table.DepartmentID, table.CreatedBy)

	var created models.InfoTable
	var columnsBytes []byte

	err = row.Scan(
		&created.ID,
		&created.Name,
		&created.Description,
		&columnsBytes,
		&created.DepartmentID,
		&created.CreatedBy,
		&created.CreatedAt,
		&created.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal(columnsBytes, &created.Columns); err != nil {
		return nil, err
	}

	return &created, nil
}

// GetVisibleTables returns tables visible to a specific employee (based on department access or direct employee access or if they created it).
// Admins see all tables. This filtering logic is better partially done in SQL.
func (r *InfoTableRepository) GetVisibleTables(ctx context.Context, employeeID uuid.UUID, role string, departmentID *uuid.UUID) ([]models.InfoTable, error) {
	// If admin, get all
	var query string
	var rows pgx.Rows
	var err error

	if role == "admin" {
		query = `SELECT id, name, description, columns, department_id, created_by, created_at, updated_at FROM info_tables ORDER BY created_at DESC`
		rows, err = r.db.Query(ctx, query)
	} else {
		query = `
			SELECT DISTINCT t.id, t.name, t.description, t.columns, t.department_id, t.created_by, t.created_at, t.updated_at
			FROM info_tables t
			LEFT JOIN info_table_department_access d_acc ON t.id = d_acc.table_id
			LEFT JOIN info_table_employee_access e_acc ON t.id = e_acc.table_id
			WHERE t.created_by = $1
			   OR (t.department_id = $2 AND $2 IS NOT NULL)
			   OR (d_acc.department_id = $2 AND $2 IS NOT NULL)
			   OR e_acc.employee_id = $1
			ORDER BY t.created_at DESC
		`
		rows, err = r.db.Query(ctx, query, employeeID, departmentID)
	}

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tables []models.InfoTable
	for rows.Next() {
		var t models.InfoTable
		var columnsBytes []byte
		if err := rows.Scan(
			&t.ID, &t.Name, &t.Description, &columnsBytes,
			&t.DepartmentID, &t.CreatedBy, &t.CreatedAt, &t.UpdatedAt,
		); err != nil {
			return nil, err
		}
		if err := json.Unmarshal(columnsBytes, &t.Columns); err != nil {
			return nil, err
		}
		tables = append(tables, t)
	}
	return tables, nil
}

// GetTableByID gets a specific table
func (r *InfoTableRepository) GetTableByID(ctx context.Context, id uuid.UUID) (*models.InfoTable, error) {
	query := `SELECT id, name, description, columns, department_id, created_by, created_at, updated_at FROM info_tables WHERE id = $1`
	var t models.InfoTable
	var columnsBytes []byte

	err := r.db.QueryRow(ctx, query, id).Scan(
		&t.ID, &t.Name, &t.Description, &columnsBytes,
		&t.DepartmentID, &t.CreatedBy, &t.CreatedAt, &t.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(columnsBytes, &t.Columns); err != nil {
		return nil, err
	}
	return &t, nil
}

func (r *InfoTableRepository) UpdateTable(ctx context.Context, table *models.InfoTable) (*models.InfoTable, error) {
	query := `
		UPDATE info_tables
		SET name = $1, description = $2, columns = $3, updated_at = CURRENT_TIMESTAMP
		WHERE id = $4
		RETURNING id, name, description, columns, department_id, created_by, created_at, updated_at
	`
	columnsJSON, err := json.Marshal(table.Columns)
	if err != nil {
		return nil, err
	}

	row := r.db.QueryRow(ctx, query, table.Name, table.Description, columnsJSON, table.ID)

	var updated models.InfoTable
	var columnsBytes []byte

	err = row.Scan(
		&updated.ID, &updated.Name, &updated.Description, &columnsBytes,
		&updated.DepartmentID, &updated.CreatedBy, &updated.CreatedAt, &updated.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(columnsBytes, &updated.Columns); err != nil {
		return nil, err
	}

	return &updated, nil
}

func (r *InfoTableRepository) DeleteTable(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.Exec(ctx, `DELETE FROM info_tables WHERE id = $1`, id)
	return err
}

// --- Rows ---

func (r *InfoTableRepository) GetTableRows(ctx context.Context, tableID uuid.UUID) ([]models.InfoTableRow, error) {
	query := `SELECT id, table_id, data, created_by, created_at, updated_at FROM info_table_rows WHERE table_id = $1 ORDER BY created_at ASC`
	rows, err := r.db.Query(ctx, query, tableID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tableRows []models.InfoTableRow
	for rows.Next() {
		var r models.InfoTableRow
		var dataBytes []byte
		if err := rows.Scan(
			&r.ID, &r.TableID, &dataBytes, &r.CreatedBy, &r.CreatedAt, &r.UpdatedAt,
		); err != nil {
			return nil, err
		}
		if err := json.Unmarshal(dataBytes, &r.Data); err != nil {
			return nil, err
		}
		tableRows = append(tableRows, r)
	}
	return tableRows, nil
}

func (r *InfoTableRepository) CreateTableRow(ctx context.Context, row *models.InfoTableRow) (*models.InfoTableRow, error) {
	query := `
		INSERT INTO info_table_rows (table_id, data, created_by)
		VALUES ($1, $2, $3)
		RETURNING id, table_id, data, created_by, created_at, updated_at
	`
	dataJSON, err := json.Marshal(row.Data)
	if err != nil {
		return nil, err
	}

	var created models.InfoTableRow
	var dataBytes []byte

	dbRow := r.db.QueryRow(ctx, query, row.TableID, dataJSON, row.CreatedBy)
	err = dbRow.Scan(
		&created.ID, &created.TableID, &dataBytes, &created.CreatedBy, &created.CreatedAt, &created.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(dataBytes, &created.Data); err != nil {
		return nil, err
	}

	return &created, nil
}

func (r *InfoTableRepository) UpdateTableRow(ctx context.Context, row *models.InfoTableRow) (*models.InfoTableRow, error) {
	query := `
		UPDATE info_table_rows
		SET data = $1, updated_at = CURRENT_TIMESTAMP
		WHERE id = $2
		RETURNING id, table_id, data, created_by, created_at, updated_at
	`
	dataJSON, err := json.Marshal(row.Data)
	if err != nil {
		return nil, err
	}

	var updated models.InfoTableRow
	var dataBytes []byte

	dbRow := r.db.QueryRow(ctx, query, dataJSON, row.ID)
	err = dbRow.Scan(
		&updated.ID, &updated.TableID, &dataBytes, &updated.CreatedBy, &updated.CreatedAt, &updated.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(dataBytes, &updated.Data); err != nil {
		return nil, err
	}

	return &updated, nil
}

func (r *InfoTableRepository) DeleteTableRow(ctx context.Context, rowID uuid.UUID) error {
	_, err := r.db.Exec(ctx, `DELETE FROM info_table_rows WHERE id = $1`, rowID)
	return err
}

// --- Access Management ---

func (r *InfoTableRepository) AddDepartmentAccess(ctx context.Context, access *models.InfoTableDepartmentAccess) error {
	query := `
		INSERT INTO info_table_department_access (table_id, department_id, granted_by)
		VALUES ($1, $2, $3)
		ON CONFLICT (table_id, department_id) DO NOTHING
	`
	_, err := r.db.Exec(ctx, query, access.TableID, access.DepartmentID, access.GrantedBy)
	return err
}

func (r *InfoTableRepository) RemoveDepartmentAccess(ctx context.Context, tableID, departmentID uuid.UUID) error {
	_, err := r.db.Exec(ctx, `DELETE FROM info_table_department_access WHERE table_id = $1 AND department_id = $2`, tableID, departmentID)
	return err
}

func (r *InfoTableRepository) GetDepartmentAccesses(ctx context.Context, tableID uuid.UUID) ([]models.InfoTableDepartmentAccess, error) {
	query := `SELECT id, table_id, department_id, granted_by, created_at FROM info_table_department_access WHERE table_id = $1`
	rows, err := r.db.Query(ctx, query, tableID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var accesses []models.InfoTableDepartmentAccess
	for rows.Next() {
		var a models.InfoTableDepartmentAccess
		if err := rows.Scan(&a.ID, &a.TableID, &a.DepartmentID, &a.GrantedBy, &a.CreatedAt); err != nil {
			return nil, err
		}
		accesses = append(accesses, a)
	}
	return accesses, nil
}

func (r *InfoTableRepository) AddEmployeeAccess(ctx context.Context, access *models.InfoTableEmployeeAccess) error {
	query := `
		INSERT INTO info_table_employee_access (table_id, employee_id, access_level, granted_by)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (table_id, employee_id) DO UPDATE SET access_level = $3
	`
	_, err := r.db.Exec(ctx, query, access.TableID, access.EmployeeID, access.AccessLevel, access.GrantedBy)
	return err
}

func (r *InfoTableRepository) RemoveEmployeeAccess(ctx context.Context, tableID, employeeID uuid.UUID) error {
	_, err := r.db.Exec(ctx, `DELETE FROM info_table_employee_access WHERE table_id = $1 AND employee_id = $2`, tableID, employeeID)
	return err
}

func (r *InfoTableRepository) GetEmployeeAccesses(ctx context.Context, tableID uuid.UUID) ([]models.InfoTableEmployeeAccess, error) {
	query := `SELECT id, table_id, employee_id, access_level, granted_by, created_at FROM info_table_employee_access WHERE table_id = $1`
	rows, err := r.db.Query(ctx, query, tableID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var accesses []models.InfoTableEmployeeAccess
	for rows.Next() {
		var a models.InfoTableEmployeeAccess
		if err := rows.Scan(&a.ID, &a.TableID, &a.EmployeeID, &a.AccessLevel, &a.GrantedBy, &a.CreatedAt); err != nil {
			return nil, err
		}
		accesses = append(accesses, a)
	}
	return accesses, nil
}
