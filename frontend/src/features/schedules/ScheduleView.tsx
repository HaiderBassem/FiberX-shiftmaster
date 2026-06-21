import { useMemo, useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import api from '@/lib/api';
import { useAuthStore } from '@/store/authStore';
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Calendar as CalendarIcon, Loader2, Users, Briefcase, Wand2, Filter, AlertTriangle } from 'lucide-react';
import { addDays, format, startOfWeek } from 'date-fns';

export const ScheduleView = () => {
  const [viewDate, setViewDate] = useState(format(new Date(), 'yyyy-MM-dd'));
  const [filterShiftId, setFilterShiftId] = useState<string>('');
  const queryClient = useQueryClient();
  const { user } = useAuthStore();
  const isAdmin = user?.role === 'admin';
  const isSupervisor = user?.role === 'admin' || user?.role === 'manager' || user?.role === 'team_leader';
  const canEdit = user?.role === 'team_leader' || user?.role === 'admin'; // Manager is VIEW ONLY

  // Manual set form
  const [setEmployeeId, setSetEmployeeId] = useState('');
  const [setShiftStatus, setSetShiftStatus] = useState<'working' | 'off'>('working');
  const [setShiftId, setSetShiftId] = useState('');
  const [setError, setSetError] = useState<string | null>(null);

  const selectClass = "w-full h-10 px-3 py-2 rounded-lg bg-background border border-input text-foreground focus:outline-none focus:ring-2 focus:ring-ring transition-colors";

  // Query for fetching daily schedule
  const { data: activeSchedule, isLoading: isLoadingSchedule } = useQuery({
    queryKey: ['schedules', viewDate],
    queryFn: async () => {
      const response = await api.get(`/schedules/daily?date=${viewDate}`);
      return response.data?.data || [];
    },
  });

  const { data: rawEmployees } = useQuery({
    queryKey: ['employees', 'all'],
    queryFn: async () => {
      const res = await api.get('/employees');
      return res.data?.data || [];
    },
  });

  const employees = useMemo(() => {
    if (!rawEmployees) return [];
    if (user?.role === 'admin') return rawEmployees;
    if (user?.role === 'manager') return rawEmployees.filter((e: any) => e.role !== 'admin');
    if (user?.role === 'team_leader') return rawEmployees.filter((e: any) => e.role !== 'admin' && e.role !== 'manager');
    return rawEmployees;
  }, [rawEmployees, user]);

  const { data: shifts } = useQuery({
    queryKey: ['shifts'],
    queryFn: async () => {
      const res = await api.get('/shifts');
      return res.data?.data || [];
    },
  });

  const weekStartKey = format(startOfWeek(new Date(), { weekStartsOn: 0 }), 'yyyy-MM-dd');
  const weekStart = useMemo(
    () => startOfWeek(new Date(), { weekStartsOn: 0 }),
    [weekStartKey],
  );
  const weekDays = useMemo(
    () => Array.from({ length: 7 }, (_, idx) => addDays(weekStart, idx)),
    [weekStartKey, weekStart],
  );

  const { data: weeklyRows, isLoading: weeklyLoading } = useQuery({
    queryKey: ['schedules', 'weekly-matrix', format(weekStart, 'yyyy-MM-dd')],
    queryFn: async () => {
      const dates = weekDays.map((d) => format(d, 'yyyy-MM-dd'));
      const dayResponses = await Promise.all(
        dates.map((d) => api.get(`/schedules/daily?date=${d}`)),
      );
      const dayMap: Record<string, any[]> = {};
      dates.forEach((d, idx) => {
        dayMap[d] = dayResponses[idx]?.data?.data || [];
      });

      const emps = (employees || []).filter((e: any) => e.status === 'active');
      return emps.map((emp: any) => {
        const days = dates.map((d) => {
          const row = (dayMap[d] || []).find((r: any) => r.employee_id === emp.id);
          return {
            date: d,
            row,
            status: row?.shift_status || 'none',
            shiftName: row?.shift_id ? (shiftMap[row.shift_id]?.name || 'Shift') : null,
          };
        });
        return { employee: emp, days };
      });
    },
    enabled: (employees || []).length > 0 && (shifts || []).length >= 0,
  });

  const filteredWeeklyRows = useMemo(() => {
    const rows = weeklyRows || [];
    if (!filterShiftId) return rows;
    return rows.filter((row: any) => row?.employee?.default_shift_id === filterShiftId);
  }, [weeklyRows, filterShiftId]);

  const { data: pendingLeaves } = useQuery({
    queryKey: ['leaves', 'pending', 'schedule-panel'],
    queryFn: async () => {
      const res = await api.get('/leaves/pending');
      return res.data?.data || [];
    },
    enabled: isSupervisor,
  });

  const employeeMap = useMemo(() => {
    const m: Record<string, any> = {};
    (employees || []).forEach((e: any) => { m[e.id] = e; });
    return m;
  }, [employees]);

  const shiftMap = useMemo(() => {
    const m: Record<string, any> = {};
    (shifts || []).forEach((s: any) => { m[s.id] = s; });
    return m;
  }, [shifts]);

  const shiftsByShift: Record<string, any[]> = useMemo(() => {
    const groups: Record<string, any[]> = {};

    // Track which employees already have a real DB record for this day
    const coveredEmployeeIds = new Set<string>();
    (activeSchedule || []).forEach((es: any) => {
      const emp = employeeMap[es.employee_id];
      if (!emp) return; // Ignore shifts for employees outside of our allowed scope/department
      
      coveredEmployeeIds.add(String(es.employee_id));
      const key = es.shift_id || (emp?.default_shift_id) || 'off_no_shift';
      if (filterShiftId && key !== filterShiftId) return;
      if (!groups[key]) groups[key] = [];
      groups[key].push(es);
    });

    // Add virtual rows for active employees with no record today
    (employees || []).forEach((emp: any) => {
      if (emp.status !== 'active') return;
      if (coveredEmployeeIds.has(String(emp.id))) return; // already has a real row
      const key = emp.default_shift_id || 'off_no_shift';
      if (filterShiftId && key !== filterShiftId) return;
      if (!groups[key]) groups[key] = [];
      groups[key].push({
        id: `virtual-${emp.id}`,
        employee_id: emp.id,
        shift_id: emp.default_shift_id || null,
        shift_status: 'working',
        _isVirtual: true,
      });
    });

    Object.values(groups).forEach((list) => list.sort((a: any, b: any) => (a.employee_id || '').localeCompare(b.employee_id || '')));
    return groups;
  }, [activeSchedule, employees, filterShiftId]);

  const stats = useMemo(() => {
    const base = { working: 0, off: 0, leave: 0, vacation: 0, other: 0, total: 0 };
    // Count real DB rows
    const coveredIds = new Set<string>();
    (activeSchedule || []).forEach((es: any) => {
      if (!employeeMap[es.employee_id]) return; // Skip out-of-scope employees
      coveredIds.add(String(es.employee_id));
      base.total++;
      const st = (es.shift_status || '').toLowerCase();
      if (st === 'working') base.working++;
      else if (st === 'off') base.off++;
      else if (st === 'leave') base.leave++;
      else if (st === 'hourly') base.other++; // we could track hourly separately, but for now we put it under other, or we can just count it in working.
      else if (st === 'vacation') base.vacation++;
      else base.other++;
    });
    // Count virtual rows (employees with no record = assumed working)
    (employees || []).forEach((emp: any) => {
      if (emp.status === 'active' && !coveredIds.has(String(emp.id))) {
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
      });
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['schedules'] });
      setSetEmployeeId(''); setSetShiftId('');
    },
    onError: (err: any) => {
      setSetError(err?.response?.data?.error || err?.message || 'Failed to set shift');
    },
  });

  const setOffQuick = useMutation({
    mutationFn: async ({ employeeId, date }: { employeeId: string; date: string }) => {
      await api.post('/schedules/shifts/set', {
        employee_id: employeeId, shift_date: date, shift_status: 'off', shift_id: null, leave_reason: null,
      });
    },
    onSuccess: () => { queryClient.invalidateQueries({ queryKey: ['schedules'] }); },
  });

  const setWorkingQuick = useMutation({
    mutationFn: async ({ employeeId, date, shiftId }: { employeeId: string; date: string; shiftId: string }) => {
      await api.post('/schedules/shifts/set', {
        employee_id: employeeId, shift_date: date, shift_status: 'working', shift_id: shiftId, leave_reason: null,
      });
    },
    onSuccess: () => { queryClient.invalidateQueries({ queryKey: ['schedules'] }); },
  });

  const deleteShift = useMutation({
    mutationFn: async (shiftId: string) => {
      await api.delete(`/schedules/shifts/${shiftId}`);
    },
    onSuccess: () => { queryClient.invalidateQueries({ queryKey: ['schedules'] }); },
    onError: (err: any) => {
      alert("Error deleting shift: " + (err?.response?.data?.error || err.message));
    }
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
                {shifts?.map((s: any) => <option key={s.id} value={s.id}>{s.name} ({s.shift_code})</option>)}
              </select>
            </div>
          </div>

          <div className="grid grid-cols-3 sm:grid-cols-5 gap-2 sm:gap-3 w-full">
            <Stat label="Total" value={stats.total} tone="default" />
            <Stat label="Working" value={stats.working} tone="emerald" />
            <Stat label="Off" value={stats.off} tone="amber" />
            <Stat label="Leave" value={stats.leave} tone="rose" />
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
                  {(employees || []).filter((e: any) => e.status === 'active').map((e: any) => <option key={e.id} value={e.id}>{e.first_name} {e.last_name} — {e.employee_code}</option>)}
                </select>
              </div>
              <div className="space-y-2">
                <Label>Status</Label>
                <select className={selectClass} value={setShiftStatus} onChange={(e) => setSetShiftStatus(e.target.value as any)}>
                  <option value="working">Working</option>
                  <option value="off">Off</option>
                </select>
              </div>
              <div className="space-y-2">
                <Label>Shift</Label>
                <select className={selectClass} value={setShiftId} onChange={(e) => setSetShiftId(e.target.value)} disabled={setShiftStatus !== 'working'}>
                  <option value="">{setShiftStatus === 'working' ? 'Select shift…' : 'Not required'}</option>
                  {shifts?.map((s: any) => <option key={s.id} value={s.id}>{s.name} ({s.shift_code})</option>)}
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
                const working = rows.filter((r: any) => r.shift_status === 'working').length;
                const off = rows.filter((r: any) => r.shift_status === 'off').length;
                const leave = rows.filter((r: any) => r.shift_status === 'leave').length;
                const vacation = rows.filter((r: any) => r.shift_status === 'vacation').length;

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
                        <span className="text-blue-500">Vacation: {vacation}</span>
                      </div>
                    </div>

                    <div className="divide-y divide-border">
                      {rows.map((es: any) => {
                        const emp = employeeMap[es.employee_id];
                        const name = emp ? `${emp.first_name} ${emp.last_name}` : 'Unknown Employee';
                        const code = emp?.employee_code || '';
                        const status = (es.shift_status || '—') as string;

                        const statusTone =
                          status === 'working' ? 'text-emerald-500' :
                          status === 'off' ? 'text-amber-500' :
                          status === 'leave' ? 'text-rose-500' :
                          status === 'vacation' ? 'text-blue-500' :
                          status === 'hourly' ? 'text-blue-500' :
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
        <CardHeader className="pb-3 border-b border-border">
          <CardTitle className="text-xl">Weekly Team Roster</CardTitle>
          <CardDescription>
            One row per employee with day-by-day status. Quick "Off" action included.
          </CardDescription>
        </CardHeader>
        <CardContent className="p-0">
          {weeklyLoading ? (
            <div className="flex items-center justify-center py-12">
              <Loader2 className="w-7 h-7 animate-spin text-muted-foreground" />
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
                  {filteredWeeklyRows.map((row: any) => (
                    <tr key={row.employee.id} className="border-b border-border align-top">
                      <td className="p-3">
                        <div className="text-foreground font-medium">
                          {row.employee.first_name} {row.employee.last_name}
                        </div>
                        <div className="text-xs text-muted-foreground">{row.employee.employee_code}</div>
                      </td>
                      {row.days.map((d: any) => {
                        const tone =
                          d.status === 'working' ? 'text-emerald-500 border-emerald-500/30 bg-emerald-500/5'
                            : d.status === 'off' ? 'text-amber-500 border-amber-500/30 bg-amber-500/5'
                              : d.status === 'leave' ? 'text-rose-500 border-rose-500/30 bg-rose-500/5'
                                : d.status === 'vacation' ? 'text-blue-500 border-blue-500/30 bg-blue-500/5'
                                  : d.status === 'hourly' ? 'text-blue-500 border-blue-500/30 bg-blue-500/5'
                                    : 'text-muted-foreground border-border bg-muted/20';

                        return (
                          <td key={`${row.employee.id}-${d.date}`} className="p-3">
                            <div className={`rounded-lg border px-2 py-2 ${tone}`}>
                              <div className="text-xs font-semibold uppercase tracking-wider">{d.status}</div>
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
                                      {shifts?.map((s: any) => (
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
            {pendingLeaves?.filter((l: any) => l.leave_type === 'annual' || l.leave_type === 'vacation').length > 0 ? (
              pendingLeaves
                .filter((l: any) => l.leave_type === 'annual' || l.leave_type === 'vacation')
                .map((leave: any) => (
                  <div key={leave.id} className="rounded-lg border border-amber-500/30 bg-amber-500/5 p-3 text-sm">
                    <span className="text-amber-500 font-medium">Vacation request:</span>{' '}
                    <span className="text-foreground font-semibold">
                      {leave.employee_name ||
                        (employeeMap[leave.employee_id]
                          ? `${employeeMap[leave.employee_id].first_name} ${employeeMap[leave.employee_id].last_name}`
                          : `Employee #${String(leave.employee_id).slice(0, 8)}`)}
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

const Stat = ({ label, value, tone }: { label: string; value: number; tone: 'default' | 'emerald' | 'amber' | 'rose' | 'blue' }) => {
  const toneMap: Record<string, string> = {
    default: 'text-foreground bg-muted/40 border-border',
    emerald: 'text-emerald-500 bg-emerald-500/5 border-emerald-500/20',
    amber: 'text-amber-500 bg-amber-500/5 border-amber-500/20',
    rose: 'text-rose-500 bg-rose-500/5 border-rose-500/20',
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
      return res.data?.data || [];
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
            {replacements?.map((e: any) => (
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
