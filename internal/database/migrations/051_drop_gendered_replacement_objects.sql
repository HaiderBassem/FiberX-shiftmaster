-- =============================================================================
-- 051_drop_gendered_replacement_objects.sql
-- =============================================================================
-- Removes two database objects that filter night-shift replacement candidates on
-- e.gender = 'male':
--
--   view_eligible_night_replacements   (migration 005)
--   find_replacement_for_night(...)    (migration 006)
--
-- WHY THIS IS NOT A POLICY CHANGE
--
-- Neither object is referenced anywhere in the application. The feature that
-- actually finds replacements is ScheduleService.GetAvailableReplacements, which
-- runs its own query in internal/repository/schedule_repository.go, selects
-- candidates purely from who was off or on leave the previous day, and applies
-- no gender condition at all. Every replacement the product has ever offered a
-- user came from that path.
--
-- So the rule encoded here has had no effect on behaviour. Dropping the objects
-- changes nothing a user can observe; leaving them in place is the riskier
-- option, because they read as current policy to anyone who queries the schema
-- directly from a report or an ad-hoc script, and would silently reintroduce a
-- demographic filter the application itself does not apply.
--
-- IF A NIGHT-SHIFT ELIGIBILITY POLICY IS ACTUALLY REQUIRED, it should be modelled
-- explicitly — employees.can_cover_night_shift already exists for exactly this
-- purpose and is set per employee — rather than inferred from a demographic
-- column. This migration deliberately does not invent one.
-- =============================================================================

DROP VIEW IF EXISTS view_eligible_night_replacements;

DROP FUNCTION IF EXISTS find_replacement_for_night(DATE, UUID);
