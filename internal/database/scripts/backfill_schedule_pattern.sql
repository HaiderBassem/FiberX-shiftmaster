-- =====================================================
-- One-time data step: give every active employee a fixed weekly pattern
-- =====================================================
-- NOT a migration, and deliberately not in migrations/: it writes data, not schema,
-- so it is run consciously rather than as part of the normal migration sweep.
--
-- Run the preview first and read the output:
--   psql -d shiftmaster -f internal/database/scripts/preview_schedule_pattern_backfill.sql
-- Then, if it looks right:
--   psql -d shiftmaster -f internal/database/scripts/backfill_schedule_pattern.sql
--
-- What it writes — no invented values anywhere:
--   a) the most recent working/off day the employee actually had on that weekday
--      within the last 8 weeks, i.e. the pattern they are really working today; or
--   b) their own default_shift_id / weekly_off_days when they have no history on
--      that weekday.
--
-- What it never does:
--   * INSERT only. Nothing is deleted, dropped or overwritten.
--   * A weekday that already has a valid pattern entry is skipped untouched.
--   * employee_shifts is not read for anything except deriving (a), and not written.
--
-- Idempotent: re-running adds nothing, because every weekday then has an entry.
--
-- Optional. Without it the system still works and every schedule a supervisor sets
-- from now on sticks permanently — but employees who have never been edited by hand
-- keep falling back to default_shift_id / weekly_off_days, which is the "it resets to
-- the same cliché every week" symptom. This is what clears that for existing staff.
-- =====================================================

BEGIN;

INSERT INTO schedule_templates (employee_id, day_of_week, shift_id, is_off, valid_from)
SELECT e.id,
       d.dow,
       CASE
           WHEN r.shift_status = 'working' THEN COALESCE(r.shift_id, e.default_shift_id)
           WHEN r.shift_status = 'off'     THEN NULL
           WHEN e.weekly_off_days = d.dow  THEN NULL
           ELSE e.default_shift_id
       END,
       CASE
           WHEN r.shift_status = 'working' THEN COALESCE(r.shift_id, e.default_shift_id) IS NULL
           WHEN r.shift_status = 'off'     THEN true
           WHEN e.weekly_off_days = d.dow  THEN true
           ELSE e.default_shift_id IS NULL
       END,
       CURRENT_DATE
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
ON CONFLICT (employee_id, day_of_week, valid_from) DO NOTHING;

COMMIT;
