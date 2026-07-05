package repository

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"

	"shiftmaster-backend/internal/models"
	"shiftmaster-backend/pkg/database"
)

type HandoverRepository interface {
	Create(ctx context.Context, handover *models.Handover) error
	GetByDepartment(ctx context.Context, departmentID uuid.UUID) ([]models.Handover, error)
	Claim(ctx context.Context, id, employeeID uuid.UUID) error
	Unclaim(ctx context.Context, id, employeeID uuid.UUID) error
	AddComment(ctx context.Context, id, employeeID uuid.UUID, comment string) error
	Complete(ctx context.Context, id, employeeID uuid.UUID) error
}

type handoverRepo struct {
	db *database.DB
}

func NewHandoverRepository(db *database.DB) HandoverRepository {
	return &handoverRepo{db: db}
}

func (r *handoverRepo) Create(ctx context.Context, handover *models.Handover) error {
	query := `
		INSERT INTO shift_handovers (department_id, creator_id, shift_summary, pending_issues, status)
		VALUES ($1, $2, $3, $4, 'open')
		RETURNING id, created_at, updated_at
	`
	return r.db.QueryRow(ctx, query, handover.DepartmentID, handover.CreatorID, handover.ShiftSummary, handover.PendingIssues).
		Scan(&handover.ID, &handover.CreatedAt, &handover.UpdatedAt)
}

	func (r *handoverRepo) GetByDepartment(ctx context.Context, departmentID uuid.UUID) ([]models.Handover, error) {
	query := `
		SELECT h.id, h.department_id, h.creator_id, h.shift_summary, h.pending_issues, h.status, h.claimed_by, h.done_by, h.created_at, h.updated_at,
		       c.first_name || ' ' || c.last_name as creator_name,
		       cl.first_name || ' ' || cl.last_name as claimer_name,
		       d.first_name || ' ' || d.last_name as done_by_name,
		       COALESCE(
		           (
		               SELECT json_agg(json_build_object(
		                   'id', hc.id,
		                   'employee_id', hc.employee_id,
		                   'author_name', e.first_name || ' ' || e.last_name,
		                   'comment', hc.comment,
		                   'created_at', hc.created_at
		               ) ORDER BY hc.created_at ASC)
		               FROM handover_comments hc
		               JOIN employees e ON e.id = hc.employee_id
		               WHERE hc.handover_id = h.id
		           ), '[]'::json
		       ) as comments
		FROM shift_handovers h
		JOIN employees c ON c.id = h.creator_id
		LEFT JOIN employees cl ON cl.id = h.claimed_by
		LEFT JOIN employees d ON d.id = h.done_by
		WHERE h.department_id = $1
		ORDER BY h.created_at DESC
	`
	rows, err := r.db.Query(ctx, query, departmentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var handovers []models.Handover
	for rows.Next() {
		var h models.Handover
		var commentsJSON []byte
		if err := rows.Scan(
			&h.ID, &h.DepartmentID, &h.CreatorID, &h.ShiftSummary, &h.PendingIssues, &h.Status, &h.ClaimedBy, &h.DoneBy, &h.CreatedAt, &h.UpdatedAt,
			&h.CreatorName, &h.ClaimerName, &h.DoneByName, &commentsJSON,
		); err != nil {
			return nil, err
		}
		
		if err := json.Unmarshal(commentsJSON, &h.Comments); err != nil {
			// ignore unmarshal errors for empty/invalid json
			h.Comments = []models.HandoverComment{}
		}
		
		handovers = append(handovers, h)
	}
	return handovers, rows.Err()
}

func (r *handoverRepo) Claim(ctx context.Context, id, employeeID uuid.UUID) error {
	query := `
		UPDATE shift_handovers
		SET status = 'claimed', claimed_by = $1, updated_at = CURRENT_TIMESTAMP
		WHERE id = $2 AND status = 'open'
	`
	_, err := r.db.Exec(ctx, query, employeeID, id)
	return err
}

func (r *handoverRepo) Unclaim(ctx context.Context, id, employeeID uuid.UUID) error {
	query := `
		UPDATE shift_handovers
		SET status = 'open', claimed_by = NULL, updated_at = CURRENT_TIMESTAMP
		WHERE id = $1 AND claimed_by = $2 AND status = 'claimed'
	`
	_, err := r.db.Exec(ctx, query, id, employeeID)
	return err
}

func (r *handoverRepo) AddComment(ctx context.Context, id, employeeID uuid.UUID, comment string) error {
	query := `
		INSERT INTO handover_comments (handover_id, employee_id, comment)
		VALUES ($1, $2, $3)
	`
	_, err := r.db.Exec(ctx, query, id, employeeID, comment)
	if err == nil {
		_, _ = r.db.Exec(ctx, "UPDATE shift_handovers SET updated_at = CURRENT_TIMESTAMP WHERE id = $1", id)
	}
	return err
}

func (r *handoverRepo) Complete(ctx context.Context, id, employeeID uuid.UUID) error {
	query := `
		UPDATE shift_handovers
		SET status = 'completed', done_by = $1, updated_at = CURRENT_TIMESTAMP
		WHERE id = $2
	`
	_, err := r.db.Exec(ctx, query, employeeID, id)
	return err
}
