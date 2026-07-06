package repository

import (
	"context"
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
		 RETURNING id, created_at, updated_at`,
		t.SourceDepartmentID, t.TargetDepartmentID, t.CreatorID, t.Title, t.Description, t.Attachments,
	).Scan(&t.ID, &t.CreatedAt, &t.UpdatedAt)
}

func (r *ticketRepo) GetTicketsForDepartment(ctx context.Context, departmentID uuid.UUID) ([]models.Ticket, error) {
	query := `
		SELECT t.id, t.source_department_id, t.target_department_id, t.creator_id, t.title, t.description, t.status, t.closed_by, t.attachments, t.created_at, t.updated_at,
		       e.first_name || ' ' || e.last_name as creator_name, COALESCE(e.profile_image, '') as creator_profile_image,
		       sd.name as source_department, td.name as target_department,
		       COALESCE(cb.first_name || ' ' || cb.last_name, '') as closed_by_name
		FROM tickets t
		JOIN employees e ON t.creator_id = e.id
		JOIN departments sd ON t.source_department_id = sd.id
		JOIN departments td ON t.target_department_id = td.id
		LEFT JOIN employees cb ON t.closed_by = cb.id
		WHERE t.source_department_id = $1 OR t.target_department_id = $1
		ORDER BY CASE WHEN t.status = 'open' THEN 0 ELSE 1 END, t.created_at DESC
	`
	rows, err := r.db.Query(ctx, query, departmentID)
	if err != nil {
		return nil, fmt.Errorf("get tickets: %w", err)
	}
	defer rows.Close()

	var tickets []models.Ticket
	for rows.Next() {
		var t models.Ticket
		if err := rows.Scan(
			&t.ID, &t.SourceDepartmentID, &t.TargetDepartmentID, &t.CreatorID, &t.Title, &t.Description, &t.Status, &t.ClosedBy, &t.Attachments, &t.CreatedAt, &t.UpdatedAt,
			&t.CreatorName, &t.CreatorProfileImage, &t.SourceDepartment, &t.TargetDepartment, &t.ClosedByName,
		); err != nil {
			return nil, fmt.Errorf("scan ticket: %w", err)
		}
		
		// Load comments
		cQuery := `
			SELECT tc.id, tc.ticket_id, tc.employee_id, tc.comment, tc.attachments, tc.created_at,
			       e.first_name || ' ' || e.last_name as author_name, COALESCE(e.profile_image, '') as author_image
			FROM ticket_comments tc
			JOIN employees e ON tc.employee_id = e.id
			WHERE tc.ticket_id = $1
			ORDER BY tc.created_at ASC
		`
		cRows, err := r.db.Query(ctx, cQuery, t.ID)
		if err == nil {
			var comments []models.TicketComment
			for cRows.Next() {
				var c models.TicketComment
				if err := cRows.Scan(&c.ID, &c.TicketID, &c.EmployeeID, &c.Comment, &c.Attachments, &c.CreatedAt, &c.AuthorName, &c.AuthorImage); err == nil {
					comments = append(comments, c)
				}
			}
			t.Comments = comments
			cRows.Close()
		}

		tickets = append(tickets, t)
	}

	return tickets, nil
}

func (r *ticketRepo) GetTicketByID(ctx context.Context, id uuid.UUID) (*models.Ticket, error) {
	query := `
		SELECT t.id, t.source_department_id, t.target_department_id, t.creator_id, t.title, t.description, t.status, t.closed_by, t.attachments, t.created_at, t.updated_at,
		       e.first_name || ' ' || e.last_name as creator_name, COALESCE(e.profile_image, '') as creator_profile_image,
		       sd.name as source_department, td.name as target_department,
		       COALESCE(cb.first_name || ' ' || cb.last_name, '') as closed_by_name
		FROM tickets t
		JOIN employees e ON t.creator_id = e.id
		JOIN departments sd ON t.source_department_id = sd.id
		JOIN departments td ON t.target_department_id = td.id
		LEFT JOIN employees cb ON t.closed_by = cb.id
		WHERE t.id = $1
	`
	var t models.Ticket
	if err := r.db.QueryRow(ctx, query, id).Scan(
		&t.ID, &t.SourceDepartmentID, &t.TargetDepartmentID, &t.CreatorID, &t.Title, &t.Description, &t.Status, &t.ClosedBy, &t.Attachments, &t.CreatedAt, &t.UpdatedAt,
		&t.CreatorName, &t.CreatorProfileImage, &t.SourceDepartment, &t.TargetDepartment, &t.ClosedByName,
	); err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("ticket not found")
		}
		return nil, fmt.Errorf("get ticket by id: %w", err)
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
	return r.db.QueryRow(ctx,
		`INSERT INTO ticket_comments (ticket_id, employee_id, comment, attachments)
		 VALUES ($1, $2, $3, $4::jsonb)
		 RETURNING id, created_at`,
		comment.TicketID, comment.EmployeeID, comment.Comment, comment.Attachments,
	).Scan(&comment.ID, &comment.CreatedAt)
}
