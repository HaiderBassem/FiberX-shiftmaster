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
	GetAll(ctx context.Context) ([]models.Province, error)
	GetByID(ctx context.Context, id uuid.UUID) (*models.Province, error)
	GetByName(ctx context.Context, name string) (*models.Province, error)
	Create(ctx context.Context, province *models.Province) error
	Update(ctx context.Context, province *models.Province) error
	Delete(ctx context.Context, id uuid.UUID) error
}

type provinceRepo struct {
	db *database.DB
}

func NewProvinceRepository(db *database.DB) ProvinceRepository {
	return &provinceRepo{db: db}
}

func (r *provinceRepo) GetAll(ctx context.Context) ([]models.Province, error) {
	query := `
		SELECT id, name, sort_order, is_active, created_at, updated_at
		FROM provinces
		ORDER BY sort_order ASC, name ASC
	`
	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("query provinces: %w", err)
	}
	defer rows.Close()

	var provinces []models.Province
	for rows.Next() {
		var p models.Province
		if err := rows.Scan(&p.ID, &p.Name, &p.SortOrder, &p.IsActive, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan province: %w", err)
		}
		provinces = append(provinces, p)
	}

	return provinces, nil
}

func (r *provinceRepo) GetByID(ctx context.Context, id uuid.UUID) (*models.Province, error) {
	query := `
		SELECT id, name, sort_order, is_active, created_at, updated_at
		FROM provinces
		WHERE id = $1
	`
	var p models.Province
	err := r.db.QueryRow(ctx, query, id).Scan(
		&p.ID, &p.Name, &p.SortOrder, &p.IsActive, &p.CreatedAt, &p.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("get province by id: %w", err)
	}
	return &p, nil
}

func (r *provinceRepo) GetByName(ctx context.Context, name string) (*models.Province, error) {
	query := `
		SELECT id, name, sort_order, is_active, created_at, updated_at
		FROM provinces
		WHERE name = $1
	`
	var p models.Province
	err := r.db.QueryRow(ctx, query, name).Scan(
		&p.ID, &p.Name, &p.SortOrder, &p.IsActive, &p.CreatedAt, &p.UpdatedAt,
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
		INSERT INTO provinces (name, sort_order, is_active)
		VALUES ($1, $2, $3)
		RETURNING id, created_at, updated_at
	`
	err := r.db.QueryRow(ctx, query, province.Name, province.SortOrder, province.IsActive).Scan(
		&province.ID, &province.CreatedAt, &province.UpdatedAt,
	)
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
	cmd, err := r.db.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("delete province: %w", err)
	}
	if cmd.RowsAffected() == 0 {
		return fmt.Errorf("province not found")
	}
	return nil
}
