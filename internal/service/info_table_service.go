package service

import (
	"context"
	"errors"
	"shiftmaster-backend/internal/models"
	"shiftmaster-backend/internal/repository"

	"github.com/google/uuid"
)

type InfoTableService struct {
	repo         *repository.InfoTableRepository
	employeeRepo repository.EmployeeRepository
}

func NewInfoTableService(repo *repository.InfoTableRepository, empRepo repository.EmployeeRepository) *InfoTableService {
	return &InfoTableService{
		repo:         repo,
		employeeRepo: empRepo,
	}
}

func (s *InfoTableService) CreateTable(ctx context.Context, table *models.InfoTable, creatorRole string, creatorCanCreate bool) (*models.InfoTable, error) {
	// Only admins, managers, team_leaders, or employees with explicit can_create_tables permission can create.
	if creatorRole != "admin" && creatorRole != "manager" && creatorRole != "team_leader" && !creatorCanCreate {
		return nil, errors.New("unauthorized to create tables")
	}

	return s.repo.CreateTable(ctx, table)
}

func (s *InfoTableService) GetVisibleTables(ctx context.Context, employeeID uuid.UUID, role string, departmentID *uuid.UUID) ([]models.InfoTable, error) {
	return s.repo.GetVisibleTables(ctx, employeeID, role, departmentID)
}

func (s *InfoTableService) GetTableByID(ctx context.Context, tableID uuid.UUID, reqEmployeeID uuid.UUID, reqRole string, reqDepID *uuid.UUID) (*models.InfoTable, error) {
	table, err := s.repo.GetTableByID(ctx, tableID)
	if err != nil {
		return nil, err
	}

	if reqRole == "admin" || (table.CreatedBy != nil && *table.CreatedBy == reqEmployeeID) {
		return table, nil
	}

	// Check explicit employee accesses first for blocking
	empAccesses, err := s.repo.GetEmployeeAccesses(ctx, tableID)
	if err == nil {
		for _, ea := range empAccesses {
			if ea.EmployeeID == reqEmployeeID && ea.AccessLevel == "none" {
				return nil, errors.New("unauthorized access to table (explicitly blocked)")
			}
		}
	}

	// Check department matching
	if reqDepID != nil && table.DepartmentID != nil && *reqDepID == *table.DepartmentID {
		return table, nil
	}

	// Check department accesses
	depAccesses, err := s.repo.GetDepartmentAccesses(ctx, tableID)
	if err == nil {
		for _, da := range depAccesses {
			if reqDepID != nil && da.DepartmentID == *reqDepID {
				return table, nil
			}
		}
	}

	// Check employee accesses (again for granting)
	if err == nil {
		for _, ea := range empAccesses {
			if ea.EmployeeID == reqEmployeeID {
				return table, nil
			}
		}
	}

	return nil, errors.New("unauthorized access to table")
}

func (s *InfoTableService) HasWriteAccess(ctx context.Context, tableID uuid.UUID, reqEmployeeID uuid.UUID, reqRole string, reqDepID *uuid.UUID) bool {
	if reqRole == "admin" {
		return true
	}
	table, err := s.repo.GetTableByID(ctx, tableID)
	if err != nil {
		return false
	}
	// Creator always has write access
	if table.CreatedBy != nil && *table.CreatedBy == reqEmployeeID {
		return true
	}
	
	// Check explicit employee access first
	empAccesses, err := s.repo.GetEmployeeAccesses(ctx, tableID)
	if err == nil {
		for _, ea := range empAccesses {
			if ea.EmployeeID == reqEmployeeID {
				if ea.AccessLevel == "none" {
					return false
				}
				if ea.AccessLevel == "write" {
					return true
				}
			}
		}
	}

	// If it's a manager/TL in the table's original department, they have full access to manage it.
	if reqDepID != nil && table.DepartmentID != nil && *reqDepID == *table.DepartmentID {
		if reqRole == "manager" || reqRole == "team_leader" {
			return true
		}
	}

	// If manager/TL in a department that was GRANTED access, they can write & manage access for their dept
	depAccesses, err := s.repo.GetDepartmentAccesses(ctx, tableID)
	if err == nil {
		for _, da := range depAccesses {
			if reqDepID != nil && da.DepartmentID == *reqDepID {
				if reqRole == "manager" || reqRole == "team_leader" {
					return true
				}
			}
		}
	}

	return false
}

func (s *InfoTableService) HasManageAccessRight(ctx context.Context, tableID uuid.UUID, reqEmployeeID uuid.UUID, reqRole string, reqDepID *uuid.UUID) bool {
	if reqRole == "admin" {
		return true
	}
	table, err := s.repo.GetTableByID(ctx, tableID)
	if err != nil {
		return false
	}
	if table.CreatedBy != nil && *table.CreatedBy == reqEmployeeID {
		return true
	}
	// Explicit block check
	empAccesses, err := s.repo.GetEmployeeAccesses(ctx, tableID)
	if err == nil {
		for _, ea := range empAccesses {
			if ea.EmployeeID == reqEmployeeID && ea.AccessLevel == "none" {
				return false
			}
		}
	}

	if reqDepID != nil && table.DepartmentID != nil && *reqDepID == *table.DepartmentID {
		if reqRole == "manager" || reqRole == "team_leader" {
			return true
		}
	}
	depAccesses, err := s.repo.GetDepartmentAccesses(ctx, tableID)
	if err == nil {
		for _, da := range depAccesses {
			if reqDepID != nil && da.DepartmentID == *reqDepID {
				if reqRole == "manager" || reqRole == "team_leader" {
					return true
				}
			}
		}
	}
	return false
}

// Update, Delete for tables...
func (s *InfoTableService) UpdateTable(ctx context.Context, table *models.InfoTable, reqEmployeeID uuid.UUID, reqRole string, reqDepID *uuid.UUID) (*models.InfoTable, error) {
	if !s.HasManageAccessRight(ctx, table.ID, reqEmployeeID, reqRole, reqDepID) {
		return nil, errors.New("unauthorized to update table")
	}
	return s.repo.UpdateTable(ctx, table)
}

func (s *InfoTableService) DeleteTable(ctx context.Context, tableID uuid.UUID, reqEmployeeID uuid.UUID, reqRole string, reqDepID *uuid.UUID) error {
	if !s.HasManageAccessRight(ctx, tableID, reqEmployeeID, reqRole, reqDepID) {
		return errors.New("unauthorized to delete table")
	}
	return s.repo.DeleteTable(ctx, tableID)
}

// Rows
func (s *InfoTableService) GetTableRows(ctx context.Context, tableID uuid.UUID, reqEmployeeID uuid.UUID, reqRole string, reqDepID *uuid.UUID) ([]models.InfoTableRow, error) {
	_, err := s.GetTableByID(ctx, tableID, reqEmployeeID, reqRole, reqDepID) // check read access
	if err != nil {
		return nil, err
	}
	return s.repo.GetTableRows(ctx, tableID)
}

func (s *InfoTableService) CreateTableRow(ctx context.Context, row *models.InfoTableRow, reqEmployeeID uuid.UUID, reqRole string, reqDepID *uuid.UUID) (*models.InfoTableRow, error) {
	if !s.HasWriteAccess(ctx, row.TableID, reqEmployeeID, reqRole, reqDepID) {
		return nil, errors.New("unauthorized to add rows")
	}
	return s.repo.CreateTableRow(ctx, row)
}

func (s *InfoTableService) UpdateTableRow(ctx context.Context, row *models.InfoTableRow, reqEmployeeID uuid.UUID, reqRole string, reqDepID *uuid.UUID) (*models.InfoTableRow, error) {
	if !s.HasWriteAccess(ctx, row.TableID, reqEmployeeID, reqRole, reqDepID) {
		return nil, errors.New("unauthorized to update rows")
	}
	return s.repo.UpdateTableRow(ctx, row)
}

func (s *InfoTableService) DeleteTableRow(ctx context.Context, rowID uuid.UUID, tableID uuid.UUID, reqEmployeeID uuid.UUID, reqRole string, reqDepID *uuid.UUID) error {
	if !s.HasWriteAccess(ctx, tableID, reqEmployeeID, reqRole, reqDepID) {
		return errors.New("unauthorized to delete rows")
	}
	return s.repo.DeleteTableRow(ctx, rowID)
}

// Access
func (s *InfoTableService) ShareWithDepartment(ctx context.Context, reqRole string, tableID uuid.UUID, targetDepID uuid.UUID, grantedBy uuid.UUID) error {
	if reqRole != "admin" {
		return errors.New("only admins can share tables with other departments")
	}
	access := &models.InfoTableDepartmentAccess{
		TableID:      tableID,
		DepartmentID: targetDepID,
		GrantedBy:    &grantedBy,
	}
	return s.repo.AddDepartmentAccess(ctx, access)
}

func (s *InfoTableService) AddEmployeeAccess(ctx context.Context, reqEmployeeID uuid.UUID, reqRole string, reqDepID *uuid.UUID, tableID uuid.UUID, targetEmployeeID uuid.UUID, accessLevel string) error {
	if !s.HasManageAccessRight(ctx, tableID, reqEmployeeID, reqRole, reqDepID) {
		return errors.New("unauthorized to manage access for this table")
	}
	// Verify target employee is in the same department, unless admin
	if reqRole != "admin" {
		emp, err := s.employeeRepo.GetByID(ctx, targetEmployeeID)
		if err != nil {
			return err
		}
		if reqDepID == nil || emp.DepartmentID == nil || *reqDepID != *emp.DepartmentID {
			return errors.New("cannot manage access for employees outside your department")
		}
	}
	
	access := &models.InfoTableEmployeeAccess{
		TableID:     tableID,
		EmployeeID:  targetEmployeeID,
		AccessLevel: accessLevel,
		GrantedBy:   &reqEmployeeID,
	}
	return s.repo.AddEmployeeAccess(ctx, access)
}

func (s *InfoTableService) GetAccessLists(ctx context.Context, tableID uuid.UUID, reqEmployeeID uuid.UUID, reqRole string, reqDepID *uuid.UUID) ([]models.InfoTableDepartmentAccess, []models.InfoTableEmployeeAccess, error) {
	if !s.HasManageAccessRight(ctx, tableID, reqEmployeeID, reqRole, reqDepID) {
		return nil, nil, errors.New("unauthorized to view access lists")
	}
	depAcc, _ := s.repo.GetDepartmentAccesses(ctx, tableID)
	empAcc, _ := s.repo.GetEmployeeAccesses(ctx, tableID)
	return depAcc, empAcc, nil
}

func (s *InfoTableService) RemoveEmployeeAccess(ctx context.Context, reqEmployeeID uuid.UUID, reqRole string, reqDepID *uuid.UUID, tableID uuid.UUID, targetEmployeeID uuid.UUID) error {
	if !s.HasManageAccessRight(ctx, tableID, reqEmployeeID, reqRole, reqDepID) {
		return errors.New("unauthorized to manage access for this table")
	}
	
	if reqRole != "admin" {
		emp, err := s.employeeRepo.GetByID(ctx, targetEmployeeID)
		if err != nil {
			return err
		}
		if reqDepID == nil || emp.DepartmentID == nil || *reqDepID != *emp.DepartmentID {
			return errors.New("cannot manage access for employees outside your department")
		}
	}

	return s.repo.RemoveEmployeeAccess(ctx, tableID, targetEmployeeID)
}
