package service

import (
	"context"

	"github.com/google/uuid"

	"shiftmaster-backend/internal/models"
	"shiftmaster-backend/internal/repository"
)

type LeaveTypeService interface {
	GetAllLeaveTypes(ctx context.Context) ([]models.LeaveType, error)
	GetActiveLeaveTypes(ctx context.Context) ([]models.LeaveType, error)
	GetLeaveTypeByID(ctx context.Context, id uuid.UUID) (*models.LeaveType, error)
	CreateLeaveType(ctx context.Context, lt *models.LeaveType) error
	UpdateLeaveType(ctx context.Context, lt *models.LeaveType) error
	DeleteLeaveType(ctx context.Context, id uuid.UUID) error
}

type leaveTypeService struct {
	repo repository.LeaveTypeRepository
}

func NewLeaveTypeService(repo repository.LeaveTypeRepository) LeaveTypeService {
	return &leaveTypeService{repo: repo}
}

func (s *leaveTypeService) GetAllLeaveTypes(ctx context.Context) ([]models.LeaveType, error) {
	return s.repo.GetAll(ctx)
}

func (s *leaveTypeService) GetActiveLeaveTypes(ctx context.Context) ([]models.LeaveType, error) {
	return s.repo.GetActive(ctx)
}

func (s *leaveTypeService) GetLeaveTypeByID(ctx context.Context, id uuid.UUID) (*models.LeaveType, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *leaveTypeService) CreateLeaveType(ctx context.Context, lt *models.LeaveType) error {
	return s.repo.Create(ctx, lt)
}

func (s *leaveTypeService) UpdateLeaveType(ctx context.Context, lt *models.LeaveType) error {
	return s.repo.Update(ctx, lt)
}

func (s *leaveTypeService) DeleteLeaveType(ctx context.Context, id uuid.UUID) error {
	return s.repo.Delete(ctx, id)
}
