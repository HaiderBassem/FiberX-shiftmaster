CREATE TABLE IF NOT EXISTS shift_handovers (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    department_id UUID NOT NULL REFERENCES departments(id) ON DELETE CASCADE,
    creator_id UUID NOT NULL REFERENCES employees(id) ON DELETE CASCADE,
    shift_summary TEXT NOT NULL,
    pending_issues TEXT NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'open', -- 'open', 'claimed', 'completed'
    claimed_by UUID REFERENCES employees(id) ON DELETE SET NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_shift_handovers_dept ON shift_handovers(department_id);
CREATE INDEX IF NOT EXISTS idx_shift_handovers_status ON shift_handovers(status);
