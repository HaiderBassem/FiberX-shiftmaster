package repository

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"shiftmaster-backend/internal/models"
	"shiftmaster-backend/pkg/database"
)

// EmployeeRepository defines the interface for employee data access.
type EmployeeRepository interface {
	GetByID(ctx context.Context, id uuid.UUID) (*models.Employee, error)
	GetByEmail(ctx context.Context, email string) (*models.Employee, error)
	GetByCode(ctx context.Context, code string) (*models.Employee, error)
	GetAll(ctx context.Context) ([]models.Employee, error)
	GetByShiftID(ctx context.Context, shiftID uuid.UUID) ([]models.Employee, error)
	GetActive(ctx context.Context) ([]models.Employee, error)
	GetByDepartment(ctx context.Context, departmentID uuid.UUID) ([]models.Employee, error)
	GetByRole(ctx context.Context, role string) ([]models.Employee, error)
	Create(ctx context.Context, emp *models.Employee) error
	Update(ctx context.Context, emp *models.Employee) error
	UpdateStatus(ctx context.Context, id uuid.UUID, status string) error
	UpdatePassword(ctx context.Context, id uuid.UUID, passwordHash string) error
	UpdateLastLogin(ctx context.Context, id uuid.UUID) error
	Delete(ctx context.Context, id uuid.UUID) error
	ForceDelete(ctx context.Context, id uuid.UUID) error
	UpdateHelpPermission(ctx context.Context, id uuid.UUID, canManage bool) error
}

type employeeRepo struct {
	db *database.DB
}

func NewEmployeeRepository(db *database.DB) EmployeeRepository {
	return &employeeRepo{db: db}
}

func (r *employeeRepo) GetByID(ctx context.Context, id uuid.UUID) (*models.Employee, error) {
	var emp models.Employee
	err := r.db.QueryRow(ctx,
		`SELECT id, employee_code, first_name, last_name, gender, phone, email, password_hash,
				hire_date, role, department_id, position, default_shift_id, weekly_off_days,
				can_cover_night_shift, status, profile_image, remember_token, last_login, secondary_phone, secondary_email, can_create_tables, can_manage_help_docs,
				created_at, updated_at, created_by
		 FROM employees WHERE id = $1`, id,
	).Scan(
		&emp.ID, &emp.EmployeeCode, &emp.FirstName, &emp.LastName, &emp.Gender,
		&emp.Phone, &emp.Email, &emp.PasswordHash,
		&emp.HireDate, &emp.Role, &emp.DepartmentID, &emp.Position,
		&emp.DefaultShiftID, &emp.WeeklyOffDays, &emp.CanCoverNightShift,
		&emp.Status, &emp.ProfileImage, &emp.RememberToken, &emp.LastLogin, &emp.SecondaryPhone, &emp.SecondaryEmail, &emp.CanCreateTables, &emp.CanManageHelpDocs,
		&emp.CreatedAt, &emp.UpdatedAt, &emp.CreatedBy,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("employee not found: %w", err)
		}
		return nil, fmt.Errorf("get employee by id: %w", err)
	}
	return &emp, nil
}

func (r *employeeRepo) GetByEmail(ctx context.Context, email string) (*models.Employee, error) {
	var emp models.Employee
	err := r.db.QueryRow(ctx,
		`SELECT id, employee_code, first_name, last_name, gender, phone, email, password_hash,
				hire_date, role, department_id, position, default_shift_id, weekly_off_days,
				can_cover_night_shift, status, profile_image, remember_token, last_login, secondary_phone, secondary_email, can_create_tables, can_manage_help_docs,
				created_at, updated_at, created_by
		 FROM employees WHERE email = $1`, email,
	).Scan(
		&emp.ID, &emp.EmployeeCode, &emp.FirstName, &emp.LastName, &emp.Gender,
		&emp.Phone, &emp.Email, &emp.PasswordHash,
		&emp.HireDate, &emp.Role, &emp.DepartmentID, &emp.Position,
		&emp.DefaultShiftID, &emp.WeeklyOffDays, &emp.CanCoverNightShift,
		&emp.Status, &emp.ProfileImage, &emp.RememberToken, &emp.LastLogin, &emp.SecondaryPhone, &emp.SecondaryEmail, &emp.CanCreateTables, &emp.CanManageHelpDocs,
		&emp.CreatedAt, &emp.UpdatedAt, &emp.CreatedBy,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("employee not found: %w", err)
		}
		return nil, fmt.Errorf("get employee by email: %w", err)
	}
	return &emp, nil
}

func (r *employeeRepo) GetByCode(ctx context.Context, code string) (*models.Employee, error) {
	var emp models.Employee
	err := r.db.QueryRow(ctx,
		`SELECT id, employee_code, first_name, last_name, gender, phone, email, password_hash,
				hire_date, role, department_id, position, default_shift_id, weekly_off_days,
				can_cover_night_shift, status, profile_image, remember_token, last_login, secondary_phone, secondary_email, can_create_tables, can_manage_help_docs,
				created_at, updated_at, created_by
		 FROM employees WHERE employee_code = $1`, code,
	).Scan(
		&emp.ID, &emp.EmployeeCode, &emp.FirstName, &emp.LastName, &emp.Gender,
		&emp.Phone, &emp.Email, &emp.PasswordHash,
		&emp.HireDate, &emp.Role, &emp.DepartmentID, &emp.Position,
		&emp.DefaultShiftID, &emp.WeeklyOffDays, &emp.CanCoverNightShift,
		&emp.Status, &emp.ProfileImage, &emp.RememberToken, &emp.LastLogin, &emp.SecondaryPhone, &emp.SecondaryEmail, &emp.CanCreateTables, &emp.CanManageHelpDocs,
		&emp.CreatedAt, &emp.UpdatedAt, &emp.CreatedBy,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("employee not found: %w", err)
		}
		return nil, fmt.Errorf("get employee by code: %w", err)
	}
	return &emp, nil
}

func (r *employeeRepo) scanEmployees(rows pgx.Rows) ([]models.Employee, error) {
	var employees []models.Employee
	for rows.Next() {
		var emp models.Employee
		err := rows.Scan(
			&emp.ID, &emp.EmployeeCode, &emp.FirstName, &emp.LastName, &emp.Gender,
			&emp.Phone, &emp.Email, &emp.PasswordHash,
			&emp.HireDate, &emp.Role, &emp.DepartmentID, &emp.Position,
			&emp.DefaultShiftID, &emp.WeeklyOffDays, &emp.CanCoverNightShift,
			&emp.Status, &emp.ProfileImage, &emp.RememberToken, &emp.LastLogin, &emp.SecondaryPhone, &emp.SecondaryEmail, &emp.CanCreateTables, &emp.CanManageHelpDocs,
			&emp.CreatedAt, &emp.UpdatedAt, &emp.CreatedBy,
		)
		if err != nil {
			return nil, fmt.Errorf("scan employee: %w", err)
		}
		employees = append(employees, emp)
	}
	return employees, rows.Err()
}

func (r *employeeRepo) GetAll(ctx context.Context) ([]models.Employee, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, employee_code, first_name, last_name, gender, phone, email, password_hash,
				hire_date, role, department_id, position, default_shift_id, weekly_off_days,
				can_cover_night_shift, status, profile_image, remember_token, last_login, secondary_phone, secondary_email, can_create_tables, can_manage_help_docs,
				created_at, updated_at, created_by
		 FROM employees ORDER BY first_name, last_name`)
	if err != nil {
		return nil, fmt.Errorf("get all employees: %w", err)
	}
	defer rows.Close()
	return r.scanEmployees(rows)
}
func (r *employeeRepo) GetByShiftID(ctx context.Context, shiftID uuid.UUID) ([]models.Employee, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, employee_code, first_name, last_name, gender, phone, email, password_hash,
				hire_date, role, department_id, position, default_shift_id, weekly_off_days,
				can_cover_night_shift, status, profile_image, remember_token, last_login, secondary_phone, secondary_email, can_create_tables, can_manage_help_docs,
				created_at, updated_at, created_by
		 FROM employees WHERE default_shift_id = $1 ORDER BY first_name, last_name`, shiftID)
	if err != nil {
		return nil, fmt.Errorf("get employees by shift id: %w", err)
	}
	defer rows.Close()
	return r.scanEmployees(rows)
}

func (r *employeeRepo) GetActive(ctx context.Context) ([]models.Employee, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, employee_code, first_name, last_name, gender, phone, email, password_hash,
				hire_date, role, department_id, position, default_shift_id, weekly_off_days,
				can_cover_night_shift, status, profile_image, remember_token, last_login, secondary_phone, secondary_email, can_create_tables, can_manage_help_docs,
				created_at, updated_at, created_by
		 FROM employees WHERE status = 'active' ORDER BY first_name, last_name`)
	if err != nil {
		return nil, fmt.Errorf("get active employees: %w", err)
	}
	defer rows.Close()
	return r.scanEmployees(rows)
}

func (r *employeeRepo) GetByDepartment(ctx context.Context, departmentID uuid.UUID) ([]models.Employee, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, employee_code, first_name, last_name, gender, phone, email, password_hash,
				hire_date, role, department_id, position, default_shift_id, weekly_off_days,
				can_cover_night_shift, status, profile_image, remember_token, last_login, secondary_phone, secondary_email, can_create_tables, can_manage_help_docs,
				created_at, updated_at, created_by
		 FROM employees WHERE department_id = $1 ORDER BY first_name, last_name`, departmentID)
	if err != nil {
		return nil, fmt.Errorf("get employees by department: %w", err)
	}
	defer rows.Close()
	return r.scanEmployees(rows)
}

func (r *employeeRepo) GetByRole(ctx context.Context, role string) ([]models.Employee, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, employee_code, first_name, last_name, gender, phone, email, password_hash,
				hire_date, role, department_id, position, default_shift_id, weekly_off_days,
				can_cover_night_shift, status, profile_image, remember_token, last_login, secondary_phone, secondary_email, can_create_tables, can_manage_help_docs,
				created_at, updated_at, created_by
		 FROM employees WHERE role = $1 ORDER BY first_name, last_name`, role)
	if err != nil {
		return nil, fmt.Errorf("get employees by role: %w", err)
	}
	defer rows.Close()
	return r.scanEmployees(rows)
}

func (r *employeeRepo) Create(ctx context.Context, emp *models.Employee) error {
	return r.db.QueryRow(ctx,
		`INSERT INTO employees (employee_code, first_name, last_name, gender, phone, email, password_hash,
			hire_date, role, department_id, position, default_shift_id, weekly_off_days,
			can_cover_night_shift, status, profile_image, secondary_phone, secondary_email, can_create_tables, can_manage_help_docs, created_by)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21)
		 RETURNING id, created_at, updated_at`,
		emp.EmployeeCode, emp.FirstName, emp.LastName, emp.Gender, emp.Phone, emp.Email, emp.PasswordHash,
		emp.HireDate, emp.Role, emp.DepartmentID, emp.Position, emp.DefaultShiftID, emp.WeeklyOffDays,
		emp.CanCoverNightShift, emp.Status, emp.ProfileImage, emp.SecondaryPhone, emp.SecondaryEmail, emp.CanCreateTables, emp.CanManageHelpDocs, emp.CreatedBy,
	).Scan(&emp.ID, &emp.CreatedAt, &emp.UpdatedAt)
}

func (r *employeeRepo) Update(ctx context.Context, emp *models.Employee) error {
	_, err := r.db.Exec(ctx,
		`UPDATE employees SET first_name=$1, last_name=$2, gender=$3, phone=$4, email=$5,
			role=$6, department_id=$7, position=$8, default_shift_id=$9, weekly_off_days=$10,
			can_cover_night_shift=$11, status=$12, profile_image=$13,
			secondary_phone=$14, secondary_email=$15, can_create_tables=$16, can_manage_help_docs=$17, updated_at=CURRENT_TIMESTAMP
		 WHERE id=$18`,
		emp.FirstName, emp.LastName, emp.Gender, emp.Phone, emp.Email,
		emp.Role, emp.DepartmentID, emp.Position, emp.DefaultShiftID, emp.WeeklyOffDays,
		emp.CanCoverNightShift, emp.Status, emp.ProfileImage,
		emp.SecondaryPhone, emp.SecondaryEmail, emp.CanCreateTables, emp.CanManageHelpDocs, emp.ID,
	)
	return err
}

func (r *employeeRepo) UpdateStatus(ctx context.Context, id uuid.UUID, status string) error {
	_, err := r.db.Exec(ctx,
		`UPDATE employees SET status=$1, updated_at=CURRENT_TIMESTAMP WHERE id=$2`, status, id)
	return err
}

func (r *employeeRepo) UpdatePassword(ctx context.Context, id uuid.UUID, passwordHash string) error {
	_, err := r.db.Exec(ctx,
		`UPDATE employees SET password_hash=$1, updated_at=CURRENT_TIMESTAMP WHERE id=$2`, passwordHash, id)
	return err
}

func (r *employeeRepo) UpdateLastLogin(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.Exec(ctx,
		`UPDATE employees SET last_login=CURRENT_TIMESTAMP WHERE id=$1`, id)
	return err
}

func (r *employeeRepo) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.Exec(ctx, `DELETE FROM employees WHERE id=$1`, id)
	return err
}

func (r *employeeRepo) ForceDelete(ctx context.Context, id uuid.UUID) error {
	return r.db.ExecTx(ctx, func(ctx context.Context, tx pgx.Tx) error {
		queries := []string{
			`DELETE FROM department_managers WHERE manager_id = $1`,
			`UPDATE employees SET created_by = NULL WHERE created_by = $1`,
			`UPDATE weekly_schedule SET published_by = NULL WHERE published_by = $1`,
			`UPDATE weekly_schedule SET created_by = NULL WHERE created_by = $1`,
			`UPDATE employee_shifts SET replaced_employee_id = NULL WHERE replaced_employee_id = $1`,
			`UPDATE employee_shifts SET replacement_approved_by = NULL WHERE replacement_approved_by = $1`,
			`UPDATE employee_shifts SET created_by = NULL WHERE created_by = $1`,
			`UPDATE leaves SET approved_by_team_leader = NULL WHERE approved_by_team_leader = $1`,
			`UPDATE leaves SET approved_by_manager = NULL WHERE approved_by_manager = $1`,
			`UPDATE task_schedules SET created_by = NULL WHERE created_by = $1`,
			`UPDATE task_assignments SET assigned_by = NULL WHERE assigned_by = $1`,
			`UPDATE shift_swaps SET approved_by_team_leader = NULL WHERE approved_by_team_leader = $1`,
			`UPDATE shift_swaps SET approved_by_manager = NULL WHERE approved_by_manager = $1`,
			`DELETE FROM shift_swaps WHERE requester_id = $1 OR target_employee_id = $1`,
			`UPDATE notifications SET sender_id = NULL WHERE sender_id = $1`,
			`UPDATE audit_logs SET employee_id = NULL WHERE employee_id = $1`,
			`DELETE FROM employees WHERE id = $1`,
		}

		for _, q := range queries {
			if _, err := tx.Exec(ctx, q, id); err != nil {
				return err
			}
		}
		return nil
	})
}





func (r *employeeRepo) UpdateHelpPermission(ctx context.Context, id uuid.UUID, canManage bool) error {
	query := `UPDATE employees SET can_manage_help_docs = $2, updated_at = CURRENT_TIMESTAMP WHERE id = $1`
	_, err := r.db.Exec(ctx, query, id, canManage)
	return err
}
