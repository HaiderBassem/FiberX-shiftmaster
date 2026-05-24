export interface InfoTableColumn {
  id: string;
  name: string;
  type: string; // 'text', 'number', 'date', 'link', 'select'
  order: number;
}

export interface InfoTable {
  id: string;
  name: string;
  description?: string;
  columns: InfoTableColumn[];
  department_id?: string;
  created_by?: string;
  created_at: string;
  updated_at: string;
}

export interface InfoTableRow {
  id: string;
  table_id: string;
  data: Record<string, any>;
  created_by?: string;
  created_at: string;
  updated_at: string;
}

export interface InfoTableDepartmentAccess {
  id: string;
  table_id: string;
  department_id: string;
  granted_by?: string;
  created_at: string;
}

export interface InfoTableEmployeeAccess {
  id: string;
  table_id: string;
  employee_id: string;
  access_level: string; // 'read', 'write'
  granted_by?: string;
  created_at: string;
}
