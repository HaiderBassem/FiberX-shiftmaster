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
    return res.data.data || [];
  },

  createTable: async (data: Partial<InfoTable>): Promise<InfoTable> => {
    const res = await api.post('/info-tables', data);
    return res.data.data;
  },

  updateTable: async (tableId: string, data: Partial<InfoTable>): Promise<InfoTable> => {
    const res = await api.put(`/info-tables/${tableId}`, data);
    return res.data.data;
  },

  deleteTable: async (tableId: string): Promise<void> => {
    await api.delete(`/info-tables/${tableId}`);
  },

  getTableRows: async (tableId: string): Promise<InfoTableRow[]> => {
    const res = await api.get(`/info-tables/${tableId}/rows`);
    return res.data.data || [];
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
    return {
      departments: res.data.departments || [],
      employees: res.data.employees || [],
    };
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

  removeEmployeeAccess: async (tableId: string, employeeId: string): Promise<void> => {
    await api.delete(`/info-tables/${tableId}/access/employee/${employeeId}`);
  },

  exportToExcel: async (tableId: string): Promise<void> => {
    const response = await api.get(`/info-tables/${tableId}/export`, {
      responseType: 'blob',
    });
    const url = window.URL.createObjectURL(new Blob([response.data]));
    const link = document.createElement('a');
    link.href = url;
    // Extract filename from header if present, otherwise default
    const contentDisposition = response.headers['content-disposition'];
    let fileName = `table-${tableId}.xlsx`;
    if (contentDisposition) {
      const fileNameMatch = contentDisposition.match(/filename="?([^"]+)"?/);
      if (fileNameMatch && fileNameMatch.length === 2) {
        fileName = fileNameMatch[1];
      }
    }
    link.setAttribute('download', fileName);
    document.body.appendChild(link);
    link.click();
    link.remove();
    window.URL.revokeObjectURL(url);
  },

  importFromExcel: async (tableId: string, file: File): Promise<void> => {
    const formData = new FormData();
    formData.append('file', file);
    await api.post(`/info-tables/${tableId}/import`, formData, {
      headers: {
        'Content-Type': 'multipart/form-data',
      },
    });
  },
};
