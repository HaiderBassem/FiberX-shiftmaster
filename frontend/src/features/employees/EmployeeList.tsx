import { useMemo, useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import api from '@/lib/api';
import { useAuthStore } from '@/store/authStore';
import { Card, CardContent, CardHeader, CardTitle, CardDescription, CardFooter } from '@/components/ui/card';
import { Users, Mail, Phone, Briefcase, UserPlus, ShieldCheck, Trash2, UserX, UserCheck, Edit3, Save, X, ChevronDown, ChevronUp } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { format } from 'date-fns';

interface Employee {
  id: string;
  employee_code: string;
  first_name: string;
  last_name: string;
  gender?: 'male' | 'female' | string;
  email: string;
  phone: string | null;
  role: 'employee' | 'team_leader' | 'manager' | 'admin';
  status: 'active' | 'inactive' | string;
  department_id: string | null;
  default_shift_id: string | null;
  position?: string | null;
  weekly_off_days?: number;
  can_cover_night_shift?: boolean;
}

interface Shift {
  id: string;
  name: string;
  shift_code: string;
}

interface Department {
  id: string;
  department_code: string;
  name: string;
  manager_id: string | null;
}

export const EmployeeList = () => {
  const { user } = useAuthStore();
  const queryClient = useQueryClient();

  const isTL = user?.role === 'team_leader';
  const isManager = user?.role === 'manager';
  const isAdmin = user?.role === 'admin';

  const allowedRoles: Array<Employee['role']> = isTL
    ? ['employee']
    : isManager
      ? ['employee', 'team_leader']
      : ['employee', 'team_leader', 'manager', 'admin'];

  // ── Create form state ──
  const [showCreateForm, setShowCreateForm] = useState(false);
  const [firstName, setFirstName] = useState('');
  const [lastName, setLastName] = useState('');
  const [gender, setGender] = useState<'male' | 'female'>('male');
  const [phone, setPhone] = useState('');
  const [email, setEmail] = useState('');
  const [hireDate, setHireDate] = useState(format(new Date(), 'yyyy-MM-dd'));
  const [role, setRole] = useState<Employee['role']>(allowedRoles[0] ?? 'employee');
  const [departmentId, setDepartmentId] = useState<string>('');
  const [defaultShiftId, setDefaultShiftId] = useState<string>('');
  const [position, setPosition] = useState('');
  const [weeklyOffDays, setWeeklyOffDays] = useState<number>(1);
  const [canCoverNightShift, setCanCoverNightShift] = useState(false);
  const [password, setPassword] = useState('');

  const [createdLogin, setCreatedLogin] = useState<{ email: string; password: string } | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [statusFilter, setStatusFilter] = useState<'all' | 'active' | 'inactive'>('all');
  const [departmentFilter, setDepartmentFilter] = useState<string>('all');
  const [roleFilter, setRoleFilter] = useState<'all' | Employee['role']>('all');
  const [search, setSearch] = useState('');
  const [editId, setEditId] = useState<string | null>(null);
  const [editFirstName, setEditFirstName] = useState('');
  const [editLastName, setEditLastName] = useState('');
  const [editGender, setEditGender] = useState<'male' | 'female'>('male');
  const [editPhone, setEditPhone] = useState('');
  const [editEmail, setEditEmail] = useState('');
  const [editRole, setEditRole] = useState<Employee['role']>('employee');
  const [editDepartmentId, setEditDepartmentId] = useState<string>('');
  const [editDefaultShiftId, setEditDefaultShiftId] = useState<string>('');
  const [editPosition, setEditPosition] = useState('');
  const [editWeeklyOffDays, setEditWeeklyOffDays] = useState<number>(1);
  const [editCanCoverNightShift, setEditCanCoverNightShift] = useState(false);
  const [editStatus, setEditStatus] = useState<'active' | 'inactive' | string>('active');

  const { data: shifts } = useQuery<Shift[]>({
    queryKey: ['shifts'],
    queryFn: async () => {
      const res = await api.get('/shifts');
      return res.data?.data || [];
    },
  });

  const { data: departments } = useQuery<Department[]>({
    queryKey: ['departments'],
    queryFn: async () => {
      const res = await api.get('/departments');
      return res.data?.data || [];
    },
    enabled: true,
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

  const { data: employees, isLoading, isError } = useQuery<Employee[]>({
    queryKey: ['employees'],
    queryFn: async () => {
      const response = await api.get('/employees');
      return response.data?.data || [];
    },
  });

  const createEmployee = useMutation({
    mutationFn: async () => {
      setCreatedLogin(null);
      setError(null);
      await api.post('/employees', {
        first_name: firstName,
        last_name: lastName,
        gender,
        phone: phone || null,
        email,
        password,
        hire_date: hireDate,
        role,
        department_id: departmentId || null,
        position: position || null,
        default_shift_id: defaultShiftId || null,
        weekly_off_days: weeklyOffDays,
        can_cover_night_shift: canCoverNightShift,
        status: 'active',
        profile_image: null,
      });
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['employees'] });
      setCreatedLogin({ email, password });
      setFirstName('');
      setLastName('');
      setPhone('');
      setEmail('');
      setPosition('');
      setPassword('');
      setCanCoverNightShift(false);
      setWeeklyOffDays(1);
      setShowCreateForm(false);
    },
    onError: (err: any) => {
      setError(err?.response?.data?.error || err?.message || 'Failed to create employee');
    },
  });

  const filteredEmployees = useMemo(() => {
    const q = search.trim().toLowerCase();
    return (employees || []).filter((emp) => {
      if (statusFilter !== 'all' && emp.status !== statusFilter) return false;
      if (departmentFilter !== 'all' && emp.department_id !== departmentFilter) return false;
      if (roleFilter !== 'all' && emp.role !== roleFilter) return false;
      if (!q) return true;
      const fullName = `${emp.first_name} ${emp.last_name}`.toLowerCase();
      return (
        fullName.includes(q) ||
        emp.employee_code.toLowerCase().includes(q) ||
        emp.email.toLowerCase().includes(q)
      );
    });
  }, [employees, statusFilter, departmentFilter, roleFilter, search]);

  const updateEmployeeStatus = useMutation({
    mutationFn: async ({ id, status }: { id: string; status: 'active' | 'inactive' }) => {
      await api.patch(`/employees/${id}/status`, { status });
    },
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ['employees'] }),
    onError: (err: any) => setError(err?.response?.data?.error || err?.message || 'Failed to update employee status'),
  });

  const updateEmployee = useMutation({
    mutationFn: async () => {
      if (!editId) return;
      await api.put(`/employees/${editId}`, {
        first_name: editFirstName,
        last_name: editLastName,
        gender: editGender,
        phone: editPhone || null,
        email: editEmail,
        role: editRole,
        department_id: editDepartmentId || null,
        position: editPosition || null,
        default_shift_id: editDefaultShiftId || null,
        weekly_off_days: editWeeklyOffDays,
        can_cover_night_shift: editCanCoverNightShift,
        status: editStatus,
        profile_image: null,
      });
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['employees'] });
      setEditId(null);
    },
    onError: (err: any) => setError(err?.response?.data?.error || err?.message || 'Failed to update employee'),
  });

  const deleteEmployee = useMutation({
    mutationFn: async (id: string) => {
      await api.delete(`/employees/${id}`);
    },
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ['employees'] }),
    onError: (err: any) => setError(err?.response?.data?.error || err?.message || 'Failed to delete employee'),
  });

  const startEdit = (emp: Employee) => {
    setEditId(emp.id);
    setEditFirstName(emp.first_name);
    setEditLastName(emp.last_name);
    setEditGender((emp.gender as 'male' | 'female') || 'male');
    setEditPhone(emp.phone || '');
    setEditEmail(emp.email || '');
    setEditRole(emp.role);
    setEditDepartmentId(emp.department_id || '');
    setEditDefaultShiftId(emp.default_shift_id || '');
    setEditPosition(emp.position || '');
    setEditWeeklyOffDays(emp.weekly_off_days ?? 1);
    setEditCanCoverNightShift(!!emp.can_cover_night_shift);
    setEditStatus(emp.status || 'active');
  };

  return (
    <div className="space-y-6">
      <div className="flex justify-between items-end">
        <div>
          <h2 className="text-3xl font-bold tracking-tight text-white">Employees</h2>
          <p className="text-zinc-400">
            {isTL || isManager ? 'Your department employees.' : 'View and manage the workforce directory.'}
          </p>
        </div>
      </div>

      {/* Create Account */}
      <Card className="bg-zinc-900/50 border-zinc-800/60">
        <CardHeader>
          <div className="flex items-center justify-between gap-3">
            <CardTitle className="text-white flex items-center gap-2">
              <UserPlus className="w-5 h-5 text-emerald-400" />
              Create Account
            </CardTitle>
            <Button
              variant="outline"
              className="border-zinc-700 text-zinc-200"
              onClick={() => setShowCreateForm((v) => !v)}
            >
              {showCreateForm ? <ChevronUp className="w-4 h-4 mr-1" /> : <ChevronDown className="w-4 h-4 mr-1" />}
              Create an employee
            </Button>
          </div>
          <CardDescription>
            {isTL && 'Team leaders can create employee accounts only.'}
            {isManager && 'Managers can create employee and team leader accounts.'}
            {isAdmin && 'Admins can create any role and assign departments/shifts.'}
          </CardDescription>
        </CardHeader>
        {showCreateForm && <CardContent className="space-y-4">
          {error && (
            <div className="p-3 rounded-lg bg-red-500/10 border border-red-500/20 text-red-300 text-sm">
              {error}
            </div>
          )}

          {createdLogin && (
            <div className="p-3 rounded-lg bg-emerald-500/10 border border-emerald-500/20 text-emerald-200 text-sm space-y-1">
              <div className="flex items-center gap-2 font-semibold">
                <ShieldCheck className="w-4 h-4" />
                Login credentials (give to the employee)
              </div>
              <div><span className="text-emerald-300/80">Email:</span> {createdLogin.email}</div>
              <div><span className="text-emerald-300/80">Password:</span> {createdLogin.password}</div>
            </div>
          )}

          <div className="grid md:grid-cols-3 gap-4">
            <div className="space-y-2">
              <Label className="text-zinc-300">First Name</Label>
              <Input value={firstName} onChange={(e) => setFirstName(e.target.value)} className="bg-black/20 border-zinc-700" />
            </div>
            <div className="space-y-2">
              <Label className="text-zinc-300">Last Name</Label>
              <Input value={lastName} onChange={(e) => setLastName(e.target.value)} className="bg-black/20 border-zinc-700" />
            </div>
            <div className="space-y-2">
              <Label className="text-zinc-300">Gender</Label>
              <select
                className="w-full h-10 px-3 py-2 rounded-md bg-zinc-950/50 border border-zinc-700 text-white"
                value={gender}
                onChange={(e) => setGender(e.target.value as any)}
              >
                <option value="male">Male</option>
                <option value="female">Female</option>
              </select>
            </div>
            <div className="space-y-2">
              <Label className="text-zinc-300">Phone</Label>
              <Input value={phone} onChange={(e) => setPhone(e.target.value)} className="bg-black/20 border-zinc-700" />
            </div>
            <div className="space-y-2">
              <Label className="text-zinc-300">Email (login)</Label>
              <Input value={email} onChange={(e) => setEmail(e.target.value)} className="bg-black/20 border-zinc-700" />
            </div>
            <div className="space-y-2">
              <Label className="text-zinc-300">Password</Label>
              <Input type="password" value={password} onChange={(e) => setPassword(e.target.value)} className="bg-black/20 border-zinc-700" />
              <p className="text-[11px] text-zinc-500">Minimum 8 characters.</p>
            </div>
            <div className="space-y-2">
              <Label className="text-zinc-300">Hire Date</Label>
              <Input type="date" value={hireDate} onChange={(e) => setHireDate(e.target.value)} className="bg-black/20 border-zinc-700" />
            </div>
            <div className="space-y-2">
              <Label className="text-zinc-300">Role</Label>
              <select
                className="w-full h-10 px-3 py-2 rounded-md bg-zinc-950/50 border border-zinc-700 text-white"
                value={role}
                onChange={(e) => setRole(e.target.value as any)}
              >
                {allowedRoles.map((r) => (
                  <option key={r} value={r}>{r.replace('_', ' ')}</option>
                ))}
              </select>
            </div>

            {(isAdmin || isManager) && (
              <div className="space-y-2">
                <Label className="text-zinc-300">Department</Label>
                <select
                  className="w-full h-10 px-3 py-2 rounded-md bg-zinc-950/50 border border-zinc-700 text-white"
                  value={departmentId}
                  onChange={(e) => setDepartmentId(e.target.value)}
                >
                  <option value="">(Optional)</option>
                  {departments?.map((d) => (
                    <option key={d.id} value={d.id}>{d.name} ({d.department_code})</option>
                  ))}
                </select>
              </div>
            )}

            <div className="space-y-2">
              <Label className="text-zinc-300">Default Shift</Label>
              <select
                className="w-full h-10 px-3 py-2 rounded-md bg-zinc-950/50 border border-zinc-700 text-white"
                value={defaultShiftId}
                onChange={(e) => setDefaultShiftId(e.target.value)}
              >
                <option value="">(Optional)</option>
                {shifts?.map((s) => (
                  <option key={s.id} value={s.id}>{s.name} ({s.shift_code})</option>
                ))}
              </select>
            </div>

            <div className="space-y-2">
              <Label className="text-zinc-300">Position</Label>
              <Input value={position} onChange={(e) => setPosition(e.target.value)} className="bg-black/20 border-zinc-700" />
            </div>

            <div className="space-y-2">
              <Label className="text-zinc-300">Weekly Off Days</Label>
              <Input
                type="number"
                min={0}
                value={weeklyOffDays}
                onChange={(e) => setWeeklyOffDays(parseInt(e.target.value) || 0)}
                className="bg-black/20 border-zinc-700"
              />
            </div>
          </div>

          <div className="flex items-center gap-2">
            <input
              id="night"
              type="checkbox"
              checked={canCoverNightShift}
              onChange={(e) => setCanCoverNightShift(e.target.checked)}
            />
            <Label htmlFor="night" className="text-zinc-300">Can cover night shift</Label>
          </div>
          <p className="text-xs text-zinc-500">Employee code is auto-generated.</p>
        </CardContent>}
        {showCreateForm && <CardFooter>
          <Button
            className="bg-emerald-600 hover:bg-emerald-500 text-white gap-2"
            onClick={() => createEmployee.mutate()}
            disabled={
              createEmployee.isPending ||
              !firstName || !lastName || !email || !password || !hireDate || !role
            }
          >
            <UserPlus className="w-4 h-4" />
            Create Account
          </Button>
        </CardFooter>}
      </Card>

      <Card className="bg-zinc-900/50 border-zinc-800/60">
        <CardHeader>
          <CardTitle className="text-white">Filters</CardTitle>
        </CardHeader>
        <CardContent className="grid md:grid-cols-4 gap-4">
          <div className="space-y-2">
            <Label className="text-zinc-300">Search</Label>
            <Input
              value={search}
              onChange={(e) => setSearch(e.target.value)}
              placeholder="Name, code, email..."
              className="bg-black/20 border-zinc-700"
            />
          </div>
          <div className="space-y-2">
            <Label className="text-zinc-300">Status</Label>
            <select
              className="w-full h-10 px-3 py-2 rounded-md bg-zinc-950/50 border border-zinc-700 text-white"
              value={statusFilter}
              onChange={(e) => setStatusFilter(e.target.value as any)}
            >
              <option value="all">All</option>
              <option value="active">Active</option>
              <option value="inactive">Inactive</option>
            </select>
          </div>
          <div className="space-y-2">
            <Label className="text-zinc-300">Department</Label>
            <select
              className="w-full h-10 px-3 py-2 rounded-md bg-zinc-950/50 border border-zinc-700 text-white"
              value={departmentFilter}
              onChange={(e) => setDepartmentFilter(e.target.value)}
            >
              <option value="all">All departments</option>
              {departments?.map((d) => (
                <option key={d.id} value={d.id}>{d.name}</option>
              ))}
            </select>
          </div>
          <div className="space-y-2">
            <Label className="text-zinc-300">Role</Label>
            <select
              className="w-full h-10 px-3 py-2 rounded-md bg-zinc-950/50 border border-zinc-700 text-white"
              value={roleFilter}
              onChange={(e) => setRoleFilter(e.target.value as any)}
            >
              <option value="all">All roles</option>
              <option value="employee">Employee</option>
              <option value="team_leader">Team Leader</option>
              <option value="manager">Manager</option>
              <option value="admin">Admin</option>
            </select>
          </div>
        </CardContent>
      </Card>

      {isLoading ? (
        <div className="grid gap-6 md:grid-cols-2 xl:grid-cols-3">
          {[1, 2, 3, 4, 5, 6].map((i) => (
            <Card key={i} className="animate-pulse bg-zinc-900/50">
              <CardHeader className="h-24 bg-zinc-800/50 rounded-t-xl" />
              <CardContent className="h-24" />
            </Card>
          ))}
        </div>
      ) : isError ? (
        <div className="p-4 rounded-xl bg-red-500/10 border border-red-500/20 text-red-400">
          Failed to load employees. Please try again.
        </div>
      ) : (
        <div className="grid gap-6 md:grid-cols-2 xl:grid-cols-3">
          {filteredEmployees?.map((emp) => (
            <Card key={emp.id} className="bg-zinc-900/40 hover:bg-zinc-800/60 transition-colors border-zinc-800/60 overflow-hidden relative">
              <div className={`absolute top-0 w-full h-1 ${emp.status === 'active' ? 'bg-green-500' : 'bg-red-500'}`} />
              <CardHeader className="flex flex-row items-center gap-4 pb-4">
                <div className="h-12 w-12 rounded-full bg-blue-500/10 flex items-center justify-center text-blue-400 flex-shrink-0 text-xl font-bold">
                  {emp.first_name?.[0]}{emp.last_name?.[0]}
                </div>
                <div className="space-y-1 overflow-hidden">
                  <CardTitle className="text-lg font-medium text-white truncate">
                    {emp.first_name} {emp.last_name}
                  </CardTitle>
                  <p className="text-xs font-mono text-zinc-500">{emp.employee_code}</p>
                </div>
              </CardHeader>
              <CardContent className="space-y-3">
                {editId === emp.id ? (
                  <div className="grid grid-cols-2 gap-2 text-sm">
                    <Input value={editFirstName} onChange={(e) => setEditFirstName(e.target.value)} className="bg-black/20 border-zinc-700" placeholder="First name" />
                    <Input value={editLastName} onChange={(e) => setEditLastName(e.target.value)} className="bg-black/20 border-zinc-700" placeholder="Last name" />
                    <select className="w-full h-10 px-3 py-2 rounded-md bg-zinc-950/50 border border-zinc-700 text-white" value={editGender} onChange={(e) => setEditGender(e.target.value as any)}>
                      <option value="male">Male</option>
                      <option value="female">Female</option>
                    </select>
                    <Input value={editPhone} onChange={(e) => setEditPhone(e.target.value)} className="bg-black/20 border-zinc-700" placeholder="Phone" />
                    <Input value={editEmail} onChange={(e) => setEditEmail(e.target.value)} className="bg-black/20 border-zinc-700 col-span-2" placeholder="Email" />
                    <select className="w-full h-10 px-3 py-2 rounded-md bg-zinc-950/50 border border-zinc-700 text-white" value={editRole} onChange={(e) => setEditRole(e.target.value as any)}>
                      {allowedRoles.map((r) => <option key={r} value={r}>{r.replace('_', ' ')}</option>)}
                    </select>
                    <select className="w-full h-10 px-3 py-2 rounded-md bg-zinc-950/50 border border-zinc-700 text-white" value={editStatus} onChange={(e) => setEditStatus(e.target.value)}>
                      <option value="active">active</option>
                      <option value="inactive">inactive</option>
                      <option value="on_leave">on_leave</option>
                      <option value="terminated">terminated</option>
                    </select>
                    <select className="w-full h-10 px-3 py-2 rounded-md bg-zinc-950/50 border border-zinc-700 text-white col-span-2" value={editDepartmentId} onChange={(e) => setEditDepartmentId(e.target.value)}>
                      <option value="">No department</option>
                      {departments?.map((d) => <option key={d.id} value={d.id}>{d.name}</option>)}
                    </select>
                    <select className="w-full h-10 px-3 py-2 rounded-md bg-zinc-950/50 border border-zinc-700 text-white col-span-2" value={editDefaultShiftId} onChange={(e) => setEditDefaultShiftId(e.target.value)}>
                      <option value="">No default shift</option>
                      {shifts?.map((s) => <option key={s.id} value={s.id}>{s.name} ({s.shift_code})</option>)}
                    </select>
                    <Input value={editPosition} onChange={(e) => setEditPosition(e.target.value)} className="bg-black/20 border-zinc-700" placeholder="Position" />
                    <Input type="number" min={0} value={editWeeklyOffDays} onChange={(e) => setEditWeeklyOffDays(parseInt(e.target.value) || 0)} className="bg-black/20 border-zinc-700" placeholder="Weekly off days" />
                    <div className="col-span-2 flex items-center gap-2">
                      <input id={`edit-night-${emp.id}`} type="checkbox" checked={editCanCoverNightShift} onChange={(e) => setEditCanCoverNightShift(e.target.checked)} />
                      <Label htmlFor={`edit-night-${emp.id}`} className="text-zinc-300">Can cover night shift</Label>
                    </div>
                  </div>
                ) : (
                  <div className="grid grid-cols-2 gap-2 text-sm">
                    <div className="flex items-center gap-2 text-zinc-400">
                      <Briefcase className="w-4 h-4 text-emerald-400/70" />
                      <span className="capitalize">{emp.role.replace('_', ' ')}</span>
                    </div>
                    <div className="flex items-center gap-2 text-zinc-400">
                      <Users className="w-4 h-4 text-purple-400/70" />
                      <span>
                        Dept: {emp.department_id && deptMap[emp.department_id]
                          ? deptMap[emp.department_id].name
                          : 'N/A'}
                      </span>
                    </div>
                    <div className="flex items-center gap-2 text-zinc-400 col-span-2">
                      <Briefcase className="w-4 h-4 text-blue-400/70 shrink-0" />
                      <span>
                        Shift: {emp.default_shift_id && shiftMap[emp.default_shift_id]
                          ? `${shiftMap[emp.default_shift_id].name} (${shiftMap[emp.default_shift_id].shift_code})`
                          : 'N/A'}
                      </span>
                    </div>
                    <div className="flex items-center gap-2 text-zinc-400 col-span-2">
                      <Mail className="w-4 h-4 text-blue-400/70 shrink-0" />
                      <span className="truncate">{emp.email || 'No email provided'}</span>
                    </div>
                    <div className="flex items-center gap-2 text-zinc-400 col-span-2">
                      <Phone className="w-4 h-4 text-orange-400/70 shrink-0" />
                      <span>{emp.phone || 'No phone provided'}</span>
                    </div>
                  </div>
                )}
                <div className="pt-2 border-t border-zinc-800/60 flex items-center justify-end gap-2">
                  {editId === emp.id ? (
                    <>
                      <Button
                        size="sm"
                        variant="outline"
                        className="border-zinc-700 text-zinc-300 hover:bg-zinc-800"
                        onClick={() => setEditId(null)}
                      >
                        <X className="w-4 h-4 mr-1" />
                        Cancel
                      </Button>
                      <Button
                        size="sm"
                        className="bg-emerald-600 hover:bg-emerald-500 text-white"
                        onClick={() => updateEmployee.mutate()}
                        disabled={updateEmployee.isPending || !editFirstName || !editLastName || !editEmail}
                      >
                        <Save className="w-4 h-4 mr-1" />
                        Save
                      </Button>
                    </>
                  ) : (
                    <Button
                      size="sm"
                      variant="outline"
                      className="border-zinc-700 text-zinc-300 hover:bg-zinc-800"
                      onClick={() => startEdit(emp)}
                    >
                      <Edit3 className="w-4 h-4 mr-1" />
                      Edit
                    </Button>
                  )}
                  {emp.status === 'active' ? (
                    <Button
                      size="sm"
                      variant="outline"
                      className="border-amber-500/30 text-amber-300 hover:bg-amber-500/10"
                      onClick={() => updateEmployeeStatus.mutate({ id: emp.id, status: 'inactive' })}
                    >
                      <UserX className="w-4 h-4 mr-1" />
                      Deactivate
                    </Button>
                  ) : (
                    <Button
                      size="sm"
                      variant="outline"
                      className="border-emerald-500/30 text-emerald-300 hover:bg-emerald-500/10"
                      onClick={() => updateEmployeeStatus.mutate({ id: emp.id, status: 'active' })}
                    >
                      <UserCheck className="w-4 h-4 mr-1" />
                      Activate
                    </Button>
                  )}
                  <Button
                    size="sm"
                    variant="outline"
                    className="border-red-500/30 text-red-300 hover:bg-red-500/10"
                    onClick={() => {
                      if (confirm(`Delete employee "${emp.first_name} ${emp.last_name}"?`)) {
                        deleteEmployee.mutate(emp.id);
                      }
                    }}
                  >
                    <Trash2 className="w-4 h-4 mr-1" />
                    Delete
                  </Button>
                </div>
              </CardContent>
            </Card>
          ))}
          {filteredEmployees?.length === 0 && (
            <div className="col-span-full p-8 text-center text-zinc-500 border border-zinc-800/60 border-dashed rounded-xl bg-zinc-900/20">
              No employees match current filters.
            </div>
          )}
        </div>
      )}
    </div>
  );
};
