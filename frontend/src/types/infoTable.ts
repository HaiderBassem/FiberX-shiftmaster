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
  my_access_level?: string;
}

/**
 * One cell of a table row. The row editor writes strings; the column is JSONB,
 * so another writer may have left a number, a boolean, or null in it.
 */
export type InfoTableCell = string | number | boolean | null;

export interface InfoTableRow {
  id: string;
  table_id: string;
  data: Record<string, InfoTableCell>;
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
