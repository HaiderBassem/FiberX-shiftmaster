-- =====================================================
-- Employees Table
-- =====================================================

CREATE TABLE employees (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    employee_code VARCHAR(20) UNIQUE NOT NULL,
    first_name VARCHAR(50) NOT NULL,
    last_name VARCHAR(50) NOT NULL,
    gender gender_type NOT NULL,
    phone VARCHAR(20),
    email VARCHAR(100) UNIQUE,
    hire_date DATE NOT NULL,
    role employee_role NOT NULL DEFAULT 'employee',
    department VARCHAR(50),
    position VARCHAR(100),
    default_shift_id UUID,          -- Will be linked later to shifts table
    weekly_off_days INTEGER DEFAULT 1,
    can_cover_night_shift BOOLEAN DEFAULT false,
    status employee_status DEFAULT 'active',
    profile_image VARCHAR(255),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    created_by UUID REFERENCES employees(id)
);

COMMENT ON TABLE employees IS 'Core employee data';
COMMENT ON COLUMN employees.can_cover_night_shift IS 'Can this employee cover a night shift?';

-- =====================================================
-- Shifts Table (Shift Types)
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
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

COMMENT ON TABLE shifts IS 'Shift types (morning, evening, night)';


-- =====================================================
-- Weekly Schedule Templates (Base Fixed Schedule)
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
-- Published Weekly Schedules
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
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (week_start_date)
);

COMMENT ON TABLE weekly_schedule IS 'Each published week';

-- =====================================================
-- Employee Shifts (Actual Daily Assignments)
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
-- Leaves Table
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
-- Task Schedules (defines tasks and their recurrence)
-- =====================================================

CREATE TABLE task_schedules (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    title VARCHAR(200) NOT NULL,
    description TEXT,
    schedule_type VARCHAR(20) NOT NULL CHECK (schedule_type IN ('daily_task', 'node_check', 'mobile_ticket')),
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
-- Task Assignments (who does what on which date)
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
-- Task Executions (completion tracking)
-- =====================================================

CREATE TABLE task_executions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    assignment_id UUID NOT NULL REFERENCES task_assignments(id) ON DELETE CASCADE,
    status task_status DEFAULT 'pending',
    completed_at TIMESTAMP,
    notes TEXT,
    attachments JSONB,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

COMMENT ON TABLE task_executions IS 'Tracks whether a task assignment was completed';

-- =====================================================
-- Permissions Table
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

-- =====================================================
-- Shift Swap Requests Table
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
-- Audit Logs Table
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
-- Notifications Table
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
-- Daily Statistics Table (can be converted to TimescaleDB)
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

-- If using TimescaleDB, convert daily_stats to a hypertable
-- SELECT create_hypertable('daily_stats', 'stat_date');

-- =====================================================
-- Link default_shift_id after creating shifts table
-- =====================================================

ALTER TABLE employees ADD CONSTRAINT fk_employees_default_shift
    FOREIGN KEY (default_shift_id) REFERENCES shifts(id);
