package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"shiftmaster-backend/internal/models"
	"shiftmaster-backend/internal/repository"
)

var (
	ErrProvinceNotFound     = errors.New("province not found")
	ErrProvinceNameRequired = errors.New("province name is required")
	ErrProvinceExists       = errors.New("province name already exists")
	ErrProvinceNotOwned     = errors.New("province belongs to another department")
)

type ProvinceService interface {
	GetAll(ctx context.Context, departmentID uuid.UUID) ([]models.Province, error)
	Create(ctx context.Context, departmentID uuid.UUID, empID uuid.UUID, req CreateProvinceRequest) (*models.Province, error)
	Update(ctx context.Context, id uuid.UUID, departmentID uuid.UUID, req UpdateProvinceRequest) (*models.Province, error)
	Delete(ctx context.Context, id uuid.UUID, departmentID uuid.UUID) error

	// Sharing. Each of these takes the acting department so ownership can be
	// enforced: only the department that owns a province may decide who else
	// sees it, or see who it has been shared with.
	ShareProvince(ctx context.Context, actingDepartmentID uuid.UUID, share *models.ProvinceShare) error
	UnshareProvince(ctx context.Context, actingDepartmentID, provinceID, departmentID uuid.UUID) error
	GetProvinceShares(ctx context.Context, actingDepartmentID, provinceID uuid.UUID) ([]models.ProvinceShare, error)
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

	// A lookup failure must not be mistaken for "no such province": treating a
	// dropped connection as absence would create a duplicate.
	existing, err := s.repo.GetByName(ctx, req.Name, departmentID)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) && err.Error() != "sql: no rows in result set" {
		return nil, fmt.Errorf("check for an existing province: %w", err)
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

	// Ownership, not merely visibility: a department that had this province
	// shared with it must not be able to rename or deactivate it.
	province, err := s.assertOwned(ctx, id, departmentID)
	if err != nil {
		return nil, err
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
	if _, err := s.assertOwned(ctx, id, departmentID); err != nil {
		return err
	}
	return s.repo.Delete(ctx, id)
}

// assertOwned resolves a province and confirms the acting department owns it.
//
// repo.GetByID deliberately returns a province that is merely *shared with* the
// caller as well as one they own, distinguishing the two through IsShared. That
// is right for reading, but every mutation and the share list need ownership:
// otherwise a department that had a province shared with it could rename it,
// re-share it onward, or revoke the owner's other shares.
func (s *provinceService) assertOwned(ctx context.Context, provinceID, departmentID uuid.UUID) (*models.Province, error) {
	province, err := s.repo.GetByID(ctx, provinceID, departmentID)
	if err != nil {
		return nil, err
	}
	if province == nil {
		return nil, ErrProvinceNotFound
	}
	if province.IsShared {
		return nil, ErrProvinceNotOwned
	}
	return province, nil
}

func (s *provinceService) ShareProvince(ctx context.Context, actingDepartmentID uuid.UUID, share *models.ProvinceShare) error {
	if _, err := s.assertOwned(ctx, share.ProvinceID, actingDepartmentID); err != nil {
		return err
	}
	// Sharing a province with the department that already owns it is meaningless
	// and would make it appear twice in that department's list.
	if share.DepartmentID == actingDepartmentID {
		return errors.New("cannot share a province with the department that owns it")
	}
	return s.repo.ShareProvince(ctx, share)
}

func (s *provinceService) UnshareProvince(ctx context.Context, actingDepartmentID, provinceID, departmentID uuid.UUID) error {
	if _, err := s.assertOwned(ctx, provinceID, actingDepartmentID); err != nil {
		return err
	}
	return s.repo.UnshareProvince(ctx, provinceID, departmentID)
}

func (s *provinceService) GetProvinceShares(ctx context.Context, actingDepartmentID, provinceID uuid.UUID) ([]models.ProvinceShare, error) {
	if _, err := s.assertOwned(ctx, provinceID, actingDepartmentID); err != nil {
		return nil, err
	}
	return s.repo.GetProvinceShares(ctx, provinceID)
}
