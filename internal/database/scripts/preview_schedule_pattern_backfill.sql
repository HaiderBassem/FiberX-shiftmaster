-- =====================================================
-- PREVIEW ONLY — reads nothing but SELECT. Changes nothing.
-- =====================================================
-- Shows exactly what backfill_schedule_pattern.sql would write, so you can check it
-- before writing anything. Nothing here is invented: every row is either read back
-- from the employee's own worked history, or from the default_shift_id /
-- weekly_off_days already on their employee record.
--
--   psql -d shiftmaster -f internal/database/scripts/preview_schedule_pattern_backfill.sql
--
-- Read the `derived_from` column:
--   'worked history'  -> taken from a day this employee actually worked on that weekday
--   'employee default'-> no history on that weekday; falls back to their record
--
-- Employees who already have a pattern entry for a weekday are skipped entirely and
-- do not appear below.
-- =====================================================

SELECT e.employee_code,
       e.first_name || ' ' || e.last_name              AS employee,
       d.dow                                            AS day_of_week,
       to_char(DATE '2024-01-07' + d.dow, 'Day')        AS weekday,
       CASE
           WHEN r.shift_status = 'working' THEN 'working'
           WHEN r.shift_status = 'off'     THEN 'off'
           WHEN e.weekly_off_days = d.dow  THEN 'off'
           WHEN e.default_shift_id IS NULL THEN 'off'
           ELSE 'working'
       END                                              AS will_be,
       COALESCE(
           (SELECT s.shift_code FROM shifts s WHERE s.id =
               CASE
                   WHEN r.shift_status = 'working' THEN COALESCE(r.shift_id, e.default_shift_id)
                   WHEN r.shift_status = 'off'     THEN NULL
                   WHEN e.weekly_off_days = d.dow  THEN NULL
                   ELSE e.default_shift_id
               END),
           '—')                                         AS shift,
       CASE WHEN r.shift_status IS NOT NULL
            THEN 'worked history'
            ELSE 'employee default'
       END                                              AS derived_from
  FROM employees e
 CROSS JOIN generate_series(0, 6) AS d(dow)
  LEFT JOIN LATERAL (
       SELECT es.shift_status::text AS shift_status, es.shift_id
         FROM employee_shifts es
        WHERE es.employee_id = e.id
          AND EXTRACT(DOW FROM es.shift_date)::int = d.dow
          AND es.shift_status IN ('working', 'off')
          AND es.shift_date <= CURRENT_DATE
          AND es.shift_date >= CURRENT_DATE - 56
        ORDER BY es.shift_date DESC
        LIMIT 1
  ) r ON true
 WHERE e.status = 'active'
   AND NOT EXISTS (
       SELECT 1 FROM schedule_templates st
        WHERE st.employee_id = e.id
          AND st.day_of_week = d.dow
          AND st.valid_to IS NULL
   )
 ORDER BY e.employee_code, d.dow;
