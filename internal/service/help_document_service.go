package service

import (
	"context"
	"errors"
	"shiftmaster-backend/internal/models"
	"shiftmaster-backend/internal/repository"

	"github.com/google/uuid"
)

type HelpDocumentService struct {
	repo    *repository.HelpDocumentRepository
	empRepo repository.EmployeeRepository
}

func NewHelpDocumentService(repo *repository.HelpDocumentRepository, empRepo repository.EmployeeRepository) *HelpDocumentService {
	return &HelpDocumentService{repo: repo, empRepo: empRepo}
}

func (s *HelpDocumentService) GetVisibleDocuments(ctx context.Context, departmentID *uuid.UUID, employeeID uuid.UUID, role string) ([]models.HelpDocument, error) {
	if departmentID == nil {
		return nil, errors.New("employee does not belong to a department")
	}
	emp, err := s.empRepo.GetByID(ctx, employeeID)
	if err != nil {
		return nil, err
	}
	return s.repo.GetVisibleDocuments(ctx, *departmentID, employeeID, role, emp.CanManageHelpDocs)
}

func (s *HelpDocumentService) GetDocumentByID(ctx context.Context, id uuid.UUID, employeeID uuid.UUID, role string) (*models.HelpDocument, error) {
	emp, err := s.empRepo.GetByID(ctx, employeeID)
	if err != nil {
		return nil, err
	}
	return s.repo.GetDocumentByID(ctx, id, employeeID, role, emp.CanManageHelpDocs)
}

func (s *HelpDocumentService) CreateDocument(ctx context.Context, doc *models.HelpDocument, role string) (*models.HelpDocument, error) {
	if doc.CreatedBy == nil {
		return nil, errors.New("creator ID is missing")
	}
	emp, err := s.empRepo.GetByID(ctx, *doc.CreatedBy)
	if err != nil {
		return nil, err
	}
	if role != "manager" && role != "team_leader" && role != "admin" && !emp.CanManageHelpDocs {
		return nil, errors.New("only managers, team leaders, and authorized employees can create help documents")
	}
	if doc.DepartmentID == uuid.Nil {
		return nil, errors.New("department ID is required")
	}
	return s.repo.CreateDocument(ctx, doc)
}

func (s *HelpDocumentService) UpdateDocument(ctx context.Context, doc *models.HelpDocument, employeeID uuid.UUID, role string) (*models.HelpDocument, error) {
	emp, err := s.empRepo.GetByID(ctx, employeeID)
	if err != nil {
		return nil, err
	}
	existing, err := s.repo.GetDocumentByID(ctx, doc.ID, employeeID, role, emp.CanManageHelpDocs)
	if err != nil {
		return nil, err
	}
	if existing == nil {
		return nil, errors.New("document not found")
	}
	if existing.AccessLevel == nil || *existing.AccessLevel != "write" {
		return nil, errors.New("you do not have write access to this document")
	}
	
	existing.Title = doc.Title
	existing.Content = doc.Content
	return s.repo.UpdateDocument(ctx, existing)
}

func (s *HelpDocumentService) DeleteDocument(ctx context.Context, id uuid.UUID, employeeID uuid.UUID, role string) error {
	emp, err := s.empRepo.GetByID(ctx, employeeID)
	if err != nil {
		return err
	}
	existing, err := s.repo.GetDocumentByID(ctx, id, employeeID, role, emp.CanManageHelpDocs)
	if err != nil {
		return err
	}
	if existing == nil {
		return errors.New("document not found")
	}
	if existing.AccessLevel == nil || *existing.AccessLevel != "write" {
		return errors.New("you do not have write access to this document")
	}

	return s.repo.DeleteDocument(ctx, id)
}

func (s *HelpDocumentService) SetEmployeeAccess(ctx context.Context, documentID, targetEmployeeID uuid.UUID, accessLevel string, employeeID uuid.UUID, role string) error {
	emp, err := s.empRepo.GetByID(ctx, employeeID)
	if err != nil {
		return err
	}
	if role != "manager" && role != "team_leader" && role != "admin" && !emp.CanManageHelpDocs {
		return errors.New("only managers, team leaders, and authorized employees can manage access")
	}

	// Verify the document belongs to the same department as the manager/TL
	// And verify the target employee belongs to the same department
	targetEmp, err := s.empRepo.GetByID(ctx, targetEmployeeID)
	if err != nil {
		return err
	}
	
	doc, err := s.repo.GetDocumentByID(ctx, documentID, employeeID, role, emp.CanManageHelpDocs)
	if err != nil {
		return err
	}
	if doc == nil {
		return errors.New("document not found")
	}

	if targetEmp.DepartmentID == nil || *targetEmp.DepartmentID != doc.DepartmentID {
		return errors.New("employee does not belong to the same department as the document")
	}

	return s.repo.SetEmployeeAccess(ctx, documentID, targetEmployeeID, accessLevel, employeeID)
}

func (s *HelpDocumentService) GetDocumentAccessList(ctx context.Context, documentID uuid.UUID, employeeID uuid.UUID, role string) ([]models.HelpDocumentAccess, error) {
	emp, err := s.empRepo.GetByID(ctx, employeeID)
	if err != nil {
		return nil, err
	}
	if role != "manager" && role != "team_leader" && role != "admin" && !emp.CanManageHelpDocs {
		return nil, errors.New("only managers, team leaders, and authorized employees can view access lists")
	}
	// Verify doc exists and they have access
	doc, err := s.repo.GetDocumentByID(ctx, documentID, employeeID, role, emp.CanManageHelpDocs)
	if err != nil {
		return nil, err
	}
	if doc == nil {
		return nil, errors.New("document not found")
	}
	
	return s.repo.GetDocumentAccessList(ctx, documentID)
}
