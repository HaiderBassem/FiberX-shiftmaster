import { useMemo, useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { Link } from 'react-router-dom';
import api from '@/lib/api';
import { useAuthStore } from '@/store/authStore';
import { Card, CardContent, CardDescription, CardHeader, CardTitle, CardFooter } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Users, Plus, Search, Loader2, X, Edit3, Trash2, Save, UserCircle, Key } from 'lucide-react';
import { ChangePasswordModal } from '@/features/auth/ChangePasswordModal';

interface Employee {
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
  can_manage_help_docs: boolean;
  status: string;
  secondary_phone: string | null;
  secondary_email: string | null;
  created_at: string;
}

interface Department {
  id: string;
  department_code: string;
  name: string;
}

interface Shift {
  id: string;
  shift_code: string;
  name: string;
}

export const EmployeeList = () => {
  const queryClient = useQueryClient();
  const { user } = useAuthStore();
  const canCreate = user?.role === 'admin' || user?.role === 'manager' || user?.role === 'team_leader';

  // Role options based on current user's role
  const availableRoles = user?.role === 'admin'
    ? ['employee', 'team_leader', 'manager', 'admin']
    : user?.role === 'manager'
    ? ['employee', 'team_leader']
    : ['employee'];

  // ── State ──
  const [searchQuery, setSearchQuery] = useState('');
  const [filterRole, setFilterRole] = useState<string>('');

  const [filterShift, setFilterShift] = useState<string>('');
  const [showCreate, setShowCreate] = useState(false);
  const [editId, setEditId] = useState<string | null>(null);
  const [resetPasswordEmpId, setResetPasswordEmpId] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);

  // Create form
  const [createCode, setCreateCode] = useState('');
  const [createFirst, setCreateFirst] = useState('');
  const [createLast, setCreateLast] = useState('');
  const [createEmail, setCreateEmail] = useState('');
  const [createPhone, setCreatePhone] = useState('');
  const [createGender, setCreateGender] = useState('male');
  const [createHireDate, setCreateHireDate] = useState('');
  const [createRole, setCreateRole] = useState('employee');
  const [createDept, setCreateDept] = useState('');
  const [createShift, setCreateShift] = useState('');
  const [createPosition, setCreatePosition] = useState('');
  const [createOffDays, setCreateOffDays] = useState(1);
  const [createNight, setCreateNight] = useState(false);
  const [createCanManageHelpDocs, setCreateCanManageHelpDocs] = useState(false);
  const [createPassword, setCreatePassword] = useState('');
  const [createSecPhone, setCreateSecPhone] = useState('');
  const [createSecEmail, setCreateSecEmail] = useState('');

  // Edit form
  const [editCode, setEditCode] = useState('');
  const [editFirst, setEditFirst] = useState('');
  const [editLast, setEditLast] = useState('');
  const [editEmail, setEditEmail] = useState('');
  const [editPhone, setEditPhone] = useState('');
  const [editGender, setEditGender] = useState('male');
  const [editRole, setEditRole] = useState('employee');
  const [editDept, setEditDept] = useState('');
  const [editShift, setEditShift] = useState('');
  const [editPosition, setEditPosition] = useState('');
  const [editOffDays, setEditOffDays] = useState(1);
  const [editNight, setEditNight] = useState(false);
  const [editCanManageHelpDocs, setEditCanManageHelpDocs] = useState(false);
  const [editSecPhone, setEditSecPhone] = useState('');
  const [editSecEmail, setEditSecEmail] = useState('');

  // ── Queries ──
  const { data: employees, isLoading } = useQuery<Employee[]>({
    queryKey: ['employees'],
    queryFn: async () => {
      const res = await api.get('/employees');
      return res.data?.data || [];
    },
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

  // ── Mutations ──
  const createEmployee = useMutation({
    mutationFn: async () => {
      setError(null);
      await api.post('/employees', {
        employee_code: createCode,
        first_name: createFirst,
        last_name: createLast,
        email: createEmail,
        phone: createPhone || null,
        gender: createGender,
        hire_date: createHireDate,
        role: createRole,
        department_id: createDept || null,
        default_shift_id: createShift || null,
        position: createPosition || null,
        weekly_off_days: createOffDays,
        can_cover_night_shift: createNight,
        can_manage_help_docs: createCanManageHelpDocs,
        password: createPassword,
        secondary_phone: createSecPhone || null,
        secondary_email: createSecEmail || null,
      });
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['employees'] });
      setShowCreate(false);
      setCreateCode(''); setCreateFirst(''); setCreateLast('');
      setCreateEmail(''); setCreatePhone(''); setCreatePassword('');
      setCreatePosition(''); setCreateDept(''); setCreateShift('');
      setCreateSecPhone(''); setCreateSecEmail('');
    },
    onError: (err: any) => {
      setError(err?.response?.data?.error || err?.message || 'Failed to create employee');
    },
  });

  const updateEmployee = useMutation({
    mutationFn: async () => {
      if (!editId) return;
      setError(null);
      await api.put(`/employees/${editId}`, {
        employee_code: editCode,
        first_name: editFirst,
        last_name: editLast,
        email: editEmail,
        phone: editPhone || null,
        gender: editGender,
        role: editRole,
        department_id: editDept || null,
        default_shift_id: editShift || null,
        position: editPosition || null,
        weekly_off_days: editOffDays,
        can_cover_night_shift: editNight,
        can_manage_help_docs: editCanManageHelpDocs,
        secondary_phone: editSecPhone || null,
        secondary_email: editSecEmail || null,
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
      setError(null);
      await api.delete(`/employees/${id}`);
    },
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ['employees'] }),
    onError: (err: any) => setError(err?.response?.data?.error || err?.message || 'Failed to delete employee'),
  });

  const startEdit = (emp: Employee) => {
    setEditId(emp.id);
    setEditCode(emp.employee_code);
    setEditFirst(emp.first_name);
    setEditLast(emp.last_name);
    setEditEmail(emp.email);
    setEditPhone(emp.phone || '');
    setEditGender(emp.gender);
    setEditRole(emp.role);
    setEditDept(emp.department_id || '');
    setEditShift(emp.default_shift_id || '');
    setEditPosition(emp.position || '');
    setEditOffDays(emp.weekly_off_days);
    setEditNight(emp.can_cover_night_shift);
    setEditCanManageHelpDocs(emp.can_manage_help_docs);
    setEditSecPhone(emp.secondary_phone || '');
    setEditSecEmail(emp.secondary_email || '');
  };

  // ── Filter ──
  const filteredEmployees = useMemo(() => {
    let list = employees || [];
    const q = searchQuery.trim().toLowerCase();
    if (q) {
      list = list.filter((e) =>
        `${e.first_name} ${e.last_name}`.toLowerCase().includes(q) ||
        e.employee_code.toLowerCase().includes(q) ||
        e.email.toLowerCase().includes(q)
      );
    }
    if (filterRole) list = list.filter((e) => e.role === filterRole);
    if (filterShift) list = list.filter((e) => e.default_shift_id === filterShift);
    return list;
  }, [employees, searchQuery, filterRole, filterShift]);

  // Helper for select styling
  const selectClass = "w-full h-10 px-3 py-2 rounded-lg bg-background border border-input text-foreground focus:outline-none focus:ring-2 focus:ring-ring transition-colors";

  return (
    <div className="space-y-6">
      <div className="flex flex-col sm:flex-row justify-between items-start sm:items-end gap-3">
        <div>
          <h2 className="text-2xl sm:text-3xl font-bold tracking-tight text-foreground flex items-center gap-2 sm:gap-3">
            <Users className="w-6 h-6 sm:w-8 sm:h-8 text-primary" />
            Employees
          </h2>
          <p className="text-sm sm:text-base text-muted-foreground">Manage employee records, roles, and shifts.</p>
        </div>
        {canCreate && (
          <Button
            onClick={() => { setShowCreate((v) => !v); setError(null); }}
            className="gap-2 w-full sm:w-auto"
          >
            {showCreate ? <X className="w-4 h-4" /> : <Plus className="w-4 h-4" />}
            {showCreate ? 'Close' : 'New Employee'}
          </Button>
        )}
      </div>

      {error && (
        <div className="p-3 rounded-xl bg-destructive/10 border border-destructive/20 text-destructive text-sm">
          {error}
        </div>
      )}

      {/* Filters */}
      <Card>
        <CardContent className="pt-4 sm:pt-6">
          <div className="grid grid-cols-1 sm:grid-cols-2 md:grid-cols-4 gap-3 sm:gap-4">
            <div className="space-y-2 md:col-span-2">
              <Label>Search</Label>
              <div className="relative">
                <Search className="w-4 h-4 text-muted-foreground absolute left-3 top-3" />
                <Input
                  value={searchQuery}
                  onChange={(e) => setSearchQuery(e.target.value)}
                  placeholder="Search by name, code, or email..."
                  className="pl-9"
                />
              </div>
            </div>
            <div className="space-y-2">
              <Label>Role</Label>
              <select className={selectClass} value={filterRole} onChange={(e) => setFilterRole(e.target.value)}>
                <option value="">All Roles</option>
                <option value="employee">Employee</option>
                <option value="team_leader">Team Leader</option>
                <option value="manager">Manager</option>
                <option value="admin">Admin</option>
              </select>
            </div>
            <div className="space-y-2">
              <Label>Shift</Label>
              <select className={selectClass} value={filterShift} onChange={(e) => setFilterShift(e.target.value)}>
                <option value="">All Shifts</option>
                {shifts?.map((s) => <option key={s.id} value={s.id}>{s.name}</option>)}
              </select>
            </div>
          </div>
        </CardContent>
      </Card>

      {/* Create Form */}
      {canCreate && showCreate && (
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <Plus className="w-5 h-5 text-primary" />
              Create Employee
            </CardTitle>
            <CardDescription>Add a new employee to the system</CardDescription>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="grid grid-cols-1 sm:grid-cols-2 md:grid-cols-3 gap-3 sm:gap-4">
              <div className="space-y-2">
                <Label>Employee Code</Label>
                <Input value={createCode} onChange={(e) => setCreateCode(e.target.value)} />
              </div>
              <div className="space-y-2">
                <Label>First Name</Label>
                <Input value={createFirst} onChange={(e) => setCreateFirst(e.target.value)} />
              </div>
              <div className="space-y-2">
                <Label>Last Name</Label>
                <Input value={createLast} onChange={(e) => setCreateLast(e.target.value)} />
              </div>
            </div>

            {/* Contact — each primary + secondary side by side */}
            <div className="grid grid-cols-1 sm:grid-cols-2 gap-3 sm:gap-4">
              <div className="space-y-2">
                <Label>Email *</Label>
                <Input type="email" value={createEmail} onChange={(e) => setCreateEmail(e.target.value)} placeholder="Primary email" />
              </div>
              <div className="space-y-2">
                <Label>Secondary Email <span className="text-muted-foreground text-xs">(optional)</span></Label>
                <Input type="email" value={createSecEmail} onChange={(e) => setCreateSecEmail(e.target.value)} placeholder="Second email" />
              </div>
              <div className="space-y-2">
                <Label>Phone</Label>
                <Input value={createPhone} onChange={(e) => setCreatePhone(e.target.value)} placeholder="Primary phone" />
              </div>
              <div className="space-y-2">
                <Label>Secondary Phone <span className="text-muted-foreground text-xs">(optional)</span></Label>
                <Input value={createSecPhone} onChange={(e) => setCreateSecPhone(e.target.value)} placeholder="Second phone" />
              </div>
            </div>

            <div className="grid grid-cols-1 sm:grid-cols-2 md:grid-cols-3 gap-3 sm:gap-4">
              <div className="space-y-2">
                <Label>Gender</Label>
                <select className={selectClass} value={createGender} onChange={(e) => setCreateGender(e.target.value)}>
                  <option value="male">Male</option>
                  <option value="female">Female</option>
                </select>
              </div>
              <div className="space-y-2">
                <Label>Hire Date</Label>
                <Input type="date" value={createHireDate} onChange={(e) => setCreateHireDate(e.target.value)} />
              </div>
              <div className="space-y-2">
                <Label>Role</Label>
                <select className={selectClass} value={createRole} onChange={(e) => setCreateRole(e.target.value)}>
                  {availableRoles.map((r) => (
                    <option key={r} value={r}>{r.replace('_', ' ').replace(/\b\w/g, c => c.toUpperCase())}</option>
                  ))}
                </select>
              </div>
              <div className="space-y-2">
                <Label>Department</Label>
                <select className={selectClass} value={createDept} onChange={(e) => setCreateDept(e.target.value)}>
                  <option value="">No Department</option>
                  {departments?.map((d) => <option key={d.id} value={d.id}>{d.name}</option>)}
                </select>
              </div>
              <div className="space-y-2">
                <Label>Default Shift</Label>
                <select className={selectClass} value={createShift} onChange={(e) => setCreateShift(e.target.value)}>
                  <option value="">No Shift</option>
                  {shifts?.map((s) => <option key={s.id} value={s.id}>{s.name} ({s.shift_code})</option>)}
                </select>
              </div>
              <div className="space-y-2">
                <Label>Position</Label>
                <Input value={createPosition} onChange={(e) => setCreatePosition(e.target.value)} />
              </div>
              <div className="space-y-2">
                <Label>Password</Label>
                <Input type="password" value={createPassword} onChange={(e) => setCreatePassword(e.target.value)} />
              </div>
              <div className="space-y-2">
                <Label>Weekly Off Days</Label>
                <select className={selectClass} value={createOffDays} onChange={(e) => setCreateOffDays(parseInt(e.target.value))}>
                  <option value="-1">None</option>
                  <option value="0">Sunday</option>
                  <option value="1">Monday</option>
                  <option value="2">Tuesday</option>
                  <option value="3">Wednesday</option>
                  <option value="4">Thursday</option>
                  <option value="5">Friday</option>
                  <option value="6">Saturday</option>
                </select>
              </div>
              <div className="flex items-center gap-2 pt-6">
                <input id="create-night" type="checkbox" checked={createNight} onChange={(e) => setCreateNight(e.target.checked)} />
                <Label htmlFor="create-night">Can cover night shifts</Label>
              </div>
              <div className="flex items-center gap-2 pt-6">
                <input id="create-help" type="checkbox" checked={createCanManageHelpDocs} onChange={(e) => setCreateCanManageHelpDocs(e.target.checked)} />
                <Label htmlFor="create-help">Can manage Help Docs</Label>
              </div>
            </div>
          </CardContent>
          <CardFooter>
            <Button
              onClick={() => createEmployee.mutate()}
              disabled={createEmployee.isPending || !createCode || !createFirst || !createLast || !createEmail || !createPassword}
              className="gap-2"
            >
              <Plus className="w-4 h-4" />
              {createEmployee.isPending ? 'Creating…' : 'Create'}
            </Button>
          </CardFooter>
        </Card>
      )}

      {/* Employee List */}
      {isLoading ? (
        <div className="flex justify-center py-16">
          <Loader2 className="w-8 h-8 animate-spin text-muted-foreground" />
        </div>
      ) : (
        <div className="grid gap-3 sm:gap-4 grid-cols-1 sm:grid-cols-2 lg:grid-cols-3">
          {filteredEmployees.map((emp) => (
            <Card key={emp.id} className="group hover:shadow-md transition-all">
              {editId === emp.id ? (
                /* Edit mode */
                <div className="p-5 space-y-3">
                  <div className="grid grid-cols-2 gap-3">
                    <div className="space-y-1"><Label className="text-xs">Code</Label><Input value={editCode} onChange={(e) => setEditCode(e.target.value)} className="h-9" /></div>
                    <div className="space-y-1"><Label className="text-xs">Role</Label>
                      <select className={selectClass + " h-9"} value={editRole} onChange={(e) => setEditRole(e.target.value)}>
                        <option value="employee">Employee</option><option value="team_leader">Team Leader</option><option value="manager">Manager</option><option value="admin">Admin</option>
                      </select>
                    </div>
                    <div className="space-y-1"><Label className="text-xs">First</Label><Input value={editFirst} onChange={(e) => setEditFirst(e.target.value)} className="h-9" /></div>
                    <div className="space-y-1"><Label className="text-xs">Last</Label><Input value={editLast} onChange={(e) => setEditLast(e.target.value)} className="h-9" /></div>
                    <div className="space-y-1"><Label className="text-xs">Email</Label><Input value={editEmail} onChange={(e) => setEditEmail(e.target.value)} className="h-9" /></div>
                    <div className="space-y-1"><Label className="text-xs">Sec. Email <span className="text-muted-foreground">(opt)</span></Label><Input value={editSecEmail} onChange={(e) => setEditSecEmail(e.target.value)} className="h-9" placeholder="Secondary email" /></div>
                    <div className="space-y-1"><Label className="text-xs">Phone</Label><Input value={editPhone} onChange={(e) => setEditPhone(e.target.value)} className="h-9" /></div>
                    <div className="space-y-1"><Label className="text-xs">Sec. Phone <span className="text-muted-foreground">(opt)</span></Label><Input value={editSecPhone} onChange={(e) => setEditSecPhone(e.target.value)} className="h-9" placeholder="Secondary phone" /></div>
                    <div className="space-y-1 flex items-center gap-2 pt-6"><input id="edit-help" type="checkbox" checked={editCanManageHelpDocs} onChange={(e) => setEditCanManageHelpDocs(e.target.checked)} /><Label htmlFor="edit-help" className="text-xs">Can manage Help Docs</Label></div>
                    <div className="space-y-1"><Label className="text-xs">Department</Label>
                      <select className={selectClass + " h-9"} value={editDept} onChange={(e) => setEditDept(e.target.value)}>
                        <option value="">None</option>{departments?.map((d) => <option key={d.id} value={d.id}>{d.name}</option>)}
                      </select>
                    </div>
                    <div className="space-y-1"><Label className="text-xs">Shift</Label>
                      <select className={selectClass + " h-9"} value={editShift} onChange={(e) => setEditShift(e.target.value)}>
                        <option value="">None</option>{shifts?.map((s) => <option key={s.id} value={s.id}>{s.name}</option>)}
                      </select>
                    </div>
                  </div>
                  <div className="flex justify-end gap-2 pt-2">
                    <Button size="sm" variant="outline" onClick={() => setEditId(null)}><X className="w-4 h-4 mr-1" /> Cancel</Button>
                    <Button size="sm" onClick={() => updateEmployee.mutate()} disabled={updateEmployee.isPending}><Save className="w-4 h-4 mr-1" /> Save</Button>
                  </div>
                </div>
              ) : (
                /* View mode */
                <Link to={`/employees/${emp.id}`} className="block">
                  <CardHeader className="pb-2 flex flex-row items-center justify-between">
                    <div className="flex items-center gap-3 min-w-0">
                      <div className="w-10 h-10 rounded-xl bg-primary/10 flex items-center justify-center shrink-0">
                        <UserCircle className="w-5 h-5 text-primary" />
                      </div>
                      <div className="min-w-0">
                        <CardTitle className="text-base truncate">{emp.first_name} {emp.last_name}</CardTitle>
                        <CardDescription className="text-xs font-mono">{emp.employee_code}</CardDescription>
                      </div>
                    </div>
                    <span className={`text-[10px] font-bold px-2 py-1 rounded-full uppercase tracking-wider ${
                      emp.role === 'admin' ? 'bg-destructive/10 text-destructive border border-destructive/20' :
                      emp.role === 'manager' ? 'bg-amber-500/10 text-amber-500 border border-amber-500/20' :
                      emp.role === 'team_leader' ? 'bg-primary/10 text-primary border border-primary/20' :
                      'bg-muted text-muted-foreground border border-border'
                    }`}>
                      {emp.role.replace('_', ' ')}
                    </span>
                  </CardHeader>
                  <CardContent className="text-sm space-y-1.5 text-muted-foreground">
                    <p>{emp.email}</p>
                    {emp.department_id && deptMap[emp.department_id] && (
                      <p className="text-xs">Dept: <span className="text-foreground font-medium">{deptMap[emp.department_id].name}</span></p>
                    )}
                    {emp.default_shift_id && shiftMap[emp.default_shift_id] && (
                      <p className="text-xs">Shift: <span className="text-foreground font-medium">{shiftMap[emp.default_shift_id].name}</span></p>
                    )}
                  </CardContent>
                  {canCreate && (
                    <CardFooter className="pt-0 flex justify-end gap-2" onClick={(e) => e.preventDefault()}>
                      {user?.role === 'admin' && (
                        <Button size="sm" variant="outline" className="text-destructive border-destructive/30 hover:bg-destructive/10"
                          onClick={() => setResetPasswordEmpId(emp.id)}
                        >
                          <Key className="w-4 h-4 mr-1" /> Reset Pwd
                        </Button>
                      )}
                      <Button size="sm" variant="outline" onClick={() => startEdit(emp)}>
                        <Edit3 className="w-4 h-4 mr-1" /> Edit
                      </Button>
                      <Button size="sm" variant="outline" className="text-destructive border-destructive/30 hover:bg-destructive/10"
                        onClick={() => { if (confirm(`Delete ${emp.first_name} ${emp.last_name}?`)) deleteEmployee.mutate(emp.id); }}
                      >
                        <Trash2 className="w-4 h-4 mr-1" /> Delete
                      </Button>
                    </CardFooter>
                  )}
                </Link>
              )}
            </Card>
          ))}
          {filteredEmployees.length === 0 && (
            <div className="col-span-full p-12 text-center text-muted-foreground border border-border border-dashed rounded-xl bg-muted/10">
              <Users className="w-12 h-12 mx-auto mb-4 opacity-20" />
              No employees match current filters.
            </div>
          )}
        </div>
      )}

      {resetPasswordEmpId && (
        <ChangePasswordModal
          isOpen={!!resetPasswordEmpId}
          onClose={() => setResetPasswordEmpId(null)}
          employeeId={resetPasswordEmpId}
          requireOldPassword={false}
        />
      )}
    </div>
  );
};
