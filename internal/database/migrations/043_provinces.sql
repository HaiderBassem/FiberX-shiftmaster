-- Create provinces table
CREATE TABLE IF NOT EXISTS provinces (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(255) NOT NULL UNIQUE,
    sort_order INTEGER NOT NULL DEFAULT 0,
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Initial seed data
INSERT INTO provinces (name, sort_order) VALUES
    ('بغداد', 1),
    ('البصرة', 2),
    ('نينوى', 3),
    ('أربيل', 4),
    ('النجف', 5),
    ('كربلاء', 6),
    ('الأنبار', 7),
    ('ديالى', 8),
    ('كركوك', 9),
    ('بابل', 10),
    ('واسط', 11),
    ('ذي قار', 12),
    ('ميسان', 13),
    ('المثنى', 14),
    ('القادسية', 15),
    ('صلاح الدين', 16),
    ('دهوك', 17),
    ('السليمانية', 18)
ON CONFLICT (name) DO NOTHING;
