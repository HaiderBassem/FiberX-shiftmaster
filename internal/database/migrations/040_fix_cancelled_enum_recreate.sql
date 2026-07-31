-- 040_fix_cancelled_enum_recreate.sql
-- Add 'cancelled' to the leave_status enum.
--
-- The original version of this file never ran on a fresh database: it dropped and
-- recreated the type against a table called `leave_requests`, which does not exist in
-- this schema -- the table is `leaves` (003_tables.sql). Every fresh install failed here
-- with: relation "leave_requests" does not exist.
--
-- The rename-and-recreate dance existed to dodge the old restriction that
-- ALTER TYPE ... ADD VALUE could not run inside a transaction block. That restriction
-- was lifted in PostgreSQL 12, and this project requires 14+, so the value can simply be
-- added. This is also strictly safer: nothing is dropped, so column defaults, views and
-- any other dependency on the type stay intact.
--
-- Must stay a bare top-level statement: ALTER TYPE ... ADD VALUE cannot be wrapped in a
-- DO block or an explicit BEGIN/COMMIT.
--
-- Idempotent via IF NOT EXISTS.

ALTER TYPE leave_status ADD VALUE IF NOT EXISTS 'cancelled';
