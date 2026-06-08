import api from '../lib/api';

export interface FiberxData {
  id: string;
  department_id: string;
  title: string;
  content: string;
  created_by: string;
  created_at: string;
  updated_at: string;
  creator_name: string;
  department_name: string;
  is_shared: boolean;
  access_level: string; // 'read', 'write'
}

export interface EmployeeAccess {
  id: string;
  employee_id: string;
  employee_name: string;
  access_level: string; // 'hide', 'read', 'write'
}

export interface DepartmentShare {
  id: string;
  department_id: string;
  department_name: string;
  access_level: string; // 'read', 'write'
}

export const fiberxDataService = {
  getVisible: async () => {
    const { data } = await api.get('/fiberx-data');
    return data.data as FiberxData[];
  },

  getById: async (id: string) => {
    const { data } = await api.get(`/fiberx-data/${id}`);
    return data.data as FiberxData;
  },

  create: async (payload: { title: string; content: string }) => {
    const { data } = await api.post('/fiberx-data', payload);
    return data.data as FiberxData;
  },

  update: async (id: string, payload: { title: string; content: string }) => {
    const { data } = await api.put(`/fiberx-data/${id}`, payload);
    return data.data as FiberxData;
  },

  delete: async (id: string) => {
    const { data } = await api.delete(`/fiberx-data/${id}`);
    return data;
  },

  getEmployeeAccessList: async (id: string) => {
    const { data } = await api.get(`/fiberx-data/${id}/access`);
    return data.data as EmployeeAccess[];
  },

  setEmployeeAccess: async (id: string, employee_id: string, access_level: string) => {
    const { data } = await api.post(`/fiberx-data/${id}/access`, { employee_id, access_level });
    return data;
  },

  getDepartmentShares: async (id: string) => {
    const { data } = await api.get(`/fiberx-data/${id}/shares`);
    return data.data as DepartmentShare[];
  },

  setDepartmentShare: async (id: string, department_id: string, access_level: string) => {
    const { data } = await api.post(`/fiberx-data/${id}/shares`, { department_id, access_level });
    return data;
  },
};
