-- transfer-ownership.sql
--
-- Moves ownership of every application object in schema "public" — tables,
-- sequences, views, functions and enum types — to the role the application
-- connects as, and makes sure that role can create new objects there.
--
-- WHEN TO USE THIS
--   The migrator stopped with "must be owner of …" (SQLSTATE 42501): the
--   database was originally created by a different role (often postgres) than
--   the DB_USER in /opt/shiftmaster/.env, so migrations that alter existing
--   objects cannot run. Transferring ownership once fixes every future deploy.
--   Existing data is not touched — only the owner attribute of each object.
--
-- USAGE (as a PostgreSQL superuser, from the repository root):
--
--   sudo -u postgres psql -d <DB_NAME> -v new_owner=<DB_USER> -f deploy/transfer-ownership.sql
--
-- The script is idempotent: objects already owned by new_owner are simply
-- re-asserted.

\set ON_ERROR_STOP on

\echo Transferring ownership of schema "public" objects to :'new_owner'

-- Tables (ALTER TABLE ... OWNER also carries any sequences owned by their
-- columns, e.g. serial/identity columns).
SELECT format('ALTER TABLE %I.%I OWNER TO %I', schemaname, tablename, :'new_owner')
FROM pg_tables WHERE schemaname = 'public'
\gexec

-- Free-standing sequences.
SELECT format('ALTER SEQUENCE %I.%I OWNER TO %I', schemaname, sequencename, :'new_owner')
FROM pg_sequences WHERE schemaname = 'public'
\gexec

-- Views.
SELECT format('ALTER VIEW %I.%I OWNER TO %I', schemaname, viewname, :'new_owner')
FROM pg_views WHERE schemaname = 'public'
\gexec

-- Functions and procedures (regprocedure carries the argument signature).
SELECT format('ALTER FUNCTION %s OWNER TO %I', p.oid::regprocedure, :'new_owner')
FROM pg_proc p
JOIN pg_namespace n ON n.oid = p.pronamespace
WHERE n.nspname = 'public'
\gexec

-- Enum types (ALTER TYPE ... ADD VALUE in future migrations requires
-- ownership). Table row types are excluded — they follow their table.
SELECT format('ALTER TYPE %I.%I OWNER TO %I', n.nspname, t.typname, :'new_owner')
FROM pg_type t
JOIN pg_namespace n ON n.oid = t.typnamespace
WHERE n.nspname = 'public' AND t.typtype = 'e'
\gexec

-- New objects (the migration ledger, future tables) need CREATE on the
-- schema; PostgreSQL 15+ no longer grants it to everyone by default.
SELECT format('GRANT USAGE, CREATE ON SCHEMA public TO %I', :'new_owner')
\gexec

\echo Done. Re-run the deploy (or: ./shiftmaster-migrate -dir internal/database/migrations adopt)
