-- =============================================================================
-- 048_auth_lockout.sql
-- =============================================================================
-- Separates *authentication lockout* from *employment status*.
--
-- Before this migration, ten failed password attempts set employees.status to
-- 'inactive'. That conflated two unrelated concepts and had three consequences:
--
--   1. The lock never expired, so every lockout required an administrator.
--   2. Because the login identifier is an email address, anyone who knew a
--      colleague's address could permanently disable that colleague's account.
--   3. 'inactive' is read throughout the application as "no longer works here",
--      so a mistyped password removed the person from schedules and rosters.
--
-- locked_until carries the lock instead. It is advisory and self-healing: once
-- the timestamp passes, the account authenticates normally again with no
-- administrator involvement.
--
-- NOTE ON BACKFILL: accounts already switched to 'inactive' by the old lockout
-- path are deliberately NOT reactivated here. There is no stored evidence that
-- distinguishes them from employees an administrator deactivated on purpose, and
-- silently re-enabling a terminated employee's login would be a worse defect than
-- the one being fixed. Those accounts must be reactivated by hand.
-- =============================================================================

ALTER TABLE employees ADD COLUMN IF NOT EXISTS locked_until TIMESTAMPTZ;

COMMENT ON COLUMN employees.locked_until IS
    'Authentication lock expiry. NULL or in the past means not locked. Distinct from employees.status, which records employment state.';

-- failed_login_attempts was added in 032 without NOT NULL, so existing rows may
-- hold NULL and break arithmetic in the atomic increment below.
UPDATE employees SET failed_login_attempts = 0 WHERE failed_login_attempts IS NULL;

ALTER TABLE employees ALTER COLUMN failed_login_attempts SET DEFAULT 0;
ALTER TABLE employees ALTER COLUMN failed_login_attempts SET NOT NULL;

-- Supports the "is this account currently locked" probe on the login path.
CREATE INDEX IF NOT EXISTS idx_employees_locked_until
    ON employees (locked_until)
    WHERE locked_until IS NOT NULL;
