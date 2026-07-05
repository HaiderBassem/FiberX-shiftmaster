-- =====================================================
-- Migration 038: Shift-based Leave and Zamania Limits & Reminders
-- =====================================================

-- Add max_hourly_leaves_per_day to departments
ALTER TABLE departments ADD COLUMN IF NOT EXISTS max_hourly_leaves_per_day INTEGER DEFAULT NULL;

-- Add reminder_sent_at to leaves for background notification cron
ALTER TABLE leaves ADD COLUMN IF NOT EXISTS reminder_sent_at TIMESTAMP;
