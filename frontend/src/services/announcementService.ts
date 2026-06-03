import api from '../lib/api';

export interface Announcement {
  id: string;
  department_id: string;
  title: string;
  message: string;
  priority: 'info' | 'normal' | 'important' | 'critical';
  is_active: boolean;
  created_by: string;
  creator_name?: string;
  created_at: string;
  updated_at: string;
}

export const announcementService = {
  getActive: async () => {
    const { data } = await api.get('/announcements/active');
    return data.data as Announcement | null;
  },

  getAll: async () => {
    const { data } = await api.get('/announcements');
    return data.data as Announcement[];
  },

  create: async (announcement: Partial<Announcement>) => {
    const { data } = await api.post('/announcements', announcement);
    return data.data as Announcement;
  },

  delete: async (id: string) => {
    const { data } = await api.delete(`/announcements/${id}`);
    return data;
  },

  setActive: async (id: string) => {
    const { data } = await api.put(`/announcements/${id}/activate`);
    return data;
  },
};
