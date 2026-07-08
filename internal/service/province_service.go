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
	GetAll(ctx context.Context) ([]models.Province, error)
	Create(ctx context.Context, req CreateProvinceRequest) (*models.Province, error)
	Update(ctx context.Context, id uuid.UUID, req UpdateProvinceRequest) (*models.Province, error)
	Delete(ctx context.Context, id uuid.UUID) error
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

func (s *provinceService) GetAll(ctx context.Context) ([]models.Province, error) {
	return s.repo.GetAll(ctx)
}

func (s *provinceService) Create(ctx context.Context, req CreateProvinceRequest) (*models.Province, error) {
	if req.Name == "" {
		return nil, ErrProvinceNameRequired
	}

	existing, err := s.repo.GetByName(ctx, req.Name)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, ErrProvinceExists
	}

	province := &models.Province{
		Name:      req.Name,
		SortOrder: req.SortOrder,
		IsActive:  req.IsActive,
	}

	if err := s.repo.Create(ctx, province); err != nil {
		return nil, err
	}

	return province, nil
}

func (s *provinceService) Update(ctx context.Context, id uuid.UUID, req UpdateProvinceRequest) (*models.Province, error) {
	if req.Name == "" {
		return nil, ErrProvinceNameRequired
	}

	province, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if province == nil {
		return nil, ErrProvinceNotFound
	}

	if req.Name != province.Name {
		existing, err := s.repo.GetByName(ctx, req.Name)
		if err != nil {
			return nil, err
		}
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

func (s *provinceService) Delete(ctx context.Context, id uuid.UUID) error {
	province, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if province == nil {
		return ErrProvinceNotFound
	}

	return s.repo.Delete(ctx, id)
}
