package repository

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"shiftmaster-backend/internal/models"
	"shiftmaster-backend/pkg/database"
)

// BoardRepository handles CRUD for task_boards.
type BoardRepository interface {
	GetAll(ctx context.Context, departmentID *uuid.UUID) ([]models.TaskBoard, error)
	GetByID(ctx context.Context, id uuid.UUID) (*models.TaskBoard, error)
	Create(ctx context.Context, b *models.TaskBoard) error
	Update(ctx context.Context, b *models.TaskBoard) error
	Delete(ctx context.Context, id uuid.UUID) error
}

type boardRepo struct {
	db *database.DB
}

func NewBoardRepository(db *database.DB) BoardRepository {
	return &boardRepo{db: db}
}

func (r *boardRepo) GetAll(ctx context.Context, departmentID *uuid.UUID) ([]models.TaskBoard, error) {
	query := `SELECT id, name, description, recurrence_type, is_active, department_id, created_by, created_at, updated_at
		 FROM task_boards WHERE ($1::uuid IS NULL OR department_id = $1) ORDER BY created_at DESC`
	rows, err := r.db.Query(ctx, query, departmentID)
	if err != nil {
		return nil, fmt.Errorf("get all boards: %w", err)
	}
	defer rows.Close()

	var boards []models.TaskBoard
	for rows.Next() {
		var b models.TaskBoard
		if err := rows.Scan(&b.ID, &b.Name, &b.Description, &b.RecurrenceType,
			&b.IsActive, &b.DepartmentID, &b.CreatedBy, &b.CreatedAt, &b.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan board: %w", err)
		}
		boards = append(boards, b)
	}
	return boards, rows.Err()
}

func (r *boardRepo) GetByID(ctx context.Context, id uuid.UUID) (*models.TaskBoard, error) {
	var b models.TaskBoard
	err := r.db.QueryRow(ctx,
		`SELECT id, name, description, recurrence_type, is_active, department_id, created_by, created_at, updated_at
		 FROM task_boards WHERE id = $1`, id,
	).Scan(&b.ID, &b.Name, &b.Description, &b.RecurrenceType,
		&b.IsActive, &b.DepartmentID, &b.CreatedBy, &b.CreatedAt, &b.UpdatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("board not found: %w", err)
		}
		return nil, fmt.Errorf("get board: %w", err)
	}
	return &b, nil
}

func (r *boardRepo) Create(ctx context.Context, b *models.TaskBoard) error {
	return r.db.QueryRow(ctx,
		`INSERT INTO task_boards (name, description, recurrence_type, is_active, department_id, created_by)
		 VALUES ($1, $2, $3, $4, $5, $6) RETURNING id, created_at, updated_at`,
		b.Name, b.Description, b.RecurrenceType, b.IsActive, b.DepartmentID, b.CreatedBy,
	).Scan(&b.ID, &b.CreatedAt, &b.UpdatedAt)
}

func (r *boardRepo) Update(ctx context.Context, b *models.TaskBoard) error {
	_, err := r.db.Exec(ctx,
		`UPDATE task_boards SET name=$1, description=$2, recurrence_type=$3, is_active=$4, department_id=$5, updated_at=CURRENT_TIMESTAMP
		 WHERE id=$6`,
		b.Name, b.Description, b.RecurrenceType, b.IsActive, b.DepartmentID, b.ID)
	return err
}

func (r *boardRepo) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.Exec(ctx, `DELETE FROM task_boards WHERE id=$1`, id)
	return err
}
