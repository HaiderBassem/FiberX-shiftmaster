import api from '@/lib/api';
import type {
  InfoTable,
  InfoTableRow,
  InfoTableDepartmentAccess,
  InfoTableEmployeeAccess,
} from '../../types/infoTable';

export const infoTableService = {
  getVisibleTables: async (): Promise<InfoTable[]> => {
    const res = await api.get('/info-tables');
    return res.data.data;
  },

  createTable: async (data: Partial<InfoTable>): Promise<InfoTable> => {
    const res = await api.post('/info-tables', data);
    return res.data.data;
  },

  getTableRows: async (tableId: string): Promise<InfoTableRow[]> => {
    const res = await api.get(`/info-tables/${tableId}/rows`);
    return res.data.data;
  },

  createTableRow: async (tableId: string, data: Record<string, any>): Promise<InfoTableRow> => {
    const res = await api.post(`/info-tables/${tableId}/rows`, { data });
    return res.data.data;
  },

  updateTableRow: async (tableId: string, rowId: string, data: Record<string, any>): Promise<InfoTableRow> => {
    const res = await api.put(`/info-tables/${tableId}/rows/${rowId}`, { data });
    return res.data.data;
  },

  deleteTableRow: async (tableId: string, rowId: string): Promise<void> => {
    await api.delete(`/info-tables/${tableId}/rows/${rowId}`);
  },

  getAccessLists: async (tableId: string): Promise<{
    departments: InfoTableDepartmentAccess[];
    employees: InfoTableEmployeeAccess[];
  }> => {
    const res = await api.get(`/info-tables/${tableId}/access`);
    return res.data;
  },

  shareWithDepartment: async (tableId: string, departmentId: string): Promise<void> => {
    await api.post(`/info-tables/${tableId}/department-access`, { department_id: departmentId });
  },

  addEmployeeAccess: async (
    tableId: string,
    employeeId: string,
    accessLevel: string
  ): Promise<void> => {
    await api.post(`/info-tables/${tableId}/access`, {
      employee_id: employeeId,
      access_level: accessLevel,
    });
  },
};
