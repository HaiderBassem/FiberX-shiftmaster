-- FiberX Data Table (idempotent)
CREATE TABLE IF NOT EXISTS fiberx_data (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    department_id UUID REFERENCES departments(id) ON DELETE CASCADE,
    title VARCHAR(255) NOT NULL,
    content TEXT NOT NULL DEFAULT '',
    created_by UUID REFERENCES employees(id) ON DELETE SET NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Share a document with another department
CREATE TABLE IF NOT EXISTS fiberx_data_department_shares (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    data_id UUID REFERENCES fiberx_data(id) ON DELETE CASCADE,
    department_id UUID REFERENCES departments(id) ON DELETE CASCADE,
    access_level VARCHAR(50) NOT NULL DEFAULT 'read',
    granted_by UUID REFERENCES employees(id) ON DELETE SET NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(data_id, department_id)
);

-- Granular employee access
CREATE TABLE IF NOT EXISTS fiberx_data_employee_access (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    data_id UUID REFERENCES fiberx_data(id) ON DELETE CASCADE,
    employee_id UUID REFERENCES employees(id) ON DELETE CASCADE,
    access_level VARCHAR(50) NOT NULL,
    granted_by UUID REFERENCES employees(id) ON DELETE SET NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(data_id, employee_id)
);

-- Add permission column to employees table (idempotent)
ALTER TABLE employees ADD COLUMN IF NOT EXISTS can_manage_fiberx_data BOOLEAN DEFAULT false;
COMMENT ON COLUMN employees.can_manage_fiberx_data IS 'Grants an employee the ability to create and edit all FiberX Data documents in their department';
