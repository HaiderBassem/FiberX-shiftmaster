-- Create provinces table
CREATE TABLE IF NOT EXISTS provinces (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(255) NOT NULL UNIQUE,
    sort_order INTEGER NOT NULL DEFAULT 0,
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Initial seed data.
--
-- Only seeded once at least one department exists. Migration 045 makes provinces
-- department-scoped and NOT NULL, attaching any pre-existing province to the oldest
-- department; on a brand-new database there are no departments yet, so seeding here
-- would produce ownerless rows that 045 then cannot attach to anything, and the
-- migration run fails with:
--   column "department_id" of relation "provinces" contains null values
-- Skipping the seed on an empty install leaves provinces to be created per department
-- through the provinces UI, which is what the province-scoped model expects anyway.
--
-- Matched on name rather than ON CONFLICT because 045 drops the unique constraint on
-- provinces.name, which would make an ON CONFLICT (name) clause invalid afterwards.
INSERT INTO provinces (name, sort_order)
SELECT v.name, v.sort_order
  FROM (VALUES
    ('بغداد', 1),
    ('البصرة', 2),
    ('نينوى', 3),
    ('أربيل', 4),
    ('النجف', 5),
    ('كربلاء', 6),
    ('الأنبار', 7),
    ('ديالى', 8),
    ('كركوك', 9),
    ('بابل', 10),
    ('واسط', 11),
    ('ذي قار', 12),
    ('ميسان', 13),
    ('المثنى', 14),
    ('القادسية', 15),
    ('صلاح الدين', 16),
    ('دهوك', 17),
    ('السليمانية', 18)
  ) AS v(name, sort_order)
 WHERE EXISTS (SELECT 1 FROM departments)
   AND NOT EXISTS (SELECT 1 FROM provinces p WHERE p.name = v.name);
