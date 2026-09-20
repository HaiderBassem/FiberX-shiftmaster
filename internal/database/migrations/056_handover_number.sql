-- 056_handover_number.sql
--
-- Handovers were only addressable by UUID, which nobody can read over a radio
-- or type into a search box. This gives each handover a short, sequential,
-- human-facing number (#1, #2, ...) so staff can search for "handover 42"
-- the same way they'd reference a ticket number.
--
-- Existing rows are backfilled in creation order so the numbering reads as a
-- timeline rather than depending on physical row order.

CREATE SEQUENCE IF NOT EXISTS shift_handovers_handover_number_seq;

ALTER TABLE shift_handovers
    ADD COLUMN IF NOT EXISTS handover_number BIGINT;

WITH ordered AS (
    SELECT id, ROW_NUMBER() OVER (ORDER BY created_at, id) AS rn
    FROM shift_handovers
    WHERE handover_number IS NULL
)
UPDATE shift_handovers h
SET handover_number = ordered.rn
FROM ordered
WHERE h.id = ordered.id;

SELECT setval('shift_handovers_handover_number_seq', COALESCE((SELECT MAX(handover_number) FROM shift_handovers), 0) + 1, false);

ALTER TABLE shift_handovers
    ALTER COLUMN handover_number SET DEFAULT nextval('shift_handovers_handover_number_seq'),
    ALTER COLUMN handover_number SET NOT NULL;

ALTER SEQUENCE shift_handovers_handover_number_seq OWNED BY shift_handovers.handover_number;

CREATE UNIQUE INDEX IF NOT EXISTS idx_shift_handovers_number ON shift_handovers(handover_number);
