-- =====================================================
-- Migration: Leave Balances System
-- =====================================================

-- Add quota and carry forward settings to leave_types
ALTER TABLE leave_types 
ADD COLUMN days_per_year INT DEFAULT 0,
ADD COLUMN carries_forward BOOLEAN DEFAULT false;

-- Create employee_leave_balances table
CREATE TABLE employee_leave_balances (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    employee_id UUID NOT NULL REFERENCES employees(id) ON DELETE CASCADE,
    leave_type_id UUID NOT NULL REFERENCES leave_types(id) ON DELETE CASCADE,
    year INT NOT NULL,
    allocated_days FLOAT NOT NULL DEFAULT 0,
    used_days FLOAT NOT NULL DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(employee_id, leave_type_id, year)
);

-- Trigger for updating updated_at timestamp
CREATE TRIGGER set_timestamp_employee_leave_balances
BEFORE UPDATE ON employee_leave_balances
FOR EACH ROW
EXECUTE FUNCTION update_updated_at_column();

COMMENT ON TABLE employee_leave_balances IS 'Tracks allocated and used leave days per employee per year';
