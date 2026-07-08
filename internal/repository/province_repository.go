package repository

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"shiftmaster-backend/internal/models"
	"shiftmaster-backend/pkg/database"
)

// ProvinceRepository defines the interface for province data access.
type ProvinceRepository interface {
	GetAll(ctx context.Context, departmentID uuid.UUID) ([]models.Province, error)
	GetByID(ctx context.Context, id uuid.UUID, departmentID uuid.UUID) (*models.Province, error)
	GetByName(ctx context.Context, name string, departmentID uuid.UUID) (*models.Province, error)
	Create(ctx context.Context, province *models.Province) error
	Update(ctx context.Context, province *models.Province) error
	Delete(ctx context.Context, id uuid.UUID) error

	// Sharing
	ShareProvince(ctx context.Context, share *models.ProvinceShare) error
	UnshareProvince(ctx context.Context, provinceID, departmentID uuid.UUID) error
	GetProvinceShares(ctx context.Context, provinceID uuid.UUID) ([]models.ProvinceShare, error)
}

type provinceRepo struct {
	db *database.DB
}

func NewProvinceRepository(db *database.DB) ProvinceRepository {
	return &provinceRepo{db: db}
}

func (r *provinceRepo) GetAll(ctx context.Context, departmentID uuid.UUID) ([]models.Province, error) {
	query := `
		SELECT DISTINCT
			p.id, p.department_id, p.name, p.sort_order, p.is_active, p.created_by, p.created_at, p.updated_at,
			e.first_name || ' ' || e.last_name AS creator_name,
			dep.name AS department_name,
			CASE WHEN p.department_id = $1 THEN false ELSE true END AS is_shared
		FROM provinces p
		LEFT JOIN employees e ON p.created_by = e.id
		JOIN departments dep ON p.department_id = dep.id
		LEFT JOIN province_department_shares ps ON p.id = ps.province_id AND ps.department_id = $1
		WHERE p.department_id = $1 OR ps.id IS NOT NULL
		ORDER BY p.sort_order ASC, p.name ASC
	`
	rows, err := r.db.Query(ctx, query, departmentID)
	if err != nil {
		return nil, fmt.Errorf("query provinces: %w", err)
	}
	defer rows.Close()

	var provinces []models.Province
	for rows.Next() {
		var p models.Province
		if err := rows.Scan(&p.ID, &p.DepartmentID, &p.Name, &p.SortOrder, &p.IsActive, &p.CreatedBy, &p.CreatedAt, &p.UpdatedAt, &p.CreatorName, &p.DepartmentName, &p.IsShared); err != nil {
			return nil, fmt.Errorf("scan province: %w", err)
		}
		provinces = append(provinces, p)
	}

	return provinces, nil
}

func (r *provinceRepo) GetByID(ctx context.Context, id uuid.UUID, departmentID uuid.UUID) (*models.Province, error) {
	query := `
		SELECT DISTINCT
			p.id, p.department_id, p.name, p.sort_order, p.is_active, p.created_by, p.created_at, p.updated_at,
			e.first_name || ' ' || e.last_name AS creator_name,
			dep.name AS department_name,
			CASE WHEN p.department_id = $2 THEN false ELSE true END AS is_shared
		FROM provinces p
		LEFT JOIN employees e ON p.created_by = e.id
		JOIN departments dep ON p.department_id = dep.id
		LEFT JOIN province_department_shares ps ON p.id = ps.province_id AND ps.department_id = $2
		WHERE p.id = $1 AND (p.department_id = $2 OR ps.id IS NOT NULL)
	`
	var p models.Province
	err := r.db.QueryRow(ctx, query, id, departmentID).Scan(
		&p.ID, &p.DepartmentID, &p.Name, &p.SortOrder, &p.IsActive, &p.CreatedBy, &p.CreatedAt, &p.UpdatedAt, &p.CreatorName, &p.DepartmentName, &p.IsShared,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("get province by id: %w", err)
	}
	return &p, nil
}

func (r *provinceRepo) GetByName(ctx context.Context, name string, departmentID uuid.UUID) (*models.Province, error) {
	query := `
		SELECT DISTINCT
			p.id, p.department_id, p.name, p.sort_order, p.is_active, p.created_by, p.created_at, p.updated_at,
			e.first_name || ' ' || e.last_name AS creator_name,
			dep.name AS department_name,
			CASE WHEN p.department_id = $2 THEN false ELSE true END AS is_shared
		FROM provinces p
		LEFT JOIN employees e ON p.created_by = e.id
		JOIN departments dep ON p.department_id = dep.id
		LEFT JOIN province_department_shares ps ON p.id = ps.province_id AND ps.department_id = $2
		WHERE p.name = $1 AND (p.department_id = $2 OR ps.id IS NOT NULL)
	`
	var p models.Province
	err := r.db.QueryRow(ctx, query, name, departmentID).Scan(
		&p.ID, &p.DepartmentID, &p.Name, &p.SortOrder, &p.IsActive, &p.CreatedBy, &p.CreatedAt, &p.UpdatedAt, &p.CreatorName, &p.DepartmentName, &p.IsShared,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("get province by name: %w", err)
	}
	return &p, nil
}

func (r *provinceRepo) Create(ctx context.Context, province *models.Province) error {
	query := `
		INSERT INTO provinces (department_id, name, sort_order, is_active, created_by)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, created_at, updated_at
	`
	err := r.db.QueryRow(ctx, query, province.DepartmentID, province.Name, province.SortOrder, province.IsActive, province.CreatedBy).
		Scan(&province.ID, &province.CreatedAt, &province.UpdatedAt)
	if err != nil {
		return fmt.Errorf("insert province: %w", err)
	}
	return nil
}

func (r *provinceRepo) Update(ctx context.Context, province *models.Province) error {
	query := `
		UPDATE provinces
		SET name = $1, sort_order = $2, is_active = $3, updated_at = NOW()
		WHERE id = $4
		RETURNING updated_at
	`
	err := r.db.QueryRow(ctx, query, province.Name, province.SortOrder, province.IsActive, province.ID).Scan(&province.UpdatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return fmt.Errorf("province not found")
		}
		return fmt.Errorf("update province: %w", err)
	}
	return nil
}

func (r *provinceRepo) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM provinces WHERE id = $1`
	_, err := r.db.Exec(ctx, query, id)
	return err
}

// ═══════════════════════════════════════════════════════════
// Sharing
// ═══════════════════════════════════════════════════════════

func (r *provinceRepo) ShareProvince(ctx context.Context, share *models.ProvinceShare) error {
	return r.db.QueryRow(ctx,
		`INSERT INTO province_department_shares (province_id, department_id, granted_by)
		 VALUES ($1, $2, $3)
		 ON CONFLICT (province_id, department_id) DO NOTHING
		 RETURNING id, created_at`,
		share.ProvinceID, share.DepartmentID, share.GrantedBy,
	).Scan(&share.ID, &share.CreatedAt)
}

func (r *provinceRepo) UnshareProvince(ctx context.Context, provinceID, departmentID uuid.UUID) error {
	_, err := r.db.Exec(ctx,
		`DELETE FROM province_department_shares WHERE province_id = $1 AND department_id = $2`,
		provinceID, departmentID)
	return err
}

func (r *provinceRepo) GetProvinceShares(ctx context.Context, provinceID uuid.UUID) ([]models.ProvinceShare, error) {
	query := `
		SELECT s.id, s.province_id, s.department_id, s.granted_by, s.created_at, d.name AS department_name
		FROM province_department_shares s
		JOIN departments d ON s.department_id = d.id
		WHERE s.province_id = $1
		ORDER BY d.name
	`
	rows, err := r.db.Query(ctx, query, provinceID)
	if err != nil {
		return nil, fmt.Errorf("get province shares: %w", err)
	}
	defer rows.Close()

	var shares []models.ProvinceShare
	for rows.Next() {
		var s models.ProvinceShare
		if err := rows.Scan(&s.ID, &s.ProvinceID, &s.DepartmentID, &s.GrantedBy, &s.CreatedAt, &s.DepartmentName); err != nil {
			return nil, fmt.Errorf("scan share: %w", err)
		}
		shares = append(shares, s)
	}
	return shares, nil
}
