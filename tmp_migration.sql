-- Fix 025
CREATE TRIGGER set_timestamp_employee_leave_balances
BEFORE UPDATE ON employee_leave_balances
FOR EACH ROW
EXECUTE FUNCTION update_updated_at_column();

-- Run 026
-- Add unit and reset_cycle to leave_types
ALTER TABLE leave_types 
ADD COLUMN unit VARCHAR(10) DEFAULT 'days',
ADD COLUMN reset_cycle VARCHAR(10) DEFAULT 'annual';

-- Update existing records to default values
UPDATE leave_types SET unit = 'days', reset_cycle = 'annual' WHERE unit IS NULL;

-- Add month to employee_leave_balances
ALTER TABLE employee_leave_balances
ADD COLUMN month INT DEFAULT 0;

-- Update existing records to month 0 (annual)
UPDATE employee_leave_balances SET month = 0 WHERE month IS NULL;

-- Drop old unique constraint and add new one
ALTER TABLE employee_leave_balances
DROP CONSTRAINT employee_leave_balances_employee_id_leave_type_id_year_key,
ADD CONSTRAINT employee_leave_balances_unique_emp_type_year_month UNIQUE (employee_id, leave_type_id, year, month);

-- Rename columns to be more generic
ALTER TABLE employee_leave_balances RENAME COLUMN allocated_days TO allocated_amount;
ALTER TABLE employee_leave_balances RENAME COLUMN used_days TO used_amount;
