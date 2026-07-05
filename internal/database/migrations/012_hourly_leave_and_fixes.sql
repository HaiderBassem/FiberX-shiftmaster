-- Migration 012: Add hourly leave support
-- Run this on the production database

-- 1. Add hourly leave type to leave_type enum
ALTER TYPE leave_type ADD VALUE IF NOT EXISTS 'hourly';

-- 2. Add start_time and end_time columns for hourly leaves
ALTER TABLE leaves ADD COLUMN IF NOT EXISTS start_time TIME;
ALTER TABLE leaves ADD COLUMN IF NOT EXISTS end_time TIME;
