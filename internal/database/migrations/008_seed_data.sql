-- =====================================================
-- Seed Data
-- =====================================================

-- Add departments
INSERT INTO departments (department_code, name, description) VALUES
('TS', 'Technical Support', 'Technical support department'),
('HR', 'Human Resources', 'Human resources and employee affairs'),
('OPS', 'Operations', 'Operations and management'),
('FIN', 'Finance', 'Finance and accounting'),
('SALES', 'Sales', 'Sales and marketing'),
('NOC', 'Network Operations Center', 'Network Operations Center')
ON CONFLICT (department_code) DO NOTHING;

-- Add shift types
INSERT INTO shifts (shift_code, name, name_en, start_time, end_time, color_code) VALUES
('M', 'Morning', 'Morning', '08:30', '16:30', '#28a745'),
('E', 'Evening', 'Evening', '16:30', '00:30', '#ffc107'),
('N', 'Night', 'Night', '00:30', '08:30', '#dc3545')
ON CONFLICT (shift_code) DO NOTHING;

-- Add base permissions
INSERT INTO permissions (role, permission_name, resource, can_view, can_create, can_edit, can_delete, can_approve) VALUES
('admin', 'full_access', 'employees', true, true, true, true, true),
('admin', 'full_access', 'daily_tasks', true, true, true, true, true),
('admin', 'full_access', 'node_check', true, true, true, true, true),
('admin', 'full_access', 'mobile_ticket', true, true, true, true, true),
('admin', 'full_access', 'schedule', true, true, true, true, true),
('admin', 'full_access', 'leaves', true, true, true, true, true),
('admin', 'full_access', 'reports', true, true, true, true, true),

('manager', 'full_access', 'employees', true, true, true, true, true),
('manager', 'full_access', 'daily_tasks', true, true, true, true, true),
('manager', 'full_access', 'node_check', true, true, true, true, true),
('manager', 'full_access', 'mobile_ticket', true, true, true, true, true),
('manager', 'full_access', 'schedule', true, true, true, true, true),
('manager', 'full_access', 'leaves', true, true, true, true, true),
('manager', 'full_access', 'reports', true, true, false, false, false),

('team_leader', 'view_team', 'employees', true, false, false, false, false),
('team_leader', 'manage_tasks', 'daily_tasks', true, true, true, false, false),
('team_leader', 'view_node_check', 'node_check', true, false, false, false, false),
('team_leader', 'view_mobile_ticket', 'mobile_ticket', true, false, false, false, false),
('team_leader', 'approve_initial', 'leaves', true, false, false, false, true),

('employee', 'view_own', 'daily_tasks', true, false, false, false, false),
('employee', 'view_own', 'node_check', true, false, false, false, false),
('employee', 'view_own', 'mobile_ticket', true, false, false, false, false),
('employee', 'view_own', 'schedule', true, false, false, false, false),
('employee', 'create', 'leaves', true, true, false, false, false);

-- Add sample employees
-- Note: password 'password' hashed with bcrypt
INSERT INTO employees (employee_code, first_name, last_name, gender, phone, email, password_hash, hire_date, role, department_id, position, default_shift_id, can_cover_night_shift) VALUES
('EMP000', 'Admin', 'User', 'male', '0555000000', 'admin@company.com', '$2a$10$M2orX.ZA8XxU/SuFmaDD1e8zqWA9xJA6waEBr8V8oPasRzPzmR0bS', '2023-01-01', 'admin', (SELECT id FROM departments WHERE department_code='TS'), 'System Admin', (SELECT id FROM shifts WHERE shift_code='M'), true),
('EMP001', 'Ahmed', 'Mohammed', 'male', '0555000011', 'manager@company.com', '$2a$10$M2orX.ZA8XxU/SuFmaDD1e8zqWA9xJA6waEBr8V8oPasRzPzmR0bS', '2023-01-15', 'manager', (SELECT id FROM departments WHERE department_code='TS'), 'IT Manager', (SELECT id FROM shifts WHERE shift_code='M'), true),

('EMP002', 'Sara', 'Ali', 'female', '0555000022', 'teamleader@company.com', '$2a$10$M2orX.ZA8XxU/SuFmaDD1e8zqWA9xJA6waEBr8V8oPasRzPzmR0bS', '2023-02-20', 'team_leader', (SELECT id FROM departments WHERE department_code='OPS'), 'Operations Team Leader', (SELECT id FROM shifts WHERE shift_code='M'), false),

('EMP003', 'Khaled', 'Ibrahim', 'male', '0555000033', 'employee1@company.com', '$2a$10$M2orX.ZA8XxU/SuFmaDD1e8zqWA9xJA6waEBr8V8oPasRzPzmR0bS', '2023-03-10', 'employee', (SELECT id FROM departments WHERE department_code='OPS'), 'Operations Employee', (SELECT id FROM shifts WHERE shift_code='M'), true),

('EMP004', 'Noura', 'Abdullah', 'female', '0555000044', 'employee2@company.com', '$2a$10$M2orX.ZA8XxU/SuFmaDD1e8zqWA9xJA6waEBr8V8oPasRzPzmR0bS', '2023-04-05', 'employee', (SELECT id FROM departments WHERE department_code='OPS'), 'Operations Employee', (SELECT id FROM shifts WHERE shift_code='E'), false),

('EMP005', 'Omar', 'Hassan', 'male', '0555000055', 'employee3@company.com', '$2a$10$M2orX.ZA8XxU/SuFmaDD1e8zqWA9xJA6waEBr8V8oPasRzPzmR0bS', '2023-05-12', 'employee', (SELECT id FROM departments WHERE department_code='OPS'), 'Operations Employee', (SELECT id FROM shifts WHERE shift_code='N'), true),

('EMP006', 'Saeed', 'Ahmed', 'male', '0555000066', 'saeed@company.com', '$2a$10$M2orX.ZA8XxU/SuFmaDD1e8zqWA9xJA6waEBr8V8oPasRzPzmR0bS', '2023-06-15', 'employee', (SELECT id FROM departments WHERE department_code='SALES'), 'Sales Employee', (SELECT id FROM shifts WHERE shift_code='M'), false),

('EMP007', 'Mona', 'Khaled', 'female', '0555000077', 'mona@company.com', '$2a$10$M2orX.ZA8XxU/SuFmaDD1e8zqWA9xJA6waEBr8V8oPasRzPzmR0bS', '2023-07-20', 'employee', (SELECT id FROM departments WHERE department_code='FIN'), 'Accountant', (SELECT id FROM shifts WHERE shift_code='M'), false);

-- Set created_by for all employees (manager created them)
UPDATE employees SET created_by = (SELECT id FROM employees WHERE employee_code = 'EMP001') WHERE employee_code != 'EMP001';

-- Set manager_id in departments
UPDATE departments SET manager_id = (SELECT id FROM employees WHERE employee_code = 'EMP001') WHERE department_code = 'TS';
UPDATE departments SET manager_id = (SELECT id FROM employees WHERE employee_code = 'EMP002') WHERE department_code = 'OPS';

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

    -- For each active employee
    FOR v_emp IN SELECT id, default_shift_id FROM employees WHERE status = 'active' LOOP
        -- Sunday to Thursday: shift based on default_shift
        FOR d IN 0..4 LOOP
            INSERT INTO schedule_templates (employee_id, day_of_week, shift_id)
            VALUES (v_emp.id, d, v_emp.default_shift_id)
            ON CONFLICT (employee_id, day_of_week, valid_from) DO NOTHING;
        END LOOP;

        -- Friday and Saturday off
        INSERT INTO schedule_templates (employee_id, day_of_week, is_off)
        VALUES (v_emp.id, 5, true), (v_emp.id, 6, true)
        ON CONFLICT (employee_id, day_of_week, valid_from) DO NOTHING;
    END LOOP;
END $$;

-- =====================================================
-- Sample Task Schedules & Assignments
-- =====================================================
DO $$
DECLARE
    v_shift_m UUID;
    v_shift_e UUID;
    v_emp_ahmed UUID;
    v_emp_khaled UUID;
    v_emp_noura UUID;
    v_emp_omar UUID;
    v_task_1 UUID;
    v_task_2 UUID;
    v_task_3 UUID;
    v_task_4 UUID;
    v_task_5 UUID;
BEGIN
    SELECT id INTO v_shift_m FROM shifts WHERE shift_code = 'M';
    SELECT id INTO v_shift_e FROM shifts WHERE shift_code = 'E';
    SELECT id INTO v_emp_ahmed FROM employees WHERE employee_code = 'EMP001';
    SELECT id INTO v_emp_khaled FROM employees WHERE employee_code = 'EMP003';
    SELECT id INTO v_emp_noura FROM employees WHERE employee_code = 'EMP004';
    SELECT id INTO v_emp_omar FROM employees WHERE employee_code = 'EMP005';

    -- Daily Tasks (morning shift)
    INSERT INTO task_schedules (title, description, schedule_type, shift_id, recurrence, recurrence_days, max_assignees, created_by)
    VALUES ('Clean Equipment', 'Clean computers and printers daily', 'daily_task', v_shift_m, 'daily', NULL, 1, v_emp_ahmed)
    RETURNING id INTO v_task_1;

    INSERT INTO task_schedules (title, description, schedule_type, shift_id, recurrence, recurrence_days, max_assignees, created_by)
    VALUES ('Deliver Documents', 'Deliver documents between departments', 'daily_task', v_shift_m, 'daily', NULL, 1, v_emp_ahmed)
    RETURNING id INTO v_task_2;

    -- Periodic daily task (Thursdays, evening shift)
    INSERT INTO task_schedules (title, description, schedule_type, shift_id, recurrence, recurrence_days, max_assignees, created_by)
    VALUES ('Weekly Inventory Review', 'Review inventory items weekly', 'daily_task', v_shift_e, 'periodic', ARRAY[4], 1, v_emp_ahmed)
    RETURNING id INTO v_task_3;

    -- Node Check (morning shift, daily, 2 max)
    INSERT INTO task_schedules (title, description, schedule_type, shift_id, recurrence, recurrence_days, max_assignees, created_by)
    VALUES ('Server Inspection', 'Inspect servers and verify operation', 'node_check', v_shift_m, 'daily', NULL, 2, v_emp_ahmed)
    RETURNING id INTO v_task_4;

    -- Mobile Ticket (evening shift, daily, 2 max)
    INSERT INTO task_schedules (title, description, schedule_type, shift_id, recurrence, recurrence_days, max_assignees, created_by)
    VALUES ('Support Ticket Follow-up', 'Follow up on support tickets', 'mobile_ticket', v_shift_e, 'daily', NULL, 2, v_emp_ahmed)
    RETURNING id INTO v_task_5;

    -- Sample assignments for today
    INSERT INTO task_assignments (schedule_id, employee_id, assigned_date, assigned_by)
    VALUES
        (v_task_1, v_emp_khaled, CURRENT_DATE, v_emp_ahmed),
        (v_task_2, v_emp_khaled, CURRENT_DATE, v_emp_ahmed),
        (v_task_4, v_emp_khaled, CURRENT_DATE, v_emp_ahmed),
        (v_task_4, v_emp_omar, CURRENT_DATE, v_emp_ahmed),
        (v_task_5, v_emp_noura, CURRENT_DATE, v_emp_ahmed),
        (v_task_5, v_emp_omar, CURRENT_DATE, v_emp_ahmed);

    -- Create execution records (all pending)
    INSERT INTO task_executions (assignment_id, status)
    SELECT id, 'pending' FROM task_assignments WHERE assigned_date = CURRENT_DATE;
END $$;

-- =====================================================
-- Generate First Weekly Schedule
-- =====================================================
DO $$
DECLARE
    v_schedule_id UUID;
BEGIN
    SELECT generate_weekly_schedule(
        DATE_TRUNC('week', CURRENT_DATE + interval '1 week')::DATE,
        (SELECT id FROM employees WHERE employee_code = 'EMP001')
    ) INTO v_schedule_id;
END $$;
