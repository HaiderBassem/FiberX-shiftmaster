-- =====================================================
-- Custom Data Types (ENUMs)
-- =====================================================

CREATE TYPE employee_role AS ENUM ('employee', 'team_leader', 'manager', 'hr');
CREATE TYPE gender_type AS ENUM ('male', 'female');
CREATE TYPE employee_status AS ENUM ('active', 'inactive', 'on_leave', 'terminated');
CREATE TYPE shift_status_type AS ENUM ('working', 'off', 'leave', 'sick', 'vacation', 'training', 'business_trip');
CREATE TYPE leave_type AS ENUM ('annual', 'sick', 'emergency', 'marriage', 'maternity', 'unpaid', 'other');
CREATE TYPE leave_status AS ENUM ('pending', 'approved_by_team_leader', 'approved_by_manager', 'rejected');
CREATE TYPE task_priority AS ENUM ('low', 'medium', 'high', 'urgent');
CREATE TYPE task_status AS ENUM ('pending', 'in_progress', 'completed', 'cancelled', 'overdue');
CREATE TYPE notification_type AS ENUM ('shift_change', 'task_assigned', 'leave_request', 'approval', 'system_alert', 'reminder');
CREATE TYPE notification_priority AS ENUM ('low', 'medium', 'high');
CREATE TYPE swap_status AS ENUM ('pending', 'employee_accepted', 'approved', 'rejected', 'cancelled');
