-- =====================================================
-- FUNCTIONS
-- =====================================================

-- Automatically update the updated_at field
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Function to generate a weekly schedule from a template
CREATE OR REPLACE FUNCTION generate_weekly_schedule(p_week_start DATE, p_created_by UUID)
RETURNS UUID AS $$
DECLARE
    v_schedule_id UUID;
    v_template_id UUID;
    v_week_end DATE := p_week_start + 6;
BEGIN
    -- Use the latest valid template (logic can be improved)
    SELECT id INTO v_template_id FROM schedule_templates 
    WHERE (valid_from IS NULL OR valid_from <= p_week_start) 
      AND (valid_to IS NULL OR valid_to >= v_week_end)
    LIMIT 1;
    
    -- Create the week record
    INSERT INTO weekly_schedule (week_start_date, week_end_date, template_id, status, created_by)
    VALUES (p_week_start, v_week_end, v_template_id, 'draft', p_created_by)
    RETURNING id INTO v_schedule_id;
    
    -- Copy the template to employee_shifts
    INSERT INTO employee_shifts (schedule_id, employee_id, shift_id, shift_date, shift_status, created_by)
    SELECT 
        v_schedule_id,
        st.employee_id,
        st.shift_id,
        p_week_start + st.day_of_week,
        CASE WHEN st.is_off THEN 'off'::shift_status_type ELSE 'working'::shift_status_type END,
        p_created_by
    FROM schedule_templates st
    WHERE (st.valid_from IS NULL OR st.valid_from <= p_week_start)
      AND (st.valid_to IS NULL OR st.valid_to >= v_week_end)
      AND EXISTS (SELECT 1 FROM employees e WHERE e.id = st.employee_id AND e.status = 'active');
    
    RETURN v_schedule_id;
END;
$$ LANGUAGE plpgsql;

-- Function to find a replacement for a night shift
CREATE OR REPLACE FUNCTION find_replacement_for_night(p_date DATE, p_exclude_employee_id UUID DEFAULT NULL)
RETURNS TABLE (
    employee_id UUID,
    full_name TEXT,
    phone VARCHAR(20)
) AS $$
BEGIN
    RETURN QUERY
    SELECT 
        e.id,
        e.first_name || ' ' || e.last_name,
        e.phone
    FROM employees e
    JOIN employee_shifts es ON e.id = es.employee_id
    WHERE e.gender = 'male'
      AND e.can_cover_night_shift = true
      AND es.shift_date = p_date
      AND es.shift_id IN (SELECT id FROM shifts WHERE shift_code IN ('M','E'))
      AND es.shift_status = 'working'
      AND (p_exclude_employee_id IS NULL OR e.id != p_exclude_employee_id)
      AND EXISTS (
          SELECT 1 FROM employee_shifts es_prev
          WHERE es_prev.employee_id = e.id
            AND es_prev.shift_date = p_date - 1
            AND es_prev.shift_status IN ('off', 'leave')
      )
    LIMIT 5;
END;
$$ LANGUAGE plpgsql;

-- =====================================================
-- Task Functions
-- =====================================================

-- Generate task assignments for a given date based on active schedules
CREATE OR REPLACE FUNCTION generate_daily_assignments(p_date DATE)
RETURNS void AS $$
DECLARE
    v_dow INTEGER := EXTRACT(DOW FROM p_date)::INTEGER;
    v_schedule RECORD;
BEGIN
    FOR v_schedule IN
        SELECT * FROM task_schedules
        WHERE is_active = true
    LOOP
        -- Check if this task should run on this date
        IF v_schedule.recurrence = 'daily' THEN
            -- Daily tasks: run every day (if recurrence_days is NULL) or on specified days
            IF v_schedule.recurrence_days IS NULL 
               OR v_dow = ANY(v_schedule.recurrence_days) THEN
                -- Task should run today, but we don't auto-assign employees
                -- The manager will assign employees via the UI
                -- We just create a placeholder assignment with no employee
                NULL; -- Manager assigns manually
            END IF;
        ELSIF v_schedule.recurrence = 'periodic' THEN
            -- Periodic tasks: only run on specified days
            IF v_schedule.recurrence_days IS NOT NULL 
               AND v_dow = ANY(v_schedule.recurrence_days) THEN
                NULL; -- Manager assigns manually
            END IF;
        END IF;
    END LOOP;
END;
$$ LANGUAGE plpgsql;

-- Get all tasks for an employee on a specific date
CREATE OR REPLACE FUNCTION get_employee_today_tasks(p_employee_id UUID, p_date DATE DEFAULT CURRENT_DATE)
RETURNS TABLE (
    assignment_id UUID,
    schedule_type VARCHAR,
    task_title VARCHAR,
    task_description TEXT,
    shift_name VARCHAR,
    execution_status task_status,
    completed_at TIMESTAMP,
    execution_notes TEXT
) AS $$
BEGIN
    RETURN QUERY
    SELECT 
        ta.id,
        ts.schedule_type::VARCHAR,
        ts.title,
        ts.description,
        s.name,
        te.status,
        te.completed_at,
        te.notes
    FROM task_assignments ta
    JOIN task_schedules ts ON ta.schedule_id = ts.id
    LEFT JOIN shifts s ON ts.shift_id = s.id
    LEFT JOIN task_executions te ON ta.id = te.assignment_id
    WHERE ta.employee_id = p_employee_id
      AND ta.assigned_date = p_date
      AND ts.is_active = true
    ORDER BY ts.schedule_type, ts.title;
END;
$$ LANGUAGE plpgsql;

-- Swap task assignments between two employees on the same date
CREATE OR REPLACE FUNCTION swap_task_assignment(p_assignment_id_1 UUID, p_assignment_id_2 UUID)
RETURNS BOOLEAN AS $$
DECLARE
    v_emp_1 UUID;
    v_emp_2 UUID;
    v_date_1 DATE;
    v_date_2 DATE;
BEGIN
    -- Get employee IDs and dates
    SELECT employee_id, assigned_date INTO v_emp_1, v_date_1
    FROM task_assignments WHERE id = p_assignment_id_1;
    
    SELECT employee_id, assigned_date INTO v_emp_2, v_date_2
    FROM task_assignments WHERE id = p_assignment_id_2;
    
    -- Validate both assignments exist and are on the same date
    IF v_emp_1 IS NULL OR v_emp_2 IS NULL THEN
        RAISE EXCEPTION 'One or both assignment IDs not found';
    END IF;
    
    IF v_date_1 != v_date_2 THEN
        RAISE EXCEPTION 'Cannot swap assignments on different dates';
    END IF;
    
    -- Temporarily remove unique constraint by using a temp value
    -- Update first assignment to a temporary employee
    UPDATE task_assignments SET employee_id = v_emp_2 WHERE id = p_assignment_id_1;
    UPDATE task_assignments SET employee_id = v_emp_1 WHERE id = p_assignment_id_2;
    
    RETURN true;
END;
$$ LANGUAGE plpgsql;

-- Check if a task schedule should appear on a given date
CREATE OR REPLACE FUNCTION should_task_run_on_date(p_schedule_id UUID, p_date DATE)
RETURNS BOOLEAN AS $$
DECLARE
    v_schedule RECORD;
    v_dow INTEGER := EXTRACT(DOW FROM p_date)::INTEGER;
BEGIN
    SELECT * INTO v_schedule FROM task_schedules WHERE id = p_schedule_id;
    
    IF v_schedule IS NULL OR NOT v_schedule.is_active THEN
        RETURN false;
    END IF;
    
    IF v_schedule.recurrence = 'daily' THEN
        IF v_schedule.recurrence_days IS NULL THEN
            RETURN true; -- every day
        ELSE
            RETURN v_dow = ANY(v_schedule.recurrence_days);
        END IF;
    ELSIF v_schedule.recurrence = 'periodic' THEN
        IF v_schedule.recurrence_days IS NOT NULL THEN
            RETURN v_dow = ANY(v_schedule.recurrence_days);
        END IF;
    END IF;
    
    RETURN false;
END;
$$ LANGUAGE plpgsql;
