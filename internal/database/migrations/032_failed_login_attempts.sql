-- Add failed_login_attempts to employees (idempotent)
ALTER TABLE employees ADD COLUMN IF NOT EXISTS failed_login_attempts INT DEFAULT 0;
