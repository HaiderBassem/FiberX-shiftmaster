-- =====================================================
-- 047: Fixed weekly pattern as the source of truth
-- =====================================================
-- Problem this fixes:
--   employee_shifts rows were mass-seeded on every read by copying the previous
--   week. Any read of a future date froze that week forever (rows exist -> skip),
--   so schedules "reset" to weekly_off_days / default_shift_id every week.
--
-- New model:
--   schedule_templates  = the employee's fixed weekly pattern (one entry per weekday).
--                         This is the permanent thing.
--   employee_shifts     = materialised days, tagged with `source`:
--                           'generated' -> derived from the pattern, safe to re-derive
--                           'manual'    -> a human decision (one-off edit, swap, replacement)
--                           'leave'     -> owned by the approved-leave overlay
--                         Only 'generated' rows are ever rewritten, and only for
--                         today onward. History is immutable.
--
-- THIS MIGRATION IS PURELY ADDITIVE.
--   * No table, column or row is dropped or deleted.
--   * No constraint or index is dropped.
--   * The only writes are to the new `source` column on existing rows.
--   Duplicate schedule_templates rows and duplicate weekly_schedule headers are left
--   exactly where they are; the application now resolves them deterministically
--   (newest still-valid entry wins) instead of picking one at random.
--
-- Idempotent: safe to re-run.
-- =====================================================

BEGIN;

-- -----------------------------------------------------
-- 1. Provenance column on employee_shifts  (ADD only)
-- -----------------------------------------------------
ALTER TABLE employee_shifts
    ADD COLUMN IF NOT EXISTS source VARCHAR(16) NOT NULL DEFAULT 'generated';

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
         WHERE conrelid = 'employee_shifts'::regclass
           AND conname = 'employee_shifts_source_check'
    ) THEN
        ALTER TABLE employee_shifts
            ADD CONSTRAINT employee_shifts_source_check
            CHECK (source IN ('generated', 'manual', 'leave'));
    END IF;
END $$;

COMMENT ON COLUMN employee_shifts.source IS
    'Who authored this day: generated = derived from the weekly pattern and safe to '
    're-derive; manual = an explicit human decision; leave = owned by an approved leave. '
    'Only generated rows are ever rewritten.';

-- -----------------------------------------------------
-- 2. Classify the rows that already exist  (UPDATE only)
-- -----------------------------------------------------
-- Runs once: the guard stops a re-run from re-classifying rows the application has
-- written since. Nothing is deleted -- every existing row keeps its shift, date and
-- status exactly as it is.
DO $$
DECLARE
    v_week_end DATE := CURRENT_DATE - EXTRACT(DOW FROM CURRENT_DATE)::int + 6;
BEGIN
    IF NOT EXISTS (SELECT 1 FROM employee_shifts WHERE source <> 'generated') THEN
        -- Everything up to the end of the current week is historical fact and is
        -- pinned so the materialiser can never rewrite it.
        UPDATE employee_shifts
           SET source = 'manual'
         WHERE shift_date <= v_week_end;

        -- Leave days belong to the leave overlay, whatever their date.
        --
        -- Compared as text on purpose. 002_types.sql was edited after the first
        -- deployments, so shift_status_type holds different labels depending on when
        -- the database was created: 'hourly' is present on databases built from the
        -- current file, absent on older ones (which store hourly leave as 'leave' with
        -- a "[hourly]" reason tag). An enum literal that the local type lacks is a hard
        -- error at parse time -- `invalid input value for enum shift_status_type` --
        -- which aborts the whole migration. Casting to text makes an absent label
        -- simply match nothing, so this runs on any vintage of the schema.
        UPDATE employee_shifts
           SET source = 'leave'
         WHERE shift_status::text IN ('leave', 'vacation', 'sick', 'hourly');

        -- Replacements and anything already worked are human decisions.
        UPDATE employee_shifts
           SET source = 'manual'
         WHERE is_replacement = true
            OR check_in_time IS NOT NULL
            OR check_out_time IS NOT NULL;

        -- Future rows keep source='generated': these are the mass-seeded rows that
        -- caused the reset, and they are now re-derived from the pattern instead of
        -- staying frozen. Their current values are overwritten only when they
        -- disagree with the employee's pattern -- which is the point of the fix.
    END IF;
END $$;

COMMENT ON TABLE schedule_templates IS
    'The employee''s fixed weekly pattern: one entry per weekday. Source of truth for '
    'recurring schedules. The still-valid entry with the newest valid_from wins.';

-- -----------------------------------------------------
-- 3. Indexes for the per-week materialisation  (ADD only)
-- -----------------------------------------------------
CREATE INDEX IF NOT EXISTS idx_employee_shifts_date_employee
    ON employee_shifts (shift_date, employee_id);

CREATE INDEX IF NOT EXISTS idx_employee_shifts_generated
    ON employee_shifts (employee_id, shift_date)
 WHERE source = 'generated';

COMMIT;
