-- =============================================================================
-- 050_leave_type_semantics.sql
-- =============================================================================
-- Replaces name matching with explicit, admin-controllable schema state.
--
-- Two behaviours were keyed off leave_types.name_en, a field administrators edit
-- freely through the leave-type screen:
--
--   1. Hourly leave. schedule_service compared name_en against 'hourly' and
--      against 'زمنية' — the latter an Arabic string tested against the English
--      column, so it could never match. Meanwhile leave_service decided the same
--      question from unit = 'hours', and the profile endpoint decided it from
--      whether the leave carried start_time and end_time. Three definitions of
--      one concept, disagreeing.
--
--   2. Emergency leave bypassing the department's daily leave cap, matched on
--      the literals 'Emergency' and 'emergency'. Renaming that type to
--      "Emergency Leave", or creating the Arabic equivalent, silently removed
--      the exemption with no error anywhere.
--
-- The columns below name what the flag *does* rather than what a type is called,
-- so the meaning survives renaming and translation.
--
-- BACKFILL: deliberately the union of every rule that was previously in force,
-- so no type changes behaviour on deploy. is_hourly is set for anything the old
-- code would have treated as hourly by either route.
-- =============================================================================

ALTER TABLE leave_types
    ADD COLUMN IF NOT EXISTS is_hourly BOOLEAN NOT NULL DEFAULT false;

ALTER TABLE leave_types
    ADD COLUMN IF NOT EXISTS bypasses_daily_limit BOOLEAN NOT NULL DEFAULT false;

COMMENT ON COLUMN leave_types.is_hourly IS
    'Leave of this type occupies part of a day and is recorded with start_time/end_time. Replaces matching on name_en.';

COMMENT ON COLUMN leave_types.bypasses_daily_limit IS
    'Leave of this type ignores the department max_leaves_per_day / max_hourly_leaves_per_day caps. Replaces matching name_en against "Emergency".';

-- Hourly: the union of the previous rules.
--   - unit = 'hours'                    (what leave_service used)
--   - name_en matched 'hourly'          (what schedule_service used)
--   - name_ar matched the Arabic term   (what schedule_service intended)
UPDATE leave_types
SET is_hourly = true
WHERE is_hourly = false
  AND (
        lower(coalesce(unit, '')) = 'hours'
     OR lower(trim(coalesce(name_en, ''))) = 'hourly'
     OR trim(coalesce(name_ar, '')) = 'زمنية'
  );

-- Daily-limit exemption: preserve exactly the types that were exempt before.
UPDATE leave_types
SET bypasses_daily_limit = true
WHERE bypasses_daily_limit = false
  AND lower(trim(coalesce(name_en, ''))) = 'emergency';

CREATE INDEX IF NOT EXISTS idx_leave_types_is_hourly
    ON leave_types (is_hourly)
    WHERE is_hourly;
