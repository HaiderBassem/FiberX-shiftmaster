import { useState } from 'react';
import { useAuthStore } from '@/store/authStore';
import { useQuery } from '@tanstack/react-query';
import api from '@/lib/api';
import { 
  useAllLinks, useCreateLink, useDeleteLink, 
  useLinkAccess, useSetDepartmentAccess, useSetEmployeeExclusion 
} from '@/hooks/useModuleAccess';
import { Switch } from '@/components/ui/switch';
import { ShieldCheck, Plus, Trash2, Link as LinkIcon, MapPin, Ticket, ExternalLink, Calendar, Users, BookOpen } from 'lucide-react';

const ICONS = ['link', 'map-pin', 'ticket', 'external-link', 'calendar', 'users', 'book-open'];

export const ModuleAccessSettings = () => {
  const { user } = useAuthStore();
  const isAdmin = user?.role === 'admin';
  const { data: departmentsResponse } = useQuery({
    queryKey: ['departments'],
    queryFn: async () => {
      const res = await api.get('/departments');
      return res.data?.data || [];
    }
  });
  const departments: any[] = departmentsResponse || [];

  const { data: employeesResponse } = useQuery({
    queryKey: ['employees'],
    queryFn: async () => {
      const res = await api.get('/employees');
      return res.data?.data || [];
    }
  });
  const employees: any[] = employeesResponse || [];

  // Links
  const { data: allLinks, isLoading: isLoadingLinks } = useAllLinks();
  const createLink = useCreateLink();
  const deleteLink = useDeleteLink();

  // UI State
  const [activeTab, setActiveTab] = useState<'access' | 'links'>('access');
  const [selectedLink, setSelectedLink] = useState<string>('');
  
  // Access data
  const { data: accessData, isLoading: isLoadingAccess } = useLinkAccess(selectedLink || null);
  const setDepartmentAccess = useSetDepartmentAccess();
  const setEmployeeExclusion = useSetEmployeeExclusion();

  // Form State
  const [newTitle, setNewTitle] = useState('');
  const [newUrl, setNewUrl] = useState('');
  const [newIcon, setNewIcon] = useState('link');

  const handleCreateLink = (e: React.FormEvent) => {
    e.preventDefault();
    if (!newTitle || !newUrl) return;
    createLink.mutate({ title: newTitle, url: newUrl, icon_name: newIcon }, {
      onSuccess: () => {
        setNewTitle('');
        setNewUrl('');
        setNewIcon('link');
      }
    });
  };

  const getIconComponent = (name: string) => {
    switch (name) {
      case 'map-pin': return <MapPin className="w-4 h-4" />;
      case 'ticket': return <Ticket className="w-4 h-4" />;
      case 'external-link': return <ExternalLink className="w-4 h-4" />;
      case 'calendar': return <Calendar className="w-4 h-4" />;
      case 'users': return <Users className="w-4 h-4" />;
      case 'book-open': return <BookOpen className="w-4 h-4" />;
      default: return <LinkIcon className="w-4 h-4" />;
    }
  };

  if (!isAdmin && user?.role !== 'manager' && user?.role !== 'team_leader') {
    return <div className="p-6 text-red-500">Access Denied</div>;
  }

  // Determine which departments the current user is allowed to manage
  const manageableDepartments = isAdmin 
    ? departments 
    : departments?.filter(d => d.id === user?.department_id);

  return (
    <div className="flex flex-col h-full bg-[#0B0F19] text-white">
      {/* Header */}
      <div className="flex-none p-6 border-b border-white/5 bg-white/[0.02]">
        <div className="flex items-center gap-3 mb-2">
          <div className="w-10 h-10 rounded-xl bg-purple-500/20 flex items-center justify-center border border-purple-500/30">
            <ShieldCheck className="w-5 h-5 text-purple-400" />
          </div>
          <div>
            <h1 className="text-xl font-bold text-white">External Tools Management</h1>
            <p className="text-sm text-muted-foreground">Manage dynamic links and control access</p>
          </div>
        </div>

        {/* Tabs */}
        {isAdmin && (
          <div className="flex gap-4 mt-6 border-b border-white/10">
            <button
              onClick={() => setActiveTab('access')}
              className={`pb-2 px-1 text-sm font-medium transition-colors ${activeTab === 'access' ? 'text-purple-400 border-b-2 border-purple-400' : 'text-gray-400 hover:text-gray-200'}`}
            >
              Access Control
            </button>
            <button
              onClick={() => setActiveTab('links')}
              className={`pb-2 px-1 text-sm font-medium transition-colors ${activeTab === 'links' ? 'text-purple-400 border-b-2 border-purple-400' : 'text-gray-400 hover:text-gray-200'}`}
            >
              Manage Links
            </button>
          </div>
        )}
      </div>

      <div className="flex-1 overflow-y-auto p-6">
        
        {/* --- LINKS MANAGEMENT TAB --- */}
        {isAdmin && activeTab === 'links' && (
          <div className="space-y-6 max-w-4xl">
            {/* Create form */}
            <form onSubmit={handleCreateLink} className="p-5 rounded-xl bg-white/5 border border-white/10 flex flex-col sm:flex-row gap-4 items-end">
              <div className="flex-1 w-full">
                <label className="text-xs text-gray-400 mb-1 block">Title</label>
                <input 
                  type="text" required value={newTitle} onChange={e => setNewTitle(e.target.value)}
                  className="w-full bg-black/40 border border-white/10 rounded-lg px-3 py-2 text-sm text-white" 
                  placeholder="e.g. HR Portal" 
                />
              </div>
              <div className="flex-1 w-full">
                <label className="text-xs text-gray-400 mb-1 block">URL</label>
                <input 
                  type="url" required value={newUrl} onChange={e => setNewUrl(e.target.value)}
                  className="w-full bg-black/40 border border-white/10 rounded-lg px-3 py-2 text-sm text-white" 
                  placeholder="https://..." 
                />
              </div>
              <div className="w-32 shrink-0">
                <label className="text-xs text-gray-400 mb-1 block">Icon</label>
                <select 
                  value={newIcon} onChange={e => setNewIcon(e.target.value)}
                  className="w-full bg-black/40 border border-white/10 rounded-lg px-3 py-2 text-sm text-white appearance-none"
                >
                  {ICONS.map(i => <option key={i} value={i}>{i}</option>)}
                </select>
              </div>
              <button 
                type="submit" disabled={createLink.isPending}
                className="bg-purple-600 hover:bg-purple-700 text-white px-4 py-2 rounded-lg text-sm font-medium flex items-center gap-2 transition-colors disabled:opacity-50 h-[38px]"
              >
                <Plus className="w-4 h-4" /> Add Link
              </button>
            </form>

            {/* List */}
            <div className="grid gap-3">
              {isLoadingLinks ? (
                <div className="text-center text-gray-500 py-10">Loading...</div>
              ) : allLinks?.length === 0 ? (
                <div className="text-center text-gray-500 py-10">No external links found.</div>
              ) : (
                allLinks?.map(link => (
                  <div key={link.id} className="flex items-center justify-between p-4 rounded-xl bg-white/5 border border-white/5">
                    <div className="flex items-center gap-4">
                      <div className="w-10 h-10 rounded-lg bg-black/40 flex items-center justify-center border border-white/10">
                        {getIconComponent(link.icon_name)}
                      </div>
                      <div>
                        <h3 className="font-medium text-white">{link.title}</h3>
                        <p className="text-xs text-blue-400">{link.url}</p>
                      </div>
                    </div>
                    <button 
                      onClick={() => {
                        if (confirm('Are you sure you want to delete this link?')) {
                          deleteLink.mutate(link.id);
                        }
                      }}
                      disabled={deleteLink.isPending}
                      className="text-red-400 hover:bg-red-400/10 p-2 rounded-lg transition-colors"
                    >
                      <Trash2 className="w-4 h-4" />
                    </button>
                  </div>
                ))
              )}
            </div>
          </div>
        )}

        {/* --- ACCESS MANAGEMENT TAB --- */}
        {activeTab === 'access' && (
          <div className="space-y-6 max-w-4xl">
            {/* Module Selector */}
            <div className="p-5 rounded-xl bg-white/5 border border-white/10">
              <label className="text-sm font-medium text-gray-300 mb-2 block">Select Tool to Manage Access</label>
              <select
                className="w-full bg-black/50 border border-white/10 rounded-lg px-4 py-3 text-white appearance-none cursor-pointer"
                value={selectedLink}
                onChange={(e) => setSelectedLink(e.target.value)}
              >
                <option value="" disabled>-- Select a Tool --</option>
                {allLinks?.map((m) => (
                  <option key={m.id} value={m.id}>{m.title}</option>
                ))}
              </select>
            </div>

            {selectedLink && (
              <div className="space-y-6">
                {isLoadingAccess ? (
                  <div className="text-center py-10 text-gray-500">Loading access data...</div>
                ) : (
                  manageableDepartments?.map((dept) => {
                    // Check if department is granted
                    const isDeptGranted = accessData?.departments.includes(dept.id);
                    // Get all employees for this dept
                    const deptEmployees = employees?.filter(e => e.department_id === dept.id) || [];

                    return (
                      <div key={dept.id} className="border border-white/10 rounded-xl overflow-hidden">
                        {/* Department Header */}
                        <div className="bg-white/[0.03] p-4 flex items-center justify-between border-b border-white/5">
                          <div>
                            <h3 className="font-semibold text-white text-lg">{dept.name}</h3>
                            <p className="text-xs text-gray-400">
                              {isDeptGranted 
                                ? 'Tool is ENABLED for this department' 
                                : 'Tool is DISABLED for this department'}
                            </p>
                          </div>
                          
                          {isAdmin && (
                            <div className="flex items-center gap-3">
                              <span className="text-sm font-medium text-gray-300">Dept. Access</span>
                              <Switch
                                checked={isDeptGranted}
                                onCheckedChange={(checked) => 
                                  setDepartmentAccess.mutate({ linkId: selectedLink, departmentId: dept.id, grant: checked })
                                }
                                disabled={setDepartmentAccess.isPending}
                              />
                            </div>
                          )}
                        </div>

                        {/* Employees List */}
                        {isDeptGranted ? (
                          <div className="p-4 bg-black/20 divide-y divide-white/5">
                            {deptEmployees.length === 0 ? (
                              <p className="text-sm text-gray-500 py-2">No employees found in this department.</p>
                            ) : (
                              deptEmployees.map(emp => {
                                // If they are IN excluded_employees, they DON'T have access
                                const isExcluded = accessData?.excluded_employees.includes(emp.id);
                                const hasAccess = !isExcluded;

                                return (
                                  <div key={emp.id} className="flex items-center justify-between py-3">
                                    <div className="flex items-center gap-3">
                                      <div className="w-8 h-8 rounded-full bg-blue-500/20 flex items-center justify-center text-blue-400 font-bold text-xs border border-blue-500/30">
                                        {emp.name.charAt(0)}
                                      </div>
                                      <div>
                                        <p className="text-sm font-medium text-gray-200">{emp.name}</p>
                                        <p className="text-xs text-gray-500">{emp.role}</p>
                                      </div>
                                    </div>
                                    <div className="flex items-center gap-3">
                                      <span className={`text-xs font-medium px-2 py-1 rounded-md ${hasAccess ? 'bg-green-500/10 text-green-400' : 'bg-red-500/10 text-red-400'}`}>
                                        {hasAccess ? 'Visible' : 'Hidden'}
                                      </span>
                                      <Switch
                                        checked={hasAccess}
                                        onCheckedChange={(checked) => 
                                          // Exclude if checked is false (meaning they shouldn't have access)
                                          setEmployeeExclusion.mutate({ linkId: selectedLink, employeeId: emp.id, exclude: !checked })
                                        }
                                        disabled={setEmployeeExclusion.isPending}
                                      />
                                    </div>
                                  </div>
                                );
                              })
                            )}
                          </div>
                        ) : (
                          <div className="p-4 bg-black/20 text-sm text-gray-500">
                            Enable department access first to manage individual employees.
                          </div>
                        )}
                      </div>
                    );
                  })
                )}
              </div>
            )}
          </div>
        )}

      </div>
    </div>
  );
};
