import { useMemo, useState } from 'react';
import { useParams, Link } from 'react-router-dom';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import api from '@/lib/api';
import { useAuthStore } from '@/store/authStore';
import { Card, CardContent, CardHeader, CardTitle, CardDescription, CardFooter } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Key, ArrowLeft, Mail, Phone, Briefcase, Building2, CalendarDays, Users, Edit3, Save, X, PhoneCall, AtSign } from 'lucide-react';
import { ChangePasswordModal } from '@/features/auth/ChangePasswordModal';
import { EmployeeLeaveBalances } from './EmployeeLeaveBalances';

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
  can_post_announcements: boolean;
  status: string;
  last_login: string | null;
  secondary_phone: string | null;
  secondary_email: string | null;
  profile_image?: string | null;
  created_at: string;
  updated_at: string;
};

type Department = { id: string; department_code: string; name: string; manager_id: string | null };
type Shift = { id: string; name: string; shift_code: string };

export const EmployeeDetail = () => {
  const { id } = useParams();
  const { user } = useAuthStore();
  const queryClient = useQueryClient();
  const [isEditing, setIsEditing] = useState(false);
  const [showResetPassword, setShowResetPassword] = useState(false);
  const [editData, setEditData] = useState<Partial<Employee>>({});
  const [error, setError] = useState<string | null>(null);

  const canEdit = user?.role === 'team_leader' || user?.role === 'manager' || user?.role === 'admin';

  const { data: employee, isLoading } = useQuery<Employee>({
    queryKey: ['employee', id],
    queryFn: async () => { const res = await api.get(`/employees/${id}`); return res.data?.data; },
    enabled: !!id,
  });

  const { data: departments } = useQuery<Department[]>({
    queryKey: ['departments'],
    queryFn: async () => { const res = await api.get('/departments'); return res.data?.data || []; },
  });

  const { data: shifts } = useQuery<Shift[]>({
    queryKey: ['shifts'],
    queryFn: async () => { const res = await api.get('/shifts'); return res.data?.data || []; },
  });

  const updateMutation = useMutation({
    mutationFn: async (data: any) => {
      setError(null);
      await api.put(`/employees/${id}`, data);
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['employee', id] });
      queryClient.invalidateQueries({ queryKey: ['employees'] });
      setIsEditing(false);
    },
    onError: (err: any) => setError(err?.response?.data?.error || err?.message || 'Failed to update'),
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

  const startEdit = () => {
    if (!employee) return;
    setEditData({
      first_name: employee.first_name,
      last_name: employee.last_name,
      gender: employee.gender,
      phone: employee.phone,
      email: employee.email,
      role: employee.role,
      department_id: employee.department_id,
      position: employee.position,
      default_shift_id: employee.default_shift_id,
      weekly_off_days: employee.weekly_off_days,
      can_cover_night_shift: employee.can_cover_night_shift,
      can_post_announcements: employee.can_post_announcements,
      status: employee.status,
      secondary_phone: employee.secondary_phone,
      secondary_email: employee.secondary_email,
    });
    setIsEditing(true);
    setError(null);
  };

  const handleSave = () => {
    updateMutation.mutate(editData);
  };

  const selectClass = "w-full h-10 px-3 py-2 rounded-lg bg-background border border-input text-foreground focus:outline-none focus:ring-2 focus:ring-ring transition-colors text-sm";

  if (isLoading) return <Card className="animate-pulse h-44" />;
  if (!employee) return (
    <div className="space-y-4">
      <p className="text-muted-foreground">Employee not found.</p>
      <Link to="/employees"><Button variant="outline">Back</Button></Link>
    </div>
  );

  const dept = employee.department_id ? deptMap[employee.department_id] : undefined;
  const shift = employee.default_shift_id ? shiftMap[employee.default_shift_id] : undefined;

  return (
    <div className="space-y-6">
      <div className="flex flex-col sm:flex-row items-start sm:items-center justify-between gap-3">
        <div className="flex items-center gap-2 sm:gap-3">
          <Link to="/employees">
            <Button variant="outline" size="sm" className="gap-2">
              <ArrowLeft className="w-4 h-4" /> Back
            </Button>
          </Link>
          <div>
            <h2 className="text-2xl sm:text-3xl font-bold tracking-tight text-foreground flex items-center gap-2 sm:gap-3">
              {employee.profile_image ? (
                <img 
                  src={`${import.meta.env.VITE_API_URL ? import.meta.env.VITE_API_URL.replace('/api', '') : (import.meta.env.DEV ? 'http://localhost:8080' : '')}${employee.profile_image.startsWith('/api') ? employee.profile_image : '/api' + employee.profile_image}`}
                  alt="Profile"
                  className="w-8 h-8 sm:w-12 sm:h-12 rounded-full object-cover border border-border"
                />
              ) : (
                <Users className="w-6 h-6 sm:w-8 sm:h-8 text-primary" />
              )}
              {employee.first_name} {employee.last_name}
            </h2>
            <p className="text-sm text-muted-foreground">{employee.employee_code} · {employee.role.replace('_', ' ')}</p>
          </div>
        </div>

        {user?.role === 'admin' && !isEditing && (
          <Button onClick={() => setShowResetPassword(true)} variant="outline" className="gap-2 text-destructive border-destructive/20 hover:bg-destructive/10">
            <Key className="w-4 h-4" /> Reset Password
          </Button>
        )}
        {canEdit && !isEditing && (
          <Button onClick={startEdit} className="gap-2">
            <Edit3 className="w-4 h-4" /> Edit Employee
          </Button>
        )}
      </div>

      <ChangePasswordModal
        isOpen={showResetPassword}
        onClose={() => setShowResetPassword(false)}
        employeeId={id || ''}
        requireOldPassword={false}
      />

      {error && (
        <div className="p-3 rounded-xl bg-destructive/10 border border-destructive/20 text-destructive text-sm">{error}</div>
      )}

      {/* ── View Mode ── */}
      {!isEditing && (
        <Card>
          <CardHeader>
            <CardTitle>Employee Details</CardTitle>
            <CardDescription>Profile information</CardDescription>
          </CardHeader>
          <CardContent className="grid sm:grid-cols-2 gap-4 text-sm">
            <DetailBlock icon={<Mail className="w-4 h-4" />} label="Email" value={employee.email} />
            <DetailBlock icon={<AtSign className="w-4 h-4" />} label="Secondary Email" value={employee.secondary_email || '—'} />
            <DetailBlock icon={<Phone className="w-4 h-4" />} label="Phone" value={employee.phone || '—'} />
            <DetailBlock icon={<PhoneCall className="w-4 h-4" />} label="Secondary Phone" value={employee.secondary_phone || '—'} />
            <DetailBlock icon={<Building2 className="w-4 h-4" />} label="Department" value={dept ? `${dept.name} (${dept.department_code})` : '—'} />
            <DetailBlock icon={<Briefcase className="w-4 h-4" />} label="Position" value={employee.position || '—'} />
            <DetailBlock icon={<CalendarDays className="w-4 h-4" />} label="Hire Date" value={employee.hire_date?.split('T')[0]} />
            <DetailBlock icon={<CalendarDays className="w-4 h-4" />} label="Shift" value={shift ? `${shift.name} (${shift.shift_code})` : '—'} />
            <DetailBlock label="Weekly Off Days" value={employee.weekly_off_days === -1 ? 'None' : ['Sunday', 'Monday', 'Tuesday', 'Wednesday', 'Thursday', 'Friday', 'Saturday'][employee.weekly_off_days] || String(employee.weekly_off_days)} />
            <DetailBlock label="Night Shift Coverage" value={employee.can_cover_night_shift ? 'Yes' : 'No'} />
            <DetailBlock label="Status" value={employee.status} />
            <DetailBlock label="Last Login" value={employee.last_login?.split('T')[0] || '—'} />
          </CardContent>
        </Card>
      )}

      {/* ── Admin / Manager Leave Balances View ── */}
      {!isEditing && canEdit && id && (
        <EmployeeLeaveBalances employeeId={id} />
      )}

      {/* ── Edit Mode ── */}
      {isEditing && (
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <Edit3 className="w-5 h-5 text-primary" /> Edit Employee
            </CardTitle>
            <CardDescription>Update employee information</CardDescription>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="grid sm:grid-cols-2 gap-4">
              <div className="space-y-2">
                <Label>First Name *</Label>
                <Input value={editData.first_name || ''} onChange={(e: React.ChangeEvent<HTMLInputElement>) => setEditData({...editData, first_name: e.target.value})} />
              </div>
              <div className="space-y-2">
                <Label>Last Name *</Label>
                <Input value={editData.last_name || ''} onChange={(e: React.ChangeEvent<HTMLInputElement>) => setEditData({...editData, last_name: e.target.value})} />
              </div>
            </div>

            <div className="grid sm:grid-cols-2 gap-4">
              <div className="space-y-2">
                <Label>Email *</Label>
                <Input type="email" value={editData.email || ''} onChange={(e: React.ChangeEvent<HTMLInputElement>) => setEditData({...editData, email: e.target.value})} />
              </div>
              <div className="space-y-2">
                <Label>Secondary Email <span className="text-muted-foreground text-xs">(optional)</span></Label>
                <Input type="email" value={editData.secondary_email || ''} onChange={(e: React.ChangeEvent<HTMLInputElement>) => setEditData({...editData, secondary_email: e.target.value || null})} placeholder="Secondary email" />
              </div>
            </div>

            <div className="grid sm:grid-cols-2 gap-4">
              <div className="space-y-2">
                <Label>Phone</Label>
                <Input value={editData.phone || ''} onChange={(e: React.ChangeEvent<HTMLInputElement>) => setEditData({...editData, phone: e.target.value || null})} placeholder="Primary phone" />
              </div>
              <div className="space-y-2">
                <Label>Secondary Phone <span className="text-muted-foreground text-xs">(optional)</span></Label>
                <Input value={editData.secondary_phone || ''} onChange={(e: React.ChangeEvent<HTMLInputElement>) => setEditData({...editData, secondary_phone: e.target.value || null})} placeholder="Secondary phone" />
              </div>
            </div>

            <div className="grid sm:grid-cols-3 gap-4">
              <div className="space-y-2">
                <Label>Gender</Label>
                <select className={selectClass} value={editData.gender || ''} onChange={(e) => setEditData({...editData, gender: e.target.value})}>
                  <option value="male">Male</option>
                  <option value="female">Female</option>
                </select>
              </div>
              <div className="space-y-2">
                <Label>Position</Label>
                <Input value={editData.position || ''} onChange={(e: React.ChangeEvent<HTMLInputElement>) => setEditData({...editData, position: e.target.value || null})} />
              </div>
              <div className="space-y-2">
                <Label>Status</Label>
                <select className={selectClass} value={editData.status || ''} onChange={(e) => setEditData({...editData, status: e.target.value})}>
                  <option value="active">Active</option>
                  <option value="inactive">Inactive</option>
                  <option value="suspended">Suspended</option>
                </select>
              </div>
            </div>

            <div className="grid sm:grid-cols-2 gap-4">
              <div className="space-y-2">
                <Label>Department</Label>
                <select className={selectClass} value={editData.department_id || ''} onChange={(e) => setEditData({...editData, department_id: e.target.value || null})}>
                  <option value="">No Department</option>
                  {departments?.map((d) => <option key={d.id} value={d.id}>{d.name} ({d.department_code})</option>)}
                </select>
              </div>
              <div className="space-y-2">
                <Label>Default Shift</Label>
                <select className={selectClass} value={editData.default_shift_id || ''} onChange={(e) => setEditData({...editData, default_shift_id: e.target.value || null})}>
                  <option value="">No Shift</option>
                  {shifts?.map((s) => <option key={s.id} value={s.id}>{s.name} ({s.shift_code})</option>)}
                </select>
              </div>
            </div>

            <div className="grid sm:grid-cols-2 gap-4">
              <div className="space-y-2">
                <Label>Weekly Off Days</Label>
                <select className={selectClass} value={editData.weekly_off_days ?? -1} onChange={(e) => setEditData({...editData, weekly_off_days: parseInt(e.target.value)})}>
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
              <div className="flex flex-col justify-end gap-3 pt-6">
                <div className="flex items-center gap-3">
                  <input type="checkbox" id="nightShift" checked={editData.can_cover_night_shift || false}
                    onChange={(e) => setEditData({...editData, can_cover_night_shift: e.target.checked})}
                    className="w-4 h-4 rounded border-input" />
                  <Label htmlFor="nightShift">Can Cover Night Shift</Label>
                </div>
                <div className="flex items-center gap-3">
                  <input type="checkbox" id="postAnnouncements" checked={editData.can_post_announcements || false}
                    onChange={(e) => setEditData({...editData, can_post_announcements: e.target.checked})}
                    className="w-4 h-4 rounded border-input" />
                  <Label htmlFor="postAnnouncements">Can Post Announcements</Label>
                </div>
              </div>
            </div>

            {user?.role === 'admin' && (
              <div className="space-y-2">
                <Label>Role</Label>
                <select className={selectClass} value={editData.role || ''} onChange={(e) => setEditData({...editData, role: e.target.value})}>
                  <option value="employee">Employee</option>
                  <option value="team_leader">Team Leader</option>
                  <option value="manager">Manager</option>
                  <option value="admin">Admin</option>
                </select>
              </div>
            )}
          </CardContent>
          <CardFooter className="flex justify-end gap-3">
            <Button variant="outline" onClick={() => setIsEditing(false)} className="gap-2">
              <X className="w-4 h-4" /> Cancel
            </Button>
            <Button onClick={handleSave} disabled={updateMutation.isPending} className="gap-2">
              <Save className="w-4 h-4" /> {updateMutation.isPending ? 'Saving...' : 'Save Changes'}
            </Button>
          </CardFooter>
        </Card>
      )}
    </div>
  );
};

const DetailBlock = ({ icon, label, value }: { icon?: React.ReactNode; label: string; value: string }) => (
  <div className="p-4 rounded-xl bg-muted/30 border border-border">
    <div className="flex items-center gap-2 text-muted-foreground mb-2 text-xs uppercase tracking-wider">
      {icon}
      {label}
    </div>
    <div className="text-foreground font-medium">{value}</div>
  </div>
);
