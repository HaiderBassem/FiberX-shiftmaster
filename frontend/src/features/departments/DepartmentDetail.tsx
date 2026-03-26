import { useMemo } from 'react';
import { useParams, Link } from 'react-router-dom';
import { useQuery } from '@tanstack/react-query';
import api from '@/lib/api';
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from '@/components/ui/card';
import { Building2, Users, Crown, ArrowLeft } from 'lucide-react';
import { Button } from '@/components/ui/button';

type Department = {
  id: string;
  department_code: string;
  name: string;
  description: string | null;
  manager_id: string | null;
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

  const { data: manager } = useQuery<Employee | null>({
    queryKey: ['employee', 'manager', dept?.manager_id],
    queryFn: async () => {
      if (!dept?.manager_id) return null;
      const res = await api.get(`/employees/${dept.manager_id}`);
      return res.data?.data;
    },
    enabled: !!dept?.manager_id,
  });

  if (deptLoading) {
    return <Card className="animate-pulse bg-zinc-900/40 border-zinc-800/60 h-40" />;
  }

  if (!dept) {
    return (
      <div className="space-y-4">
        <p className="text-zinc-400">Department not found.</p>
        <Link to="/departments"><Button variant="outline" className="border-zinc-700 text-zinc-300">Back</Button></Link>
      </div>
    );
  }

  const employeesOnly = (employees || []).filter((e) => e.role === 'employee');

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-3">
          <Link to="/departments">
            <Button variant="outline" className="border-zinc-700 text-zinc-300 gap-2">
              <ArrowLeft className="w-4 h-4" /> Back
            </Button>
          </Link>
          <div>
            <h2 className="text-3xl font-bold tracking-tight text-white flex items-center gap-3">
              <Building2 className="w-8 h-8 text-blue-400" />
              {dept.name}
            </h2>
            <p className="text-zinc-400">
              {dept.department_code}
              {dept.description ? ` · ${dept.description}` : ''}
            </p>
          </div>
        </div>
      </div>

      {/* Manager */}
      <Card className="bg-zinc-900/40 border-zinc-800/60">
        <CardHeader>
          <CardTitle className="text-white flex items-center gap-2">
            <Crown className="w-5 h-5 text-amber-400" />
            Department Manager
          </CardTitle>
          <CardDescription>Assigned manager for this department</CardDescription>
        </CardHeader>
        <CardContent>
          {manager ? (
            <Link to={`/employees/${manager.id}`} className="block">
              <div className="p-4 rounded-xl bg-zinc-800/30 border border-zinc-700/30 hover:bg-zinc-800/50 transition-colors">
                <div className="font-semibold text-zinc-100">
                  {manager.first_name} {manager.last_name}
                </div>
                <div className="text-sm text-zinc-400">
                  {manager.email} · {manager.employee_code}
                </div>
              </div>
            </Link>
          ) : (
            <p className="text-zinc-500">No manager assigned yet.</p>
          )}
        </CardContent>
      </Card>

      {/* Employees */}
      <Card className="bg-zinc-900/40 border-zinc-800/60">
        <CardHeader>
          <CardTitle className="text-white flex items-center gap-2">
            <Users className="w-5 h-5 text-emerald-400" />
            Employees ({employeesOnly.length})
          </CardTitle>
          <CardDescription>Employees in {dept.name}</CardDescription>
        </CardHeader>
        <CardContent>
          {empLoading ? (
            <div className="space-y-3">
              {[1, 2, 3].map((i) => <div key={i} className="animate-pulse h-14 bg-zinc-800/40 rounded-xl" />)}
            </div>
          ) : employeesOnly.length ? (
            <div className="grid md:grid-cols-2 gap-4">
              {employeesOnly.map((e) => {
                const shift = e.default_shift_id ? shiftMap[e.default_shift_id] : undefined;
                return (
                  <Link key={e.id} to={`/employees/${e.id}`} className="block">
                    <div className="p-4 rounded-xl bg-zinc-800/30 border border-zinc-700/30 hover:bg-zinc-800/50 transition-colors">
                      <div className="font-semibold text-zinc-100">
                        {e.first_name} {e.last_name}
                      </div>
                      <div className="text-sm text-zinc-400">
                        {e.employee_code} · {e.email}
                      </div>
                      {shift && (
                        <div className="text-xs text-zinc-500 mt-1">
                          Shift: {shift.name} ({shift.shift_code})
                        </div>
                      )}
                    </div>
                  </Link>
                );
              })}
            </div>
          ) : (
            <p className="text-zinc-500">No employees in this department yet.</p>
          )}
        </CardContent>
      </Card>
    </div>
  );
};

