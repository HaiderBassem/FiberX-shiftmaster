import { useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import api from '@/lib/api';
import { Card, CardContent, CardHeader, CardTitle, CardDescription, CardFooter } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { ClipboardList, Plus, Calendar, CheckSquare, Trash2, ToggleLeft, ToggleRight, Users } from 'lucide-react';
import { format } from 'date-fns';

const DAYS = ['Sunday', 'Monday', 'Tuesday', 'Wednesday', 'Thursday', 'Friday', 'Saturday'];

export const TaskManagement = () => {
  const queryClient = useQueryClient();

  // ── Form State ──
  const [title, setTitle] = useState('');
  const [description, setDescription] = useState('');
  const [scheduleType] = useState('daily_task');
  const [recurrence, setRecurrence] = useState('daily');
  const [recurrenceDays, setRecurrenceDays] = useState<number[]>([]);
  const [maxAssignees, setMaxAssignees] = useState(1);
  const [selectedShiftId, setSelectedShiftId] = useState('');
  const [selectedBoardId, setSelectedBoardId] = useState('');

  // Assignment form
  const [assignDate, setAssignDate] = useState(format(new Date(), 'yyyy-MM-dd'));
  const [assignScheduleId, setAssignScheduleId] = useState('');
  const [assignEmployeeId, setAssignEmployeeId] = useState('');
  const [assignShiftId, setAssignShiftId] = useState('');

  // ── Queries ──
  const { data: taskSchedules, isLoading: schedulesLoading } = useQuery({
    queryKey: ['tasks', 'schedules'],
    queryFn: async () => { const res = await api.get('/tasks/schedules'); return res.data?.data || []; },
  });

  const { data: shifts } = useQuery({
    queryKey: ['shifts'],
    queryFn: async () => { const res = await api.get('/shifts'); return res.data?.data || []; },
  });

  const { data: boards } = useQuery({
    queryKey: ['tasks', 'boards'],
    queryFn: async () => { const res = await api.get('/tasks/boards'); return res.data?.data || []; },
  });

  const { data: eligibleAssignees, isLoading: assigneesLoading } = useQuery({
    queryKey: ['tasks', 'eligibleAssignees', assignShiftId, assignDate],
    queryFn: async () => {
      if (!assignShiftId) return [];
      const res = await api.get(`/tasks/eligible-assignees?shift_id=${assignShiftId}&date=${assignDate}`);
      return res.data?.data || [];
    },
    enabled: !!assignShiftId && !!assignDate,
  });

  const { data: dailyAssignments } = useQuery({
    queryKey: ['tasks', 'assignments', assignDate],
    queryFn: async () => { const res = await api.get(`/tasks/assignments?date=${assignDate}`); return res.data?.data || []; },
  });

  // ── Mutations ──
  const createSchedule = useMutation({
    mutationFn: async () => {
      await api.post('/tasks/schedules', {
        title, description: description || null, schedule_type: scheduleType,
        board_id: selectedBoardId || null, shift_id: selectedShiftId || null,
        recurrence, recurrence_days: recurrenceDays, max_assignees: maxAssignees, is_active: true,
      });
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['tasks'] });
      setTitle(''); setDescription(''); setRecurrenceDays([]); setSelectedBoardId('');
    },
  });

  const assignTask = useMutation({
    mutationFn: async () => {
      await api.post('/tasks/assign', { schedule_id: assignScheduleId, employee_id: assignEmployeeId, assigned_date: assignDate });
    },
    onSuccess: () => { queryClient.invalidateQueries({ queryKey: ['tasks'] }); setAssignEmployeeId(''); },
  });

  const toggleActive = useMutation({
    mutationFn: async ({ id, active }: { id: string; active: boolean }) => {
      await api.patch(`/tasks/schedules/${id}/toggle`, { is_active: active });
    },
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ['tasks'] }),
  });

  const deleteSchedule = useMutation({
    mutationFn: async (id: string) => { await api.delete(`/tasks/schedules/${id}`); },
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ['tasks'] }),
  });

  const toggleDay = (day: number) => {
    setRecurrenceDays(prev => prev.includes(day) ? prev.filter(d => d !== day) : [...prev, day]);
  };

  const selectClass = "w-full h-10 px-3 py-2 rounded-lg bg-background border border-input text-foreground focus:outline-none focus:ring-2 focus:ring-ring transition-colors";

  return (
    <div className="space-y-8">
      <div>
        <h2 className="text-3xl font-bold tracking-tight text-foreground mb-2 flex items-center gap-3">
          <ClipboardList className="w-8 h-8 text-primary" />
          Task Management
        </h2>
        <p className="text-muted-foreground">Create, assign, and schedule recurring tasks for your team.</p>
      </div>

      <div className="grid lg:grid-cols-3 gap-6">
        {/* ── Left: Create Task Schedule ── */}
        <div className="space-y-6">
          <Card>
            <CardHeader>
              <CardTitle className="flex items-center gap-2">
                <Plus className="w-5 h-5 text-primary" />
                New Task Schedule
              </CardTitle>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="space-y-2">
                <Label>Task Title</Label>
                <Input value={title} onChange={(e: React.ChangeEvent<HTMLInputElement>) => setTitle(e.target.value)} placeholder="e.g. Clean lab equipment" />
              </div>
              <div className="space-y-2">
                <Label>Description</Label>
                <textarea
                  className="w-full h-20 px-3 py-2 rounded-lg bg-background border border-input text-foreground focus:outline-none focus:ring-2 focus:ring-ring resize-none text-sm transition-colors"
                  placeholder="Details about the task..."
                  value={description}
                  onChange={(e) => setDescription(e.target.value)}
                />
              </div>
              <div className="space-y-2">
                <Label>Board</Label>
                <select className={selectClass} value={selectedBoardId} onChange={(e) => setSelectedBoardId(e.target.value)}>
                  <option value="">No board (optional)</option>
                  {boards?.map((b: any) => <option key={b.id} value={b.id}>{b.name}</option>)}
                </select>
              </div>
              <div className="space-y-2">
                <Label>Shift</Label>
                <select className={selectClass} value={selectedShiftId} onChange={(e) => setSelectedShiftId(e.target.value)}>
                  <option value="">Any Shift</option>
                  {shifts?.map((s: any) => <option key={s.id} value={s.id}>{s.name} ({s.shift_code})</option>)}
                </select>
              </div>
              <div className="space-y-2">
                <Label>Recurrence</Label>
                <select className={selectClass} value={recurrence} onChange={(e) => setRecurrence(e.target.value)}>
                  <option value="daily">Daily</option>
                  <option value="weekly">Specific Days</option>
                  <option value="once">One-Time</option>
                </select>
              </div>
              {recurrence === 'weekly' && (
                <div className="space-y-2">
                  <Label>Select Days</Label>
                  <div className="flex flex-wrap gap-2">
                    {DAYS.map((day, i) => (
                      <button key={i} onClick={() => toggleDay(i)}
                        className={`px-3 py-1.5 rounded-md text-xs font-medium transition-all ${
                          recurrenceDays.includes(i) ? 'bg-primary text-primary-foreground shadow-lg' : 'bg-muted text-muted-foreground hover:bg-muted/80'
                        }`}
                      >{day.slice(0, 3)}</button>
                    ))}
                  </div>
                </div>
              )}
              <div className="space-y-2">
                <Label>Max Assignees</Label>
                <Input type="number" min={1} value={maxAssignees} onChange={(e: React.ChangeEvent<HTMLInputElement>) => setMaxAssignees(parseInt(e.target.value) || 1)} />
              </div>
            </CardContent>
            <CardFooter>
              <Button className="w-full gap-2" onClick={() => createSchedule.mutate()} disabled={createSchedule.isPending || !title}>
                <Plus className="w-4 h-4" /> Create Task Schedule
              </Button>
            </CardFooter>
          </Card>

          {/* ── Quick Assign ── */}
          <Card>
            <CardHeader>
              <CardTitle className="flex items-center gap-2">
                <Users className="w-5 h-5 text-primary" />
                Assign Task
              </CardTitle>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="space-y-2">
                <Label>Task Schedule</Label>
                <select className={selectClass} value={assignScheduleId}
                  onChange={(e) => {
                    setAssignScheduleId(e.target.value);
                    const sched = taskSchedules?.find((s: any) => s.id === e.target.value);
                    if (sched?.shift_id) setAssignShiftId(sched.shift_id);
                    setAssignEmployeeId('');
                  }}>
                  <option value="" disabled>Select a task schedule</option>
                  {taskSchedules?.filter((s: any) => s.is_active).map((s: any) => (
                    <option key={s.id} value={s.id}>{s.title} ({s.recurrence})</option>
                  ))}
                </select>
              </div>
              <div className="space-y-2">
                <Label>Shift</Label>
                <select className={selectClass} value={assignShiftId}
                  onChange={(e) => { setAssignShiftId(e.target.value); setAssignEmployeeId(''); }}>
                  <option value="" disabled>Select a shift</option>
                  {shifts?.map((s: any) => <option key={s.id} value={s.id}>{s.name} ({s.shift_code})</option>)}
                </select>
              </div>
              <div className="space-y-2">
                <Label>Assignment Date</Label>
                <Input type="date" value={assignDate}
                  onChange={(e: React.ChangeEvent<HTMLInputElement>) => { setAssignDate(e.target.value); setAssignEmployeeId(''); }} />
              </div>
              <div className="space-y-2">
                <Label>Employee</Label>
                <select className={selectClass} value={assignEmployeeId}
                  onChange={(e) => setAssignEmployeeId(e.target.value)} disabled={assigneesLoading || !assignShiftId}>
                  <option value="" disabled>{!assignShiftId ? 'Select a shift first' : 'Select an employee'}</option>
                  {eligibleAssignees?.map((emp: any) => (
                    <option key={emp.id} value={emp.id}>{emp.first_name} {emp.last_name} — {emp.employee_code}</option>
                  ))}
                </select>
                {assignShiftId && eligibleAssignees?.length === 0 && !assigneesLoading && (
                  <p className="text-xs text-amber-500 mt-1">No eligible employees for this shift/date.</p>
                )}
              </div>
            </CardContent>
            <CardFooter>
              <Button className="w-full gap-2" onClick={() => assignTask.mutate()}
                disabled={assignTask.isPending || !assignScheduleId || !assignEmployeeId}>
                <CheckSquare className="w-4 h-4" /> Assign
              </Button>
            </CardFooter>
          </Card>
        </div>

        {/* ── Right: Task Schedules + Assignments ── */}
        <div className="lg:col-span-2 space-y-6">
          <h3 className="text-xl font-semibold text-foreground">Task Schedules</h3>

          {schedulesLoading ? (
            <div className="grid gap-4 md:grid-cols-2">
              {[1,2,3].map(i => <Card key={i} className="animate-pulse h-32" />)}
            </div>
          ) : (
            <div className="grid gap-4 md:grid-cols-2">
              {taskSchedules?.map((ts: any) => (
                <Card key={ts.id} className={`transition-all ${ts.is_active ? '' : 'opacity-60'}`}>
                  <CardHeader className="pb-2">
                    <div className="flex items-start justify-between">
                      <CardTitle className="text-base flex items-center gap-2">
                        <CheckSquare className="w-4 h-4 text-primary" />
                        {ts.title}
                      </CardTitle>
                      <div className="flex gap-1">
                        <button onClick={() => toggleActive.mutate({ id: ts.id, active: !ts.is_active })}
                          className="p-1 rounded hover:bg-muted transition-colors"
                          title={ts.is_active ? 'Deactivate' : 'Activate'}>
                          {ts.is_active ? <ToggleRight className="w-5 h-5 text-emerald-500" /> : <ToggleLeft className="w-5 h-5 text-muted-foreground" />}
                        </button>
                        <button onClick={() => deleteSchedule.mutate(ts.id)}
                          className="p-1 rounded hover:bg-destructive/10 transition-colors" title="Delete">
                          <Trash2 className="w-4 h-4 text-destructive" />
                        </button>
                      </div>
                    </div>
                    <CardDescription>
                      {ts.recurrence === 'daily' ? 'Every day' : 
                       ts.recurrence === 'weekly' ? `Days: ${ts.recurrence_days?.map((d: number) => DAYS[d]?.slice(0, 3)).join(', ')}` :
                       'One-time'}
                      {ts.max_assignees > 1 && ` · Up to ${ts.max_assignees} assignees`}
                    </CardDescription>
                  </CardHeader>
                  {ts.description && (
                    <CardContent className="pt-0">
                      <p className="text-sm text-muted-foreground">{ts.description}</p>
                    </CardContent>
                  )}
                </Card>
              ))}
              {(!taskSchedules || taskSchedules.length === 0) && (
                <div className="col-span-full p-12 text-center text-muted-foreground border border-border border-dashed rounded-xl bg-muted/10">
                  <ClipboardList className="w-12 h-12 mx-auto mb-4 opacity-20" />
                  No task schedules yet. Create one to get started.
                </div>
              )}
            </div>
          )}

          {/* ── Daily Assignment View ── */}
          <div className="space-y-4 pt-4 border-t border-border">
            <div className="flex items-center justify-between">
              <h3 className="text-xl font-semibold text-foreground flex items-center gap-2">
                <Calendar className="w-5 h-5 text-amber-500" />
                Assignments for {format(new Date(assignDate), 'MMM d, yyyy')}
              </h3>
            </div>
            {dailyAssignments?.length > 0 ? (
              <div className="grid gap-3 md:grid-cols-2">
                {dailyAssignments.map((a: any) => (
                  <Card key={a.id} className="p-4">
                    <div className="flex items-center justify-between">
                      <div>
                        <p className="text-sm font-medium text-foreground">Employee #{a.employee_id?.slice(0, 8)}</p>
                        <p className="text-xs text-muted-foreground">Schedule #{a.schedule_id?.slice(0, 8)}</p>
                      </div>
                      <span className="text-xs px-2 py-1 rounded bg-primary/10 text-primary border border-primary/20">
                        Assigned
                      </span>
                    </div>
                  </Card>
                ))}
              </div>
            ) : (
              <p className="text-muted-foreground text-sm">No assignments for this date yet.</p>
            )}
          </div>
        </div>
      </div>
    </div>
  );
};
