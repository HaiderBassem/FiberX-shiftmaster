-- =====================================================
-- TRIGGERS
-- =====================================================

-- Auto-update updated_at on important tables
CREATE TRIGGER trigger_update_employees_updated_at
    BEFORE UPDATE ON employees
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER trigger_update_departments_updated_at
    BEFORE UPDATE ON departments
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER trigger_update_employee_shifts_updated_at
    BEFORE UPDATE ON employee_shifts
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER trigger_update_leaves_updated_at
    BEFORE UPDATE ON leaves
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER trigger_update_task_schedules_updated_at
    BEFORE UPDATE ON task_schedules
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER trigger_update_task_executions_updated_at
    BEFORE UPDATE ON task_executions
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER trigger_update_shift_swaps_updated_at
    BEFORE UPDATE ON shift_swaps
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Trigger to log modifications in audit_logs (example on employee_shifts)
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

CREATE TRIGGER trigger_audit_employee_shifts
    AFTER INSERT OR UPDATE OR DELETE ON employee_shifts
    FOR EACH ROW EXECUTE FUNCTION audit_employee_shifts();
