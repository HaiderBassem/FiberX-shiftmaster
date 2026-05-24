-- =====================================================
-- Comprehensive Database for Shift & Task Management System
-- Version: 5.0 (Unified with migrations)
-- =====================================================

-- =====================================================
-- 1. Enable Core Extensions
-- =====================================================
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- =====================================================
-- 2. Create Custom Data Types (ENUMs)
-- =====================================================
DO $$ BEGIN
    CREATE TYPE employee_role AS ENUM ('employee', 'team_leader', 'manager', 'admin', 'hr');
EXCEPTION WHEN duplicate_object THEN NULL; END $$;

DO $$ BEGIN
    CREATE TYPE gender_type AS ENUM ('male', 'female');
EXCEPTION WHEN duplicate_object THEN NULL; END $$;

DO $$ BEGIN
    CREATE TYPE employee_status AS ENUM ('active', 'inactive', 'on_leave', 'terminated');
EXCEPTION WHEN duplicate_object THEN NULL; END $$;

DO $$ BEGIN
    CREATE TYPE shift_status_type AS ENUM ('working', 'off', 'leave', 'sick', 'vacation', 'training', 'business_trip');
EXCEPTION WHEN duplicate_object THEN NULL; END $$;

DO $$ BEGIN
    CREATE TYPE leave_type AS ENUM ('annual', 'sick', 'emergency', 'marriage', 'maternity', 'unpaid', 'other');
EXCEPTION WHEN duplicate_object THEN NULL; END $$;

DO $$ BEGIN
    CREATE TYPE leave_status AS ENUM ('pending', 'approved_by_team_leader', 'approved_by_manager', 'rejected');
EXCEPTION WHEN duplicate_object THEN NULL; END $$;

DO $$ BEGIN
    CREATE TYPE task_priority AS ENUM ('low', 'medium', 'high', 'urgent');
EXCEPTION WHEN duplicate_object THEN NULL; END $$;

DO $$ BEGIN
    CREATE TYPE task_status AS ENUM ('pending', 'in_progress', 'completed', 'cancelled', 'overdue');
EXCEPTION WHEN duplicate_object THEN NULL; END $$;

DO $$ BEGIN
    CREATE TYPE notification_type AS ENUM ('shift_change', 'task_assigned', 'leave_request', 'approval', 'system_alert', 'reminder');
EXCEPTION WHEN duplicate_object THEN NULL; END $$;

DO $$ BEGIN
    CREATE TYPE notification_priority AS ENUM ('low', 'medium', 'high');
EXCEPTION WHEN duplicate_object THEN NULL; END $$;

DO $$ BEGIN
    CREATE TYPE swap_status AS ENUM ('pending', 'employee_accepted', 'approved', 'rejected', 'cancelled');
EXCEPTION WHEN duplicate_object THEN NULL; END $$;

-- =====================================================
-- 3. Departments Table
-- =====================================================
CREATE TABLE departments (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    department_code VARCHAR(20) UNIQUE NOT NULL,
    name VARCHAR(100) NOT NULL,
    description TEXT,
    manager_id UUID, -- Will be linked after creating the employees table
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

COMMENT ON TABLE departments IS 'Company departments';

INSERT INTO departments (department_code, name, description) VALUES
('TS', 'Technical Support', 'Technical support department'),
('HR', 'Human Resources', 'Human resources and employee affairs'),
('OPS', 'Operations', 'Operations and management'),
('FIN', 'Finance', 'Finance and accounting'),
('SALES', 'Sales', 'Sales and marketing'),
('NOC', 'Network Operations Center', 'Network Operations Center');

-- =====================================================
-- 4. Employees Table
-- =====================================================
CREATE TABLE employees (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    employee_code VARCHAR(20) UNIQUE NOT NULL,
    first_name VARCHAR(50) NOT NULL,
    last_name VARCHAR(50) NOT NULL,
    gender gender_type NOT NULL,
    phone VARCHAR(20),
    email VARCHAR(100) UNIQUE NOT NULL,
    password_hash VARCHAR(255), -- Encrypted password

    -- Job information
    hire_date DATE NOT NULL,
    role employee_role NOT NULL DEFAULT 'employee',
    department_id UUID REFERENCES departments(id),
    position VARCHAR(100),

    -- Shifts
    default_shift_id UUID,
    weekly_off_days INTEGER DEFAULT 1,
    can_cover_night_shift BOOLEAN DEFAULT false,

    -- Status
    status employee_status DEFAULT 'active',
    profile_image VARCHAR(255),

    -- Login token (remember me)
    remember_token VARCHAR(100),
    last_login TIMESTAMP,

    -- Secondary contact info
    secondary_phone VARCHAR(20),
    secondary_email VARCHAR(255),

    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    created_by UUID REFERENCES employees(id)
);

COMMENT ON TABLE employees IS 'Core employee data with authentication';
COMMENT ON COLUMN employees.password_hash IS 'Password encrypted using bcrypt';
COMMENT ON COLUMN employees.can_cover_night_shift IS 'Can this employee cover a night shift?';

-- Link manager_id in departments table
ALTER TABLE departments ADD CONSTRAINT fk_departments_manager
    FOREIGN KEY (manager_id) REFERENCES employees(id);

-- =====================================================
-- 5. Shifts Table (Shift Types)
-- =====================================================
CREATE TABLE shifts (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    shift_code VARCHAR(10) UNIQUE NOT NULL,
    name VARCHAR(50) NOT NULL,
    name_en VARCHAR(50),
    start_time TIME NOT NULL,
    end_time TIME NOT NULL,
    color_code VARCHAR(7) DEFAULT '#3788d8',
    requires_vehicle BOOLEAN DEFAULT false,
    min_rest_hours INTEGER DEFAULT 8,
    department_id UUID REFERENCES departments(id) ON DELETE CASCADE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

COMMENT ON TABLE shifts IS 'Shift types (morning, evening, night)';

INSERT INTO shifts (shift_code, name, name_en, start_time, end_time, color_code) VALUES
('M', 'Morning', 'Morning', '08:30', '16:30', '#28a745'),
('E', 'Evening', 'Evening', '16:30', '00:30', '#ffc107'),
('N', 'Night', 'Night', '00:30', '08:30', '#dc3545');

-- =====================================================
-- 6. Weekly Schedule Templates (Base Fixed Schedule)
-- =====================================================
CREATE TABLE schedule_templates (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    employee_id UUID NOT NULL REFERENCES employees(id) ON DELETE CASCADE,
    day_of_week INTEGER CHECK (day_of_week BETWEEN 0 AND 6),
    shift_id UUID REFERENCES shifts(id) ON DELETE SET NULL,
    is_off BOOLEAN DEFAULT false,
    valid_from DATE,
    valid_to DATE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (employee_id, day_of_week, valid_from)
);

COMMENT ON TABLE schedule_templates IS 'Fixed employee schedule template (without dates)';
COMMENT ON COLUMN schedule_templates.is_off IS 'Whether this day is a fixed day off';

-- =====================================================
-- 7. Published Weekly Schedules
-- =====================================================
CREATE TABLE weekly_schedule (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    week_start_date DATE NOT NULL,
    week_end_date DATE NOT NULL,
    template_id UUID REFERENCES schedule_templates(id),
    status VARCHAR(20) DEFAULT 'draft',  -- draft, published, archived
    published_by UUID REFERENCES employees(id),
    published_at TIMESTAMP,
    notes TEXT,
    department_id UUID REFERENCES departments(id) ON DELETE CASCADE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    created_by UUID REFERENCES employees(id),
    UNIQUE (department_id, week_start_date)
);

COMMENT ON TABLE weekly_schedule IS 'Each published week';

-- =====================================================
-- 8. Employee Shifts (Actual Daily Assignments)
-- =====================================================
CREATE TABLE employee_shifts (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    schedule_id UUID NOT NULL REFERENCES weekly_schedule(id) ON DELETE CASCADE,
    employee_id UUID NOT NULL REFERENCES employees(id) ON DELETE CASCADE,
    shift_id UUID REFERENCES shifts(id) ON DELETE SET NULL,
    shift_date DATE NOT NULL,
    shift_status shift_status_type NOT NULL DEFAULT 'working',
    leave_reason TEXT,
    is_replacement BOOLEAN DEFAULT false,
    replaced_employee_id UUID REFERENCES employees(id),
    replacement_approved_by UUID REFERENCES employees(id),
    check_in_time TIMESTAMP,
    check_out_time TIMESTAMP,
    actual_worked_hours DECIMAL(4,2),
    overtime_hours DECIMAL(4,2),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    created_by UUID REFERENCES employees(id),
    UNIQUE (employee_id, shift_date)
);

COMMENT ON TABLE employee_shifts IS 'Actual daily schedule for employees';

-- =====================================================
-- 9. Task Schedules (defines tasks and their recurrence)
-- =====================================================
CREATE TABLE task_schedules (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    title VARCHAR(200) NOT NULL,
    description TEXT,
    schedule_type VARCHAR(20) CHECK (schedule_type IN ('daily_task', 'node_check', 'mobile_ticket')),
    shift_id UUID REFERENCES shifts(id),
    recurrence VARCHAR(10) NOT NULL DEFAULT 'daily' CHECK (recurrence IN ('daily', 'periodic')),
    recurrence_days INTEGER[],              -- for periodic: e.g. [4] = Thursday, NULL = every day
    max_assignees INTEGER DEFAULT 1,        -- node_check/mobile_ticket can have 1 or 2
    is_active BOOLEAN DEFAULT true,
    created_by UUID REFERENCES employees(id),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

COMMENT ON TABLE task_schedules IS 'Task definitions with recurrence pattern (daily_task, node_check, mobile_ticket)';
COMMENT ON COLUMN task_schedules.schedule_type IS 'Type of table this task appears in: daily_task, node_check, or mobile_ticket';
COMMENT ON COLUMN task_schedules.recurrence IS 'daily = every day, periodic = only on recurrence_days';
COMMENT ON COLUMN task_schedules.recurrence_days IS 'Days of week (0=Sun..6=Sat). NULL means every day when recurrence=daily';

-- =====================================================
-- 10. Task Assignments (who does what on which date)
-- =====================================================
CREATE TABLE task_assignments (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    schedule_id UUID NOT NULL REFERENCES task_schedules(id) ON DELETE CASCADE,
    employee_id UUID NOT NULL REFERENCES employees(id) ON DELETE CASCADE,
    assigned_date DATE NOT NULL,
    assigned_by UUID REFERENCES employees(id),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(schedule_id, employee_id, assigned_date)
);

COMMENT ON TABLE task_assignments IS 'Links a task schedule to an employee for a specific date';

-- =====================================================
-- 11. Task Executions (completion tracking)
-- =====================================================
CREATE TABLE task_executions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    assignment_id UUID NOT NULL REFERENCES task_assignments(id) ON DELETE CASCADE,
    status task_status DEFAULT 'pending',
    started_at TIMESTAMP,
    completed_at TIMESTAMP,
    notes TEXT,
    attachments JSONB,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

COMMENT ON TABLE task_executions IS 'Tracks whether a task assignment was completed';

-- =====================================================
-- 12. Leaves Table
-- =====================================================
CREATE TABLE leaves (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    employee_id UUID NOT NULL REFERENCES employees(id) ON DELETE CASCADE,
    leave_type leave_type NOT NULL,
    start_date DATE NOT NULL,
    end_date DATE NOT NULL,
    total_days INTEGER GENERATED ALWAYS AS (end_date - start_date + 1) STORED,
    reason TEXT,
    status leave_status DEFAULT 'pending',
    applied_date DATE DEFAULT CURRENT_DATE,
    approved_by_team_leader UUID REFERENCES employees(id),
    approved_by_manager UUID REFERENCES employees(id),
    rejection_reason TEXT,
    attachments JSONB,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

COMMENT ON TABLE leaves IS 'Leave requests';

-- =====================================================
-- 13. Shift Swap Requests Table
-- =====================================================
CREATE TABLE shift_swaps (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    requester_id UUID NOT NULL REFERENCES employees(id),
    target_employee_id UUID NOT NULL REFERENCES employees(id),
    shift_date DATE NOT NULL,
    shift_id UUID NOT NULL REFERENCES shifts(id),
    reason TEXT,
    status swap_status DEFAULT 'pending',
    approved_by_team_leader UUID REFERENCES employees(id),
    approved_by_manager UUID REFERENCES employees(id),
    approval_date TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

COMMENT ON TABLE shift_swaps IS 'Shift swap requests between employees';

-- =====================================================
-- 14. Permissions Table
-- =====================================================
CREATE TABLE permissions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    role employee_role NOT NULL,
    permission_name VARCHAR(100) NOT NULL,
    resource VARCHAR(50) NOT NULL,  -- schedule, employees, tasks, leaves, reports
    can_view BOOLEAN DEFAULT false,
    can_create BOOLEAN DEFAULT false,
    can_edit BOOLEAN DEFAULT false,
    can_delete BOOLEAN DEFAULT false,
    can_approve BOOLEAN DEFAULT false,
    department_restricted BOOLEAN DEFAULT true,
    max_edit_days_ahead INTEGER,
    UNIQUE (role, permission_name, resource)
);

COMMENT ON TABLE permissions IS 'Role-based permissions';

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

-- =====================================================
-- 15. Notifications Table
-- =====================================================
CREATE TABLE notifications (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    recipient_id UUID NOT NULL REFERENCES employees(id) ON DELETE CASCADE,
    sender_id UUID REFERENCES employees(id),
    type notification_type NOT NULL,
    title VARCHAR(200) NOT NULL,
    message TEXT,
    related_entity_type VARCHAR(30),  -- schedule, task, leave, swap
    related_entity_id UUID,
    priority notification_priority DEFAULT 'medium',
    is_read BOOLEAN DEFAULT false,
    read_at TIMESTAMP,
    action_url VARCHAR(255),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

COMMENT ON TABLE notifications IS 'Internal notifications';

-- =====================================================
-- 16. Audit Logs Table
-- =====================================================
CREATE TABLE audit_logs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    employee_id UUID REFERENCES employees(id),
    action VARCHAR(50) NOT NULL,  -- INSERT, UPDATE, DELETE
    table_name VARCHAR(50) NOT NULL,
    record_id UUID,
    old_data JSONB,
    new_data JSONB,
    ip_address INET,
    user_agent TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

COMMENT ON TABLE audit_logs IS 'Log of all important modifications';

-- =====================================================
-- 17. Daily Statistics Table
-- =====================================================
CREATE TABLE daily_stats (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    stat_date DATE UNIQUE NOT NULL,
    total_employees INTEGER,
    present_count INTEGER,
    absent_count INTEGER,
    leave_count INTEGER,
    vacation_count INTEGER,
    night_shift_count INTEGER,
    morning_shift_count INTEGER,
    evening_shift_count INTEGER,
    open_tasks_count INTEGER,
    completed_tasks_count INTEGER,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

COMMENT ON TABLE daily_stats IS 'Quick daily statistics';

-- =====================================================
-- 18. Task Boards Table (dynamic, user-defined boards)
-- =====================================================
CREATE TABLE task_boards (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(100) NOT NULL,
    description TEXT,
    recurrence_type VARCHAR(10) NOT NULL DEFAULT 'weekly' CHECK (recurrence_type IN ('daily', 'weekly')),
    is_active BOOLEAN DEFAULT true,
    department_id UUID REFERENCES departments(id) ON DELETE CASCADE,
    created_by UUID REFERENCES employees(id),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

COMMENT ON TABLE task_boards IS 'Custom task boards created by team leaders';
COMMENT ON COLUMN task_boards.recurrence_type IS 'daily = tasks repeat every day, weekly = tasks on specific days of the week';

-- Add board_id to task_schedules
ALTER TABLE task_schedules ADD COLUMN board_id UUID REFERENCES task_boards(id) ON DELETE CASCADE;

-- Make schedule_type nullable (kept for backward compat / boards)
ALTER TABLE task_schedules DROP CONSTRAINT IF EXISTS task_schedules_schedule_type_check;
ALTER TABLE task_schedules ALTER COLUMN schedule_type DROP NOT NULL;

-- =====================================================
-- 19. Link default_shift_id
-- =====================================================
ALTER TABLE employees ADD CONSTRAINT fk_employees_default_shift
    FOREIGN KEY (default_shift_id) REFERENCES shifts(id);

-- =====================================================
-- 20. Indexes
-- =====================================================

-- employee_shifts indexes
CREATE INDEX idx_employee_shifts_date ON employee_shifts(shift_date);
CREATE INDEX idx_employee_shifts_employee ON employee_shifts(employee_id);
CREATE INDEX idx_employee_shifts_status ON employee_shifts(shift_status);

-- leaves indexes
CREATE INDEX idx_leaves_employee_dates ON leaves(employee_id, start_date, end_date);
CREATE INDEX idx_leaves_status ON leaves(status);

-- task_schedules indexes
CREATE INDEX idx_task_schedules_type ON task_schedules(schedule_type);
CREATE INDEX idx_task_schedules_shift ON task_schedules(shift_id);
CREATE INDEX idx_task_schedules_active ON task_schedules(is_active);

-- task_assignments indexes
CREATE INDEX idx_task_assignments_date ON task_assignments(assigned_date);
CREATE INDEX idx_task_assignments_employee ON task_assignments(employee_id);
CREATE INDEX idx_task_assignments_schedule ON task_assignments(schedule_id);
CREATE INDEX idx_task_assignments_employee_date ON task_assignments(employee_id, assigned_date);

-- task_executions indexes
CREATE INDEX idx_task_executions_status ON task_executions(status);
CREATE INDEX idx_task_executions_assignment ON task_executions(assignment_id);

-- audit_logs indexes
CREATE INDEX idx_audit_logs_employee ON audit_logs(employee_id);
CREATE INDEX idx_audit_logs_table_record ON audit_logs(table_name, record_id);

-- notifications indexes
CREATE INDEX idx_notifications_recipient_read ON notifications(recipient_id, is_read);
CREATE INDEX idx_notifications_created ON notifications(created_at);
CREATE INDEX idx_notifications_related ON notifications(related_entity_type, related_entity_id);

-- schedule_templates indexes
CREATE INDEX idx_schedule_templates_employee ON schedule_templates(employee_id);
CREATE INDEX idx_schedule_templates_dates ON schedule_templates(valid_from, valid_to);

-- shift_swaps indexes
CREATE INDEX idx_shift_swaps_date ON shift_swaps(shift_date);

-- =====================================================
-- 21. Uniqueness Constraints
-- =====================================================

-- Ensure email is unique (case-insensitive)
CREATE UNIQUE INDEX IF NOT EXISTS uq_employees_email_ci
ON employees (LOWER(email));

-- Ensure phone number is unique when provided
CREATE UNIQUE INDEX IF NOT EXISTS uq_employees_phone_not_null
ON employees (phone)
WHERE phone IS NOT NULL;

-- Ensure one manager cannot manage multiple departments
CREATE UNIQUE INDEX IF NOT EXISTS uq_departments_manager_single
ON departments (manager_id)
WHERE manager_id IS NOT NULL;

-- Ensure department names are unique
CREATE UNIQUE INDEX IF NOT EXISTS uq_departments_name_ci
ON departments (LOWER(name));

-- =====================================================
-- 22. Views
-- =====================================================

-- Weekly schedule view with employee and shift names
CREATE VIEW view_weekly_schedule AS
SELECT
    ws.week_start_date,
    ws.week_end_date,
    e.id AS employee_id,
    e.first_name || ' ' || e.last_name AS employee_name,
    e.gender,
    es.shift_date,
    s.name AS shift_name,
    s.shift_code,
    es.shift_status,
    es.is_replacement,
    es.check_in_time,
    es.check_out_time
FROM weekly_schedule ws
JOIN employee_shifts es ON ws.id = es.schedule_id
JOIN employees e ON es.employee_id = e.id
LEFT JOIN shifts s ON es.shift_id = s.id
ORDER BY es.shift_date, e.first_name;

-- View of employees eligible to cover night shifts
CREATE VIEW view_eligible_night_replacements AS
SELECT
    e.id,
    e.first_name,
    e.last_name,
    e.gender,
    es.shift_date AS missing_date
FROM employees e
JOIN employee_shifts es ON e.id = es.employee_id
WHERE e.gender = 'male'
  AND e.can_cover_night_shift = true
  AND es.shift_id IN (SELECT id FROM shifts WHERE shift_code IN ('M','E'))
  AND es.shift_status = 'working'
  AND EXISTS (
      SELECT 1 FROM employee_shifts es_prev
      WHERE es_prev.employee_id = e.id
        AND es_prev.shift_date = es.shift_date - 1
        AND es_prev.shift_status IN ('off', 'leave')
  );

-- Daily Tasks view filtered by shift
CREATE VIEW view_daily_tasks_by_shift AS
SELECT
    s.id AS shift_id,
    s.name AS shift_name,
    s.shift_code,
    ts.id AS schedule_id,
    ts.title AS task_title,
    ts.description AS task_description,
    ts.recurrence,
    ts.recurrence_days,
    ta.id AS assignment_id,
    ta.assigned_date,
    ta.employee_id,
    e.first_name || ' ' || e.last_name AS employee_name,
    e.employee_code,
    te.status AS execution_status,
    te.completed_at,
    te.notes AS execution_notes
FROM task_schedules ts
JOIN shifts s ON ts.shift_id = s.id
LEFT JOIN task_assignments ta ON ts.id = ta.schedule_id
LEFT JOIN employees e ON ta.employee_id = e.id
LEFT JOIN task_executions te ON ta.id = te.assignment_id
WHERE ts.schedule_type = 'daily_task'
  AND ts.is_active = true;

-- Node Check view filtered by shift
CREATE VIEW view_node_check_by_shift AS
SELECT
    s.id AS shift_id,
    s.name AS shift_name,
    s.shift_code,
    ts.id AS schedule_id,
    ts.title AS task_title,
    ts.description AS task_description,
    ts.recurrence,
    ts.recurrence_days,
    ts.max_assignees,
    ta.id AS assignment_id,
    ta.assigned_date,
    ta.employee_id,
    e.first_name || ' ' || e.last_name AS employee_name,
    e.employee_code,
    te.status AS execution_status,
    te.completed_at,
    te.notes AS execution_notes
FROM task_schedules ts
JOIN shifts s ON ts.shift_id = s.id
LEFT JOIN task_assignments ta ON ts.id = ta.schedule_id
LEFT JOIN employees e ON ta.employee_id = e.id
LEFT JOIN task_executions te ON ta.id = te.assignment_id
WHERE ts.schedule_type = 'node_check'
  AND ts.is_active = true;

-- Mobile Ticket view filtered by shift
CREATE VIEW view_mobile_ticket_by_shift AS
SELECT
    s.id AS shift_id,
    s.name AS shift_name,
    s.shift_code,
    ts.id AS schedule_id,
    ts.title AS task_title,
    ts.description AS task_description,
    ts.recurrence,
    ts.recurrence_days,
    ts.max_assignees,
    ta.id AS assignment_id,
    ta.assigned_date,
    ta.employee_id,
    e.first_name || ' ' || e.last_name AS employee_name,
    e.employee_code,
    te.status AS execution_status,
    te.completed_at,
    te.notes AS execution_notes
FROM task_schedules ts
JOIN shifts s ON ts.shift_id = s.id
LEFT JOIN task_assignments ta ON ts.id = ta.schedule_id
LEFT JOIN employees e ON ta.employee_id = e.id
LEFT JOIN task_executions te ON ta.id = te.assignment_id
WHERE ts.schedule_type = 'mobile_ticket'
  AND ts.is_active = true;

-- Employee today's tasks view (all types combined)
CREATE VIEW view_employee_today_tasks AS
SELECT
    ta.employee_id,
    e.first_name || ' ' || e.last_name AS employee_name,
    ts.schedule_type,
    ts.title AS task_title,
    ts.description AS task_description,
    s.name AS shift_name,
    ta.assigned_date,
    te.status AS execution_status,
    te.completed_at,
    te.notes AS execution_notes
FROM task_assignments ta
JOIN task_schedules ts ON ta.schedule_id = ts.id
JOIN employees e ON ta.employee_id = e.id
LEFT JOIN shifts s ON ts.shift_id = s.id
LEFT JOIN task_executions te ON ta.id = te.assignment_id
WHERE ta.assigned_date = CURRENT_DATE
  AND ts.is_active = true;

-- =====================================================
-- 23. Functions
-- =====================================================

-- Automatically update the updated_at field
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Function to generate a weekly schedule from a template
CREATE OR REPLACE FUNCTION generate_weekly_schedule(p_week_start DATE, p_created_by UUID)
RETURNS UUID AS $$
DECLARE
    v_schedule_id UUID;
    v_template_id UUID;
    v_week_end DATE := p_week_start + 6;
BEGIN
    -- Use the latest valid template
    SELECT id INTO v_template_id FROM schedule_templates
    WHERE (valid_from IS NULL OR valid_from <= p_week_start)
      AND (valid_to IS NULL OR valid_to >= v_week_end)
    LIMIT 1;

    -- Create the week record
    INSERT INTO weekly_schedule (week_start_date, week_end_date, template_id, status, created_by)
    VALUES (p_week_start, v_week_end, v_template_id, 'draft', p_created_by)
    RETURNING id INTO v_schedule_id;

    -- Copy the template to employee_shifts
    INSERT INTO employee_shifts (schedule_id, employee_id, shift_id, shift_date, shift_status, created_by)
    SELECT
        v_schedule_id,
        st.employee_id,
        st.shift_id,
        p_week_start + st.day_of_week,
        CASE WHEN st.is_off THEN 'off'::shift_status_type ELSE 'working'::shift_status_type END,
        p_created_by
    FROM schedule_templates st
    WHERE (st.valid_from IS NULL OR st.valid_from <= p_week_start)
      AND (st.valid_to IS NULL OR st.valid_to >= v_week_end)
      AND EXISTS (SELECT 1 FROM employees e WHERE e.id = st.employee_id AND e.status = 'active');

    RETURN v_schedule_id;
END;
$$ LANGUAGE plpgsql;

-- Function to find a replacement for a night shift
CREATE OR REPLACE FUNCTION find_replacement_for_night(p_date DATE, p_exclude_employee_id UUID DEFAULT NULL)
RETURNS TABLE (
    employee_id UUID,
    full_name TEXT,
    phone VARCHAR(20)
) AS $$
BEGIN
    RETURN QUERY
    SELECT
        e.id,
        e.first_name || ' ' || e.last_name,
        e.phone
    FROM employees e
    JOIN employee_shifts es ON e.id = es.employee_id
    WHERE e.gender = 'male'
      AND e.can_cover_night_shift = true
      AND es.shift_date = p_date
      AND es.shift_id IN (SELECT id FROM shifts WHERE shift_code IN ('M','E'))
      AND es.shift_status = 'working'
      AND (p_exclude_employee_id IS NULL OR e.id != p_exclude_employee_id)
      AND EXISTS (
          SELECT 1 FROM employee_shifts es_prev
          WHERE es_prev.employee_id = e.id
            AND es_prev.shift_date = p_date - 1
            AND es_prev.shift_status IN ('off', 'leave')
      )
    LIMIT 5;
END;
$$ LANGUAGE plpgsql;

-- Generate task assignments for a given date based on active schedules
CREATE OR REPLACE FUNCTION generate_daily_assignments(p_date DATE)
RETURNS void AS $$
DECLARE
    v_dow INTEGER := EXTRACT(DOW FROM p_date)::INTEGER;
    v_schedule RECORD;
BEGIN
    FOR v_schedule IN
        SELECT * FROM task_schedules
        WHERE is_active = true
    LOOP
        -- Check if this task should run on this date
        IF v_schedule.recurrence = 'daily' THEN
            IF v_schedule.recurrence_days IS NULL
               OR v_dow = ANY(v_schedule.recurrence_days) THEN
                NULL; -- Manager assigns manually via UI
            END IF;
        ELSIF v_schedule.recurrence = 'periodic' THEN
            IF v_schedule.recurrence_days IS NOT NULL
               AND v_dow = ANY(v_schedule.recurrence_days) THEN
                NULL; -- Manager assigns manually via UI
            END IF;
        END IF;
    END LOOP;
END;
$$ LANGUAGE plpgsql;

-- Get all tasks for an employee on a specific date
CREATE OR REPLACE FUNCTION get_employee_today_tasks(p_employee_id UUID, p_date DATE DEFAULT CURRENT_DATE)
RETURNS TABLE (
    assignment_id UUID,
    schedule_type VARCHAR,
    task_title VARCHAR,
    task_description TEXT,
    shift_name VARCHAR,
    execution_status task_status,
    completed_at TIMESTAMP,
    execution_notes TEXT
) AS $$
BEGIN
    RETURN QUERY
    SELECT
        ta.id,
        ts.schedule_type::VARCHAR,
        ts.title,
        ts.description,
        s.name,
        te.status,
        te.completed_at,
        te.notes
    FROM task_assignments ta
    JOIN task_schedules ts ON ta.schedule_id = ts.id
    LEFT JOIN shifts s ON ts.shift_id = s.id
    LEFT JOIN task_executions te ON ta.id = te.assignment_id
    WHERE ta.employee_id = p_employee_id
      AND ta.assigned_date = p_date
      AND ts.is_active = true
    ORDER BY ts.schedule_type, ts.title;
END;
$$ LANGUAGE plpgsql;

-- Swap task assignments between two employees on the same date
CREATE OR REPLACE FUNCTION swap_task_assignment(p_assignment_id_1 UUID, p_assignment_id_2 UUID)
RETURNS BOOLEAN AS $$
DECLARE
    v_emp_1 UUID;
    v_emp_2 UUID;
    v_date_1 DATE;
    v_date_2 DATE;
BEGIN
    SELECT employee_id, assigned_date INTO v_emp_1, v_date_1
    FROM task_assignments WHERE id = p_assignment_id_1;

    SELECT employee_id, assigned_date INTO v_emp_2, v_date_2
    FROM task_assignments WHERE id = p_assignment_id_2;

    IF v_emp_1 IS NULL OR v_emp_2 IS NULL THEN
        RAISE EXCEPTION 'One or both assignment IDs not found';
    END IF;

    IF v_date_1 != v_date_2 THEN
        RAISE EXCEPTION 'Cannot swap assignments on different dates';
    END IF;

    UPDATE task_assignments SET employee_id = v_emp_2 WHERE id = p_assignment_id_1;
    UPDATE task_assignments SET employee_id = v_emp_1 WHERE id = p_assignment_id_2;

    RETURN true;
END;
$$ LANGUAGE plpgsql;

-- Check if a task schedule should appear on a given date
CREATE OR REPLACE FUNCTION should_task_run_on_date(p_schedule_id UUID, p_date DATE)
RETURNS BOOLEAN AS $$
DECLARE
    v_schedule RECORD;
    v_dow INTEGER := EXTRACT(DOW FROM p_date)::INTEGER;
BEGIN
    SELECT * INTO v_schedule FROM task_schedules WHERE id = p_schedule_id;

    IF v_schedule IS NULL OR NOT v_schedule.is_active THEN
        RETURN false;
    END IF;

    IF v_schedule.recurrence = 'daily' THEN
        IF v_schedule.recurrence_days IS NULL THEN
            RETURN true; -- every day
        ELSE
            RETURN v_dow = ANY(v_schedule.recurrence_days);
        END IF;
    ELSIF v_schedule.recurrence = 'periodic' THEN
        IF v_schedule.recurrence_days IS NOT NULL THEN
            RETURN v_dow = ANY(v_schedule.recurrence_days);
        END IF;
    END IF;

    RETURN false;
END;
$$ LANGUAGE plpgsql;

-- =====================================================
-- 24. Triggers
-- =====================================================

CREATE TRIGGER trigger_update_employees_updated_at
    BEFORE UPDATE ON employees
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

DROP TRIGGER IF EXISTS trigger_update_departments_updated_at ON departments;
CREATE TRIGGER trigger_update_departments_updated_at
    BEFORE UPDATE ON departments
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

DROP TRIGGER IF EXISTS trigger_update_employee_shifts_updated_at ON employee_shifts;
CREATE TRIGGER trigger_update_employee_shifts_updated_at
    BEFORE UPDATE ON employee_shifts
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

DROP TRIGGER IF EXISTS trigger_update_leaves_updated_at ON leaves;
CREATE TRIGGER trigger_update_leaves_updated_at
    BEFORE UPDATE ON leaves
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

DROP TRIGGER IF EXISTS trigger_update_task_schedules_updated_at ON task_schedules;
CREATE TRIGGER trigger_update_task_schedules_updated_at
    BEFORE UPDATE ON task_schedules
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

DROP TRIGGER IF EXISTS trigger_update_task_executions_updated_at ON task_executions;
CREATE TRIGGER trigger_update_task_executions_updated_at
    BEFORE UPDATE ON task_executions
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

DROP TRIGGER IF EXISTS trigger_update_shift_swaps_updated_at ON shift_swaps;
CREATE TRIGGER trigger_update_shift_swaps_updated_at
    BEFORE UPDATE ON shift_swaps
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Audit trigger for employee_shifts
CREATE OR REPLACE FUNCTION audit_employee_shifts()
RETURNS TRIGGER AS $$
BEGIN
    IF TG_OP = 'INSERT' THEN
        INSERT INTO audit_logs (employee_id, action, table_name, record_id, new_data, created_at)
        VALUES (NEW.created_by, 'INSERT', 'employee_shifts', NEW.id, row_to_json(NEW), CURRENT_TIMESTAMP);
    ELSIF TG_OP = 'UPDATE' THEN
        INSERT INTO audit_logs (employee_id, action, table_name, record_id, old_data, new_data, created_at)
        VALUES (NEW.created_by, 'UPDATE', 'employee_shifts', NEW.id, row_to_json(OLD), row_to_json(NEW), CURRENT_TIMESTAMP);
    ELSIF TG_OP = 'DELETE' THEN
        INSERT INTO audit_logs (employee_id, action, table_name, record_id, old_data, created_at)
        VALUES (OLD.created_by, 'DELETE', 'employee_shifts', OLD.id, row_to_json(OLD), CURRENT_TIMESTAMP);
    END IF;
    RETURN NULL;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trigger_audit_employee_shifts ON employee_shifts;
DROP TRIGGER IF EXISTS trg_audit_employee_shifts ON employee_shifts;

CREATE TRIGGER trg_audit_employee_shifts
    AFTER INSERT OR UPDATE OR DELETE ON employee_shifts
    FOR EACH ROW EXECUTE FUNCTION audit_employee_shifts();

-- =====================================================
-- 25. Add Sample Employees (with passwords)
-- =====================================================
-- Password: password (hashed with bcrypt)

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

-- =====================================================
-- 26. Add Default Schedule Templates
-- =====================================================
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
-- 27. Sample Task Schedules & Assignments
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
-- 28. Generate First Weekly Schedule
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

-- =====================================================
-- End of Script
-- =====================================================