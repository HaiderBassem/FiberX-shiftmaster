package repository

import (
	"context"
	"fmt"

	"shiftmaster-backend/internal/models"
	"shiftmaster-backend/pkg/database"
)

// PermissionRepository defines the interface for permission data access.
type PermissionRepository interface {
	GetByRole(ctx context.Context, role string) ([]models.Permission, error)
	GetByRoleAndResource(ctx context.Context, role, resource string) (*models.Permission, error)
	GetAll(ctx context.Context) ([]models.Permission, error)
	Upsert(ctx context.Context, perm *models.Permission) error
}

type permissionRepo struct {
	db *database.DB
}

func NewPermissionRepository(db *database.DB) PermissionRepository {
	return &permissionRepo{db: db}
}

const permColumns = `id, role, permission_name, resource, can_view, can_create, can_edit, can_delete, can_approve, department_restricted`

func (r *permissionRepo) GetByRole(ctx context.Context, role string) ([]models.Permission, error) {
	rows, err := r.db.Query(ctx,
		`SELECT `+permColumns+` FROM permissions WHERE role = $1 ORDER BY resource`, role)
	if err != nil {
		return nil, fmt.Errorf("get permissions by role: %w", err)
	}
	defer rows.Close()

	var perms []models.Permission
	for rows.Next() {
		var p models.Permission
		if err := rows.Scan(&p.ID, &p.Role, &p.PermissionName, &p.Resource,
			&p.CanView, &p.CanCreate, &p.CanEdit, &p.CanDelete, &p.CanApprove,
			&p.DepartmentRestricted); err != nil {
			return nil, fmt.Errorf("scan permission: %w", err)
		}
		perms = append(perms, p)
	}
	return perms, rows.Err()
}

func (r *permissionRepo) GetByRoleAndResource(ctx context.Context, role, resource string) (*models.Permission, error) {
	var p models.Permission
	err := r.db.QueryRow(ctx,
		`SELECT `+permColumns+` FROM permissions WHERE role = $1 AND resource = $2`, role, resource,
	).Scan(&p.ID, &p.Role, &p.PermissionName, &p.Resource,
		&p.CanView, &p.CanCreate, &p.CanEdit, &p.CanDelete, &p.CanApprove,
		&p.DepartmentRestricted)
	if err != nil {
		return nil, fmt.Errorf("get permission: %w", err)
	}
	return &p, nil
}

func (r *permissionRepo) GetAll(ctx context.Context) ([]models.Permission, error) {
	rows, err := r.db.Query(ctx,
		`SELECT `+permColumns+` FROM permissions ORDER BY role, resource`)
	if err != nil {
		return nil, fmt.Errorf("get all permissions: %w", err)
	}
	defer rows.Close()

	var perms []models.Permission
	for rows.Next() {
		var p models.Permission
		if err := rows.Scan(&p.ID, &p.Role, &p.PermissionName, &p.Resource,
			&p.CanView, &p.CanCreate, &p.CanEdit, &p.CanDelete, &p.CanApprove,
			&p.DepartmentRestricted); err != nil {
			return nil, fmt.Errorf("scan permission: %w", err)
		}
		perms = append(perms, p)
	}
	return perms, rows.Err()
}

func (r *permissionRepo) Upsert(ctx context.Context, perm *models.Permission) error {
	return r.db.QueryRow(ctx,
		`INSERT INTO permissions (role, permission_name, resource, can_view, can_create, can_edit, can_delete, can_approve, department_restricted)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)
		 ON CONFLICT (role, permission_name, resource) DO UPDATE SET
			can_view=EXCLUDED.can_view, can_create=EXCLUDED.can_create, can_edit=EXCLUDED.can_edit,
			can_delete=EXCLUDED.can_delete, can_approve=EXCLUDED.can_approve, department_restricted=EXCLUDED.department_restricted
		 RETURNING id`,
		perm.Role, perm.PermissionName, perm.Resource, perm.CanView, perm.CanCreate,
		perm.CanEdit, perm.CanDelete, perm.CanApprove, perm.DepartmentRestricted,
	).Scan(&perm.ID)
}
