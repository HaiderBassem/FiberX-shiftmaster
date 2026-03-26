-- Additional uniqueness constraints for data integrity.

-- Ensure email is unique (case-insensitive).
CREATE UNIQUE INDEX IF NOT EXISTS uq_employees_email_ci
ON employees (LOWER(email));

-- Ensure phone number is unique when provided.
CREATE UNIQUE INDEX IF NOT EXISTS uq_employees_phone_not_null
ON employees (phone)
WHERE phone IS NOT NULL;

-- Ensure one manager cannot manage multiple departments.
CREATE UNIQUE INDEX IF NOT EXISTS uq_departments_manager_single
ON departments (manager_id)
WHERE manager_id IS NOT NULL;

-- Ensure department names are unique.
CREATE UNIQUE INDEX IF NOT EXISTS uq_departments_name_ci
ON departments (LOWER(name));
