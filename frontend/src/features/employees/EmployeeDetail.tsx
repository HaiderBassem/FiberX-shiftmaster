import { useMemo } from 'react';
import { useParams, Link } from 'react-router-dom';
import { useQuery } from '@tanstack/react-query';
import api from '@/lib/api';
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { ArrowLeft, Mail, Phone, Briefcase, Building2, CalendarDays, Users } from 'lucide-react';

type Employee = {
  id: string;
  employee_code: string;
  first_name: string;
  last_name: string;
  gender: string;
  phone: string | null;
  email: string;
  hire_date: string;
  role: string;
  department_id: string | null;
  position: string | null;
  default_shift_id: string | null;
  weekly_off_days: number;
  can_cover_night_shift: boolean;
  status: string;
  last_login: string | null;
  created_at: string;
  updated_at: string;
};

type Department = { id: string; department_code: string; name: string; manager_id: string | null };
type Shift = { id: string; name: string; shift_code: string };

export const EmployeeDetail = () => {
  const { id } = useParams();

  const { data: employee, isLoading } = useQuery<Employee>({
    queryKey: ['employee', id],
    queryFn: async () => {
      const res = await api.get(`/employees/${id}`);
      return res.data?.data;
    },
    enabled: !!id,
  });

  const { data: departments } = useQuery<Department[]>({
    queryKey: ['departments'],
    queryFn: async () => {
      const res = await api.get('/departments');
      return res.data?.data || [];
    },
  });

  const { data: shifts } = useQuery<Shift[]>({
    queryKey: ['shifts'],
    queryFn: async () => {
      const res = await api.get('/shifts');
      return res.data?.data || [];
    },
  });

  const deptMap = useMemo(() => {
    const m: Record<string, Department> = {};
    (departments || []).forEach((d) => { m[d.id] = d; });
    return m;
  }, [departments]);

  const shiftMap = useMemo(() => {
    const m: Record<string, Shift> = {};
    (shifts || []).forEach((s) => { m[s.id] = s; });
    return m;
  }, [shifts]);

  if (isLoading) {
    return <Card className="animate-pulse bg-zinc-900/40 border-zinc-800/60 h-44" />;
  }

  if (!employee) {
    return (
      <div className="space-y-4">
        <p className="text-zinc-400">Employee not found.</p>
        <Link to="/employees"><Button variant="outline" className="border-zinc-700 text-zinc-300">Back</Button></Link>
      </div>
    );
  }

  const dept = employee.department_id ? deptMap[employee.department_id] : undefined;
  const shift = employee.default_shift_id ? shiftMap[employee.default_shift_id] : undefined;

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-3">
          <Link to="/employees">
            <Button variant="outline" className="border-zinc-700 text-zinc-300 gap-2">
              <ArrowLeft className="w-4 h-4" /> Back
            </Button>
          </Link>
          <div>
            <h2 className="text-3xl font-bold tracking-tight text-white flex items-center gap-3">
              <Users className="w-8 h-8 text-emerald-400" />
              {employee.first_name} {employee.last_name}
            </h2>
            <p className="text-zinc-400">{employee.employee_code} · {employee.role.replace('_', ' ')}</p>
          </div>
        </div>
      </div>

      <Card className="bg-zinc-900/40 border-zinc-800/60">
        <CardHeader>
          <CardTitle className="text-white">Employee Details</CardTitle>
          <CardDescription>Readable profile data (no raw IDs)</CardDescription>
        </CardHeader>
        <CardContent className="grid md:grid-cols-2 gap-4 text-sm">
          <div className="p-4 rounded-xl bg-zinc-800/30 border border-zinc-700/30">
            <div className="flex items-center gap-2 text-zinc-400 mb-2"><Mail className="w-4 h-4" /> Email</div>
            <div className="text-zinc-100 font-medium">{employee.email}</div>
          </div>

          <div className="p-4 rounded-xl bg-zinc-800/30 border border-zinc-700/30">
            <div className="flex items-center gap-2 text-zinc-400 mb-2"><Phone className="w-4 h-4" /> Phone</div>
            <div className="text-zinc-100 font-medium">{employee.phone || '—'}</div>
          </div>

          <div className="p-4 rounded-xl bg-zinc-800/30 border border-zinc-700/30">
            <div className="flex items-center gap-2 text-zinc-400 mb-2"><Building2 className="w-4 h-4" /> Department</div>
            <div className="text-zinc-100 font-medium">{dept ? `${dept.name} (${dept.department_code})` : '—'}</div>
          </div>

          <div className="p-4 rounded-xl bg-zinc-800/30 border border-zinc-700/30">
            <div className="flex items-center gap-2 text-zinc-400 mb-2"><Briefcase className="w-4 h-4" /> Position</div>
            <div className="text-zinc-100 font-medium">{employee.position || '—'}</div>
          </div>

          <div className="p-4 rounded-xl bg-zinc-800/30 border border-zinc-700/30">
            <div className="flex items-center gap-2 text-zinc-400 mb-2"><CalendarDays className="w-4 h-4" /> Hire Date</div>
            <div className="text-zinc-100 font-medium">{employee.hire_date?.split('T')[0]}</div>
          </div>

          <div className="p-4 rounded-xl bg-zinc-800/30 border border-zinc-700/30">
            <div className="flex items-center gap-2 text-zinc-400 mb-2"><CalendarDays className="w-4 h-4" /> Shift</div>
            <div className="text-zinc-100 font-medium">{shift ? `${shift.name} (${shift.shift_code})` : '—'}</div>
          </div>

          <div className="p-4 rounded-xl bg-zinc-800/30 border border-zinc-700/30">
            <div className="text-zinc-400 mb-2">Weekly Off Days</div>
            <div className="text-zinc-100 font-medium">{employee.weekly_off_days}</div>
          </div>

          <div className="p-4 rounded-xl bg-zinc-800/30 border border-zinc-700/30">
            <div className="text-zinc-400 mb-2">Night Shift Coverage</div>
            <div className="text-zinc-100 font-medium">{employee.can_cover_night_shift ? 'Yes' : 'No'}</div>
          </div>

          <div className="p-4 rounded-xl bg-zinc-800/30 border border-zinc-700/30">
            <div className="text-zinc-400 mb-2">Status</div>
            <div className="text-zinc-100 font-medium capitalize">{employee.status}</div>
          </div>

          <div className="p-4 rounded-xl bg-zinc-800/30 border border-zinc-700/30">
            <div className="text-zinc-400 mb-2">Last Login</div>
            <div className="text-zinc-100 font-medium">{employee.last_login ? employee.last_login : '—'}</div>
          </div>
        </CardContent>
      </Card>
    </div>
  );
};

