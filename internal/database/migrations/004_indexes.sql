-- =====================================================
-- Indexes
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
