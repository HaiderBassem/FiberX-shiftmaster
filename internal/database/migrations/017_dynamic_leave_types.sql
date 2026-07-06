-- =====================================================
-- Migration: Dynamic Leave Types (idempotent)
-- =====================================================

CREATE TABLE IF NOT EXISTS leave_types (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name_ar VARCHAR(100) NOT NULL,
    name_en VARCHAR(100) NOT NULL,
    is_paid BOOLEAN DEFAULT false,
    color_code VARCHAR(7) DEFAULT '#3788d8',
    is_active BOOLEAN DEFAULT true,
    requires_approval BOOLEAN DEFAULT true,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

COMMENT ON TABLE leave_types IS 'Dynamic types of leaves available to employees';

-- Add the new foreign key to leaves table (idempotent)
ALTER TABLE leaves ADD COLUMN IF NOT EXISTS leave_type_id UUID REFERENCES leave_types(id);

-- Drop the old enum column (if it exists)
ALTER TABLE leaves DROP COLUMN IF EXISTS leave_type;

-- Drop the enum type (if not used elsewhere)
DROP TYPE IF EXISTS leave_type;
