-- Add can_post_announcements to employees
ALTER TABLE employees ADD COLUMN IF NOT EXISTS can_post_announcements BOOLEAN DEFAULT false;

-- Create announcements table
CREATE TABLE IF NOT EXISTS announcements (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    department_id UUID NOT NULL REFERENCES departments(id) ON DELETE CASCADE,
    title VARCHAR(200) NOT NULL,
    message TEXT NOT NULL,
    priority VARCHAR(20) DEFAULT 'normal' CHECK (priority IN ('info', 'normal', 'important', 'critical')),
    is_active BOOLEAN DEFAULT false,
    created_by UUID REFERENCES employees(id),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

COMMENT ON TABLE announcements IS 'Department announcements';

-- Indexes
CREATE INDEX IF NOT EXISTS idx_announcements_department ON announcements(department_id);
CREATE INDEX IF NOT EXISTS idx_announcements_active ON announcements(is_active);
