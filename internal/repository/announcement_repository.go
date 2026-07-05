package repository

import (
	"context"

	"shiftmaster-backend/internal/models"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type AnnouncementRepository interface {
	Create(ctx context.Context, announcement *models.Announcement) error
	GetActiveByDepartment(ctx context.Context, departmentID uuid.UUID) (*models.Announcement, error)
	GetActiveTickerByDepartment(ctx context.Context, departmentID uuid.UUID) (*models.Announcement, error)
	GetAllByDepartment(ctx context.Context, departmentID uuid.UUID) ([]models.Announcement, error)
	SetInactiveByDepartment(ctx context.Context, departmentID uuid.UUID) error
	SetInactive(ctx context.Context, id uuid.UUID, departmentID uuid.UUID) error
	SetTickerInactiveByDepartment(ctx context.Context, departmentID uuid.UUID) error
	SetActive(ctx context.Context, id uuid.UUID, departmentID uuid.UUID) error
	Delete(ctx context.Context, id uuid.UUID, departmentID uuid.UUID) error
}

type announcementRepository struct {
	db *pgxpool.Pool
}

func NewAnnouncementRepository(db *pgxpool.Pool) AnnouncementRepository {
	return &announcementRepository{db: db}
}

func (r *announcementRepository) Create(ctx context.Context, a *models.Announcement) error {
	query := `
		INSERT INTO announcements (department_id, title, message, priority, is_active, is_ticker, created_by, images)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, created_at, updated_at
	`
	err := r.db.QueryRow(ctx, query,
		a.DepartmentID,
		a.Title,
		a.Message,
		a.Priority,
		a.IsActive,
		a.IsTicker,
		a.CreatedBy,
		a.Images,
	).Scan(&a.ID, &a.CreatedAt, &a.UpdatedAt)
	return err
}

func (r *announcementRepository) GetActiveByDepartment(ctx context.Context, departmentID uuid.UUID) (*models.Announcement, error) {
	query := `
		SELECT a.id, a.department_id, a.title, a.message, a.priority, a.is_active, a.is_ticker, a.created_by, a.created_at, a.updated_at,
		       e.first_name || ' ' || e.last_name as creator_name,
		       COALESCE(a.images, '{}')
		FROM announcements a
		LEFT JOIN employees e ON a.created_by = e.id
		WHERE a.department_id = $1 AND a.is_active = true
		ORDER BY a.created_at DESC
		LIMIT 1
	`
	a := &models.Announcement{}
	err := r.db.QueryRow(ctx, query, departmentID).Scan(
		&a.ID,
		&a.DepartmentID,
		&a.Title,
		&a.Message,
		&a.Priority,
		&a.IsActive,
		&a.IsTicker,
		&a.CreatedBy,
		&a.CreatedAt,
		&a.UpdatedAt,
		&a.CreatorName,
		&a.Images,
	)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return a, nil
}

func (r *announcementRepository) GetActiveTickerByDepartment(ctx context.Context, departmentID uuid.UUID) (*models.Announcement, error) {
	query := `
		SELECT a.id, a.department_id, a.title, a.message, a.priority, a.is_active, a.is_ticker, a.created_by, a.created_at, a.updated_at,
		       e.first_name || ' ' || e.last_name as creator_name,
		       COALESCE(a.images, '{}')
		FROM announcements a
		LEFT JOIN employees e ON a.created_by = e.id
		WHERE a.department_id = $1 AND a.is_ticker = true
		ORDER BY a.created_at DESC
		LIMIT 1
	`
	a := &models.Announcement{}
	err := r.db.QueryRow(ctx, query, departmentID).Scan(
		&a.ID,
		&a.DepartmentID,
		&a.Title,
		&a.Message,
		&a.Priority,
		&a.IsActive,
		&a.IsTicker,
		&a.CreatedBy,
		&a.CreatedAt,
		&a.UpdatedAt,
		&a.CreatorName,
		&a.Images,
	)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return a, nil
}

func (r *announcementRepository) GetAllByDepartment(ctx context.Context, departmentID uuid.UUID) ([]models.Announcement, error) {
	query := `
		SELECT a.id, a.department_id, a.title, a.message, a.priority, a.is_active, a.is_ticker, a.created_by, a.created_at, a.updated_at,
		       e.first_name || ' ' || e.last_name as creator_name,
		       COALESCE(a.images, '{}')
		FROM announcements a
		LEFT JOIN employees e ON a.created_by = e.id
		WHERE a.department_id = $1
		ORDER BY a.created_at DESC
	`
	rows, err := r.db.Query(ctx, query, departmentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []models.Announcement
	for rows.Next() {
		var a models.Announcement
		err := rows.Scan(
			&a.ID,
			&a.DepartmentID,
			&a.Title,
			&a.Message,
			&a.Priority,
			&a.IsActive,
			&a.IsTicker,
			&a.CreatedBy,
			&a.CreatedAt,
			&a.UpdatedAt,
			&a.CreatorName,
			&a.Images,
		)
		if err != nil {
			return nil, err
		}
		list = append(list, a)
	}
	return list, nil
}

func (r *announcementRepository) SetInactiveByDepartment(ctx context.Context, departmentID uuid.UUID) error {
	query := `UPDATE announcements SET is_active = false WHERE department_id = $1`
	_, err := r.db.Exec(ctx, query, departmentID)
	return err
}

func (r *announcementRepository) SetInactive(ctx context.Context, id uuid.UUID, departmentID uuid.UUID) error {
	query := `UPDATE announcements SET is_active = false, is_ticker = false WHERE id = $1 AND department_id = $2`
	_, err := r.db.Exec(ctx, query, id, departmentID)
	return err
}

func (r *announcementRepository) SetTickerInactiveByDepartment(ctx context.Context, departmentID uuid.UUID) error {
	query := `UPDATE announcements SET is_ticker = false WHERE department_id = $1`
	_, err := r.db.Exec(ctx, query, departmentID)
	return err
}

func (r *announcementRepository) SetActive(ctx context.Context, id uuid.UUID, departmentID uuid.UUID) error {
	query := `UPDATE announcements SET is_active = true WHERE id = $1 AND department_id = $2`
	_, err := r.db.Exec(ctx, query, id, departmentID)
	return err
}

func (r *announcementRepository) Delete(ctx context.Context, id uuid.UUID, departmentID uuid.UUID) error {
	query := `DELETE FROM announcements WHERE id = $1 AND department_id = $2`
	_, err := r.db.Exec(ctx, query, id, departmentID)
	return err
}
