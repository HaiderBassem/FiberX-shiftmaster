package repository

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"shiftmaster-backend/internal/models"
	"shiftmaster-backend/pkg/database"
)

// ServiceRepository defines CRUD operations for the FTTH service catalog.
type ServiceRepository interface {
	// Categories
	CreateCategory(ctx context.Context, cat *models.ServiceCategory) error
	GetCategoriesByProvince(ctx context.Context, provinceID uuid.UUID) ([]models.ServiceCategory, error)
	GetCategoryByID(ctx context.Context, id uuid.UUID) (*models.ServiceCategory, error)
	UpdateCategory(ctx context.Context, cat *models.ServiceCategory) error
	DeleteCategory(ctx context.Context, id uuid.UUID) error

	// Plans
	CreatePlan(ctx context.Context, plan *models.ServicePlan) error
	GetPlansByCategory(ctx context.Context, categoryID uuid.UUID) ([]models.ServicePlan, error)
	GetPlanByID(ctx context.Context, id uuid.UUID) (*models.ServicePlan, error)
	UpdatePlan(ctx context.Context, plan *models.ServicePlan) error
	DeletePlan(ctx context.Context, id uuid.UUID) error
}

type serviceRepo struct {
	db *database.DB
}

// NewServiceRepository creates a new ServiceRepository.
func NewServiceRepository(db *database.DB) ServiceRepository {
	return &serviceRepo{db: db}
}

// ═══════════════════════════════════════════════════════════
// Categories
// ═══════════════════════════════════════════════════════════

func (r *serviceRepo) CreateCategory(ctx context.Context, cat *models.ServiceCategory) error {
	return r.db.QueryRow(ctx,
		`INSERT INTO service_categories (province_id, name, description, is_active, sort_order, created_by)
		 VALUES ($1, $2, $3, $4, $5, $6)
		 RETURNING id, created_at, updated_at`,
		cat.ProvinceID, cat.Name, cat.Description, cat.IsActive, cat.SortOrder, cat.CreatedBy,
	).Scan(&cat.ID, &cat.CreatedAt, &cat.UpdatedAt)
}

func (r *serviceRepo) GetCategoriesByProvince(ctx context.Context, provinceID uuid.UUID) ([]models.ServiceCategory, error) {
	query := `
		SELECT
			sc.id, sc.province_id, sc.name, sc.description, sc.is_active, sc.sort_order,
			sc.created_by, sc.created_at, sc.updated_at,
			e.first_name || ' ' || e.last_name AS creator_name,
			p.name AS province_name,
			COALESCE(pc.plan_count, 0) AS plan_count
		FROM service_categories sc
		JOIN employees e ON sc.created_by = e.id
		JOIN provinces p ON sc.province_id = p.id
		LEFT JOIN (
			SELECT category_id, COUNT(*) AS plan_count
			FROM service_plans
			GROUP BY category_id
		) pc ON pc.category_id = sc.id
		WHERE sc.province_id = $1
		ORDER BY sc.sort_order, sc.created_at DESC
	`

	rows, err := r.db.Query(ctx, query, provinceID)
	if err != nil {
		return nil, fmt.Errorf("get categories: %w", err)
	}
	defer rows.Close()

	var cats []models.ServiceCategory
	for rows.Next() {
		var c models.ServiceCategory
		if err := rows.Scan(
			&c.ID, &c.ProvinceID, &c.Name, &c.Description, &c.IsActive, &c.SortOrder,
			&c.CreatedBy, &c.CreatedAt, &c.UpdatedAt,
			&c.CreatorName, &c.ProvinceName, &c.PlanCount,
		); err != nil {
			return nil, fmt.Errorf("scan category: %w", err)
		}
		cats = append(cats, c)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate categories: %w", err)
	}
	return cats, nil
}

func (r *serviceRepo) GetCategoryByID(ctx context.Context, id uuid.UUID) (*models.ServiceCategory, error) {
	var c models.ServiceCategory
	err := r.db.QueryRow(ctx,
		`SELECT
			sc.id, sc.province_id, sc.name, sc.description, sc.is_active, sc.sort_order,
			sc.created_by, sc.created_at, sc.updated_at,
			e.first_name || ' ' || e.last_name AS creator_name,
			p.name AS province_name,
			COALESCE(pc.plan_count, 0) AS plan_count
		FROM service_categories sc
		JOIN employees e ON sc.created_by = e.id
		JOIN provinces p ON sc.province_id = p.id
		LEFT JOIN (
			SELECT category_id, COUNT(*) AS plan_count
			FROM service_plans
			GROUP BY category_id
		) pc ON pc.category_id = sc.id
		WHERE sc.id = $1`, id,
	).Scan(
		&c.ID, &c.ProvinceID, &c.Name, &c.Description, &c.IsActive, &c.SortOrder,
		&c.CreatedBy, &c.CreatedAt, &c.UpdatedAt,
		&c.CreatorName, &c.ProvinceName, &c.PlanCount,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("category not found")
		}
		return nil, fmt.Errorf("get category: %w", err)
	}
	return &c, nil
}

func (r *serviceRepo) UpdateCategory(ctx context.Context, cat *models.ServiceCategory) error {
	_, err := r.db.Exec(ctx,
		`UPDATE service_categories
		 SET name=$1, description=$2, is_active=$3, sort_order=$4, updated_at=CURRENT_TIMESTAMP
		 WHERE id=$5`,
		cat.Name, cat.Description, cat.IsActive, cat.SortOrder, cat.ID)
	return err
}

func (r *serviceRepo) DeleteCategory(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.Exec(ctx, `DELETE FROM service_categories WHERE id=$1`, id)
	return err
}

// ═══════════════════════════════════════════════════════════
// Sharing
// ═══════════════════════════════════════════════════════════

func (r *serviceRepo) ShareCategory(ctx context.Context, share *models.ServiceCategoryShare) error {
	return r.db.QueryRow(ctx,
		`INSERT INTO service_category_department_shares (category_id, department_id, granted_by)
		 VALUES ($1, $2, $3)
		 ON CONFLICT (category_id, department_id) DO NOTHING
		 RETURNING id, created_at`,
		share.CategoryID, share.DepartmentID, share.GrantedBy,
	).Scan(&share.ID, &share.CreatedAt)
}

func (r *serviceRepo) UnshareCategory(ctx context.Context, categoryID, departmentID uuid.UUID) error {
	_, err := r.db.Exec(ctx,
		`DELETE FROM service_category_department_shares WHERE category_id = $1 AND department_id = $2`,
		categoryID, departmentID)
	return err
}

func (r *serviceRepo) GetCategoryShares(ctx context.Context, categoryID uuid.UUID) ([]models.ServiceCategoryShare, error) {
	query := `
		SELECT s.id, s.category_id, s.department_id, s.granted_by, s.created_at, d.name AS department_name
		FROM service_category_department_shares s
		JOIN departments d ON s.department_id = d.id
		WHERE s.category_id = $1
		ORDER BY d.name
	`
	rows, err := r.db.Query(ctx, query, categoryID)
	if err != nil {
		return nil, fmt.Errorf("get category shares: %w", err)
	}
	defer rows.Close()

	var shares []models.ServiceCategoryShare
	for rows.Next() {
		var s models.ServiceCategoryShare
		if err := rows.Scan(&s.ID, &s.CategoryID, &s.DepartmentID, &s.GrantedBy, &s.CreatedAt, &s.DepartmentName); err != nil {
			return nil, fmt.Errorf("scan share: %w", err)
		}
		shares = append(shares, s)
	}
	return shares, nil
}

// ═══════════════════════════════════════════════════════════
// Plans
// ═══════════════════════════════════════════════════════════

func (r *serviceRepo) CreatePlan(ctx context.Context, plan *models.ServicePlan) error {
	return r.db.QueryRow(ctx,
		`INSERT INTO service_plans
			(category_id, name, price, duration_days,
			 speed_download, speed_upload, data_cap,
			 connection_type, installation_fee,
			 router_included, ip_type, description,
			 cabinet_notes, features, is_active, sort_order, created_by)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14::jsonb,$15,$16,$17)
		 RETURNING id, created_at, updated_at`,
		plan.CategoryID, plan.Name, plan.Price, plan.DurationDays,
		plan.SpeedDownload, plan.SpeedUpload, plan.DataCap,
		plan.ConnectionType, plan.InstallationFee,
		plan.RouterIncluded, plan.IPType, plan.Description,
		plan.CabinetNotes, plan.Features, plan.IsActive, plan.SortOrder, plan.CreatedBy,
	).Scan(&plan.ID, &plan.CreatedAt, &plan.UpdatedAt)
}

func (r *serviceRepo) GetPlansByCategory(ctx context.Context, categoryID uuid.UUID) ([]models.ServicePlan, error) {
	query := `
		SELECT
			sp.id, sp.category_id, sp.name, sp.price, sp.duration_days,
			sp.speed_download, sp.speed_upload, sp.data_cap,
			sp.connection_type, sp.installation_fee,
			sp.router_included, sp.ip_type, sp.description,
			sp.cabinet_notes, sp.features, sp.is_active, sp.sort_order,
			sp.created_by, sp.created_at, sp.updated_at,
			e.first_name || ' ' || e.last_name AS creator_name,
			sc.name                             AS category_name
		FROM service_plans sp
		JOIN employees e          ON sp.created_by  = e.id
		JOIN service_categories sc ON sp.category_id = sc.id
		WHERE sp.category_id = $1
		ORDER BY sp.sort_order, sp.created_at DESC
	`

	rows, err := r.db.Query(ctx, query, categoryID)
	if err != nil {
		return nil, fmt.Errorf("get plans: %w", err)
	}
	defer rows.Close()

	var plans []models.ServicePlan
	for rows.Next() {
		var p models.ServicePlan
		if err := rows.Scan(
			&p.ID, &p.CategoryID, &p.Name, &p.Price, &p.DurationDays,
			&p.SpeedDownload, &p.SpeedUpload, &p.DataCap,
			&p.ConnectionType, &p.InstallationFee,
			&p.RouterIncluded, &p.IPType, &p.Description,
			&p.CabinetNotes, &p.Features, &p.IsActive, &p.SortOrder,
			&p.CreatedBy, &p.CreatedAt, &p.UpdatedAt,
			&p.CreatorName, &p.CategoryName,
		); err != nil {
			return nil, fmt.Errorf("scan plan: %w", err)
		}
		plans = append(plans, p)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate plans: %w", err)
	}
	return plans, nil
}

func (r *serviceRepo) GetPlanByID(ctx context.Context, id uuid.UUID) (*models.ServicePlan, error) {
	var p models.ServicePlan
	err := r.db.QueryRow(ctx,
		`SELECT
			sp.id, sp.category_id, sp.name, sp.price, sp.duration_days,
			sp.speed_download, sp.speed_upload, sp.data_cap,
			sp.connection_type, sp.installation_fee,
			sp.router_included, sp.ip_type, sp.description,
			sp.cabinet_notes, sp.features, sp.is_active, sp.sort_order,
			sp.created_by, sp.created_at, sp.updated_at,
			e.first_name || ' ' || e.last_name AS creator_name,
			sc.name                             AS category_name
		FROM service_plans sp
		JOIN employees e          ON sp.created_by  = e.id
		JOIN service_categories sc ON sp.category_id = sc.id
		WHERE sp.id = $1`, id,
	).Scan(
		&p.ID, &p.CategoryID, &p.Name, &p.Price, &p.DurationDays,
		&p.SpeedDownload, &p.SpeedUpload, &p.DataCap,
		&p.ConnectionType, &p.InstallationFee,
		&p.RouterIncluded, &p.IPType, &p.Description,
		&p.CabinetNotes, &p.Features, &p.IsActive, &p.SortOrder,
		&p.CreatedBy, &p.CreatedAt, &p.UpdatedAt,
		&p.CreatorName, &p.CategoryName,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("plan not found")
		}
		return nil, fmt.Errorf("get plan: %w", err)
	}
	return &p, nil
}

func (r *serviceRepo) UpdatePlan(ctx context.Context, plan *models.ServicePlan) error {
	_, err := r.db.Exec(ctx,
		`UPDATE service_plans SET
			name=$1, price=$2, duration_days=$3,
			speed_download=$4, speed_upload=$5, data_cap=$6,
			connection_type=$7, installation_fee=$8,
			router_included=$9, ip_type=$10, description=$11,
			cabinet_notes=$12, features=$13::jsonb, is_active=$14, sort_order=$15,
			updated_at=CURRENT_TIMESTAMP
		 WHERE id=$16`,
		plan.Name, plan.Price, plan.DurationDays,
		plan.SpeedDownload, plan.SpeedUpload, plan.DataCap,
		plan.ConnectionType, plan.InstallationFee,
		plan.RouterIncluded, plan.IPType, plan.Description,
		plan.CabinetNotes, plan.Features, plan.IsActive, plan.SortOrder,
		plan.ID)
	return err
}

func (r *serviceRepo) DeletePlan(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.Exec(ctx, `DELETE FROM service_plans WHERE id=$1`, id)
	return err
}
