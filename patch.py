import sys

with open('internal/repository/employee_repository.go', 'r') as f:
    text = f.read()

# Add to interface
text = text.replace('UpdatePreferences(ctx context.Context, id uuid.UUID, prefs map[string]interface{}) error\n}', 'UpdatePreferences(ctx context.Context, id uuid.UUID, prefs map[string]interface{}) error\n\tGetEmailsByDepartment(ctx context.Context, departmentID uuid.UUID) ([]string, error)\n}')

# Add implementation
impl = """
func (r *employeeRepo) GetEmailsByDepartment(ctx context.Context, departmentID uuid.UUID) ([]string, error) {
	rows, err := r.db.Query(ctx, `SELECT email FROM employees WHERE department_id = $1 AND status = 'active' AND email IS NOT NULL`, departmentID)
	if err != nil {
		return nil, fmt.Errorf("get emails by department: %w", err)
	}
	defer rows.Close()

	var emails []string
	for rows.Next() {
		var email string
		if err := rows.Scan(&email); err != nil {
			return nil, err
		}
		if email != "" {
			emails = append(emails, email)
		}
	}
	return emails, nil
}
"""

text = text + '\n' + impl

with open('internal/repository/employee_repository.go', 'w') as f:
    f.write(text)
