-- Add fiberx_enabled toggle to departments (admin controls which departments can see FiberX Data)
ALTER TABLE departments ADD COLUMN IF NOT EXISTS fiberx_enabled BOOLEAN DEFAULT false;

COMMENT ON COLUMN departments.fiberx_enabled IS 'Controls whether this department can access FiberX Data feature';
