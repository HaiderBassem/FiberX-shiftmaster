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
    queryFn: async () => {
      const res = await api.get('/tasks/schedules');
      return res.data?.data || [];
    },
  });

  const { data: shifts } = useQuery({
    queryKey: ['shifts'],
    queryFn: async () => {
      const res = await api.get('/shifts');
      return res.data?.data || [];
    },
  });

  const { data: boards } = useQuery({
    queryKey: ['tasks', 'boards'],
    queryFn: async () => {
      const res = await api.get('/tasks/boards');
      return res.data?.data || [];
    },
  });

  // Eligible assignees based on selected shift + date
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
    queryFn: async () => {
      const res = await api.get(`/tasks/assignments?date=${assignDate}`);
      return res.data?.data || [];
    },
  });

  // ── Mutations ──
  const createSchedule = useMutation({
    mutationFn: async () => {
      await api.post('/tasks/schedules', {
        title,
        description: description || null,
        schedule_type: scheduleType,
        board_id: selectedBoardId || null,
        shift_id: selectedShiftId || null,
        recurrence,
        recurrence_days: recurrenceDays,
        max_assignees: maxAssignees,
        is_active: true,
      });
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['tasks'] });
      setTitle('');
      setDescription('');
      setRecurrenceDays([]);
      setSelectedBoardId('');
    },
  });

  const assignTask = useMutation({
    mutationFn: async () => {
      await api.post('/tasks/assign', {
        schedule_id: assignScheduleId,
        employee_id: assignEmployeeId,
        assigned_date: assignDate,
      });
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['tasks'] });
      setAssignEmployeeId('');
    },
  });

  const toggleActive = useMutation({
    mutationFn: async ({ id, active }: { id: string; active: boolean }) => {
      await api.patch(`/tasks/schedules/${id}/toggle`, { is_active: active });
    },
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ['tasks'] }),
  });

  const deleteSchedule = useMutation({
    mutationFn: async (id: string) => {
      await api.delete(`/tasks/schedules/${id}`);
    },
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ['tasks'] }),
  });

  const toggleDay = (day: number) => {
    setRecurrenceDays(prev =>
      prev.includes(day) ? prev.filter(d => d !== day) : [...prev, day]
    );
  };

  return (
    <div className="space-y-8">
      <div>
        <h2 className="text-3xl font-bold tracking-tight text-white mb-2 flex items-center gap-3">
          <ClipboardList className="w-8 h-8 text-indigo-400" />
          Task Management
        </h2>
        <p className="text-zinc-400">Create, assign, and schedule recurring tasks for your team.</p>
      </div>

      <div className="grid lg:grid-cols-3 gap-6">
        {/* ── Left: Create Task Schedule ── */}
        <div className="space-y-6">
          <Card className="bg-zinc-900/80 border-indigo-500/20 shadow-[0_0_30px_rgba(99,102,241,0.05)]">
            <CardHeader>
              <CardTitle className="flex items-center gap-2 text-indigo-100">
                <Plus className="w-5 h-5 text-indigo-400" />
                New Task Schedule
              </CardTitle>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="space-y-2">
                <Label className="text-indigo-200">Task Title</Label>
                <Input
                  value={title}
                  onChange={(e: React.ChangeEvent<HTMLInputElement>) => setTitle(e.target.value)}
                  placeholder="e.g. Clean lab equipment"
                  className="bg-black/20 border-indigo-500/20"
                />
              </div>
              <div className="space-y-2">
                <Label>Description</Label>
                <textarea
                  className="w-full h-20 px-3 py-2 rounded-md bg-zinc-950/50 border border-indigo-500/30 text-white focus:outline-none focus:ring-2 focus:ring-indigo-500 resize-none text-sm"
                  placeholder="Details about the task..."
                  value={description}
                  onChange={(e) => setDescription(e.target.value)}
                />
              </div>
              <div className="space-y-2">
                <Label>Board</Label>
                <select
                  className="w-full h-10 px-3 py-2 rounded-md bg-zinc-950/50 border border-indigo-500/30 text-white focus:outline-none focus:ring-2 focus:ring-indigo-500"
                  value={selectedBoardId}
                  onChange={(e) => setSelectedBoardId(e.target.value)}
                >
                  <option value="">No board (optional)</option>
                  {boards?.map((b: any) => (
                    <option key={b.id} value={b.id}>{b.name}</option>
                  ))}
                </select>
              </div>
              <div className="space-y-2">
                <Label>Shift</Label>
                <select
                  className="w-full h-10 px-3 py-2 rounded-md bg-zinc-950/50 border border-indigo-500/30 text-white focus:outline-none focus:ring-2 focus:ring-indigo-500"
                  value={selectedShiftId}
                  onChange={(e) => setSelectedShiftId(e.target.value)}
                >
                  <option value="">Any Shift</option>
                  {shifts?.map((s: any) => (
                    <option key={s.id} value={s.id}>{s.name} ({s.shift_code})</option>
                  ))}
                </select>
              </div>
              <div className="space-y-2">
                <Label>Recurrence</Label>
                <select
                  className="w-full h-10 px-3 py-2 rounded-md bg-zinc-950/50 border border-indigo-500/30 text-white focus:outline-none focus:ring-2 focus:ring-indigo-500"
                  value={recurrence}
                  onChange={(e) => setRecurrence(e.target.value)}
                >
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
                      <button
                        key={i}
                        onClick={() => toggleDay(i)}
                        className={`px-3 py-1.5 rounded-md text-xs font-medium transition-all ${
                          recurrenceDays.includes(i)
                            ? 'bg-indigo-600 text-white shadow-lg'
                            : 'bg-zinc-800 text-zinc-400 hover:bg-zinc-700'
                        }`}
                      >
                        {day.slice(0, 3)}
                      </button>
                    ))}
                  </div>
                </div>
              )}
              <div className="space-y-2">
                <Label>Max Assignees</Label>
                <Input
                  type="number"
                  min={1}
                  value={maxAssignees}
                  onChange={(e: React.ChangeEvent<HTMLInputElement>) => setMaxAssignees(parseInt(e.target.value) || 1)}
                  className="bg-black/20 border-indigo-500/20"
                />
              </div>
            </CardContent>
            <CardFooter>
              <Button
                className="w-full bg-indigo-600 hover:bg-indigo-500 text-white gap-2"
                onClick={() => createSchedule.mutate()}
                disabled={createSchedule.isPending || !title}
              >
                <Plus className="w-4 h-4" /> Create Task Schedule
              </Button>
            </CardFooter>
          </Card>

          {/* ── Quick Assign ── */}
          <Card className="bg-zinc-900/80 border-emerald-500/20">
            <CardHeader>
              <CardTitle className="flex items-center gap-2 text-emerald-100">
                <Users className="w-5 h-5 text-emerald-400" />
                Assign Task
              </CardTitle>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="space-y-2">
                <Label className="text-emerald-200">Task Schedule</Label>
                <select
                  className="w-full h-10 px-3 py-2 rounded-md bg-zinc-950/50 border border-emerald-500/30 text-white focus:outline-none focus:ring-2 focus:ring-emerald-500"
                  value={assignScheduleId}
                  onChange={(e) => {
                    setAssignScheduleId(e.target.value);
                    // Auto-select shift from task schedule
                    const sched = taskSchedules?.find((s: any) => s.id === e.target.value);
                    if (sched?.shift_id) {
                      setAssignShiftId(sched.shift_id);
                    }
                    setAssignEmployeeId('');
                  }}
                >
                  <option value="" disabled>Select a task schedule</option>
                  {taskSchedules?.filter((s: any) => s.is_active).map((s: any) => (
                    <option key={s.id} value={s.id}>{s.title} ({s.recurrence})</option>
                  ))}
                </select>
              </div>
              <div className="space-y-2">
                <Label>Shift</Label>
                <select
                  className="w-full h-10 px-3 py-2 rounded-md bg-zinc-950/50 border border-emerald-500/30 text-white focus:outline-none focus:ring-2 focus:ring-emerald-500"
                  value={assignShiftId}
                  onChange={(e) => { setAssignShiftId(e.target.value); setAssignEmployeeId(''); }}
                >
                  <option value="" disabled>Select a shift</option>
                  {shifts?.map((s: any) => (
                    <option key={s.id} value={s.id}>{s.name} ({s.shift_code})</option>
                  ))}
                </select>
              </div>
              <div className="space-y-2">
                <Label>Assignment Date</Label>
                <Input
                  type="date"
                  value={assignDate}
                  onChange={(e: React.ChangeEvent<HTMLInputElement>) => { setAssignDate(e.target.value); setAssignEmployeeId(''); }}
                  className="bg-black/20 border-emerald-500/20"
                />
              </div>
              <div className="space-y-2">
                <Label>Employee</Label>
                <select
                  className="w-full h-10 px-3 py-2 rounded-md bg-zinc-950/50 border border-emerald-500/30 text-white focus:outline-none focus:ring-2 focus:ring-emerald-500"
                  value={assignEmployeeId}
                  onChange={(e) => setAssignEmployeeId(e.target.value)}
                  disabled={assigneesLoading || !assignShiftId}
                >
                  <option value="" disabled>
                    {!assignShiftId ? 'Select a shift first' : 'Select an employee'}
                  </option>
                  {eligibleAssignees?.map((emp: any) => (
                    <option key={emp.id} value={emp.id}>
                      {emp.first_name} {emp.last_name} — {emp.employee_code}
                    </option>
                  ))}
                </select>
                {assignShiftId && eligibleAssignees?.length === 0 && !assigneesLoading && (
                  <p className="text-xs text-amber-400 mt-1">No eligible employees for this shift/date.</p>
                )}
              </div>
            </CardContent>
            <CardFooter>
              <Button
                className="w-full bg-emerald-600 hover:bg-emerald-500 text-white gap-2"
                onClick={() => assignTask.mutate()}
                disabled={assignTask.isPending || !assignScheduleId || !assignEmployeeId}
              >
                <CheckSquare className="w-4 h-4" /> Assign
              </Button>
            </CardFooter>
          </Card>
        </div>

        {/* ── Right: Task Schedules + Assignments ── */}
        <div className="lg:col-span-2 space-y-6">
          <h3 className="text-xl font-semibold text-white">Task Schedules</h3>

          {schedulesLoading ? (
            <div className="grid gap-4 md:grid-cols-2">
              {[1,2,3].map(i => <Card key={i} className="animate-pulse bg-zinc-900/50 h-32" />)}
            </div>
          ) : (
            <div className="grid gap-4 md:grid-cols-2">
              {taskSchedules?.map((ts: any) => (
                <Card key={ts.id} className={`border-zinc-800/60 transition-all ${ts.is_active ? 'bg-zinc-900/40' : 'bg-zinc-900/20 opacity-60'}`}>
                  <CardHeader className="pb-2">
                    <div className="flex items-start justify-between">
                      <CardTitle className="text-base flex items-center gap-2">
                        <CheckSquare className="w-4 h-4 text-indigo-400" />
                        {ts.title}
                      </CardTitle>
                      <div className="flex gap-1">
                        <button
                          onClick={() => toggleActive.mutate({ id: ts.id, active: !ts.is_active })}
                          className="p-1 rounded hover:bg-zinc-800 transition-colors"
                          title={ts.is_active ? 'Deactivate' : 'Activate'}
                        >
                          {ts.is_active ? <ToggleRight className="w-5 h-5 text-emerald-400" /> : <ToggleLeft className="w-5 h-5 text-zinc-500" />}
                        </button>
                        <button
                          onClick={() => deleteSchedule.mutate(ts.id)}
                          className="p-1 rounded hover:bg-red-500/10 transition-colors"
                          title="Delete"
                        >
                          <Trash2 className="w-4 h-4 text-red-400" />
                        </button>
                      </div>
                    </div>
                    <CardDescription className="text-zinc-500">
                      {ts.recurrence === 'daily' ? 'Every day' : 
                       ts.recurrence === 'weekly' ? `Days: ${ts.recurrence_days?.map((d: number) => DAYS[d]?.slice(0, 3)).join(', ')}` :
                       'One-time'}
                      {ts.max_assignees > 1 && ` · Up to ${ts.max_assignees} assignees`}
                    </CardDescription>
                  </CardHeader>
                  {ts.description && (
                    <CardContent className="pt-0">
                      <p className="text-sm text-zinc-400">{ts.description}</p>
                    </CardContent>
                  )}
                </Card>
              ))}
              {(!taskSchedules || taskSchedules.length === 0) && (
                <div className="col-span-full p-12 text-center text-zinc-500 border border-zinc-800/60 border-dashed rounded-xl bg-zinc-900/20">
                  <ClipboardList className="w-12 h-12 mx-auto mb-4 opacity-20" />
                  No task schedules yet. Create one to get started.
                </div>
              )}
            </div>
          )}

          {/* ── Daily Assignment View ── */}
          <div className="space-y-4 pt-4 border-t border-zinc-800/40">
            <div className="flex items-center justify-between">
              <h3 className="text-xl font-semibold text-white flex items-center gap-2">
                <Calendar className="w-5 h-5 text-amber-400" />
                Assignments for {format(new Date(assignDate), 'MMM d, yyyy')}
              </h3>
            </div>
            {dailyAssignments?.length > 0 ? (
              <div className="grid gap-3 md:grid-cols-2">
                {dailyAssignments.map((a: any) => (
                  <Card key={a.id} className="bg-zinc-900/30 border-zinc-800/40 p-4">
                    <div className="flex items-center justify-between">
                      <div>
                        <p className="text-sm font-medium text-white">Employee #{a.employee_id?.slice(0, 8)}</p>
                        <p className="text-xs text-zinc-500">Schedule #{a.schedule_id?.slice(0, 8)}</p>
                      </div>
                      <span className="text-xs px-2 py-1 rounded bg-indigo-500/10 text-indigo-400 border border-indigo-500/20">
                        Assigned
                      </span>
                    </div>
                  </Card>
                ))}
              </div>
            ) : (
              <p className="text-zinc-500 text-sm">No assignments for this date yet.</p>
            )}
          </div>
        </div>
      </div>
    </div>
  );
};
