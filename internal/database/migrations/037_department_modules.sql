-- Migration 037: Department Modules
-- Add active_modules column to departments (idempotent)
ALTER TABLE departments ADD COLUMN IF NOT EXISTS active_modules JSONB DEFAULT '["tasks", "handovers", "calendar", "requests", "references", "help", "fiberx_data", "task_center"]'::jsonb;

-- Backfill NULL values for existing rows
UPDATE departments
SET active_modules = '["tasks", "handovers", "calendar", "requests", "references", "help", "fiberx_data", "task_center"]'::jsonb
WHERE active_modules IS NULL;