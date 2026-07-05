-- Add 'cancelled' to leave_status
ALTER TYPE leave_status ADD VALUE IF NOT EXISTS 'cancelled';
