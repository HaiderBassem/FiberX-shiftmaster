-- Drop old tables if they exist
DROP TABLE IF EXISTS module_exclusions;
DROP TABLE IF EXISTS module_departments;

-- Create dynamic links table
CREATE TABLE IF NOT EXISTS external_links (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    title VARCHAR(255) NOT NULL,
    url VARCHAR(2048) NOT NULL,
    icon_name VARCHAR(100) DEFAULT 'link',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    created_by UUID REFERENCES employees(id) ON DELETE SET NULL
);

-- Table to track which departments have access to which link
CREATE TABLE IF NOT EXISTS link_departments (
    link_id UUID NOT NULL REFERENCES external_links(id) ON DELETE CASCADE,
    department_id UUID NOT NULL REFERENCES departments(id) ON DELETE CASCADE,
    granted_by UUID REFERENCES employees(id) ON DELETE SET NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (link_id, department_id)
);

-- Table to track explicit exclusions for specific employees
CREATE TABLE IF NOT EXISTS link_exclusions (
    link_id UUID NOT NULL REFERENCES external_links(id) ON DELETE CASCADE,
    employee_id UUID NOT NULL REFERENCES employees(id) ON DELETE CASCADE,
    excluded_by UUID REFERENCES employees(id) ON DELETE SET NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (link_id, employee_id)
);

-- Optional: Insert the defaults (Live Map, Ticket System) if you want them there immediately
INSERT INTO external_links (id, title, url, icon_name) 
VALUES 
    (gen_random_uuid(), 'Live Map', 'https://maps.shift-master.org/', 'map-pin'),
    (gen_random_uuid(), 'Ticket System', 'https://ticket.shift-master.org/', 'ticket')
ON CONFLICT DO NOTHING;
