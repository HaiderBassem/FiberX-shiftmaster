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
    return <Card className="animate-pulse h-44" />;
  }

  if (!employee) {
    return (
      <div className="space-y-4">
        <p className="text-muted-foreground">Employee not found.</p>
        <Link to="/employees"><Button variant="outline">Back</Button></Link>
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
            <Button variant="outline" className="gap-2">
              <ArrowLeft className="w-4 h-4" /> Back
            </Button>
          </Link>
          <div>
            <h2 className="text-3xl font-bold tracking-tight text-foreground flex items-center gap-3">
              <Users className="w-8 h-8 text-primary" />
              {employee.first_name} {employee.last_name}
            </h2>
            <p className="text-muted-foreground">{employee.employee_code} · {employee.role.replace('_', ' ')}</p>
          </div>
        </div>
      </div>

      <Card>
        <CardHeader>
          <CardTitle>Employee Details</CardTitle>
          <CardDescription>Readable profile data (no raw IDs)</CardDescription>
        </CardHeader>
        <CardContent className="grid md:grid-cols-2 gap-4 text-sm">
          <DetailBlock icon={<Mail className="w-4 h-4" />} label="Email" value={employee.email} />
          <DetailBlock icon={<Phone className="w-4 h-4" />} label="Phone" value={employee.phone || '—'} />
          <DetailBlock icon={<Building2 className="w-4 h-4" />} label="Department" value={dept ? `${dept.name} (${dept.department_code})` : '—'} />
          <DetailBlock icon={<Briefcase className="w-4 h-4" />} label="Position" value={employee.position || '—'} />
          <DetailBlock icon={<CalendarDays className="w-4 h-4" />} label="Hire Date" value={employee.hire_date?.split('T')[0]} />
          <DetailBlock icon={<CalendarDays className="w-4 h-4" />} label="Shift" value={shift ? `${shift.name} (${shift.shift_code})` : '—'} />
          <DetailBlock label="Weekly Off Days" value={String(employee.weekly_off_days)} />
          <DetailBlock label="Night Shift Coverage" value={employee.can_cover_night_shift ? 'Yes' : 'No'} />
          <DetailBlock label="Status" value={employee.status} />
          <DetailBlock label="Last Login" value={employee.last_login || '—'} />
        </CardContent>
      </Card>
    </div>
  );
};

const DetailBlock = ({ icon, label, value }: { icon?: React.ReactNode; label: string; value: string }) => (
  <div className="p-4 rounded-xl bg-muted/30 border border-border">
    <div className="flex items-center gap-2 text-muted-foreground mb-2">
      {icon}
      {label}
    </div>
    <div className="text-foreground font-medium">{value}</div>
  </div>
);
