import api from '../../lib/api';

export interface HelpDocument {
  id: string;
  department_id: string;
  title: string;
  content: string;
  created_by: string;
  created_at: string;
  updated_at: string;
  access_level?: string;
}

export interface HelpDocumentAccess {
  id: string;
  document_id: string;
  employee_id: string;
  access_level: string;
  granted_by: string;
  created_at: string;
}

export const helpDocumentService = {
  getVisibleDocuments: async (): Promise<HelpDocument[]> => {
    const response = await api.get('/help-docs');
    return response.data.data || [];
  },

  getDocument: async (id: string): Promise<HelpDocument> => {
    const response = await api.get(`/help-docs/${id}`);
    return response.data.data;
  },

  createDocument: async (data: { title: string; content: string }): Promise<HelpDocument> => {
    const response = await api.post('/help-docs', data);
    return response.data.data;
  },

  updateDocument: async (id: string, data: { title: string; content: string }): Promise<HelpDocument> => {
    const response = await api.put(`/help-docs/${id}`, data);
    return response.data.data;
  },

  deleteDocument: async (id: string): Promise<void> => {
    await api.delete(`/help-docs/${id}`);
  },

  getAccessList: async (id: string): Promise<HelpDocumentAccess[]> => {
    const response = await api.get(`/help-docs/${id}/access`);
    return response.data.data || [];
  },

  setEmployeeAccess: async (id: string, employee_id: string, access_level: string): Promise<void> => {
    await api.post(`/help-docs/${id}/access`, { employee_id, access_level });
  },
};
