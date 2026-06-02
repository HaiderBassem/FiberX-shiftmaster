-- Add can_manage_help_docs to employees table
ALTER TABLE employees 
ADD COLUMN can_manage_help_docs BOOLEAN DEFAULT false;

COMMENT ON COLUMN employees.can_manage_help_docs IS 'Grants an employee the ability to create and edit all help documents in their department';
