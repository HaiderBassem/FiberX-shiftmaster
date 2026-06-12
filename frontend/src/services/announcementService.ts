import api from '../lib/api';

export interface Announcement {
  id: string;
  department_id: string;
  title: string;
  message: string;
  priority: 'info' | 'normal' | 'important' | 'critical';
  is_active: boolean;
  is_ticker: boolean;
  images: string[];
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

  getActiveTicker: async () => {
    const { data } = await api.get('/announcements/active-ticker');
    return data.data as Announcement | null;
  },

  getAll: async () => {
    const { data } = await api.get('/announcements');
    return data.data as Announcement[];
  },

  create: async (announcement: Partial<Announcement>, imageFiles?: File[]) => {
    if (imageFiles && imageFiles.length > 0) {
      // Use FormData for multipart upload
      const formData = new FormData();
      formData.append('title', announcement.title || '');
      formData.append('message', announcement.message || '');
      formData.append('priority', announcement.priority || 'normal');
      formData.append('is_active', String(announcement.is_active ?? true));
      formData.append('is_ticker', String(announcement.is_ticker ?? false));

      for (const file of imageFiles) {
        formData.append('images', file);
      }

      const { data } = await api.post('/announcements', formData, {
        headers: { 'Content-Type': 'multipart/form-data' },
      });
      return data.data as Announcement;
    } else {
      // Plain JSON (no images)
      const { data } = await api.post('/announcements', announcement);
      return data.data as Announcement;
    }
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
