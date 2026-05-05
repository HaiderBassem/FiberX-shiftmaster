-- Migration 015: Add completion_type column to task_executions
-- This column was referenced in Go code but missing from the DB schema.

ALTER TABLE task_executions
    ADD COLUMN IF NOT EXISTS completion_type VARCHAR(20)
        CHECK (completion_type IN ('without_issue', 'with_issue'));

COMMENT ON COLUMN task_executions.completion_type IS 'How the task was completed: without_issue or with_issue';
