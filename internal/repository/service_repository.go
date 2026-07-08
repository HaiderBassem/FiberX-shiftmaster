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
			sc.id, sc.province_id, sc.name, sc.description, sc.is_active, sc.disabled_at, sc.sort_order,
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
			&c.ID, &c.ProvinceID, &c.Name, &c.Description, &c.IsActive, &c.DisabledAt, &c.SortOrder,
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
			sc.id, sc.province_id, sc.name, sc.description, sc.is_active, sc.disabled_at, sc.sort_order,
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
		&c.ID, &c.ProvinceID, &c.Name, &c.Description, &c.IsActive, &c.DisabledAt, &c.SortOrder,
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
		 SET name=$1, description=$2, is_active=$3, disabled_at=$4, sort_order=$5, updated_at=CURRENT_TIMESTAMP
		 WHERE id=$6`,
		cat.Name, cat.Description, cat.IsActive, cat.DisabledAt, cat.SortOrder, cat.ID)
	return err
}

func (r *serviceRepo) DeleteCategory(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.Exec(ctx, `DELETE FROM service_categories WHERE id=$1`, id)
	return err
}



// ═══════════════════════════════════════════════════════════
// Plans
// ═══════════════════════════════════════════════════════════

func (r *serviceRepo) CreatePlan(ctx context.Context, plan *models.ServicePlan) error {
	return r.db.QueryRow(ctx,
		`INSERT INTO service_plans
			(category_id, name, price, duration_days,
			 speed, data_cap,
			 connection_type, installation_fee,
			 router_included, description,
			 cabinet_notes, features, is_active, sort_order, created_by)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12::jsonb,$13,$14,$15)
		 RETURNING id, created_at, updated_at`,
		plan.CategoryID, plan.Name, plan.Price, plan.DurationDays,
		plan.Speed, plan.DataCap,
		plan.ConnectionType, plan.InstallationFee,
		plan.RouterIncluded, plan.Description,
		plan.CabinetNotes, plan.Features, plan.IsActive, plan.SortOrder, plan.CreatedBy,
	).Scan(&plan.ID, &plan.CreatedAt, &plan.UpdatedAt)
}

func (r *serviceRepo) GetPlansByCategory(ctx context.Context, categoryID uuid.UUID) ([]models.ServicePlan, error) {
	query := `
		SELECT
			sp.id, sp.category_id, sp.name, sp.price, sp.duration_days,
			sp.speed, sp.data_cap,
			sp.connection_type, sp.installation_fee,
			sp.router_included, sp.description,
			sp.cabinet_notes, sp.features, sp.is_active, sp.disabled_at, sp.sort_order,
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
			&p.Speed, &p.DataCap,
			&p.ConnectionType, &p.InstallationFee,
			&p.RouterIncluded, &p.Description,
			&p.CabinetNotes, &p.Features, &p.IsActive, &p.DisabledAt, &p.SortOrder,
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
			sp.speed, sp.data_cap,
			sp.connection_type, sp.installation_fee,
			sp.router_included, sp.description,
			sp.cabinet_notes, sp.features, sp.is_active, sp.disabled_at, sp.sort_order,
			sp.created_by, sp.created_at, sp.updated_at,
			e.first_name || ' ' || e.last_name AS creator_name,
			sc.name                             AS category_name
		FROM service_plans sp
		JOIN employees e          ON sp.created_by  = e.id
		JOIN service_categories sc ON sp.category_id = sc.id
		WHERE sp.id = $1`, id,
	).Scan(
		&p.ID, &p.CategoryID, &p.Name, &p.Price, &p.DurationDays,
		&p.Speed, &p.DataCap,
		&p.ConnectionType, &p.InstallationFee,
		&p.RouterIncluded, &p.Description,
		&p.CabinetNotes, &p.Features, &p.IsActive, &p.DisabledAt, &p.SortOrder,
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
			speed=$4, data_cap=$5,
			connection_type=$6, installation_fee=$7,
			router_included=$8, description=$9,
			cabinet_notes=$10, features=$11::jsonb, is_active=$12, disabled_at=$13, sort_order=$14,
			updated_at=CURRENT_TIMESTAMP
		 WHERE id=$15`,
		plan.Name, plan.Price, plan.DurationDays,
		plan.Speed, plan.DataCap,
		plan.ConnectionType, plan.InstallationFee,
		plan.RouterIncluded, plan.Description,
		plan.CabinetNotes, plan.Features, plan.IsActive, plan.DisabledAt, plan.SortOrder,
		plan.ID)
	return err
}

func (r *serviceRepo) DeletePlan(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.Exec(ctx, `DELETE FROM service_plans WHERE id=$1`, id)
	return err
}
