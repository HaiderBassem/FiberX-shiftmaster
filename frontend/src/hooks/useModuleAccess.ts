import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import api from '@/lib/api';

export const useMyModules = () => {
  return useQuery({
    queryKey: ['my-modules'],
    queryFn: async () => {
      const { data } = await api.get('/modules/my-access');
      return data.data as string[];
    },
    staleTime: 5 * 60 * 1000,
  });
};

export const useModuleAccess = (moduleName: string) => {
  return useQuery({
    queryKey: ['module-access', moduleName],
    queryFn: async () => {
      const { data } = await api.get(`/modules/${moduleName}/access`);
      return data.data as {
        module_name: string;
        departments: string[];
        excluded_employees: string[];
      };
    },
  });
};

export const useSetDepartmentAccess = (moduleName: string) => {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async ({ departmentId, grant }: { departmentId: string; grant: boolean }) => {
      await api.post(`/modules/${moduleName}/departments`, {
        department_id: departmentId,
        grant,
      });
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['module-access', moduleName] });
      queryClient.invalidateQueries({ queryKey: ['my-modules'] });
    },
  });
};

export const useSetEmployeeExclusion = (moduleName: string) => {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async ({ employeeId, exclude }: { employeeId: string; exclude: boolean }) => {
      await api.post(`/modules/${moduleName}/employees`, {
        employee_id: employeeId,
        exclude,
      });
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['module-access', moduleName] });
      queryClient.invalidateQueries({ queryKey: ['my-modules'] });
    },
  });
};
