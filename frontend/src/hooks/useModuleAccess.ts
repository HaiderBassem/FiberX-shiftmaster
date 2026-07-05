import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import api from '@/lib/api';

export interface ExternalLink {
  id: string;
  title: string;
  url: string;
  icon_name: string;
  created_at: string;
  created_by: string | null;
}

export interface LinkAccessResponse {
  link_id: string;
  title: string;
  departments: string[];
  excluded_employees: string[];
}

export const useMyLinks = () => {
  return useQuery<ExternalLink[]>({
    queryKey: ['my-links'],
    queryFn: async () => {
      try {
        const { data } = await api.get('/external-links/my-links');
        return data.data || [];
      } catch (err) {
        console.error('[useMyLinks] API error:', err);
        return [];
      }
    },
    retry: false,
  });
};

export const useAllLinks = () => {
  return useQuery<ExternalLink[]>({
    queryKey: ['all-links'],
    queryFn: async () => {
      try {
        const { data } = await api.get('/external-links');
        return data.data || [];
      } catch (err) {
        console.error('[useAllLinks] API error:', err);
        return [];
      }
    },
    retry: false,
  });
};

export const useCreateLink = () => {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (payload: { title: string; url: string; icon_name: string }) => {
      const { data } = await api.post('/external-links', payload);
      return data;
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['all-links'] });
      queryClient.invalidateQueries({ queryKey: ['my-links'] });
    },
  });
};

export const useUpdateLink = () => {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async ({ id, ...payload }: { id: string; title: string; url: string; icon_name: string }) => {
      const { data } = await api.put(`/external-links/${id}`, payload);
      return data;
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['all-links'] });
      queryClient.invalidateQueries({ queryKey: ['my-links'] });
    },
  });
};

export const useDeleteLink = () => {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (id: string) => {
      const { data } = await api.delete(`/external-links/${id}`);
      return data;
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['all-links'] });
      queryClient.invalidateQueries({ queryKey: ['my-links'] });
    },
  });
};

export const useLinkAccess = (linkId: string | null) => {
  return useQuery<LinkAccessResponse>({
    queryKey: ['link-access', linkId],
    queryFn: async () => {
      try {
        const { data } = await api.get(`/external-links/${linkId}/access`);
        return data.data || { link_id: linkId || '', title: '', departments: [], excluded_employees: [] };
      } catch (err) {
        console.error('[useLinkAccess] API error:', err);
        return { link_id: linkId || '', title: '', departments: [], excluded_employees: [] };
      }
    },
    enabled: !!linkId,
    retry: false,
  });
};

export const useSetDepartmentAccess = () => {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async ({ linkId, departmentId, grant }: { linkId: string; departmentId: string; grant: boolean }) => {
      const { data } = await api.post(`/external-links/${linkId}/departments`, {
        department_id: departmentId,
        grant,
      });
      return data;
    },
    onSuccess: (_, variables) => {
      queryClient.invalidateQueries({ queryKey: ['link-access', variables.linkId] });
      queryClient.invalidateQueries({ queryKey: ['my-links'] });
    },
  });
};

export const useSetEmployeeExclusion = () => {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async ({ linkId, employeeId, exclude }: { linkId: string; employeeId: string; exclude: boolean }) => {
      const { data } = await api.post(`/external-links/${linkId}/employees`, {
        employee_id: employeeId,
        exclude,
      });
      return data;
    },
    onSuccess: (_, variables) => {
      queryClient.invalidateQueries({ queryKey: ['link-access', variables.linkId] });
      queryClient.invalidateQueries({ queryKey: ['my-links'] });
    },
  });
};
