-- =====================================================
-- Migration: Monthly Hourly Leaves (idempotent)
-- =====================================================

-- Add unit and reset_cycle to leave_types
ALTER TABLE leave_types
ADD COLUMN IF NOT EXISTS unit VARCHAR(10) DEFAULT 'days',
ADD COLUMN IF NOT EXISTS reset_cycle VARCHAR(10) DEFAULT 'annual';

-- Update existing records to default values
UPDATE leave_types SET unit = 'days', reset_cycle = 'annual' WHERE unit IS NULL;

-- Add month to employee_leave_balances
ALTER TABLE employee_leave_balances
ADD COLUMN IF NOT EXISTS month INT DEFAULT 0;

-- Update existing records to month 0 (annual)
UPDATE employee_leave_balances SET month = 0 WHERE month IS NULL;

-- Drop old unique constraint and add new one (idempotent)
ALTER TABLE employee_leave_balances
DROP CONSTRAINT IF EXISTS employee_leave_balances_employee_id_leave_type_id_year_key;

ALTER TABLE employee_leave_balances
DROP CONSTRAINT IF EXISTS employee_leave_balances_unique_emp_type_year_month;

ALTER TABLE employee_leave_balances
ADD CONSTRAINT employee_leave_balances_unique_emp_type_year_month UNIQUE (employee_id, leave_type_id, year, month);

-- Rename columns to be more generic (idempotent via DO block)
DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'employee_leave_balances' AND column_name = 'allocated_days'
    ) THEN
        ALTER TABLE employee_leave_balances RENAME COLUMN allocated_days TO allocated_amount;
    END IF;
    IF EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'employee_leave_balances' AND column_name = 'used_days'
    ) THEN
        ALTER TABLE employee_leave_balances RENAME COLUMN used_days TO used_amount;
    END IF;
END $$;
