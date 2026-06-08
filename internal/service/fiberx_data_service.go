package service

import (
	"context"
	"errors"
	"shiftmaster-backend/internal/models"
	"shiftmaster-backend/internal/repository"

	"github.com/google/uuid"
)

type FiberxDataService struct {
	repo    *repository.FiberxDataRepository
	empRepo repository.EmployeeRepository
}

func NewFiberxDataService(repo *repository.FiberxDataRepository, empRepo repository.EmployeeRepository) *FiberxDataService {
	return &FiberxDataService{repo: repo, empRepo: empRepo}
}

func (s *FiberxDataService) GetVisibleDocuments(ctx context.Context, departmentID *uuid.UUID, employeeID uuid.UUID, role string) ([]models.FiberxDataResponse, error) {
	if departmentID == nil {
		return nil, errors.New("employee does not belong to a department")
	}
	emp, err := s.empRepo.GetByID(ctx, employeeID)
	if err != nil {
		return nil, err
	}
	return s.repo.GetVisibleDocuments(ctx, *departmentID, employeeID, role, emp.CanManageFiberxData)
}

func (s *FiberxDataService) GetDocumentByID(ctx context.Context, id uuid.UUID, departmentID *uuid.UUID, employeeID uuid.UUID, role string) (*models.FiberxDataResponse, error) {
	if departmentID == nil {
		return nil, errors.New("employee does not belong to a department")
	}
	emp, err := s.empRepo.GetByID(ctx, employeeID)
	if err != nil {
		return nil, err
	}
	return s.repo.GetDocumentByID(ctx, id, *departmentID, employeeID, role, emp.CanManageFiberxData)
}

func (s *FiberxDataService) CreateDocument(ctx context.Context, doc *models.FiberxData, role string) (*models.FiberxData, error) {
	if doc.CreatedBy == nil {
		return nil, errors.New("creator ID is missing")
	}
	emp, err := s.empRepo.GetByID(ctx, *doc.CreatedBy)
	if err != nil {
		return nil, err
	}
	if role != "manager" && role != "admin" && !emp.CanManageFiberxData {
		return nil, errors.New("only managers and authorized employees can create FiberX Data documents")
	}
	if doc.DepartmentID == uuid.Nil {
		return nil, errors.New("department ID is required")
	}
	return s.repo.CreateDocument(ctx, doc)
}

func (s *FiberxDataService) UpdateDocument(ctx context.Context, doc *models.FiberxData, employeeID uuid.UUID, role string) (*models.FiberxData, error) {
	emp, err := s.empRepo.GetByID(ctx, employeeID)
	if err != nil {
		return nil, err
	}
	if emp.DepartmentID == nil {
		return nil, errors.New("employee does not belong to a department")
	}
	existing, err := s.repo.GetDocumentByID(ctx, doc.ID, *emp.DepartmentID, employeeID, role, emp.CanManageFiberxData)
	if err != nil {
		return nil, err
	}
	if existing == nil {
		return nil, errors.New("document not found")
	}
	if existing.AccessLevel != "write" {
		return nil, errors.New("you do not have write access to this document")
	}
	
	existing.Title = doc.Title
	existing.Content = doc.Content
	return s.repo.UpdateDocument(ctx, &existing.FiberxData)
}

func (s *FiberxDataService) DeleteDocument(ctx context.Context, id uuid.UUID, employeeID uuid.UUID, role string) error {
	emp, err := s.empRepo.GetByID(ctx, employeeID)
	if err != nil {
		return err
	}
	if emp.DepartmentID == nil {
		return errors.New("employee does not belong to a department")
	}
	existing, err := s.repo.GetDocumentByID(ctx, id, *emp.DepartmentID, employeeID, role, emp.CanManageFiberxData)
	if err != nil {
		return err
	}
	if existing == nil {
		return errors.New("document not found")
	}
	if existing.AccessLevel != "write" {
		return errors.New("you do not have write access to this document")
	}

	return s.repo.DeleteDocument(ctx, id)
}

func (s *FiberxDataService) SetEmployeeAccess(ctx context.Context, documentID, targetEmployeeID uuid.UUID, accessLevel string, employeeID uuid.UUID, role string) error {
	emp, err := s.empRepo.GetByID(ctx, employeeID)
	if err != nil {
		return err
	}
	if role != "manager" && role != "admin" && !emp.CanManageFiberxData {
		return errors.New("only managers and authorized employees can manage access")
	}

	// targetEmp, err := s.empRepo.GetByID(ctx, targetEmployeeID) // if we needed to verify target emp
	
	if emp.DepartmentID == nil {
		return errors.New("employee does not belong to a department")
	}
	doc, err := s.repo.GetDocumentByID(ctx, documentID, *emp.DepartmentID, employeeID, role, emp.CanManageFiberxData)
	if err != nil {
		return err
	}
	if doc == nil {
		return errors.New("document not found")
	}
	
	if doc.AccessLevel != "write" {
		return errors.New("you do not have write access to this document")
	}

	return s.repo.SetEmployeeAccess(ctx, documentID, targetEmployeeID, accessLevel, employeeID)
}

func (s *FiberxDataService) GetEmployeeAccessList(ctx context.Context, documentID uuid.UUID, employeeID uuid.UUID, role string) ([]models.FiberxDataEmployeeAccess, error) {
	emp, err := s.empRepo.GetByID(ctx, employeeID)
	if err != nil {
		return nil, err
	}
	if role != "manager" && role != "admin" && !emp.CanManageFiberxData {
		return nil, errors.New("only managers and authorized employees can view access lists")
	}
	if emp.DepartmentID == nil {
		return nil, errors.New("employee does not belong to a department")
	}
	doc, err := s.repo.GetDocumentByID(ctx, documentID, *emp.DepartmentID, employeeID, role, emp.CanManageFiberxData)
	if err != nil {
		return nil, err
	}
	if doc == nil {
		return nil, errors.New("document not found")
	}
	
	return s.repo.GetEmployeeAccessList(ctx, documentID)
}

func (s *FiberxDataService) SetDepartmentShare(ctx context.Context, documentID, targetDepartmentID uuid.UUID, accessLevel string, employeeID uuid.UUID, role string) error {
	emp, err := s.empRepo.GetByID(ctx, employeeID)
	if err != nil {
		return err
	}
	if role != "manager" && role != "admin" && !emp.CanManageFiberxData {
		return errors.New("only managers and authorized employees can manage shares")
	}

	if emp.DepartmentID == nil {
		return errors.New("employee does not belong to a department")
	}
	doc, err := s.repo.GetDocumentByID(ctx, documentID, *emp.DepartmentID, employeeID, role, emp.CanManageFiberxData)
	if err != nil {
		return err
	}
	if doc == nil {
		return errors.New("document not found")
	}
	if doc.AccessLevel != "write" {
		return errors.New("you do not have write access to this document")
	}

	return s.repo.SetDepartmentShare(ctx, documentID, targetDepartmentID, accessLevel, employeeID)
}

func (s *FiberxDataService) GetDepartmentShares(ctx context.Context, documentID uuid.UUID, employeeID uuid.UUID, role string) ([]models.FiberxDataDepartmentShare, error) {
	emp, err := s.empRepo.GetByID(ctx, employeeID)
	if err != nil {
		return nil, err
	}
	if role != "manager" && role != "admin" && !emp.CanManageFiberxData {
		return nil, errors.New("only managers and authorized employees can view shares")
	}
	if emp.DepartmentID == nil {
		return nil, errors.New("employee does not belong to a department")
	}
	doc, err := s.repo.GetDocumentByID(ctx, documentID, *emp.DepartmentID, employeeID, role, emp.CanManageFiberxData)
	if err != nil {
		return nil, err
	}
	if doc == nil {
		return nil, errors.New("document not found")
	}
	
	return s.repo.GetDepartmentShares(ctx, documentID)
}
