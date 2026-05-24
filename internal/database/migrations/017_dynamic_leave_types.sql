-- =====================================================
-- Migration: Dynamic Leave Types
-- =====================================================

CREATE TABLE leave_types (
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

-- Insert default leave types based on the old ENUM values
INSERT INTO leave_types (name_en, name_ar, is_paid, color_code) VALUES
('Annual', 'إجازة سنوية', true, '#28a745'),
('Sick', 'إجازة مرضية', true, '#dc3545'),
('Emergency', 'إجازة طارئة', true, '#fd7e14'),
('Marriage', 'إجازة زواج', true, '#17a2b8'),
('Maternity', 'إجازة أمومة', true, '#e83e8c'),
('Hourly', 'إجازة ساعية', true, '#6f42c1'),
('Unpaid', 'إجازة بدون راتب', false, '#6c757d'),
('Other', 'أخرى', false, '#343a40');

-- Add the new foreign key to leaves table
ALTER TABLE leaves ADD COLUMN leave_type_id UUID REFERENCES leave_types(id);

-- Migrate existing data: match the ENUM string with the new English name
UPDATE leaves 
SET leave_type_id = (
    SELECT id FROM leave_types 
    WHERE LOWER(name_en) = CAST(leaves.leave_type AS VARCHAR)
    LIMIT 1
);

-- For any mismatched records (should not happen), fallback to 'Other'
UPDATE leaves 
SET leave_type_id = (SELECT id FROM leave_types WHERE name_en = 'Other' LIMIT 1) 
WHERE leave_type_id IS NULL;

-- Make the new column required
ALTER TABLE leaves ALTER COLUMN leave_type_id SET NOT NULL;

-- Drop the old enum column
ALTER TABLE leaves DROP COLUMN leave_type;

-- Drop the enum type (if not used elsewhere)
DROP TYPE IF EXISTS leave_type;
