-- Migration 044: Add department_id to service_categories and create shares table
-- =========================================================================

-- 1. Since we are adding a NOT NULL constraint and the user instructed to ignore existing data,
-- we truncate the services tables to avoid constraint violations.
TRUNCATE TABLE service_plans CASCADE;
TRUNCATE TABLE service_categories CASCADE;

-- 2. Add department_id to service_categories
ALTER TABLE service_categories 
ADD COLUMN department_id UUID NOT NULL REFERENCES departments(id) ON DELETE CASCADE;

CREATE INDEX IF NOT EXISTS idx_service_categories_dept ON service_categories(department_id);

-- 3. Create shares table for cross-department sharing
CREATE TABLE IF NOT EXISTS service_category_department_shares (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    category_id UUID NOT NULL REFERENCES service_categories(id) ON DELETE CASCADE,
    department_id UUID NOT NULL REFERENCES departments(id) ON DELETE CASCADE,
    granted_by UUID REFERENCES employees(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(category_id, department_id)
);

CREATE INDEX IF NOT EXISTS idx_service_category_shares_dept ON service_category_department_shares(department_id);
