-- 040_fix_cancelled_enum_recreate.sql
-- Safely recreate the leave_status enum to include 'cancelled'
-- This avoids transaction block restrictions of ALTER TYPE ADD VALUE

DO $$ 
BEGIN
    -- Only proceed if 'cancelled' is not already in the enum
    IF NOT EXISTS (
        SELECT 1 FROM pg_enum e 
        JOIN pg_type t ON e.enumtypid = t.oid 
        WHERE t.typname = 'leave_status' AND e.enumlabel = 'cancelled'
    ) THEN
        
        -- 1. Drop default from leave_requests
        ALTER TABLE leave_requests ALTER COLUMN status DROP DEFAULT;
        
        -- 2. Rename old type
        ALTER TYPE leave_status RENAME TO leave_status_old;
        
        -- 3. Create new type with the new value
        CREATE TYPE leave_status AS ENUM ('pending', 'approved_by_team_leader', 'approved_by_manager', 'rejected', 'cancelled');
        
        -- 4. Alter table to use new type
        ALTER TABLE leave_requests ALTER COLUMN status TYPE leave_status USING status::text::leave_status;
        
        -- 5. Restore default
        ALTER TABLE leave_requests ALTER COLUMN status SET DEFAULT 'pending';
        
        -- 6. Drop old type
        DROP TYPE leave_status_old;
        
    END IF;
END $$;
