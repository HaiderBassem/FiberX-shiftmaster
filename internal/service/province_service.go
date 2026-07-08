package service

import (
	"context"
	"errors"

	"github.com/google/uuid"

	"shiftmaster-backend/internal/models"
	"shiftmaster-backend/internal/repository"
)

var (
	ErrProvinceNotFound     = errors.New("province not found")
	ErrProvinceNameRequired = errors.New("province name is required")
	ErrProvinceExists       = errors.New("province name already exists")
)

type ProvinceService interface {
	GetAll(ctx context.Context, departmentID uuid.UUID) ([]models.Province, error)
	Create(ctx context.Context, departmentID uuid.UUID, empID uuid.UUID, req CreateProvinceRequest) (*models.Province, error)
	Update(ctx context.Context, id uuid.UUID, departmentID uuid.UUID, req UpdateProvinceRequest) (*models.Province, error)
	Delete(ctx context.Context, id uuid.UUID, departmentID uuid.UUID) error

	// Sharing
	ShareProvince(ctx context.Context, share *models.ProvinceShare) error
	UnshareProvince(ctx context.Context, provinceID, departmentID uuid.UUID) error
	GetProvinceShares(ctx context.Context, provinceID uuid.UUID) ([]models.ProvinceShare, error)
}

type CreateProvinceRequest struct {
	Name      string `json:"name"`
	SortOrder int    `json:"sort_order"`
	IsActive  bool   `json:"is_active"`
}

type UpdateProvinceRequest struct {
	Name      string `json:"name"`
	SortOrder int    `json:"sort_order"`
	IsActive  bool   `json:"is_active"`
}

type provinceService struct {
	repo repository.ProvinceRepository
}

func NewProvinceService(repo repository.ProvinceRepository) ProvinceService {
	return &provinceService{repo: repo}
}

func (s *provinceService) GetAll(ctx context.Context, departmentID uuid.UUID) ([]models.Province, error) {
	return s.repo.GetAll(ctx, departmentID)
}

func (s *provinceService) Create(ctx context.Context, departmentID uuid.UUID, empID uuid.UUID, req CreateProvinceRequest) (*models.Province, error) {
	if req.Name == "" {
		return nil, ErrProvinceNameRequired
	}

	existing, err := s.repo.GetByName(ctx, req.Name, departmentID)
	if err != nil && err.Error() != "sql: no rows in result set" {
		// handle existing properly if it's not a no-rows error
	}
	if existing != nil && existing.ID != uuid.Nil {
		return nil, ErrProvinceExists
	}

	province := &models.Province{
		DepartmentID: departmentID,
		Name:         req.Name,
		SortOrder:    req.SortOrder,
		IsActive:     req.IsActive,
		CreatedBy:    empID,
	}

	if err := s.repo.Create(ctx, province); err != nil {
		return nil, err
	}

	return province, nil
}

func (s *provinceService) Update(ctx context.Context, id uuid.UUID, departmentID uuid.UUID, req UpdateProvinceRequest) (*models.Province, error) {
	if req.Name == "" {
		return nil, ErrProvinceNameRequired
	}

	province, err := s.repo.GetByID(ctx, id, departmentID)
	if err != nil {
		return nil, err
	}
	if province == nil {
		return nil, ErrProvinceNotFound
	}

	if req.Name != province.Name {
		existing, _ := s.repo.GetByName(ctx, req.Name, departmentID)
		if existing != nil && existing.ID != id {
			return nil, ErrProvinceExists
		}
	}

	province.Name = req.Name
	province.SortOrder = req.SortOrder
	province.IsActive = req.IsActive

	if err := s.repo.Update(ctx, province); err != nil {
		return nil, err
	}

	return province, nil
}

func (s *provinceService) Delete(ctx context.Context, id uuid.UUID, departmentID uuid.UUID) error {
	province, err := s.repo.GetByID(ctx, id, departmentID)
	if err != nil {
		return err
	}
	if province == nil {
		return ErrProvinceNotFound
	}
	if province.IsShared {
		return errors.New("cannot delete shared province")
	}

	return s.repo.Delete(ctx, id)
}

func (s *provinceService) ShareProvince(ctx context.Context, share *models.ProvinceShare) error {
	return s.repo.ShareProvince(ctx, share)
}

func (s *provinceService) UnshareProvince(ctx context.Context, provinceID, departmentID uuid.UUID) error {
	return s.repo.UnshareProvince(ctx, provinceID, departmentID)
}

func (s *provinceService) GetProvinceShares(ctx context.Context, provinceID uuid.UUID) ([]models.ProvinceShare, error) {
	return s.repo.GetProvinceShares(ctx, provinceID)
}
