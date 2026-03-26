package service

import (
	"context"

	"github.com/google/uuid"

	"shiftmaster-backend/internal/models"
	"shiftmaster-backend/internal/repository"
)

// AuditService provides read access to audit logs (activity history).
type AuditService struct {
	auditRepo repository.AuditLogRepository
}

func NewAuditService(auditRepo repository.AuditLogRepository) *AuditService {
	return &AuditService{auditRepo: auditRepo}
}

// GetActivityForEmployee returns recent audit logs for an employee.
func (s *AuditService) GetActivityForEmployee(ctx context.Context, employeeID uuid.UUID, limit int) ([]models.AuditLog, error) {
	return s.auditRepo.GetByEmployee(ctx, employeeID, limit)
}

