package repository

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"shiftmaster-backend/internal/models"
	"shiftmaster-backend/pkg/database"
)

// AuditLogRepository defines the interface for audit log data access.
type AuditLogRepository interface {
	Create(ctx context.Context, log *models.AuditLog) error
	GetByTable(ctx context.Context, tableName string, limit int) ([]models.AuditLog, error)
	GetByEmployee(ctx context.Context, employeeID uuid.UUID, limit int) ([]models.AuditLog, error)
	GetByRecord(ctx context.Context, tableName string, recordID uuid.UUID) ([]models.AuditLog, error)
}

type auditLogRepo struct {
	db *database.DB
}

func NewAuditLogRepository(db *database.DB) AuditLogRepository {
	return &auditLogRepo{db: db}
}

const auditColumns = `id, employee_id, action, table_name, record_id, old_data, new_data, ip_address, user_agent, created_at`

func (r *auditLogRepo) scanAuditLogs(rows interface{ Next() bool; Scan(...interface{}) error; Err() error }) ([]models.AuditLog, error) {
	var logs []models.AuditLog
	for rows.Next() {
		var l models.AuditLog
		if err := rows.Scan(&l.ID, &l.EmployeeID, &l.Action, &l.TableName, &l.RecordID,
			&l.OldData, &l.NewData, &l.IPAddress, &l.UserAgent, &l.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan audit log: %w", err)
		}
		logs = append(logs, l)
	}
	return logs, rows.Err()
}

func (r *auditLogRepo) Create(ctx context.Context, log *models.AuditLog) error {
	return r.db.QueryRow(ctx,
		`INSERT INTO audit_logs (employee_id, action, table_name, record_id, old_data, new_data, ip_address, user_agent)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8) RETURNING id, created_at`,
		log.EmployeeID, log.Action, log.TableName, log.RecordID,
		log.OldData, log.NewData, log.IPAddress, log.UserAgent,
	).Scan(&log.ID, &log.CreatedAt)
}

func (r *auditLogRepo) GetByTable(ctx context.Context, tableName string, limit int) ([]models.AuditLog, error) {
	rows, err := r.db.Query(ctx,
		`SELECT `+auditColumns+` FROM audit_logs WHERE table_name = $1 ORDER BY created_at DESC LIMIT $2`, tableName, limit)
	if err != nil {
		return nil, fmt.Errorf("get audit logs by table: %w", err)
	}
	defer rows.Close()
	return r.scanAuditLogs(rows)
}

func (r *auditLogRepo) GetByEmployee(ctx context.Context, employeeID uuid.UUID, limit int) ([]models.AuditLog, error) {
	rows, err := r.db.Query(ctx,
		`SELECT `+auditColumns+` FROM audit_logs WHERE employee_id = $1 ORDER BY created_at DESC LIMIT $2`, employeeID, limit)
	if err != nil {
		return nil, fmt.Errorf("get audit logs by employee: %w", err)
	}
	defer rows.Close()
	return r.scanAuditLogs(rows)
}

func (r *auditLogRepo) GetByRecord(ctx context.Context, tableName string, recordID uuid.UUID) ([]models.AuditLog, error) {
	rows, err := r.db.Query(ctx,
		`SELECT `+auditColumns+` FROM audit_logs WHERE table_name = $1 AND record_id = $2 ORDER BY created_at DESC`, tableName, recordID)
	if err != nil {
		return nil, fmt.Errorf("get audit logs by record: %w", err)
	}
	defer rows.Close()
	return r.scanAuditLogs(rows)
}
