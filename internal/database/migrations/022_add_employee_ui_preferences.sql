-- Add ui_preferences to employees table (idempotent)
ALTER TABLE employees
ADD COLUMN IF NOT EXISTS ui_preferences JSONB DEFAULT '{}'::jsonb;

COMMENT ON COLUMN employees.ui_preferences IS 'Stores user-specific UI layout preferences, such as drag-and-drop folders for Info Tables and Help Docs';
