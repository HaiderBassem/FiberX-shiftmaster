import { useState, useMemo } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { X, Search, Shield, UserCircle, Loader2 } from 'lucide-react';
import api from '@/lib/api';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';

interface HelpPermissionsModalProps {
  isOpen: boolean;
  onClose: () => void;
}

export const HelpPermissionsModal = ({ isOpen, onClose }: HelpPermissionsModalProps) => {
  const queryClient = useQueryClient();
  const [search, setSearch] = useState('');

  // Fetch employees
  const { data: employees, isLoading } = useQuery({
    queryKey: ['employees'],
    queryFn: async () => {
      const res = await api.get('/employees');
      return res.data?.data || [];
    },
    enabled: isOpen,
  });

  const togglePermission = useMutation({
    mutationFn: async ({ id, canManage }: { id: string; canManage: boolean }) => {
      await api.put(`/employees/${id}/help-permission`, {
        can_manage_help_docs: canManage,
      });
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['employees'] });
    },
  });

  const filteredEmployees = useMemo(() => {
    if (!employees) return [];
    return employees.filter((emp: any) =>
      `${emp.first_name} ${emp.last_name}`.toLowerCase().includes(search.toLowerCase()) ||
      emp.employee_code.toLowerCase().includes(search.toLowerCase())
    );
  }, [employees, search]);

  if (!isOpen) return null;

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-background/80 backdrop-blur-sm">
      <div className="bg-card w-full max-w-lg rounded-xl shadow-lg border border-border flex flex-col max-h-[85vh]">
        <div className="flex items-center justify-between p-4 border-b border-border">
          <div className="flex items-center gap-2">
            <Shield className="w-5 h-5 text-primary" />
            <h2 className="text-lg font-semibold">Manage Document Permissions</h2>
          </div>
          <Button variant="ghost" size="sm" onClick={onClose} className="w-8 h-8 p-0 rounded-full">
            <X className="w-4 h-4" />
          </Button>
        </div>

        <div className="p-4 border-b border-border bg-muted/30">
          <div className="relative">
            <Search className="w-4 h-4 text-muted-foreground absolute left-3 top-3" />
            <Input
              value={search}
              onChange={(e) => setSearch(e.target.value)}
              placeholder="Search employees..."
              className="pl-9 h-10"
            />
          </div>
        </div>

        <div className="overflow-y-auto p-4 flex-1">
          {isLoading ? (
            <div className="flex justify-center py-8">
              <Loader2 className="w-6 h-6 animate-spin text-muted-foreground" />
            </div>
          ) : filteredEmployees.length === 0 ? (
            <div className="text-center py-8 text-muted-foreground text-sm">
              No employees found.
            </div>
          ) : (
            <div className="space-y-2">
              {filteredEmployees.map((emp: any) => (
                <div key={emp.id} className="flex items-center justify-between p-3 rounded-lg border border-border bg-background hover:bg-muted/50 transition-colors">
                  <div className="flex items-center gap-3">
                    <UserCircle className="w-8 h-8 text-muted-foreground" />
                    <div>
                      <div className="font-medium text-sm">
                        {emp.first_name} {emp.last_name}
                      </div>
                      <div className="text-xs text-muted-foreground">
                        {emp.employee_code} • {emp.role.replace('_', ' ')}
                      </div>
                    </div>
                  </div>
                  
                  <div className="flex items-center gap-2">
                    <label className="text-xs font-medium text-muted-foreground mr-2">Can Create</label>
                    <label className="relative inline-flex items-center cursor-pointer">
                      <input
                        type="checkbox"
                        className="sr-only peer"
                        checked={emp.can_manage_help_docs}
                        onChange={(e) => togglePermission.mutate({ id: emp.id, canManage: e.target.checked })}
                        disabled={togglePermission.isPending || emp.role === 'admin'}
                      />
                      <div className="w-9 h-5 bg-muted peer-focus:outline-none peer-focus:ring-2 peer-focus:ring-primary/20 rounded-full peer peer-checked:after:translate-x-full rtl:peer-checked:after:-translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:start-[2px] after:bg-white after:border-gray-300 after:border after:rounded-full after:h-4 after:w-4 after:transition-all peer-checked:bg-primary"></div>
                    </label>
                  </div>
                </div>
              ))}
            </div>
          )}
        </div>
      </div>
    </div>
  );
};
