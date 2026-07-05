-- =====================================================
-- Task Boards Table (dynamic, user-defined boards)
-- =====================================================

CREATE TABLE task_boards (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(100) NOT NULL,
    description TEXT,
    recurrence_type VARCHAR(10) NOT NULL DEFAULT 'weekly' CHECK (recurrence_type IN ('daily', 'weekly')),
    is_active BOOLEAN DEFAULT true,
    created_by UUID REFERENCES employees(id),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

COMMENT ON TABLE task_boards IS 'Custom task boards created by team leaders';
COMMENT ON COLUMN task_boards.recurrence_type IS 'daily = tasks repeat every day, weekly = tasks on specific days of the week';

-- =====================================================
-- Add board_id to task_schedules
-- =====================================================

ALTER TABLE task_schedules ADD COLUMN board_id UUID REFERENCES task_boards(id) ON DELETE CASCADE;

-- Drop the old CHECK constraint on schedule_type
ALTER TABLE task_schedules DROP CONSTRAINT IF EXISTS task_schedules_schedule_type_check;

-- Make schedule_type nullable (kept for backward compat / migration)
ALTER TABLE task_schedules ALTER COLUMN schedule_type DROP NOT NULL;
