package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"shiftmaster-backend/internal/models"
	"shiftmaster-backend/pkg/database"
)

type TicketRepository interface {
	Create(ctx context.Context, t *models.Ticket) error
	GetTicketsForDepartment(ctx context.Context, departmentID uuid.UUID) ([]models.Ticket, error)
	GetTicketByID(ctx context.Context, id uuid.UUID) (*models.Ticket, error)
	UpdateStatus(ctx context.Context, id uuid.UUID, status string, closedBy *uuid.UUID) error
	AddComment(ctx context.Context, comment *models.TicketComment) error
}

type ticketRepo struct {
	db *database.DB
}

func NewTicketRepository(db *database.DB) TicketRepository {
	return &ticketRepo{db: db}
}

func (r *ticketRepo) Create(ctx context.Context, t *models.Ticket) error {
	return r.db.QueryRow(ctx,
		`INSERT INTO tickets (source_department_id, target_department_id, creator_id, title, description, attachments)
		 VALUES ($1, $2, $3, $4, $5, $6::jsonb)
		 RETURNING id, status, created_at, updated_at`,
		t.SourceDepartmentID, t.TargetDepartmentID, t.CreatorID, t.Title, t.Description, t.Attachments,
	).Scan(&t.ID, &t.Status, &t.CreatedAt, &t.UpdatedAt)
}

func (r *ticketRepo) GetTicketsForDepartment(ctx context.Context, departmentID uuid.UUID) ([]models.Ticket, error) {
	query := `
		SELECT
			t.id, t.source_department_id, t.target_department_id, t.creator_id,
			t.title, t.description, t.status, t.closed_by, t.attachments,
			t.created_at, t.updated_at,
			e.first_name || ' ' || e.last_name AS creator_name,
			COALESCE(e.profile_image, '')       AS creator_profile_image,
			sd.name                             AS source_department,
			td.name                             AS target_department,
			cb.first_name || ' ' || cb.last_name AS closed_by_name
		FROM tickets t
		JOIN employees   e  ON t.creator_id           = e.id
		JOIN departments sd ON t.source_department_id  = sd.id
		JOIN departments td ON t.target_department_id  = td.id
		LEFT JOIN employees cb ON t.closed_by = cb.id
		WHERE t.source_department_id = $1 OR t.target_department_id = $1
		ORDER BY
			CASE WHEN t.status = 'open' THEN 0 ELSE 1 END,
			t.created_at DESC
	`

	rows, err := r.db.Query(ctx, query, departmentID)
	if err != nil {
		return nil, fmt.Errorf("get tickets: %w", err)
	}
	defer rows.Close()

	var tickets []models.Ticket
	for rows.Next() {
		var t models.Ticket
		var closedByName sql.NullString

		if err := rows.Scan(
			&t.ID, &t.SourceDepartmentID, &t.TargetDepartmentID, &t.CreatorID,
			&t.Title, &t.Description, &t.Status, &t.ClosedBy, &t.Attachments,
			&t.CreatedAt, &t.UpdatedAt,
			&t.CreatorName, &t.CreatorProfileImage,
			&t.SourceDepartment, &t.TargetDepartment,
			&closedByName,
		); err != nil {
			return nil, fmt.Errorf("scan ticket: %w", err)
		}

		if closedByName.Valid {
			t.ClosedByName = &closedByName.String
		}

		// Load comments for this ticket
		comments, err := r.getCommentsByTicketID(ctx, t.ID)
		if err != nil {
			return nil, fmt.Errorf("load comments for ticket %s: %w", t.ID, err)
		}
		t.Comments = comments

		tickets = append(tickets, t)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate tickets: %w", err)
	}

	return tickets, nil
}

func (r *ticketRepo) GetTicketByID(ctx context.Context, id uuid.UUID) (*models.Ticket, error) {
	query := `
		SELECT
			t.id, t.source_department_id, t.target_department_id, t.creator_id,
			t.title, t.description, t.status, t.closed_by, t.attachments,
			t.created_at, t.updated_at,
			e.first_name || ' ' || e.last_name AS creator_name,
			COALESCE(e.profile_image, '')       AS creator_profile_image,
			sd.name                             AS source_department,
			td.name                             AS target_department,
			cb.first_name || ' ' || cb.last_name AS closed_by_name
		FROM tickets t
		JOIN employees   e  ON t.creator_id           = e.id
		JOIN departments sd ON t.source_department_id  = sd.id
		JOIN departments td ON t.target_department_id  = td.id
		LEFT JOIN employees cb ON t.closed_by = cb.id
		WHERE t.id = $1
	`

	var t models.Ticket
	var closedByName sql.NullString

	if err := r.db.QueryRow(ctx, query, id).Scan(
		&t.ID, &t.SourceDepartmentID, &t.TargetDepartmentID, &t.CreatorID,
		&t.Title, &t.Description, &t.Status, &t.ClosedBy, &t.Attachments,
		&t.CreatedAt, &t.UpdatedAt,
		&t.CreatorName, &t.CreatorProfileImage,
		&t.SourceDepartment, &t.TargetDepartment,
		&closedByName,
	); err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("ticket not found")
		}
		return nil, fmt.Errorf("get ticket by id: %w", err)
	}

	if closedByName.Valid {
		t.ClosedByName = &closedByName.String
	}

	return &t, nil
}

func (r *ticketRepo) UpdateStatus(ctx context.Context, id uuid.UUID, status string, closedBy *uuid.UUID) error {
	_, err := r.db.Exec(ctx,
		`UPDATE tickets SET status=$1, closed_by=$2, updated_at=CURRENT_TIMESTAMP WHERE id=$3`,
		status, closedBy, id)
	return err
}

func (r *ticketRepo) AddComment(ctx context.Context, comment *models.TicketComment) error {
	// Insert the comment and JOIN back to get author info in one query
	return r.db.QueryRow(ctx,
		`WITH inserted AS (
			INSERT INTO ticket_comments (ticket_id, employee_id, comment, attachments)
			VALUES ($1, $2, $3, $4::jsonb)
			RETURNING id, ticket_id, employee_id, comment, attachments, created_at
		)
		SELECT i.id, i.ticket_id, i.employee_id, i.comment, i.attachments, i.created_at,
		       e.first_name || ' ' || e.last_name AS author_name,
		       COALESCE(e.profile_image, '')       AS author_image
		FROM inserted i
		JOIN employees e ON i.employee_id = e.id`,
		comment.TicketID, comment.EmployeeID, comment.Comment, comment.Attachments,
	).Scan(
		&comment.ID, &comment.TicketID, &comment.EmployeeID,
		&comment.Comment, &comment.Attachments, &comment.CreatedAt,
		&comment.AuthorName, &comment.AuthorImage,
	)
}

// getCommentsByTicketID loads all comments for a ticket with author info.
func (r *ticketRepo) getCommentsByTicketID(ctx context.Context, ticketID uuid.UUID) ([]models.TicketComment, error) {
	query := `
		SELECT
			tc.id, tc.ticket_id, tc.employee_id, tc.comment, tc.attachments, tc.created_at,
			e.first_name || ' ' || e.last_name AS author_name,
			COALESCE(e.profile_image, '')       AS author_image
		FROM ticket_comments tc
		JOIN employees e ON tc.employee_id = e.id
		WHERE tc.ticket_id = $1
		ORDER BY tc.created_at ASC
	`

	rows, err := r.db.Query(ctx, query, ticketID)
	if err != nil {
		return nil, fmt.Errorf("get comments: %w", err)
	}
	defer rows.Close()

	var comments []models.TicketComment
	for rows.Next() {
		var c models.TicketComment
		if err := rows.Scan(
			&c.ID, &c.TicketID, &c.EmployeeID,
			&c.Comment, &c.Attachments, &c.CreatedAt,
			&c.AuthorName, &c.AuthorImage,
		); err != nil {
			return nil, fmt.Errorf("scan comment: %w", err)
		}
		comments = append(comments, c)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate comments: %w", err)
	}

	return comments, nil
}
