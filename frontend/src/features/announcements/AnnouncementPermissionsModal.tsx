import React, { useState, useMemo } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import api from '@/lib/api';
import { X, Search, Loader2 } from 'lucide-react';
import { Button } from '@/components/ui/button';

interface AnnouncementPermissionsModalProps {
  isOpen: boolean;
  onClose: () => void;
}

export const AnnouncementPermissionsModal: React.FC<AnnouncementPermissionsModalProps> = ({ isOpen, onClose }) => {
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
    mutationFn: async ({ id, canPost }: { id: string; canPost: boolean }) => {
      await api.put(`/employees/${id}/announcement-permission`, {
        can_post_announcements: canPost,
      });
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['employees'] });
    },
  });

  const filteredEmployees = useMemo(() => {
    if (!employees) return [];
    return employees.filter((emp: any) =>
      `${emp.first_name} ${emp.last_name} ${emp.email}`.toLowerCase().includes(search.toLowerCase())
    );
  }, [employees, search]);

  if (!isOpen) return null;

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50 p-4 animate-in fade-in duration-200">
      <div className="w-full max-w-lg rounded-xl bg-background shadow-xl animate-in zoom-in-95 duration-200 flex flex-col max-h-[85vh]">
        
        {/* Header */}
        <div className="flex items-center justify-between border-b p-4">
          <div>
            <h3 className="text-lg font-semibold">Announcement Permissions</h3>
            <p className="text-sm text-muted-foreground">Select who can create and publish announcements.</p>
          </div>
          <Button variant="ghost" size="icon" onClick={onClose} className="rounded-full">
            <X className="h-5 w-5" />
          </Button>
        </div>

        {/* Body */}
        <div className="p-4 border-b">
          <div className="relative">
            <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
            <input
              type="text"
              value={search}
              onChange={(e) => setSearch(e.target.value)}
              placeholder="Search employees..."
              className="w-full pl-9 pr-4 py-2 bg-muted/50 border-transparent focus:bg-background focus:border-primary focus:ring-1 focus:ring-primary rounded-lg text-sm transition-all"
            />
          </div>
        </div>

        <div className="flex-1 overflow-y-auto p-2">
          {isLoading ? (
            <div className="flex justify-center py-8">
              <Loader2 className="h-6 w-6 animate-spin text-primary" />
            </div>
          ) : filteredEmployees.length === 0 ? (
            <div className="text-center py-8 text-muted-foreground text-sm">
              No employees found.
            </div>
          ) : (
            <div className="space-y-1">
              {filteredEmployees.map((emp: any) => (
                <div key={emp.id} className="flex items-center justify-between p-3 rounded-lg hover:bg-muted/50 transition-colors">
                  <div>
                    <div className="font-medium text-sm flex items-center gap-2">
                      {emp.first_name} {emp.last_name}
                      {emp.can_post_announcements && (
                         <span className="bg-primary/10 text-primary text-[10px] px-2 py-0.5 rounded-full uppercase tracking-wider font-semibold">
                           Can Post
                         </span>
                      )}
                    </div>
                    <div className="text-xs text-muted-foreground">{emp.email} &bull; {emp.role.replace('_', ' ')}</div>
                  </div>
                  
                  <label className="relative inline-flex items-center cursor-pointer">
                    <input 
                      type="checkbox" 
                      className="sr-only peer"
                      checked={emp.can_post_announcements || false}
                      onChange={(e) => togglePermission.mutate({ id: emp.id, canPost: e.target.checked })}
                      disabled={togglePermission.isPending}
                    />
                    <div className="w-9 h-5 bg-muted peer-focus:outline-none peer-focus:ring-2 peer-focus:ring-primary/20 rounded-full peer peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:border-gray-300 after:border after:rounded-full after:h-4 after:w-4 after:transition-all peer-checked:bg-primary"></div>
                  </label>
                </div>
              ))}
            </div>
          )}
        </div>
      </div>
    </div>
  );
};
