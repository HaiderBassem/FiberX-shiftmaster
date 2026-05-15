-- =====================================================
-- Migration 016: Multi-Manager Departments
-- Allows a manager to manage more than one department
-- by replacing the single manager_id FK on departments
-- with a dedicated junction table.
-- Also promotes the inline schema patches that were
-- previously applied at startup in main.go into proper
-- migration SQL so they are idempotent and version-tracked.
-- =====================================================

-- -----------------------------------------------------
-- Part 1: task_boards table (was patched in main.go)
-- -----------------------------------------------------
CREATE TABLE IF NOT EXISTS task_boards (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(100) NOT NULL,
    description TEXT,
    recurrence_type VARCHAR(10) NOT NULL DEFAULT 'weekly'
        CHECK (recurrence_type IN ('daily', 'weekly')),
    is_active BOOLEAN DEFAULT true,
    created_by UUID REFERENCES employees(id),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

DO $$ BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'task_schedules' AND column_name = 'board_id'
    ) THEN
        ALTER TABLE task_schedules
            ADD COLUMN board_id UUID REFERENCES task_boards(id) ON DELETE CASCADE;
    END IF;
END $$;

DO $$ BEGIN
    BEGIN
        ALTER TABLE task_schedules ALTER COLUMN schedule_type DROP NOT NULL;
    EXCEPTION WHEN others THEN
        NULL; -- already nullable
    END;
END $$;

DO $$ BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'task_executions' AND column_name = 'started_at'
    ) THEN
        ALTER TABLE task_executions ADD COLUMN started_at TIMESTAMP;
    END IF;
END $$;

-- -----------------------------------------------------
-- Part 2: swap_status enum extension (was patched in main.go)
-- -----------------------------------------------------
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM pg_type WHERE typname = 'swap_status') THEN
        IF NOT EXISTS (
            SELECT 1
            FROM pg_enum e
            JOIN pg_type t ON t.oid = e.enumtypid
            WHERE t.typname = 'swap_status'
              AND e.enumlabel = 'employee_accepted'
        ) THEN
            ALTER TYPE swap_status ADD VALUE 'employee_accepted';
        END IF;
    END IF;
END $$;

-- -----------------------------------------------------
-- Part 3: department_managers junction table
-- (replaces the single manager_id column)
-- -----------------------------------------------------

-- 3a. Create the junction table
CREATE TABLE IF NOT EXISTS department_managers (
    department_id UUID NOT NULL REFERENCES departments(id) ON DELETE CASCADE,
    manager_id    UUID NOT NULL REFERENCES employees(id)  ON DELETE CASCADE,
    assigned_at   TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (department_id, manager_id)
);

COMMENT ON TABLE department_managers IS
    'Many-to-many: one manager can oversee multiple departments';

-- 3b. Migrate existing data from departments.manager_id
INSERT INTO department_managers (department_id, manager_id)
SELECT id, manager_id
FROM   departments
WHERE  manager_id IS NOT NULL
ON CONFLICT DO NOTHING;

-- 3c. Drop the old FK constraint and column
ALTER TABLE departments DROP CONSTRAINT IF EXISTS fk_departments_manager;
ALTER TABLE departments DROP COLUMN IF EXISTS manager_id;
