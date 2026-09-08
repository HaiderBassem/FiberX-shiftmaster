import { useEffect, useMemo, useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import api, { apiError } from '@/lib/api';
import { useAuthStore } from '@/store/authStore';
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import {
  Calendar as CalendarIcon, Loader2, Users, Briefcase, Wand2, Filter, AlertTriangle,
  ChevronLeft, ChevronRight, Repeat, Pin, X,
} from 'lucide-react';
import { addDays, addWeeks, format, startOfWeek } from 'date-fns';
import { toast } from 'sonner';
import type { Leave, Employee, EmployeeShift, Shift, PatternDay } from '@/types/domain';

const DAY_NAMES = ['Sunday', 'Monday', 'Tuesday', 'Wednesday', 'Thursday', 'Friday', 'Saturday'];

/**
 * The parts of a scheduled row these helpers read. A row may be absent for a
 * day the schedule has not materialised yet, which is why every one of them
 * accepts null.
 */
type ScheduleRowLike =
  | Partial<Pick<EmployeeShift, 'shift_status' | 'leave_reason' | 'shift_id'>>
  | null
  | undefined;

/**
 * A row in the day's roster: either a real scheduled record, or a stand-in for
 * an active employee who has none yet and is therefore shown on their default
 * shift. The stand-ins are marked so the UI can label them and refuse to offer
 * actions that need a real record to act on.
 */
type RosterRow = Partial<EmployeeShift> &
  Pick<EmployeeShift, 'employee_id' | 'shift_status'> & { id: string; _isVirtual?: boolean };

/** The fallback, plus the server's own message when it sent one. */
function describeError(err: unknown, fallback: string): string {
  const detail = apiError(err);
  return detail ? `${fallback}: ${detail}` : fallback;
}

/** Detect if a shift row is actually an hourly leave (stored as 'leave' with [hourly] in reason). */
function resolveDisplayStatus(row: ScheduleRowLike): string {
  const raw = String(row?.shift_status || '').toLowerCase();
  if (raw === 'leave' && row?.leave_reason && String(row.leave_reason).startsWith('[hourly]')) {
    return 'hourly';
  }
  if (raw === 'hourly') return 'hourly'; // direct DB enum (after restart)
  return raw;
}

/** Resolve display status for a day when DB row may be missing (matches daily view virtual rows). */
function resolveDayStatus(
  emp: Pick<Employee, 'weekly_off_days' | 'default_shift_id'>,
  row: ScheduleRowLike,
  dateStr: string,
): string {
  if (row?.shift_status) return resolveDisplayStatus(row);
  const dayOfWeek = new Date(`${dateStr}T12:00:00`).getDay();
  if (emp.weekly_off_days >= 0 && emp.weekly_off_days === dayOfWeek) return 'off';
  if (emp.default_shift_id) return 'working';
  return 'none';
}

function resolveShiftName(
  row: ScheduleRowLike,
  emp: Pick<Employee, 'default_shift_id'> | null | undefined,
  shiftLookup: Record<string, Shift>,
): string | null {
  const shiftId = row?.shift_id || emp?.default_shift_id;
  if (!shiftId) return null;
  return shiftLookup[shiftId]?.name || 'Shift';
}

export const ScheduleView = () => {
  const [viewDate, setViewDate] = useState(format(new Date(), 'yyyy-MM-dd'));
  const [filterShiftId, setFilterShiftId] = useState<string>('');
  const queryClient = useQueryClient();
  const { user, adminSelectedDepartmentId, managerSelectedDepartmentId } = useAuthStore();
  const isAdmin = user?.role === 'admin';
  const isSupervisor = user?.role === 'admin' || user?.role === 'manager' || user?.role === 'team_leader';
  const canEdit = user?.role === 'team_leader' || user?.role === 'admin' || user?.role === 'manager';
  
  const deptId = user?.role === 'admin' ? adminSelectedDepartmentId : (user?.role === 'manager' ? managerSelectedDepartmentId : 'self');

  // Manual set form
  const [setEmployeeId, setSetEmployeeId] = useState('');
  const [setShiftStatus, setSetShiftStatus] = useState<'working' | 'off'>('working');
  const [setShiftId, setSetShiftId] = useState('');
  const [setError, setSetError] = useState<string | null>(null);

  const selectClass = "w-full h-10 px-3 py-2 rounded-lg bg-background border border-input text-foreground focus:outline-none focus:ring-2 focus:ring-ring transition-colors";

  // Query for fetching daily schedule.
  // Failures are deliberately NOT swallowed: falling back to an empty array renders
  // every employee's default shift as though it were the real roster, so a broken
  // request looks identical to "nothing is scheduled" and edits appear to do nothing.
  const { data: activeSchedule, isLoading: isLoadingSchedule, error: scheduleError } = useQuery({
    queryKey: ['schedules', viewDate, deptId],
    queryFn: async () => {
      const response = await api.get(`/schedules/daily?date=${viewDate}`);
      return (response.data?.data || []) as EmployeeShift[];
    },
  });

  const { data: rawEmployees } = useQuery({
    queryKey: ['employees', 'all', deptId],
    queryFn: async () => {
      const res = await api.get('/employees');
      return (res.data?.data || []) as Employee[];
    },
  });

  const employees = useMemo(() => {
    if (!rawEmployees) return [];
    if (user?.role === 'admin') return rawEmployees;
    if (user?.role === 'manager') return rawEmployees.filter(e => e.role !== 'admin');
    if (user?.role === 'team_leader') return rawEmployees.filter(e => e.role !== 'admin' && e.role !== 'manager' && (e.role !== 'team_leader' || e.id === user.id));
    return rawEmployees;
  }, [rawEmployees, user]);

  const { data: shifts } = useQuery({
    queryKey: ['shifts', deptId],
    queryFn: async () => {
      const res = await api.get('/shifts');
      return (res.data?.data || []) as Shift[];
    },
  });

  // Roster week, navigable. 0 = current week.
  const [weekOffset, setWeekOffset] = useState(0);
  const weekStart = useMemo(
    () => startOfWeek(addWeeks(new Date(), weekOffset), { weekStartsOn: 0 }),
    [weekOffset],
  );
  const weekStartKey = format(weekStart, 'yyyy-MM-dd');
  const weekDays = useMemo(
    () => Array.from({ length: 7 }, (_, idx) => addDays(weekStart, idx)),
    [weekStartKey, weekStart],
  );

  // Whether a roster edit updates the employee's fixed weekly pattern (repeats every
  // week) or applies to that date only. Permanent matches the long-standing behaviour.
  const [applyPermanently, setApplyPermanently] = useState(true);
  const [patternEmployee, setPatternEmployee] = useState<Employee | null>(null);

  const { data: weeklyRows, isLoading: weeklyLoading, error: weeklyError } = useQuery({
    queryKey: ['schedules', 'weekly-matrix', weekStartKey, deptId, shifts?.length ?? 0],
    queryFn: async () => {
      const shiftLookup: Record<string, Shift> = {};
      (shifts || []).forEach(s => { shiftLookup[s.id] = s; });

      const dates = weekDays.map((d) => format(d, 'yyyy-MM-dd'));
      // A rejected day used to become an empty array, so a failing API silently
      // rendered the whole week as defaults. Let it fail loudly instead.
      const dayResponses = await Promise.all(
        dates.map((d) => api.get(`/schedules/daily?date=${d}`)),
      );
      const dayMap: Record<string, EmployeeShift[]> = {};
      dates.forEach((d, idx) => {
        dayMap[d] = dayResponses[idx]?.data?.data || [];
      });

      const emps = (employees || []).filter(e => !e.status || e.status.toLowerCase() === 'active');
      return emps.map(emp => {
        const days = dates.map((d) => {
          const row = (dayMap[d] || []).find(r => String(r.employee_id) === String(emp.id));
          const status = resolveDayStatus(emp, row, d);
          return {
            date: d,
            row: row || null,
            status,
            shiftName: resolveShiftName(row, emp, shiftLookup),
          };
        });
        return { employee: emp, days };
      });
    },
    enabled: (employees || []).length > 0,
  });

  const filteredWeeklyRows = useMemo(() => {
    const rows = weeklyRows || [];
    if (!filterShiftId) return rows;
    return rows.filter(row => row?.employee?.default_shift_id === filterShiftId);
  }, [weeklyRows, filterShiftId]);

  const { data: pendingLeaves } = useQuery({
    queryKey: ['leaves', 'pending', 'schedule-panel', deptId],
    queryFn: async () => {
      const res = await api.get('/leaves/pending');
      return (res.data?.data || []) as Leave[];
    },
    enabled: isSupervisor,
  });

  // The day-long requests among the pending ones, for the alerts panel below.
  //
  // This used to filter on `leave_type === 'annual' || leave_type === 'vacation'`.
  // There is no `leave_type` field on the wire — a leave carries
  // `leave_type_id`, `leave_type_name_ar/_en` and `leave_type_is_hourly` — so
  // the comparison was undefined against a string, the list was empty every
  // time, and the panel showed "No pending vacation requests" no matter how
  // many were waiting. Matching on the type NAME would be just as fragile,
  // because leave types are named by whoever configures them; `is_hourly` is
  // the property that actually separates a زمنية from a day off, and a day
  // off is what this panel means by a vacation request.
  const dayLeaveAlerts = (pendingLeaves || []).filter(l => !l.leave_type_is_hourly);

  const employeeMap = useMemo(() => {
    const m: Record<string, Employee> = {};
    (employees || []).forEach(e => { m[e.id] = e; });
    return m;
  }, [employees]);

  const shiftMap = useMemo(() => {
    const m: Record<string, Shift> = {};
    (shifts || []).forEach(s => { m[s.id] = s; });
    return m;
  }, [shifts]);

  const shiftsByShift: Record<string, RosterRow[]> = useMemo(() => {
    const groups: Record<string, RosterRow[]> = {};

    // Track which employees already have a real DB record for this day
    const coveredEmployeeIds = new Set<string>();
    (activeSchedule || []).forEach(es => {
      const emp = employeeMap[es.employee_id];
      if (!emp) return; // Ignore shifts for employees outside of our allowed scope/department
      
      coveredEmployeeIds.add(String(es.employee_id));
      const key = es.shift_id || (emp?.default_shift_id) || 'off_no_shift';
      if (filterShiftId && key !== filterShiftId) return;
      if (!groups[key]) groups[key] = [];
      groups[key].push(es);
    });

    // Add virtual rows for active employees with no record today
    (employees || []).forEach(emp => {
      if (emp.status && emp.status.toLowerCase() !== 'active') return;
      if (coveredEmployeeIds.has(String(emp.id))) return; // already has a real row
      const key = emp.default_shift_id || 'off_no_shift';
      if (filterShiftId && key !== filterShiftId) return;
      if (!groups[key]) groups[key] = [];
      groups[key].push({
        id: `virtual-${emp.id}`,
        employee_id: emp.id,
        shift_id: emp.default_shift_id || null,
        shift_status: 'working',
        leave_reason: null,
        _isVirtual: true,
      });
    });

    Object.values(groups).forEach((list) => list.sort((a, b) => (a.employee_id || '').localeCompare(b.employee_id || '')));
    return groups;
  }, [activeSchedule, employees, filterShiftId]);

  const stats = useMemo(() => {
    const base = { working: 0, off: 0, leave: 0, vacation: 0, hourly: 0, other: 0, total: 0 };
    // Count real DB rows
    const coveredIds = new Set<string>();
    (activeSchedule || []).forEach(es => {
      if (!employeeMap[es.employee_id]) return; // Skip out-of-scope employees
      coveredIds.add(String(es.employee_id));
      base.total++;
      const st = resolveDisplayStatus(es);
      if (st === 'working') base.working++;
      else if (st === 'off') base.off++;
      else if (st === 'hourly') base.hourly++;
      else if (st === 'leave') base.leave++;
      else if (st === 'vacation') base.vacation++;
      else base.other++;
    });
    // Count virtual rows (employees with no record = assumed working)
    (employees || []).forEach(emp => {
      if ((!emp.status || emp.status.toLowerCase() === 'active') && !coveredIds.has(String(emp.id))) {
        base.total++;
        base.working++;
      }
    });
    return base;
  }, [activeSchedule, employees]);

  const assignReplacement = useMutation({
    mutationFn: async ({ employeeShiftId, replacementEmployeeId }: { employeeShiftId: string; replacementEmployeeId: string }) => {
      await api.post(`/schedules/shifts/${employeeShiftId}/replace`, { replacement_employee_id: replacementEmployeeId });
    },
    onSuccess: () => { queryClient.invalidateQueries({ queryKey: ['schedules'] }); },
  });

  const setEmployeeShift = useMutation({
    mutationFn: async () => {
      setSetError(null);
      await api.post('/schedules/shifts/set', {
        employee_id: setEmployeeId,
        shift_date: viewDate,
        shift_status: setShiftStatus,
        shift_id: setShiftStatus === 'working' ? (setShiftId || null) : null,
        leave_reason: null,
        permanent: applyPermanently,
      });
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['schedules'] });
      setSetEmployeeId(''); setSetShiftId('');
    },
    onError: (err: unknown) => {
      setSetError(describeError(err, 'Failed to set shift'));
    },
  });

  // These three had no onError, so a rejected save produced no toast, no console
  // trace and no cache invalidation — the cell simply snapped back and the change
  // looked like it had been ignored.
  const setOffQuick = useMutation({
    mutationFn: async ({ employeeId, date }: { employeeId: string; date: string }) => {
      await api.post('/schedules/shifts/set', {
        employee_id: employeeId, shift_date: date, shift_status: 'off', shift_id: null, leave_reason: null,
        permanent: applyPermanently,
      });
    },
    onSuccess: () => {
      toast.success(applyPermanently ? 'Off day saved — repeats every week' : 'Off day saved for this date');
      queryClient.invalidateQueries({ queryKey: ['schedules'] });
    },
    onError: (err: unknown) => toast.error(describeError(err, 'Could not set the off day')),
  });

  const setWorkingQuick = useMutation({
    mutationFn: async ({ employeeId, date, shiftId }: { employeeId: string; date: string; shiftId: string }) => {
      await api.post('/schedules/shifts/set', {
        employee_id: employeeId, shift_date: date, shift_status: 'working', shift_id: shiftId, leave_reason: null,
        permanent: applyPermanently,
      });
    },
    onSuccess: () => {
      toast.success(applyPermanently ? 'Shift saved — repeats every week' : 'Shift saved for this date');
      queryClient.invalidateQueries({ queryKey: ['schedules'] });
    },
    onError: (err: unknown) => toast.error(describeError(err, 'Could not assign the shift')),
  });

  const deleteShift = useMutation({
    mutationFn: async (shiftId: string) => {
      await api.delete(`/schedules/shifts/${shiftId}`);
    },
    onSuccess: () => {
      toast.success('Assignment cleared');
      queryClient.invalidateQueries({ queryKey: ['schedules'] });
    },
    onError: (err: unknown) => toast.error(describeError(err, 'Could not clear the assignment')),
  });

  return (
    <div className="space-y-8">
      <div>
        <h2 className="text-2xl sm:text-2xl sm:text-3xl font-bold tracking-tight text-foreground mb-2 flex items-center gap-2 sm:gap-3">
          <CalendarIcon className="w-6 h-6 sm:w-8 sm:h-8 text-primary" />
          Schedules
        </h2>
        <p className="text-sm sm:text-base text-muted-foreground">Daily staffing view grouped by shift, with clear status and quick actions.</p>
      </div>

      {/* Filters */}
      <Card>
        <CardContent className="p-3 sm:p-5 flex flex-col gap-3 sm:gap-4">
          <div className="space-y-2">
            <Label>View day</Label>
            <div className="relative">
              <CalendarIcon className="w-4 h-4 text-muted-foreground absolute left-3 top-3" />
              <Input type="date" value={viewDate}
                onChange={(e: React.ChangeEvent<HTMLInputElement>) => setViewDate(e.target.value)}
                className="pl-9" />
            </div>
          </div>

          <div className="space-y-2">
            <Label>Shift filter</Label>
            <div className="relative">
              <Filter className="w-4 h-4 text-muted-foreground absolute left-3 top-3" />
              <select className={selectClass + " pl-9"} value={filterShiftId} onChange={(e) => setFilterShiftId(e.target.value)}>
                <option value="">All shifts</option>
                {shifts?.map(s => <option key={s.id} value={s.id}>{s.name} ({s.shift_code})</option>)}
              </select>
            </div>
          </div>

          <div className="grid grid-cols-3 sm:grid-cols-6 gap-2 sm:gap-3 w-full">
            <Stat label="Total" value={stats.total} tone="default" />
            <Stat label="Working" value={stats.working} tone="emerald" />
            <Stat label="Off" value={stats.off} tone="amber" />
            <Stat label="Leave" value={stats.leave} tone="rose" />
            <Stat label="Hourly" value={stats.hourly} tone="cyan" />
            <Stat label="Vacation" value={stats.vacation} tone="blue" />
          </div>
        </CardContent>
      </Card>

      {/* Manual schedule creation / update — team_leader + admin only */}
      {canEdit && (
        <Card>
          <CardHeader className="pb-3">
            <CardTitle className="text-lg flex items-center gap-2">
              <Wand2 className="w-5 h-5 text-primary" />
              Assign / Update Shift (manual)
            </CardTitle>
            <CardDescription>Set one employee's shift status for the selected day.</CardDescription>
          </CardHeader>
          <CardContent className="space-y-4">
            {setError && (
              <div className="p-3 rounded-lg bg-destructive/10 border border-destructive/20 text-destructive text-sm">{setError}</div>
            )}
            <div className="grid grid-cols-1 sm:grid-cols-2 md:grid-cols-4 gap-3 sm:gap-4">
              <div className="space-y-2 md:col-span-2">
                <Label>Employee</Label>
                <select className={selectClass} value={setEmployeeId} onChange={(e) => setSetEmployeeId(e.target.value)}>
                  <option value="">Select employee…</option>
                  {(employees || []).filter(e => !e.status || e.status.toLowerCase() === 'active').map(e => <option key={e.id} value={e.id}>{e.first_name} {e.last_name} — {e.employee_code}</option>)}
                </select>
              </div>
              <div className="space-y-2">
                <Label>Status</Label>
                <select className={selectClass} value={setShiftStatus} onChange={(e) => setSetShiftStatus(e.target.value as 'working' | 'off')}>
                  <option value="working">Working</option>
                  <option value="off">Off</option>
                </select>
              </div>
              <div className="space-y-2">
                <Label>Shift</Label>
                <select className={selectClass} value={setShiftId} onChange={(e) => setSetShiftId(e.target.value)} disabled={setShiftStatus !== 'working'}>
                  <option value="">{setShiftStatus === 'working' ? 'Select shift…' : 'Not required'}</option>
                  {shifts?.map(s => <option key={s.id} value={s.id}>{s.name} ({s.shift_code})</option>)}
                </select>
              </div>
            </div>
            <div className="flex justify-end">
              <Button onClick={() => setEmployeeShift.mutate()}
                disabled={setEmployeeShift.isPending || !setEmployeeId || !viewDate || (setShiftStatus === 'working' && !setShiftId)}>
                {setEmployeeShift.isPending ? 'Saving…' : 'Save'}
              </Button>
            </div>
          </CardContent>
        </Card>
      )}



      {/* Daily schedule */}
      <Card>
        <CardHeader className="pb-3 border-b border-border">
          <CardTitle className="text-xl">Daily Schedule · {format(new Date(viewDate), 'MMM d, yyyy')}</CardTitle>
          <CardDescription>Grouped by shift with readable employee info.</CardDescription>
        </CardHeader>
        <CardContent className="p-6">
          {isLoadingSchedule ? (
            <div className="flex items-center justify-center py-16">
              <Loader2 className="w-8 h-8 animate-spin text-muted-foreground" />
            </div>
          ) : scheduleError ? (
            <div className="py-12 text-center">
              <AlertTriangle className="w-10 h-10 mx-auto mb-3 text-destructive" />
              <p className="text-destructive font-medium">Could not load the schedule for this day.</p>
              <p className="text-sm text-muted-foreground mt-1">{describeError(scheduleError, 'Server error')}</p>
            </div>
          ) : !activeSchedule || activeSchedule.length === 0 ? (
            <div className="py-16 text-center text-muted-foreground">
              <CalendarIcon className="w-12 h-12 mx-auto mb-3 opacity-20" />
              <p>No schedule rows found for this day.</p>
            </div>
          ) : (
            <div className="space-y-5">
              {Object.entries(shiftsByShift).map(([shiftId, rows]) => {
                const shift = shiftMap[shiftId];
                const title = shift ? `${shift.name} (${shift.shift_code})` : 'Off / No Shift';
                const working = rows.filter(r => r.shift_status === 'working').length;
                const off = rows.filter(r => r.shift_status === 'off').length;
                const leave = rows.filter(r => r.shift_status === 'leave').length;
                const hourly = rows.filter(r => r.shift_status === 'hourly').length;
                const vacation = rows.filter(r => r.shift_status === 'vacation').length;

                return (
                  <div key={shiftId} className="rounded-2xl border border-border overflow-hidden">
                    <div className="px-4 py-3 bg-muted/30 flex items-center justify-between gap-3">
                      <div className="flex items-center gap-2 min-w-0">
                        <Briefcase className="w-4 h-4 text-muted-foreground" />
                        <span className="text-foreground font-semibold truncate">{title}</span>
                      </div>
                      <div className="text-xs text-muted-foreground flex gap-3 shrink-0">
                        <span className="text-emerald-500">Working: {working}</span>
                        <span className="text-amber-500">Off: {off}</span>
                        <span className="text-rose-500">Leave: {leave}</span>
                        <span className="text-cyan-500">Hourly: {hourly}</span>
                        <span className="text-blue-500">Vacation: {vacation}</span>
                      </div>
                    </div>

                    <div className="divide-y divide-border">
                      {rows.map(es => {
                        const emp = employeeMap[es.employee_id];
                        const name = emp ? `${emp.first_name} ${emp.last_name}` : 'Unknown Employee';
                        const code = emp?.employee_code || '';
                        const status = resolveDisplayStatus(es) || '—';

                        const statusTone =
                          status === 'working' ? 'text-emerald-500' :
                          status === 'off' ? 'text-amber-500' :
                          status === 'leave' ? 'text-rose-500' :
                          status === 'hourly' ? 'text-cyan-500' :
                          status === 'vacation' ? 'text-blue-500' :
                          'text-muted-foreground';

                        return (
                          <div key={es.id} className="px-4 py-3 flex items-center justify-between gap-4 bg-card">
                            <div className="min-w-0 flex items-center gap-2 sm:gap-3">
                              <div className="w-9 h-9 rounded-xl bg-muted/60 border border-border flex items-center justify-center text-foreground font-semibold">
                                {name?.[0] || '?'}
                              </div>
                              <div className="min-w-0">
                                <div className="text-foreground font-medium truncate flex items-center gap-2">
                                  <Users className="w-4 h-4 text-muted-foreground" />
                                  {name}
                                </div>
                                <div className="text-xs text-muted-foreground truncate">{code}</div>
                              </div>
                            </div>

                            <div className="flex items-center gap-3 shrink-0">
                              <span className={`text-xs font-semibold uppercase tracking-wider ${statusTone}`}>
                                {status}
                              </span>
                              {es._isVirtual && (
                                <span className="text-[10px] text-muted-foreground/50 italic">default</span>
                              )}

                              {isAdmin && !es._isVirtual && (
                                <ReplacementButton
                                  employeeShiftId={es.id}
                                  date={viewDate}
                                  disabled={assignReplacement.isPending}
                                  onAssign={(replacementEmployeeId) =>
                                    assignReplacement.mutate({ employeeShiftId: es.id, replacementEmployeeId })
                                  }
                                />
                              )}
                            </div>
                          </div>
                        );
                      })}
                    </div>
                  </div>
                );
              })}
            </div>
          )}
        </CardContent>
      </Card>

      {/* Weekly roster table */}
      <Card>
        <CardHeader className="pb-3 border-b border-border space-y-3">
          <div className="flex flex-wrap items-start justify-between gap-3">
            <div>
              <CardTitle className="text-xl">Weekly Team Roster</CardTitle>
              <CardDescription>
                {format(weekStart, 'MMM d')} – {format(addDays(weekStart, 6), 'MMM d, yyyy')}
                {weekOffset === 0 && <span className="ml-2 text-primary">· current week</span>}
              </CardDescription>
            </div>
            <div className="flex items-center gap-1">
              <Button variant="outline" size="sm" onClick={() => setWeekOffset((w) => w - 1)} aria-label="Previous week">
                <ChevronLeft className="w-4 h-4" />
              </Button>
              <Button variant="outline" size="sm" onClick={() => setWeekOffset(0)} disabled={weekOffset === 0}>
                Today
              </Button>
              <Button variant="outline" size="sm" onClick={() => setWeekOffset((w) => w + 1)} aria-label="Next week">
                <ChevronRight className="w-4 h-4" />
              </Button>
            </div>
          </div>

          {canEdit && (
            <div className="flex flex-wrap items-center gap-2 pt-1">
              <span className="text-xs text-muted-foreground">Edits apply:</span>
              <div className="inline-flex rounded-lg border border-border overflow-hidden">
                <button
                  type="button"
                  onClick={() => setApplyPermanently(true)}
                  className={`flex items-center gap-1.5 px-3 py-1.5 text-xs transition-colors ${
                    applyPermanently ? 'bg-primary text-primary-foreground' : 'bg-background text-muted-foreground hover:text-foreground'
                  }`}
                >
                  <Repeat className="w-3.5 h-3.5" />
                  Every week
                </button>
                <button
                  type="button"
                  onClick={() => setApplyPermanently(false)}
                  className={`flex items-center gap-1.5 px-3 py-1.5 text-xs border-l border-border transition-colors ${
                    !applyPermanently ? 'bg-primary text-primary-foreground' : 'bg-background text-muted-foreground hover:text-foreground'
                  }`}
                >
                  <Pin className="w-3.5 h-3.5" />
                  This day only
                </button>
              </div>
              <span className="text-[11px] text-muted-foreground">
                {applyPermanently
                  ? 'Becomes part of the fixed weekly schedule and repeats automatically.'
                  : 'Applies to this date only — the fixed weekly schedule is unchanged.'}
              </span>
            </div>
          )}
        </CardHeader>
        <CardContent className="p-0">
          {weeklyLoading ? (
            <div className="flex items-center justify-center py-12">
              <Loader2 className="w-7 h-7 animate-spin text-muted-foreground" />
            </div>
          ) : weeklyError ? (
            <div className="py-12 text-center">
              <AlertTriangle className="w-10 h-10 mx-auto mb-3 text-destructive" />
              <p className="text-destructive font-medium">Could not load the weekly roster.</p>
              <p className="text-sm text-muted-foreground mt-1">{describeError(weeklyError, 'Server error')}</p>
              <p className="text-xs text-muted-foreground mt-3">
                Nothing below is shown rather than displaying default shifts that are not the real roster.
              </p>
            </div>
          ) : (
            <div className="overflow-x-auto">
              <table className="w-full min-w-[700px]">
                <thead>
                  <tr className="bg-muted/30 border-b border-border">
                    <th className="text-left p-3 text-xs uppercase tracking-wider text-muted-foreground">Employee</th>
                    {weekDays.map((d) => (
                      <th key={d.toISOString()} className="text-left p-3 text-xs uppercase tracking-wider text-muted-foreground">
                        {format(d, 'EEE dd')}
                      </th>
                    ))}
                  </tr>
                </thead>
                <tbody>
                  {filteredWeeklyRows.map(row => (
                    <tr key={row.employee.id} className="border-b border-border align-top">
                      <td className="p-3">
                        <div className="text-foreground font-medium">
                          {row.employee.first_name} {row.employee.last_name}
                        </div>
                        <div className="text-xs text-muted-foreground">{row.employee.employee_code}</div>
                        {canEdit && (
                          <button
                            type="button"
                            onClick={() => setPatternEmployee(row.employee)}
                            className="mt-1 inline-flex items-center gap-1 text-[11px] text-primary hover:underline"
                          >
                            <Repeat className="w-3 h-3" />
                            Fixed schedule
                          </button>
                        )}
                      </td>
                      {row.days.map(d => {
                        const tone =
                          d.status === 'working' ? 'text-emerald-500 border-emerald-500/30 bg-emerald-500/5'
                            : d.status === 'off' ? 'text-amber-500 border-amber-500/30 bg-amber-500/5'
                              : d.status === 'leave' ? 'text-rose-500 border-rose-500/30 bg-rose-500/5'
                                : d.status === 'hourly' ? 'text-cyan-500 border-cyan-500/30 bg-cyan-500/5'
                                  : d.status === 'vacation' ? 'text-blue-500 border-blue-500/30 bg-blue-500/5'
                                    : 'text-muted-foreground border-border bg-muted/20';

                        return (
                          <td key={`${row.employee.id}-${d.date}`} className="p-3">
                            <div className={`rounded-lg border px-2 py-2 ${tone}`}>
                              <div className="text-xs font-semibold uppercase tracking-wider flex items-center gap-1">
                                {d.status}
                                {d.row?.source === 'manual' && (
                                  <Pin className="w-3 h-3 opacity-60" aria-label="Set for this day only" />
                                )}
                              </div>
                              <div className="text-[11px] opacity-80 mt-1">{d.shiftName || '-'}</div>
                              {canEdit && (
                                <div className="mt-2">
                                  <select
                                    className={`w-full h-7 text-[10px] rounded border px-1 outline-none focus:ring-1 transition-colors ${
                                      d.status === 'working' ? 'border-emerald-500/30 text-emerald-600 bg-emerald-500/10' :
                                      d.status === 'off' ? 'border-amber-500/30 text-amber-600 bg-amber-500/10' :
                                      'border-border text-foreground bg-background'
                                    }`}
                                    value={d.status === 'working' ? (d.row?.shift_id || row.employee.default_shift_id || 'working') : d.status}
                                    onChange={(e) => {
                                      const val = e.target.value;
                                      if (!val) return;
                                      if (val === 'off') {
                                        setOffQuick.mutate({ employeeId: row.employee.id, date: d.date });
                                      } else if (val === 'remove') {
                                        if (d.row?.id) deleteShift.mutate(d.row.id);
                                        else alert('Cannot clear: record missing. Please refresh.');
                                      } else {
                                        setWorkingQuick.mutate({ employeeId: row.employee.id, date: d.date, shiftId: val });
                                      }
                                    }}
                                    disabled={setOffQuick.isPending || setWorkingQuick.isPending || deleteShift.isPending}
                                  >
                                    <option value="" disabled hidden>
                                      {d.status === 'leave' || d.status === 'vacation' || d.status === 'hourly' 
                                        ? d.status.toUpperCase() 
                                        : 'Assign...'}
                                    </option>
                                    
                                    <optgroup label="Assign Shift">
                                      {shifts?.map(s => (
                                        <option key={s.id} value={s.id}>{s.name} ({s.shift_code})</option>
                                      ))}
                                    </optgroup>
                                    
                                    <optgroup label="Other Actions">
                                      <option value="off">Set Off Day</option>
                                      {d.row && <option value="remove">Clear Assignment</option>}
                                    </optgroup>
                                  </select>
                                </div>
                              )}
                            </div>
                          </td>
                        );
                      })}
                    </tr>
                  ))}
                  {filteredWeeklyRows.length === 0 && (
                    <tr>
                      <td className="p-8 text-center text-muted-foreground" colSpan={8}>
                        No employees available for this shift filter.
                      </td>
                    </tr>
                  )}
                </tbody>
              </table>
            </div>
          )}
        </CardContent>
      </Card>

      {patternEmployee && (
        <WeeklyPatternModal
          employee={patternEmployee}
          shifts={shifts || []}
          onClose={() => setPatternEmployee(null)}
          onSaved={() => {
            setPatternEmployee(null);
            queryClient.invalidateQueries({ queryKey: ['schedules'] });
          }}
        />
      )}

      {/* Vacation request alarms */}
      {isSupervisor && (
        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="text-lg flex items-center gap-2">
              <AlertTriangle className="w-5 h-5 text-amber-500" />
              Vacation / Leave Alerts
            </CardTitle>
            <CardDescription>Pending requests that need attention.</CardDescription>
          </CardHeader>
          <CardContent className="space-y-2">
            {dayLeaveAlerts.length > 0 ? (
              dayLeaveAlerts.map(leave => (
                  <div key={leave.id} className="rounded-lg border border-amber-500/30 bg-amber-500/5 p-3 text-sm">
                    <span className="text-amber-500 font-medium">Vacation request:</span>{' '}
                    <span className="text-foreground font-semibold">
                      {employeeMap[leave.employee_id]
                        ? `${employeeMap[leave.employee_id].first_name} ${employeeMap[leave.employee_id].last_name}`
                        : `Employee #${String(leave.employee_id).slice(0, 8)}`}
                    </span>{' '}
                    <span className="text-muted-foreground">({leave.start_date?.split('T')[0]} → {leave.end_date?.split('T')[0]})</span>
                  </div>
                ))
            ) : (
              <p className="text-muted-foreground text-sm">No pending vacation requests.</p>
            )}
          </CardContent>
        </Card>
      )}
    </div>
  );
};

/**
 * Editor for an employee's fixed weekly pattern — the schedule that repeats every
 * week by itself. Saving applies it to every future week at once; today and the past
 * are left alone, as are days a supervisor pinned or an approved leave owns.
 */
const WeeklyPatternModal = ({
  employee,
  shifts,
  onClose,
  onSaved,
}: {
  employee: Employee;
  shifts: Shift[];
  onClose: () => void;
  onSaved: () => void;
}) => {
  const [days, setDays] = useState<{ day_of_week: number; is_off: boolean; shift_id: string | null }[]>([]);
  const [error, setError] = useState<string | null>(null);

  const { data, isLoading } = useQuery({
    queryKey: ['schedules', 'pattern', employee.id],
    queryFn: async () => {
      const res = await api.get(`/schedules/pattern/${employee.id}`);
      return (res.data?.data || []) as PatternDay[];
    },
  });

  useEffect(() => {
    if (!data) return;
    setDays(
      data.map(d => ({
        day_of_week: d.day_of_week,
        is_off: !!d.is_off,
        shift_id: d.shift_id || null,
      })),
    );
  }, [data]);

  const save = useMutation({
    mutationFn: async () => {
      setError(null);
      await api.put(`/schedules/pattern/${employee.id}`, { days });
    },
    onSuccess: onSaved,
    onError: (err: unknown) => {
      setError(describeError(err, 'Failed to save the weekly pattern'));
    },
  });

  const selectClass =
    'w-full h-9 px-2 rounded-md bg-background border border-input text-foreground text-sm focus:outline-none focus:ring-2 focus:ring-ring';

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50 p-4" onClick={onClose}>
      <div
        className="w-full max-w-lg max-h-[90vh] overflow-y-auto rounded-2xl bg-card border border-border shadow-2xl"
        onClick={(e) => e.stopPropagation()}
      >
        <div className="flex items-start justify-between gap-3 p-5 border-b border-border">
          <div>
            <h3 className="text-lg font-semibold text-foreground flex items-center gap-2">
              <Repeat className="w-5 h-5 text-primary" />
              Fixed weekly schedule
            </h3>
            <p className="text-sm text-muted-foreground mt-1">
              {employee.first_name} {employee.last_name} — {employee.employee_code}
            </p>
          </div>
          <Button variant="outline" size="sm" onClick={onClose} aria-label="Close">
            <X className="w-4 h-4" />
          </Button>
        </div>

        <div className="p-5 space-y-3">
          <p className="text-xs text-muted-foreground">
            This is the schedule that repeats every week on its own. Saving applies it to all future
            weeks; today and past days stay as they are, and approved leaves are not affected.
          </p>

          {error && (
            <div className="p-3 rounded-lg bg-destructive/10 border border-destructive/20 text-destructive text-sm">
              {error}
            </div>
          )}

          {isLoading ? (
            <div className="flex items-center justify-center py-10">
              <Loader2 className="w-6 h-6 animate-spin text-muted-foreground" />
            </div>
          ) : (
            days.map((d) => (
              <div key={d.day_of_week} className="flex items-center gap-3">
                <div className="w-24 shrink-0 text-sm text-foreground">{DAY_NAMES[d.day_of_week]}</div>
                <select
                  className={selectClass}
                  value={d.is_off ? 'off' : d.shift_id || ''}
                  onChange={(e) => {
                    const val = e.target.value;
                    setDays((prev) =>
                      prev.map((p) =>
                        p.day_of_week === d.day_of_week
                          ? val === 'off'
                            ? { ...p, is_off: true, shift_id: null }
                            : { ...p, is_off: false, shift_id: val || null }
                          : p,
                      ),
                    );
                  }}
                >
                  <option value="">Select shift…</option>
                  <option value="off">Off day</option>
                  {shifts.map(s => (
                    <option key={s.id} value={s.id}>
                      {s.name} ({s.shift_code})
                    </option>
                  ))}
                </select>
              </div>
            ))
          )}
        </div>

        <div className="flex justify-end gap-2 p-5 border-t border-border">
          <Button variant="outline" onClick={onClose}>
            Cancel
          </Button>
          <Button
            onClick={() => save.mutate()}
            disabled={save.isPending || isLoading || days.some((d) => !d.is_off && !d.shift_id)}
          >
            {save.isPending ? 'Saving…' : 'Save pattern'}
          </Button>
        </div>
      </div>
    </div>
  );
};

const Stat = ({ label, value, tone }: { label: string; value: number; tone: 'default' | 'emerald' | 'amber' | 'rose' | 'cyan' | 'blue' }) => {
  const toneMap: Record<string, string> = {
    default: 'text-foreground bg-muted/40 border-border',
    emerald: 'text-emerald-500 bg-emerald-500/5 border-emerald-500/20',
    amber: 'text-amber-500 bg-amber-500/5 border-amber-500/20',
    rose: 'text-rose-500 bg-rose-500/5 border-rose-500/20',
    cyan: 'text-cyan-500 bg-cyan-500/5 border-cyan-500/20',
    blue: 'text-blue-500 bg-blue-500/5 border-blue-500/20',
  };
  return (
    <div className={`rounded-xl border px-3 py-2 ${toneMap[tone]}`}>
      <div className="text-[10px] uppercase tracking-widest opacity-80">{label}</div>
      <div className="text-lg font-bold leading-tight">{value}</div>
    </div>
  );
};

const ReplacementButton = ({
  employeeShiftId,
  date,
  disabled,
  onAssign,
}: {
  employeeShiftId: string;
  date: string;
  disabled: boolean;
  onAssign: (employeeId: string) => void;
}) => {
  const [open, setOpen] = useState(false);
  const [selected, setSelected] = useState('');

  const selectClass = "w-full h-9 px-2 rounded-md bg-background border border-input text-foreground text-sm focus:outline-none focus:ring-2 focus:ring-ring";

  const { data: replacements, isLoading } = useQuery({
    queryKey: ['replacements', date],
    queryFn: async () => {
      const res = await api.get(`/schedules/replacements?date=${date}`);
      return (res.data?.data || []) as Employee[];
    },
    enabled: open,
  });

  return (
    <div className="relative">
      <Button variant="outline" size="sm" onClick={() => { setOpen((v) => !v); setSelected(''); }} disabled={disabled}>
        Replace
      </Button>
      {open && (
        <div className="absolute right-0 mt-2 w-72 p-3 rounded-xl bg-popover border border-border shadow-2xl z-20">
          <div className="text-xs text-muted-foreground mb-2">Assign replacement for this shift</div>
          <select className={selectClass} value={selected} onChange={(e) => setSelected(e.target.value)} disabled={isLoading}>
            <option value="">Select employee…</option>
            {replacements?.map(e => (
              <option key={e.id} value={e.id}>{e.first_name} {e.last_name} — {e.employee_code}</option>
            ))}
          </select>
          <div className="flex justify-end gap-2 mt-3">
            <Button variant="outline" size="sm" onClick={() => setOpen(false)}>Cancel</Button>
            <Button size="sm" disabled={!selected || disabled} onClick={() => { onAssign(selected); setOpen(false); }}>Assign</Button>
          </div>
          <div className="mt-2 text-[10px] text-muted-foreground/60">Shift row: {employeeShiftId.slice(0, 8)}</div>
        </div>
      )}
    </div>
  );
};
