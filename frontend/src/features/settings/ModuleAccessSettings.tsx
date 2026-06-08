import { useState } from 'react';
import { useAuthStore } from '@/store/authStore';
import { useModuleAccess, useSetDepartmentAccess, useSetEmployeeExclusion } from '@/hooks/useModuleAccess';
import { useQuery } from '@tanstack/react-query';
import api from '@/lib/api';
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from '@/components/ui/card';
import { Switch } from '@/components/ui/switch';
import { Button } from '@/components/ui/button';
import { MapPin, Ticket, ShieldCheck, Users } from 'lucide-react';
import { toast } from 'sonner';

const MODULES = [
  { id: 'live_map', name: 'Live Map', icon: MapPin },
  { id: 'ticket_system', name: 'Ticket System', icon: Ticket },
];

export const ModuleAccessSettings = () => {
  const { user } = useAuthStore();
  const isAdmin = user?.role === 'admin';
  const isSupervisor = ['admin', 'manager', 'team_leader'].includes(user?.role || '');

  const [selectedModule, setSelectedModule] = useState(MODULES[0].id);
  const [selectedDepartmentId, setSelectedDepartmentId] = useState<string | null>(null);

  // Fetch departments if admin
  const { data: departments } = useQuery({
    queryKey: ['departments'],
    queryFn: async () => {
      const res = await api.get('/departments');
      return res.data?.data || [];
    },
    enabled: isAdmin,
  });

  // Determine which department we are managing
  // Admin selects from list, Manager/TL manages their own department context
  const activeDepartmentId = isAdmin ? selectedDepartmentId : user?.department_id;

  // Fetch employees for the active department
  const { data: employees } = useQuery({
    queryKey: ['employees', 'department', activeDepartmentId],
    queryFn: async () => {
      if (!activeDepartmentId) return [];
      const res = await api.get(`/employees?department_id=${activeDepartmentId}`);
      return res.data?.data || [];
    },
    enabled: !!activeDepartmentId,
  });

  // Fetch module access data
  const { data: accessData, isLoading: accessLoading } = useModuleAccess(selectedModule);

  const { mutateAsync: setDeptAccess } = useSetDepartmentAccess(selectedModule);
  const { mutateAsync: setEmpExclusion } = useSetEmployeeExclusion(selectedModule);

  if (!isSupervisor) {
    return <div className="p-8 text-center text-muted-foreground">You do not have access to this page.</div>;
  }

  const handleToggleDept = async (deptId: string, grant: boolean) => {
    try {
      await setDeptAccess({ departmentId: deptId, grant });
      toast.success(grant ? 'Module enabled for department' : 'Module disabled for department');
    } catch (err: any) {
      toast.error(err.response?.data?.error || 'Failed to update department access');
    }
  };

  const handleToggleEmp = async (empId: string, exclude: boolean) => {
    try {
      await setEmpExclusion({ employeeId: empId, exclude });
      toast.success(exclude ? 'Access revoked for employee' : 'Access restored for employee');
    } catch (err: any) {
      toast.error(err.response?.data?.error || 'Failed to update employee access');
    }
  };

  const isModuleEnabledForActiveDept = activeDepartmentId && accessData?.departments?.includes(activeDepartmentId);

  return (
    <div className="space-y-6 max-w-5xl mx-auto pb-12">
      <div className="flex items-center gap-3">
        <div className="w-12 h-12 rounded-xl bg-primary/10 flex items-center justify-center">
          <ShieldCheck className="w-6 h-6 text-primary" />
        </div>
        <div>
          <h2 className="text-3xl font-bold tracking-tight text-foreground">External Modules Access</h2>
          <p className="text-muted-foreground">Manage who can see Live Maps and Ticket System</p>
        </div>
      </div>

      <div className="flex gap-4 border-b border-border/50 pb-4">
        {MODULES.map(m => (
          <Button
            key={m.id}
            variant={selectedModule === m.id ? 'default' : 'outline'}
            onClick={() => setSelectedModule(m.id)}
            className="gap-2"
          >
            <m.icon className="w-4 h-4" /> {m.name}
          </Button>
        ))}
      </div>

      {isAdmin && (
        <Card>
          <CardHeader>
            <CardTitle>Department Access</CardTitle>
            <CardDescription>Select which departments have this module enabled by default.</CardDescription>
          </CardHeader>
          <CardContent>
            <div className="space-y-4">
              {departments?.map((dept: any) => {
                const isEnabled = accessData?.departments?.includes(dept.id);
                return (
                  <div key={dept.id} className="flex items-center justify-between p-4 rounded-xl border border-border/50 bg-muted/20">
                    <div>
                      <div className="font-medium text-foreground">{dept.name}</div>
                      <div className="text-sm text-muted-foreground">{dept.department_code}</div>
                    </div>
                    <div className="flex items-center gap-4">
                      <Button 
                        variant="outline" 
                        size="sm" 
                        onClick={() => setSelectedDepartmentId(dept.id)}
                        className={selectedDepartmentId === dept.id ? "bg-primary/10 border-primary/30" : ""}
                      >
                        Manage Employees
                      </Button>
                      <Switch
                        checked={isEnabled}
                        onCheckedChange={(checked) => handleToggleDept(dept.id, checked)}
                      />
                    </div>
                  </div>
                );
              })}
            </div>
          </CardContent>
        </Card>
      )}

      {activeDepartmentId && (
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <Users className="w-5 h-5 text-primary" />
              Employee Access Control
            </CardTitle>
            <CardDescription>
              {isModuleEnabledForActiveDept 
                ? "The module is enabled for this department. You can explicitly revoke access for specific employees." 
                : "The module is currently disabled for this department. Employees will not see it even if allowed here."}
            </CardDescription>
          </CardHeader>
          <CardContent>
            {!isModuleEnabledForActiveDept && (
              <div className="mb-6 p-4 rounded-xl bg-destructive/10 border border-destructive/20 text-destructive text-sm font-medium">
                Warning: The module must be enabled for the department by an Admin before any employee can access it.
              </div>
            )}
            
            <div className="grid sm:grid-cols-2 gap-4">
              {employees?.map((emp: any) => {
                const isExcluded = accessData?.excluded_employees?.includes(emp.id);
                const hasAccess = !isExcluded;
                
                return (
                  <div key={emp.id} className="flex items-center justify-between p-4 rounded-xl border border-border/50 hover:bg-muted/30 transition-colors">
                    <div>
                      <div className="font-medium text-foreground flex items-center gap-2">
                        {emp.first_name} {emp.last_name}
                        {emp.role !== 'employee' && (
                          <span className="text-[10px] uppercase tracking-wider bg-primary/10 text-primary px-2 py-0.5 rounded-full">
                            {emp.role.replace('_', ' ')}
                          </span>
                        )}
                      </div>
                      <div className="text-xs text-muted-foreground">{emp.employee_code}</div>
                    </div>
                    <div className="flex items-center gap-2">
                      <span className="text-xs text-muted-foreground mr-2">
                        {hasAccess ? 'Allowed' : 'Denied'}
                      </span>
                      <Switch
                        checked={hasAccess}
                        onCheckedChange={(checked) => handleToggleEmp(emp.id, !checked)}
                      />
                    </div>
                  </div>
                );
              })}
            </div>
          </CardContent>
        </Card>
      )}
    </div>
  );
};
