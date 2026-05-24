-- Migrate live database to add multi-tenancy columns

-- Add columns
ALTER TABLE shifts ADD COLUMN IF NOT EXISTS department_id UUID REFERENCES departments(id) ON DELETE CASCADE;
ALTER TABLE task_boards ADD COLUMN IF NOT EXISTS department_id UUID REFERENCES departments(id) ON DELETE CASCADE;
ALTER TABLE weekly_schedule ADD COLUMN IF NOT EXISTS department_id UUID REFERENCES departments(id) ON DELETE CASCADE;

-- Update weekly_schedule unique constraint
ALTER TABLE weekly_schedule DROP CONSTRAINT IF EXISTS weekly_schedule_week_start_date_key;
-- Add the new unique constraint if we can
ALTER TABLE weekly_schedule ADD CONSTRAINT weekly_schedule_department_week_start_key UNIQUE (department_id, week_start_date);

-- Get the first department ID to use as a fallback (since user said there is only one currently)
DO $$
DECLARE
    v_dept_id UUID;
BEGIN
    SELECT id INTO v_dept_id FROM departments LIMIT 1;
    
    IF v_dept_id IS NOT NULL THEN
        UPDATE shifts SET department_id = v_dept_id WHERE department_id IS NULL;
        UPDATE task_boards SET department_id = v_dept_id WHERE department_id IS NULL;
        UPDATE weekly_schedule SET department_id = v_dept_id WHERE department_id IS NULL;
    END IF;
END $$;
