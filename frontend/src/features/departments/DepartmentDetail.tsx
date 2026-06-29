import { useMemo } from 'react';
import { useParams, Link } from 'react-router-dom';
import { useQuery, useQueryClient } from '@tanstack/react-query';
import api from '@/lib/api';
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from '@/components/ui/card';
import { Building2, Users, Crown, ArrowLeft, Shield } from 'lucide-react';
import { Button } from '@/components/ui/button';

type Department = {
  id: string;
  department_code: string;
  name: string;
  description: string | null;
  fiberx_enabled: boolean;
  max_leaves_per_day?: number | null;
  max_hourly_leaves_per_day?: number | null;
  manager_ids: string[]; // array from department_managers junction table
  active_modules: string[];
  created_at: string;
};

type Employee = {
  id: string;
  employee_code: string;
  first_name: string;
  last_name: string;
  email: string;
  phone: string | null;
  role: string;
  status: string;
  department_id: string | null;
  default_shift_id: string | null;
};

type Shift = { id: string; name: string; shift_code: string };

export const DepartmentDetail = () => {
  const { id } = useParams();
  const queryClient = useQueryClient();

  const { data: dept, isLoading: deptLoading } = useQuery<Department>({
    queryKey: ['department', id],
    queryFn: async () => {
      const res = await api.get(`/departments/${id}`);
      return res.data?.data;
    },
    enabled: !!id,
  });

  const { data: shifts } = useQuery<Shift[]>({
    queryKey: ['shifts'],
    queryFn: async () => {
      const res = await api.get('/shifts');
      return res.data?.data || [];
    },
  });

  const shiftMap = useMemo(() => {
    const m: Record<string, Shift> = {};
    (shifts || []).forEach((s) => { m[s.id] = s; });
    return m;
  }, [shifts]);

  const { data: employees, isLoading: empLoading } = useQuery<Employee[]>({
    queryKey: ['employees', 'department', id],
    queryFn: async () => {
      const res = await api.get(`/employees?department_id=${id}`);
      return res.data?.data || [];
    },
    enabled: !!id,
  });

  // Fetch all managers assigned to this department in parallel
  const managerIds = dept?.manager_ids ?? [];

  const { data: managers } = useQuery<Employee[]>({
    queryKey: ['department-managers', id, managerIds],
    queryFn: async () => {
      if (!managerIds.length) return [];
      const results = await Promise.all(
        managerIds.map((mId) =>
          api.get(`/employees/${mId}`).then((r) => r.data?.data as Employee).catch(() => null),
        ),
      );
      return results.filter(Boolean) as Employee[];
    },
    enabled: !!dept && managerIds.length > 0,
  });

  if (deptLoading) return <Card className="animate-pulse h-40" />;

  if (!dept) {
    return (
      <div className="space-y-4">
        <p className="text-muted-foreground">Department not found.</p>
        <Link to="/departments"><Button variant="outline">Back</Button></Link>
      </div>
    );
  }

  const employeesOnly = (employees || []).filter((e) => e.role === 'employee');

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-2 sm:gap-3">
          <Link to="/departments">
            <Button variant="outline" className="gap-2">
              <ArrowLeft className="w-4 h-4" /> Back
            </Button>
          </Link>
          <div>
            <h2 className="text-2xl sm:text-3xl font-bold tracking-tight text-foreground flex items-center gap-2 sm:gap-3">
              <Building2 className="w-6 h-6 sm:w-8 sm:h-8 text-primary" />
              {dept.name}
            </h2>
            <p className="text-muted-foreground">
              {dept.department_code}
              {dept.description ? ` · ${dept.description}` : ''}
            </p>
          </div>
        </div>
      </div>

      {/* Managers card */}
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <Crown className="w-5 h-5 text-amber-500" />
            Department Manager{managerIds.length !== 1 ? 's' : ''}
          </CardTitle>
          <CardDescription>
            {managerIds.length > 1
              ? `${managerIds.length} managers assigned to this department`
              : 'Assigned manager for this department'}
          </CardDescription>
        </CardHeader>
        <CardContent>
          {!managers || managers.length === 0 ? (
            <p className="text-muted-foreground">No manager assigned yet.</p>
          ) : (
            <div className="space-y-3">
              {managers.map((manager) => (
                <Link key={manager.id} to={`/employees/${manager.id}`} className="block">
                  <div className="p-4 rounded-xl bg-muted/30 border border-border hover:bg-muted/50 transition-colors flex items-center gap-3">
                    <div className="h-9 w-9 rounded-full bg-amber-500/10 flex items-center justify-center text-amber-500 shrink-0">
                      <Crown className="w-4 h-4" />
                    </div>
                    <div>
                      <div className="font-semibold text-foreground">
                        {manager.first_name} {manager.last_name}
                      </div>
                      <div className="text-sm text-muted-foreground">
                        {manager.email} · {manager.employee_code}
                      </div>
                    </div>
                  </div>
                </Link>
              ))}
            </div>
          )}
        </CardContent>
      </Card>

      {/* Module Access Card */}
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <Shield className="w-5 h-5 text-indigo-500" />
            Module Access
          </CardTitle>
          <CardDescription>Control which tabs and features are available for this department</CardDescription>
        </CardHeader>
        <CardContent>
          <div className="space-y-3">
            {[
              { id: 'tasks', label: 'My Tasks', desc: 'Allow employees to see and complete their tasks' },
              { id: 'handovers', label: 'Handovers', desc: 'Enable shift handover boards and logs' },
              { id: 'calendar', label: 'Calendar', desc: 'Show the interactive shift and leave calendar' },
              { id: 'task_center', label: 'Task Center', desc: 'Allow leaders to manage task boards' },
              { id: 'references', label: 'References', desc: 'Allow access to informational data tables' },
              { id: 'info_bank', label: 'Info Bank', desc: 'Provide access to help documents and guides' },
              { id: 'fiberx_data', label: 'FiberX Data', desc: 'Access to FiberX specific documentation' },
            ].map((module) => {
              const isActive = (dept.active_modules || []).includes(module.id);
              return (
                <div key={module.id} className="flex items-center justify-between p-4 rounded-xl bg-muted/30 border border-border">
                  <div>
                    <p className="font-semibold text-foreground">{module.label}</p>
                    <p className="text-sm text-muted-foreground mt-0.5">{module.desc}</p>
                  </div>
                  <label className="relative inline-flex items-center cursor-pointer">
                    <input
                      type="checkbox"
                      className="sr-only peer"
                      checked={isActive}
                      onChange={(e) => {
                        const current = dept.active_modules || [];
                        const next = e.target.checked 
                          ? [...current, module.id] 
                          : current.filter(m => m !== module.id);
                        
                        api.put(`/departments/${dept.id}`, {
                          name: dept.name,
                          description: dept.description,
                          max_leaves_per_day: dept.max_leaves_per_day,
                          max_hourly_leaves_per_day: dept.max_hourly_leaves_per_day,
                          manager_ids: dept.manager_ids,
                          active_modules: next,
                        }).then(() => {
                          queryClient.invalidateQueries({ queryKey: ['department', id] });
                        });
                      }}
                    />
                    <div className="w-11 h-6 bg-muted peer-focus:outline-none peer-focus:ring-2 peer-focus:ring-primary/20 rounded-full peer peer-checked:after:translate-x-full rtl:peer-checked:after:-translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:start-[2px] after:bg-white after:border-gray-300 after:border after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:bg-indigo-600"></div>
                  </label>
                </div>
              );
            })}
          </div>
        </CardContent>
      </Card>

      {/* Employees card */}
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <Users className="w-5 h-5 text-primary" />
            Employees ({employeesOnly.length})
          </CardTitle>
          <CardDescription>Employees in {dept.name}</CardDescription>
        </CardHeader>
        <CardContent>
          {empLoading ? (
            <div className="space-y-3">
              {[1, 2, 3].map((i) => (
                <div key={i} className="animate-pulse h-14 bg-muted/40 rounded-xl" />
              ))}
            </div>
          ) : employeesOnly.length ? (
            <div className="grid md:grid-cols-2 gap-4">
              {employeesOnly.map((e) => {
                const shift = e.default_shift_id ? shiftMap[e.default_shift_id] : undefined;
                return (
                  <Link key={e.id} to={`/employees/${e.id}`} className="block">
                    <div className="p-4 rounded-xl bg-muted/30 border border-border hover:bg-muted/50 transition-colors">
                      <div className="font-semibold text-foreground">
                        {e.first_name} {e.last_name}
                      </div>
                      <div className="text-sm text-muted-foreground">
                        {e.employee_code} · {e.email}
                      </div>
                      {shift && (
                        <div className="text-xs text-muted-foreground mt-1">
                          Shift: {shift.name} ({shift.shift_code})
                        </div>
                      )}
                    </div>
                  </Link>
                );
              })}
            </div>
          ) : (
            <p className="text-muted-foreground">No employees in this department yet.</p>
          )}
        </CardContent>
      </Card>
    </div>
  );
};
