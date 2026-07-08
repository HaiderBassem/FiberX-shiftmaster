-- Migration 046: Simplify Service Plans and Add Disabled At tracking
-- ==================================================================

-- 1. Simplify Service Plans
ALTER TABLE service_plans RENAME COLUMN speed_download TO speed;
ALTER TABLE service_plans DROP COLUMN IF EXISTS speed_upload;
ALTER TABLE service_plans DROP COLUMN IF EXISTS ip_type;

-- 2. Add DisabledAt timestamps
ALTER TABLE service_categories ADD COLUMN IF NOT EXISTS disabled_at TIMESTAMPTZ;
ALTER TABLE service_plans ADD COLUMN IF NOT EXISTS disabled_at TIMESTAMPTZ;

-- Note: We do not retroactively populate disabled_at for already-disabled items
-- because we don't have historical log. They will remain null until toggled.
