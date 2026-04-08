-- =====================================================
-- Custom Data Types (ENUMs)
-- =====================================================

DO $$ BEGIN
    CREATE TYPE employee_role AS ENUM ('employee', 'team_leader', 'manager', 'admin', 'hr');
EXCEPTION WHEN duplicate_object THEN NULL; END $$;

DO $$ BEGIN
    CREATE TYPE gender_type AS ENUM ('male', 'female');
EXCEPTION WHEN duplicate_object THEN NULL; END $$;

DO $$ BEGIN
    CREATE TYPE employee_status AS ENUM ('active', 'inactive', 'on_leave', 'terminated');
EXCEPTION WHEN duplicate_object THEN NULL; END $$;

DO $$ BEGIN
    CREATE TYPE shift_status_type AS ENUM ('working', 'off', 'leave', 'sick', 'vacation', 'training', 'business_trip');
EXCEPTION WHEN duplicate_object THEN NULL; END $$;

DO $$ BEGIN
    CREATE TYPE leave_type AS ENUM ('annual', 'sick', 'emergency', 'marriage', 'maternity', 'unpaid', 'other');
EXCEPTION WHEN duplicate_object THEN NULL; END $$;

DO $$ BEGIN
    CREATE TYPE leave_status AS ENUM ('pending', 'approved_by_team_leader', 'approved_by_manager', 'rejected');
EXCEPTION WHEN duplicate_object THEN NULL; END $$;

DO $$ BEGIN
    CREATE TYPE task_priority AS ENUM ('low', 'medium', 'high', 'urgent');
EXCEPTION WHEN duplicate_object THEN NULL; END $$;

DO $$ BEGIN
    CREATE TYPE task_status AS ENUM ('pending', 'in_progress', 'completed', 'cancelled', 'overdue');
EXCEPTION WHEN duplicate_object THEN NULL; END $$;

DO $$ BEGIN
    CREATE TYPE notification_type AS ENUM ('shift_change', 'task_assigned', 'leave_request', 'approval', 'system_alert', 'reminder');
EXCEPTION WHEN duplicate_object THEN NULL; END $$;

DO $$ BEGIN
    CREATE TYPE notification_priority AS ENUM ('low', 'medium', 'high');
EXCEPTION WHEN duplicate_object THEN NULL; END $$;

DO $$ BEGIN
    CREATE TYPE swap_status AS ENUM ('pending', 'employee_accepted', 'approved', 'rejected', 'cancelled');
EXCEPTION WHEN duplicate_object THEN NULL; END $$;
