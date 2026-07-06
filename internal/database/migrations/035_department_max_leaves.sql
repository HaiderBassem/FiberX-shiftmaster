-- Migration 035: Department Max Leaves Per Day (idempotent)
ALTER TABLE departments ADD COLUMN IF NOT EXISTS max_leaves_per_day INTEGER DEFAULT NULL;
