-- =====================================================
-- Seed Data
-- =====================================================

-- Add shift types
INSERT INTO shifts (shift_code, name, name_en, start_time, end_time, color_code) VALUES
('M', 'Morning', 'Morning', '08:00', '16:00', '#28a745'),
('E', 'Evening', 'Evening', '16:00', '00:00', '#ffc107'),
('N', 'Night', 'Night', '00:00', '08:00', '#dc3545')
ON CONFLICT (shift_code) DO NOTHING;

-- Add base permissions
INSERT INTO permissions (role, permission_name, resource, can_view, can_create, can_edit, can_delete, can_approve) VALUES
('admin', 'full_access', 'schedule', true, true, true, true, true),
('admin', 'full_access', 'employees', true, true, true, true, true),
('admin', 'full_access', 'daily_tasks', true, true, true, true, true),
('admin', 'full_access', 'node_check', true, true, true, true, true),
('admin', 'full_access', 'mobile_ticket', true, true, true, true, true),
('admin', 'full_access', 'leaves', true, true, true, true, true),
('admin', 'full_access', 'reports', true, true, true, true, true),

('manager', 'full_access', 'schedule', true, true, true, true, true),
('manager', 'full_access', 'employees', true, true, true, true, true),
('manager', 'full_access', 'daily_tasks', true, true, true, true, true),
('manager', 'full_access', 'node_check', true, true, true, true, true),
('manager', 'full_access', 'mobile_ticket', true, true, true, true, true),
('manager', 'full_access', 'leaves', true, true, true, true, true),
('manager', 'full_access', 'reports', true, true, false, false, false),

('team_leader', 'view_team_schedule', 'schedule', true, false, true, false, false),
('team_leader', 'manage_tasks', 'daily_tasks', true, true, true, false, false),
('team_leader', 'view_node_check', 'node_check', true, false, false, false, false),
('team_leader', 'view_mobile_ticket', 'mobile_ticket', true, false, false, false, false),
('team_leader', 'view_team_leaves', 'leaves', true, false, false, false, true),

('employee', 'view_own_schedule', 'schedule', true, false, false, false, false),
('employee', 'view_own_tasks', 'daily_tasks', true, false, false, false, false),
('employee', 'view_own_tasks', 'node_check', true, false, false, false, false),
('employee', 'view_own_tasks', 'mobile_ticket', true, false, false, false, false),
('employee', 'request_leave', 'leaves', true, true, false, false, false);

-- Add sample employees
-- Note: password 'password' hashed with bcrypt
INSERT INTO employees (employee_code, first_name, last_name, gender, phone, email, password_hash, hire_date, role, department_id, position, default_shift_id, can_cover_night_shift) VALUES
('EMP000', 'Admin', 'User', 'male', '0555000000', 'admin@company.com', '$2a$10$M2orX.ZA8XxU/SuFmaDD1e8zqWA9xJA6waEBr8V8oPasRzPzmR0bS', '2023-01-01', 'admin', (SELECT id FROM departments WHERE department_code='IT'), 'System Admin', (SELECT id FROM shifts WHERE shift_code='M'), true),
('EMP001', 'Ahmed', 'Mohammed', 'male', '0555000011', 'manager@company.com', '$2a$10$M2orX.ZA8XxU/SuFmaDD1e8zqWA9xJA6waEBr8V8oPasRzPzmR0bS', '2023-01-15', 'manager', (SELECT id FROM departments WHERE department_code='IT'), 'IT Manager', (SELECT id FROM shifts WHERE shift_code='M'), true),
('EMP002', 'Sara', 'Ali', 'female', '0555000022', 'teamleader@company.com', '$2a$10$M2orX.ZA8XxU/SuFmaDD1e8zqWA9xJA6waEBr8V8oPasRzPzmR0bS', '2023-02-20', 'team_leader', (SELECT id FROM departments WHERE department_code='HR'), 'HR Team Leader', (SELECT id FROM shifts WHERE shift_code='M'), false),
('EMP003', 'Khaled', 'Ibrahim', 'male', '0555000033', 'employee1@company.com', '$2a$10$M2orX.ZA8XxU/SuFmaDD1e8zqWA9xJA6waEBr8V8oPasRzPzmR0bS', '2023-03-10', 'employee', (SELECT id FROM departments WHERE department_code='OPS'), 'Operations Employee 1', (SELECT id FROM shifts WHERE shift_code='E'), true),
('EMP004', 'Noura', 'Abdullah', 'female', '0555000044', 'employee2@company.com', '$2a$10$M2orX.ZA8XxU/SuFmaDD1e8zqWA9xJA6waEBr8V8oPasRzPzmR0bS', '2023-04-05', 'employee', (SELECT id FROM departments WHERE department_code='OPS'), 'Operations Employee 2', (SELECT id FROM shifts WHERE shift_code='N'), false),
('EMP005', 'Omar', 'Hassan', 'male', '0555000055', 'employee3@company.com', '$2a$10$M2orX.ZA8XxU/SuFmaDD1e8zqWA9xJA6waEBr8V8oPasRzPzmR0bS', '2023-05-12', 'employee', (SELECT id FROM departments WHERE department_code='OPS'), 'Operations Employee 3', (SELECT id FROM shifts WHERE shift_code='M'), true);

-- Add default schedule templates
DO $$
DECLARE
    v_shift_m UUID;
    v_shift_e UUID;
    v_shift_n UUID;
    v_emp RECORD;
BEGIN
    SELECT id INTO v_shift_m FROM shifts WHERE shift_code = 'M';
    SELECT id INTO v_shift_e FROM shifts WHERE shift_code = 'E';
    SELECT id INTO v_shift_n FROM shifts WHERE shift_code = 'N';
    
    FOR v_emp IN SELECT id, default_shift_id FROM employees WHERE status = 'active' LOOP
        FOR d IN 0..4 LOOP
            INSERT INTO schedule_templates (employee_id, day_of_week, shift_id) 
            VALUES (v_emp.id, d, v_emp.default_shift_id);
        END LOOP;
        
        -- Friday and Saturday off
        INSERT INTO schedule_templates (employee_id, day_of_week, is_off) 
        VALUES (v_emp.id, 5, true), (v_emp.id, 6, true);
    END LOOP;
END $$;
