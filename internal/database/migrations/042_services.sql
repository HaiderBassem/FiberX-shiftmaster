-- Migration 042: FTTH Service Catalog
-- =====================================================
-- Adds service categories and FTTH internet service plans
-- with cabinet notes, speed tiers, and province mapping.

-- 1. Permission flag: who can manage services
ALTER TABLE employees ADD COLUMN IF NOT EXISTS can_manage_services BOOLEAN DEFAULT false;

-- 2. Service Categories (top-level grouping cards)
CREATE TABLE IF NOT EXISTS service_categories (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name        VARCHAR(255) NOT NULL,
    description TEXT,
    is_active   BOOLEAN DEFAULT true,
    sort_order  INTEGER DEFAULT 0,
    created_by  UUID NOT NULL REFERENCES employees(id) ON DELETE CASCADE,
    created_at  TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    updated_at  TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
);

-- 3. Service Plans (FTTH internet packages inside a category)
CREATE TABLE IF NOT EXISTS service_plans (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    category_id       UUID NOT NULL REFERENCES service_categories(id) ON DELETE CASCADE,
    name              VARCHAR(255) NOT NULL,
    price             DECIMAL(10,2) NOT NULL,
    duration_days     INTEGER NOT NULL,
    speed_download    VARCHAR(50),              -- e.g. "100 Mbps"
    speed_upload      VARCHAR(50),              -- e.g. "50 Mbps"
    data_cap          VARCHAR(50) DEFAULT 'Unlimited',
    province          VARCHAR(100) NOT NULL,    -- Iraqi governorate
    connection_type   VARCHAR(50) DEFAULT 'FTTH',
    installation_fee  DECIMAL(10,2) DEFAULT 0,
    router_included   BOOLEAN DEFAULT false,
    ip_type           VARCHAR(20) DEFAULT 'Dynamic', -- Static / Dynamic
    description       TEXT,
    cabinet_notes     TEXT,                     -- FTTH cabinet coverage notes
    features          JSONB,                    -- flexible features list
    is_active         BOOLEAN DEFAULT true,
    sort_order        INTEGER DEFAULT 0,
    created_by        UUID NOT NULL REFERENCES employees(id) ON DELETE CASCADE,
    created_at        TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    updated_at        TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
);

-- 4. Indexes
CREATE INDEX IF NOT EXISTS idx_service_categories_active   ON service_categories(is_active);
CREATE INDEX IF NOT EXISTS idx_service_plans_category      ON service_plans(category_id);
CREATE INDEX IF NOT EXISTS idx_service_plans_province      ON service_plans(province);
CREATE INDEX IF NOT EXISTS idx_service_plans_active         ON service_plans(is_active);
