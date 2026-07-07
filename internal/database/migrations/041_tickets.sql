-- Migration 041: Cross-department tickets and comments
-- =====================================================

CREATE TABLE IF NOT EXISTS tickets (
    id                   UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    source_department_id UUID NOT NULL REFERENCES departments(id) ON DELETE CASCADE,
    target_department_id UUID NOT NULL REFERENCES departments(id) ON DELETE CASCADE,
    creator_id           UUID NOT NULL REFERENCES employees(id) ON DELETE CASCADE,
    title                VARCHAR(255) NOT NULL,
    description          TEXT NOT NULL,
    status               VARCHAR(50) DEFAULT 'open',
    closed_by            UUID REFERENCES employees(id) ON DELETE SET NULL,
    attachments          JSONB,
    created_at           TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    updated_at           TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS ticket_comments (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    ticket_id   UUID NOT NULL REFERENCES tickets(id) ON DELETE CASCADE,
    employee_id UUID NOT NULL REFERENCES employees(id) ON DELETE CASCADE,
    comment     TEXT NOT NULL,
    attachments JSONB,
    created_at  TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_tickets_source_dept ON tickets(source_department_id);
CREATE INDEX IF NOT EXISTS idx_tickets_target_dept ON tickets(target_department_id);
CREATE INDEX IF NOT EXISTS idx_tickets_status ON tickets(status);
CREATE INDEX IF NOT EXISTS idx_ticket_comments_ticket ON ticket_comments(ticket_id);
