-- Add fiberx_enabled toggle to departments (admin controls which departments can see FiberX Data)
--
-- The column add and the enable-for-existing-departments backfill are one
-- guarded unit: both run only when the column does not exist yet. The
-- original form guarded only the ADD (IF NOT EXISTS) and left the UPDATE
-- bare, so `migrate adopt` — which replays pre-ledger-era files against a
-- database that already ran them — re-executed the UPDATE and re-enabled
-- FiberX for every department, including ones an administrator had
-- deliberately disabled. First-run behaviour is unchanged: new column,
-- default false, all departments existing at introduction time enabled.
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns
         WHERE table_name = 'departments' AND column_name = 'fiberx_enabled'
    ) THEN
        ALTER TABLE departments ADD COLUMN fiberx_enabled BOOLEAN DEFAULT false;

        -- Enable FiberX Data for all departments that predate the toggle
        -- (so nothing breaks).
        UPDATE departments SET fiberx_enabled = true;
    END IF;
END $$;

COMMENT ON COLUMN departments.fiberx_enabled IS 'Controls whether this department can access FiberX Data feature';
