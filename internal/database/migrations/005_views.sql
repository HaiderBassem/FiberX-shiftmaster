-- =====================================================
-- VIEWS (Data Views)
-- =====================================================

-- Weekly schedule view with employee and shift names
CREATE VIEW view_weekly_schedule AS
SELECT 
    ws.week_start_date,
    ws.week_end_date,
    e.id AS employee_id,
    e.first_name || ' ' || e.last_name AS employee_name,
    e.gender,
    es.shift_date,
    s.name AS shift_name,
    s.shift_code,
    es.shift_status,
    es.is_replacement,
    es.check_in_time,
    es.check_out_time
FROM weekly_schedule ws
JOIN employee_shifts es ON ws.id = es.schedule_id
JOIN employees e ON es.employee_id = e.id
LEFT JOIN shifts s ON es.shift_id = s.id
ORDER BY es.shift_date, e.first_name;

-- View of employees eligible to cover night shifts
CREATE VIEW view_eligible_night_replacements AS
SELECT 
    e.id,
    e.first_name,
    e.last_name,
    e.gender,
    es.shift_date AS missing_date
FROM employees e
JOIN employee_shifts es ON e.id = es.employee_id
WHERE e.gender = 'male'
  AND e.can_cover_night_shift = true
  AND es.shift_id IN (SELECT id FROM shifts WHERE shift_code IN ('M','E'))
  AND es.shift_status = 'working'
  AND EXISTS (
      SELECT 1 FROM employee_shifts es_prev
      WHERE es_prev.employee_id = e.id
        AND es_prev.shift_date = es.shift_date - 1
        AND es_prev.shift_status IN ('off', 'leave')
  );

-- =====================================================
-- Task Views (3 table types + employee view)
-- =====================================================

-- Daily Tasks view filtered by shift
CREATE VIEW view_daily_tasks_by_shift AS
SELECT 
    s.id AS shift_id,
    s.name AS shift_name,
    s.shift_code,
    ts.id AS schedule_id,
    ts.title AS task_title,
    ts.description AS task_description,
    ts.recurrence,
    ts.recurrence_days,
    ta.id AS assignment_id,
    ta.assigned_date,
    ta.employee_id,
    e.first_name || ' ' || e.last_name AS employee_name,
    e.employee_code,
    te.status AS execution_status,
    te.completed_at,
    te.notes AS execution_notes
FROM task_schedules ts
JOIN shifts s ON ts.shift_id = s.id
LEFT JOIN task_assignments ta ON ts.id = ta.schedule_id
LEFT JOIN employees e ON ta.employee_id = e.id
LEFT JOIN task_executions te ON ta.id = te.assignment_id
WHERE ts.schedule_type = 'daily_task'
  AND ts.is_active = true;

-- Node Check view filtered by shift
CREATE VIEW view_node_check_by_shift AS
SELECT 
    s.id AS shift_id,
    s.name AS shift_name,
    s.shift_code,
    ts.id AS schedule_id,
    ts.title AS task_title,
    ts.description AS task_description,
    ts.recurrence,
    ts.recurrence_days,
    ts.max_assignees,
    ta.id AS assignment_id,
    ta.assigned_date,
    ta.employee_id,
    e.first_name || ' ' || e.last_name AS employee_name,
    e.employee_code,
    te.status AS execution_status,
    te.completed_at,
    te.notes AS execution_notes
FROM task_schedules ts
JOIN shifts s ON ts.shift_id = s.id
LEFT JOIN task_assignments ta ON ts.id = ta.schedule_id
LEFT JOIN employees e ON ta.employee_id = e.id
LEFT JOIN task_executions te ON ta.id = te.assignment_id
WHERE ts.schedule_type = 'node_check'
  AND ts.is_active = true;

-- Mobile Ticket view filtered by shift
CREATE VIEW view_mobile_ticket_by_shift AS
SELECT 
    s.id AS shift_id,
    s.name AS shift_name,
    s.shift_code,
    ts.id AS schedule_id,
    ts.title AS task_title,
    ts.description AS task_description,
    ts.recurrence,
    ts.recurrence_days,
    ts.max_assignees,
    ta.id AS assignment_id,
    ta.assigned_date,
    ta.employee_id,
    e.first_name || ' ' || e.last_name AS employee_name,
    e.employee_code,
    te.status AS execution_status,
    te.completed_at,
    te.notes AS execution_notes
FROM task_schedules ts
JOIN shifts s ON ts.shift_id = s.id
LEFT JOIN task_assignments ta ON ts.id = ta.schedule_id
LEFT JOIN employees e ON ta.employee_id = e.id
LEFT JOIN task_executions te ON ta.id = te.assignment_id
WHERE ts.schedule_type = 'mobile_ticket'
  AND ts.is_active = true;

-- Employee today's tasks view (all types combined)
CREATE VIEW view_employee_today_tasks AS
SELECT 
    ta.employee_id,
    e.first_name || ' ' || e.last_name AS employee_name,
    ts.schedule_type,
    ts.title AS task_title,
    ts.description AS task_description,
    s.name AS shift_name,
    ta.assigned_date,
    te.status AS execution_status,
    te.completed_at,
    te.notes AS execution_notes
FROM task_assignments ta
JOIN task_schedules ts ON ta.schedule_id = ts.id
JOIN employees e ON ta.employee_id = e.id
LEFT JOIN shifts s ON ts.shift_id = s.id
LEFT JOIN task_executions te ON ta.id = te.assignment_id
WHERE ta.assigned_date = CURRENT_DATE
  AND ts.is_active = true;
