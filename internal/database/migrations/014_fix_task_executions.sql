-- =====================================================
-- Fix task_executions: unique constraint + backfill
-- =====================================================

-- Step 1: Remove any duplicate executions (keep the most recent one per assignment)
DELETE FROM task_executions
WHERE id NOT IN (
    SELECT DISTINCT ON (assignment_id) id
    FROM task_executions
    ORDER BY assignment_id, created_at DESC NULLS LAST
);

-- Step 2: Add UNIQUE constraint on assignment_id
ALTER TABLE task_executions
    ADD CONSTRAINT uq_task_executions_assignment
    UNIQUE (assignment_id);

-- Step 3: Backfill executions for any task_assignments that are missing one
INSERT INTO task_executions (assignment_id, status)
SELECT ta.id, 'pending'
FROM task_assignments ta
WHERE NOT EXISTS (
    SELECT 1 FROM task_executions te WHERE te.assignment_id = ta.id
)
ON CONFLICT (assignment_id) DO NOTHING;
