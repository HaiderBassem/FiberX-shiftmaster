CREATE TABLE IF NOT EXISTS handover_comments (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    handover_id UUID NOT NULL REFERENCES shift_handovers(id) ON DELETE CASCADE,
    employee_id UUID NOT NULL REFERENCES employees(id) ON DELETE CASCADE,
    comment TEXT NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

ALTER TABLE shift_handovers ADD COLUMN IF NOT EXISTS done_by UUID REFERENCES employees(id) ON DELETE SET NULL;
