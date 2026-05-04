-- =====================================================
-- Task Recurring Assignments Table
-- =====================================================

CREATE TABLE task_recurring_assignments (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    schedule_id UUID NOT NULL REFERENCES task_schedules(id) ON DELETE CASCADE,
    employee_id UUID NOT NULL REFERENCES employees(id) ON DELETE CASCADE,
    day_of_week INT NOT NULL CHECK (day_of_week >= 0 AND day_of_week <= 6),
    assigned_by UUID REFERENCES employees(id),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(schedule_id, employee_id, day_of_week)
);

COMMENT ON TABLE task_recurring_assignments IS 'Recurring task assignments (which day of week an employee performs a schedule)';
