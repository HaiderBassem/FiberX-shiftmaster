-- Migration 011: Add secondary contact fields + leave_approvals table
-- Run this on the production database

-- 1. Secondary contact fields
ALTER TABLE employees ADD COLUMN IF NOT EXISTS secondary_phone VARCHAR(20);
ALTER TABLE employees ADD COLUMN IF NOT EXISTS secondary_email VARCHAR(255);

-- 2. Leave approvals tracking table (multi-TL approval workflow)
CREATE TABLE IF NOT EXISTS leave_approvals (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    leave_id UUID NOT NULL REFERENCES leaves(id) ON DELETE CASCADE,
    approver_id UUID NOT NULL REFERENCES employees(id),
    approver_role VARCHAR(20) NOT NULL DEFAULT 'team_leader',
    status VARCHAR(20) NOT NULL DEFAULT 'pending', -- pending, approved, rejected
    comment TEXT,
    decided_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_leave_approvals_leave_id ON leave_approvals(leave_id);
CREATE INDEX IF NOT EXISTS idx_leave_approvals_approver ON leave_approvals(approver_id);
