-- =============================================================================
-- 049_external_links_seed.sql
-- =============================================================================
-- Moves the last piece of schema and seed management out of application startup.
--
-- NewModuleAccessRepository used to call an autoMigrate() that ran CREATE TABLE
-- statements, a deduplicating DELETE, and this seed on every boot. Three problems
-- with that:
--
--   1. Schema changed outside the migration series, so the series alone did not
--      describe the database.
--   2. Failures were only logged. A boot against a database where the DDL failed
--      carried on and served traffic against missing tables.
--   3. It ran on every start of every replica, repeating work forever.
--
-- The CREATE TABLE statements were byte-identical to 028, so they are simply
-- dropped. The dedup and the seed are real work and move here.
--
-- 028 carried a note saying the seed had to live in Go because gen_random_uuid()
-- would create duplicates on every deploy run. That was true when the whole
-- series was re-executed on each deploy; migrations are now recorded in
-- schema_migrations and run once. The guard below makes it idempotent anyway.
-- =============================================================================

-- Remove duplicates created by earlier deploys that re-ran the unguarded insert.
-- Keeps the earliest row for each (title, url) pair.
DELETE FROM external_links a
USING external_links b
WHERE a.id > b.id
  AND a.title = b.title
  AND a.url = b.url;

-- Seed the default Live Map link, but only when no link with that title exists.
INSERT INTO external_links (title, url, icon_name)
SELECT 'Live Map', 'https://maps.shift-master.org/', 'map-pin'
WHERE NOT EXISTS (
    SELECT 1 FROM external_links WHERE title = 'Live Map'
);
