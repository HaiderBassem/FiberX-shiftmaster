-- Migration 045: Province-Centric Service Architecture
-- =========================================================================

-- 1. Clear existing data to avoid foreign key or NOT NULL constraint violations
TRUNCATE TABLE service_plans CASCADE;
TRUNCATE TABLE service_category_department_shares CASCADE;
TRUNCATE TABLE service_categories CASCADE;
TRUNCATE TABLE provinces CASCADE;

-- 2. Modify provinces to belong to departments
ALTER TABLE provinces ADD COLUMN department_id UUID NOT NULL REFERENCES departments(id) ON DELETE CASCADE;
ALTER TABLE provinces ADD COLUMN created_by UUID REFERENCES employees(id) ON DELETE SET NULL;

-- Remove global unique constraint on province name (so departments can have overlapping names if needed)
ALTER TABLE provinces DROP CONSTRAINT IF EXISTS provinces_name_key;
CREATE UNIQUE INDEX idx_provinces_dept_name ON provinces(department_id, name);

-- 3. Create province sharing table
CREATE TABLE IF NOT EXISTS province_department_shares (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    province_id UUID NOT NULL REFERENCES provinces(id) ON DELETE CASCADE,
    department_id UUID NOT NULL REFERENCES departments(id) ON DELETE CASCADE,
    granted_by UUID REFERENCES employees(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(province_id, department_id)
);

CREATE INDEX IF NOT EXISTS idx_province_shares_dept ON province_department_shares(department_id);

-- 4. Move service_categories to belong to province_id instead of department_id
DROP TABLE IF EXISTS service_category_department_shares;
ALTER TABLE service_categories DROP COLUMN IF EXISTS department_id;
ALTER TABLE service_categories ADD COLUMN province_id UUID NOT NULL REFERENCES provinces(id) ON DELETE CASCADE;

-- 5. Remove province text column from service_plans (it inherits from category -> province)
ALTER TABLE service_plans DROP COLUMN IF EXISTS province;
