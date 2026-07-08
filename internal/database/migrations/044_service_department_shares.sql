-- Migration 044: Add department_id to service_categories and create shares table
-- =========================================================================

-- 1. No longer truncating existing data to prevent data loss.

-- 2. Add department_id to service_categories
ALTER TABLE service_categories 
ADD COLUMN department_id UUID REFERENCES departments(id) ON DELETE CASCADE;

-- Assign a fallback department to existing categories so we can apply NOT NULL
UPDATE service_categories SET department_id = (SELECT id FROM departments ORDER BY created_at ASC LIMIT 1) WHERE department_id IS NULL;
ALTER TABLE service_categories ALTER COLUMN department_id SET NOT NULL;

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
