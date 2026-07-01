import { useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import api from '@/lib/api';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import {
  CheckSquare, ChevronLeft, ChevronRight, Calendar, Play,
  CheckCircle2, Clock, Loader2, ArrowRight, Timer, AlertTriangle, X,
  LayoutGrid, ClipboardList,
} from 'lucide-react';
import { format, startOfWeek, addDays, subDays, isToday, isBefore } from 'date-fns';
import { useAuthStore } from '@/store/authStore';
import { RichTextEditor } from '@/components/RichTextEditor';

// ───────────────────────────────────────────────────────────
// Types
// ───────────────────────────────────────────────────────────

interface MyTask {
  assignment_id: string;
  assigned_date: string;
  task_title: string;
  task_description: string | null;
  board_name: string | null;
  shift_name: string | null;
  shift_code: string | null;
  shift_color: string | null;
  execution_id: string | null;
  status: 'pending' | 'in_progress' | 'completed';
  completion_type: string | null;
  started_at: string | null;
  completed_at: string | null;
  notes: string | null;
}

// ───────────────────────────────────────────────────────────
// Completion Dialog
// ───────────────────────────────────────────────────────────

const CompletionDialog = ({
  task,
  onClose,
  onComplete,
  isPending,
}: {
  task: MyTask;
  onClose: () => void;
  onComplete: (completionType: string, notes: string) => void;
  isPending: boolean;
}) => {
  const [notes, setNotes] = useState('');

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50 backdrop-blur-sm">
      <div className="bg-card border border-border rounded-2xl shadow-2xl w-full max-w-md mx-4 overflow-hidden">
        {/* Header */}
        <div className="px-6 py-4 border-b border-border flex items-center justify-between">
          <div>
            <h3 className="text-lg font-semibold text-foreground">Complete Task</h3>
            <p className="text-sm text-muted-foreground">{task.task_title}</p>
          </div>
          <button onClick={onClose} className="p-1.5 rounded-lg hover:bg-muted transition-colors">
            <X className="w-5 h-5 text-muted-foreground" />
          </button>
        </div>

        {/* Notes */}
        <div className="px-6 py-4 space-y-4">
          <div>
            <label className="block text-sm font-medium text-foreground mb-2">Notes & Attachments (optional)</label>
            <div className="overflow-y-auto max-h-[30vh]">
              <RichTextEditor
                value={notes}
                onChange={setNotes}
                placeholder="Add any remarks, details, or paste screenshots..."
                height={200}
              />
            </div>
          </div>

          {/* Completion Buttons */}
          <div className="space-y-2.5">
            <Button
              onClick={() => onComplete('without_issue', notes)}
              disabled={isPending}
              className="w-full h-12 bg-emerald-500 hover:bg-emerald-400 text-white gap-2.5 text-sm font-semibold rounded-xl"
            >
              <CheckCircle2 className="w-5 h-5" />
              Complete — Without Issue
            </Button>
            <Button
              onClick={() => onComplete('with_issue', notes)}
              disabled={isPending}
              variant="outline"
              className="w-full h-12 border-amber-500/40 text-amber-600 hover:bg-amber-500/10 gap-2.5 text-sm font-semibold rounded-xl"
            >
              <AlertTriangle className="w-5 h-5" />
              Complete — With Issue
            </Button>
          </div>
        </div>
      </div>
    </div>
  );
};

// ───────────────────────────────────────────────────────────
// Component
// ───────────────────────────────────────────────────────────

export const MyTasksWeekly = () => {
  const queryClient = useQueryClient();
  const { user } = useAuthStore();

  const [view, setView] = useState<'weekly' | 'boards'>('weekly');
  const [weekStart, setWeekStart] = useState(() => startOfWeek(new Date(), { weekStartsOn: 0 }));
  const [expandedDay, setExpandedDay] = useState<number | null>(() => new Date().getDay());
  const [completingTask, setCompletingTask] = useState<MyTask | null>(null);

  const weekStartStr = format(weekStart, 'yyyy-MM-dd');
  const weekDates = Array.from({ length: 7 }, (_, i) => addDays(weekStart, i));

  // ── Query ──
  const { data: tasks, isLoading } = useQuery({
    queryKey: ['my-weekly-tasks', weekStartStr],
    queryFn: async () => {
      const res = await api.get(`/tasks/my-week?week_start=${weekStartStr}`);
      return (res.data?.data || []) as MyTask[];
    },
  });

  const { data: leaves } = useQuery({
    queryKey: ['leaves', 'me'],
    queryFn: async () => {
      const response = await api.get('/leaves/me');
      return response.data?.data || [];
    },
  });

  const weekEndStr = format(addDays(weekStart, 6), 'yyyy-MM-dd');
  const { data: scheduleRows } = useQuery({
    queryKey: ['my-schedule', user?.id, weekStartStr, weekEndStr],
    queryFn: async () => {
      if (!user?.id) return [];
      const res = await api.get(`/schedules/employee/${user?.id}?from=${weekStartStr}&to=${weekEndStr}`);
      return res.data?.data || [];
    },
    enabled: !!user?.id,
  });

  // ── Mutations ──
  const startTask = useMutation({
    mutationFn: async (executionId: string) => { await api.post(`/tasks/executions/${executionId}/start`); },
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ['my-weekly-tasks'] }),
  });

  const completeTask = useMutation({
    mutationFn: async ({ executionId, completionType, notes }: { executionId: string; completionType: string; notes?: string }) => {
      await api.post(`/tasks/executions/${executionId}/complete`, {
        completion_type: completionType,
        notes: notes || null,
      });
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['my-weekly-tasks'] });
      setCompletingTask(null);
    },
  });

  // ── Group tasks by day ──
  const tasksByDay: Record<string, MyTask[]> = {};
  weekDates.forEach((d) => { tasksByDay[format(d, 'yyyy-MM-dd')] = []; });
  tasks?.forEach((t) => {
    const key = t.assigned_date.split('T')[0];
    if (tasksByDay[key]) tasksByDay[key].push(t);
  });

  // ── Stats ──
  const totalTasks = tasks?.length || 0;
  const completedTasks = tasks?.filter((t) => t.status === 'completed').length || 0;
  const inProgressTasks = tasks?.filter((t) => t.status === 'in_progress').length || 0;
  const completionPct = totalTasks > 0 ? Math.round((completedTasks / totalTasks) * 100) : 0;

  const statusConfig = {
    pending: { icon: <Clock className="w-4 h-4" />, color: 'text-muted-foreground', bg: 'bg-muted/30 border-border', label: 'Pending' },
    in_progress: { icon: <Play className="w-4 h-4" />, color: 'text-amber-500', bg: 'bg-amber-500/10 border-amber-500/20', label: 'In Progress' },
    completed: { icon: <CheckCircle2 className="w-4 h-4" />, color: 'text-emerald-500', bg: 'bg-emerald-500/10 border-emerald-500/20', label: 'Completed' },
  };

  const formatDuration = (startedAt: string, completedAt?: string | null) => {
    const start = new Date(startedAt);
    const end = completedAt ? new Date(completedAt) : new Date();
    const diff = Math.floor((end.getTime() - start.getTime()) / 1000);
    if (diff < 60) return `${diff}s`;
    if (diff < 3600) return `${Math.floor(diff / 60)}m`;
    const hrs = Math.floor(diff / 3600);
    const mins = Math.floor((diff % 3600) / 60);
    return `${hrs}h ${mins}m`;
  };

  return (
    <div className="space-y-6">
      {/* ── Completion Dialog ── */}
      {completingTask && (
        <CompletionDialog
          task={completingTask}
          onClose={() => setCompletingTask(null)}
          onComplete={(completionType, notes) => {
            if (completingTask.execution_id) {
              completeTask.mutate({ executionId: completingTask.execution_id, completionType, notes });
            }
          }}
          isPending={completeTask.isPending}
        />
      )}

      {/* ── Header ── */}
      <div className="flex items-center justify-between">
        <div>
          <h2 className="text-2xl sm:text-3xl font-bold tracking-tight text-foreground mb-1 flex items-center gap-2 sm:gap-3">
            <CheckSquare className="w-6 h-6 sm:w-8 sm:h-8 text-primary" />
            My Tasks
          </h2>
          <p className="text-muted-foreground">Your weekly task assignments and progress</p>
        </div>
      </div>

      {/* ── Stats Bar ── */}
      <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
        <StatCard label="Total Tasks" value={totalTasks} color="text-foreground" icon={<CheckSquare className="w-5 h-5 text-primary" />} />
        <StatCard label="Completed" value={completedTasks} color="text-emerald-500" icon={<CheckCircle2 className="w-5 h-5 text-emerald-500" />} />
        <StatCard label="In Progress" value={inProgressTasks} color="text-amber-500" icon={<Play className="w-5 h-5 text-amber-500" />} />
        <StatCard label="Completion" value={`${completionPct}%`} color="text-primary" icon={<Timer className="w-5 h-5 text-primary" />} />
      </div>

      {/* ── View Tabs ── */}
      <div className="flex items-center gap-2 p-1 bg-muted/30 rounded-xl border border-border w-fit">
        <button
          onClick={() => setView('weekly')}
          className={`flex items-center gap-2 px-4 py-2 rounded-lg text-sm font-medium transition-all duration-200 ${
            view === 'weekly' ? 'bg-card text-primary shadow-sm border border-border' : 'text-muted-foreground hover:text-foreground'
          }`}
        >
          <Calendar className="w-4 h-4" /> Weekly View
        </button>
        <button
          onClick={() => setView('boards')}
          className={`flex items-center gap-2 px-4 py-2 rounded-lg text-sm font-medium transition-all duration-200 ${
            view === 'boards' ? 'bg-card text-primary shadow-sm border border-border' : 'text-muted-foreground hover:text-foreground'
          }`}
        >
          <LayoutGrid className="w-4 h-4" /> Board View
        </button>
      </div>

      {/* ── Week Navigation ── */}
      <div className="flex items-center justify-between bg-muted/30 rounded-xl p-3 border border-border">
        <Button variant="outline" size="sm" onClick={() => setWeekStart((prev) => subDays(prev, 7))} className="gap-2">
          <ChevronLeft className="w-4 h-4" /> Previous
        </Button>
        <span className="text-sm text-foreground font-semibold flex items-center gap-2">
          <Calendar className="w-4 h-4 text-muted-foreground" />
          {format(weekStart, 'MMM d')} – {format(addDays(weekStart, 6), 'MMM d, yyyy')}
        </span>
        <Button variant="outline" size="sm" onClick={() => setWeekStart((prev) => addDays(prev, 7))} className="gap-2">
          Next <ChevronRight className="w-4 h-4" />
        </Button>
      </div>

      {/* ── Loading/Data ── */}
      {isLoading ? (
        <div className="flex justify-center py-16">
          <Loader2 className="w-8 h-8 animate-spin text-muted-foreground" />
        </div>
      ) : totalTasks === 0 ? (
        <div className="py-16 text-center border border-border border-dashed rounded-xl bg-muted/10">
          <CheckSquare className="w-16 h-16 mx-auto mb-4 text-muted-foreground/30" />
          <h3 className="text-xl font-semibold text-muted-foreground mb-2">No tasks this week</h3>
          <p className="text-muted-foreground">You're clear! Check back later or navigate to a different week.</p>
        </div>
      ) : view === 'boards' ? (
        <BoardView tasks={tasks || []} statusConfig={statusConfig} formatDuration={formatDuration}
          onStart={(id) => startTask.mutate(id)} onComplete={setCompletingTask}
          startPending={startTask.isPending} />
      ) : (
        <div className="space-y-3">
          {weekDates.map((date, dayIdx) => {
            const dateKey = format(date, 'yyyy-MM-dd');
            const dayTasks = tasksByDay[dateKey] || [];
            const dayCompleted = dayTasks.filter((t) => t.status === 'completed').length;
            const dayTotal = dayTasks.length;
            const isExpanded = expandedDay === dayIdx;
            const today = isToday(date);
            const isPast = isBefore(date, new Date()) && !today;

            const dayLeave = leaves?.find((l: any) => {
              if (l.status !== 'approved_by_manager' && l.status !== 'approved_by_team_leader') return false;
              const sDate = l.start_date?.split('T')[0];
              if (!sDate) return false;
              const eDate = l.end_date ? l.end_date.split('T')[0] : sDate;
              return dateKey >= sDate && dateKey <= eDate;
            });

            const dayOffSchedule = scheduleRows?.find((s: any) => s.shift_date?.startsWith(dateKey) && ['off', 'vacation', 'leave'].includes(s.shift_status));
            
            const isHourlyLeave = dayLeave?.leave_type_name_en?.toLowerCase() === 'hourly';
            const isOff = (!!dayLeave && !isHourlyLeave) || !!dayOffSchedule;

            return (
              <Card key={dayIdx} className={`transition-all duration-300 overflow-hidden ${today ? 'border-primary/30 shadow-[0_0_30px_rgba(12,204,204,0.06)]' : ''}`}>
                <button onClick={() => setExpandedDay(isExpanded ? null : dayIdx)}
                  className="w-full text-left px-5 py-4 flex items-center justify-between group">
                  <div className="flex items-center gap-4">
                    <div className={`w-12 h-12 rounded-xl flex flex-col items-center justify-center text-xs font-bold ${
                      today ? 'bg-primary/15 text-primary border border-primary/20'
                        : isPast ? 'bg-muted/40 text-muted-foreground border border-border'
                        : 'bg-muted/60 text-foreground border border-border'
                    }`}>
                      <span className="text-[10px] uppercase">{format(date, 'EEE')}</span>
                      <span className="text-lg leading-none">{format(date, 'd')}</span>
                    </div>
                    <div>
                      <div className="flex items-center gap-2">
                        <span className={`font-semibold ${today ? 'text-primary' : 'text-foreground'}`}>{format(date, 'EEEE')}</span>
                        {today && <span className="px-2 py-0.5 rounded-full text-[10px] font-bold bg-primary/15 text-primary border border-primary/20 uppercase tracking-wider">Today</span>}
                      </div>
                      <span className="text-xs text-muted-foreground">{format(date, 'MMMM d, yyyy')}</span>
                    </div>
                  </div>
                  <div className="flex items-center gap-2 sm:gap-3">
                    {dayTotal > 0 ? (
                      <div className="flex items-center gap-2">
                        {dayLeave && isHourlyLeave && (
                          <span className="text-[10px] font-semibold px-1.5 py-0.5 rounded bg-amber-500/10 text-amber-600 border border-amber-500/20 mr-1">
                            ⏳ {dayLeave.leave_type_name_ar || dayLeave.leave_type_name_en || 'زمنية'}
                          </span>
                        )}
                        <div className="w-24 h-2 bg-muted rounded-full overflow-hidden">
                          <div className="h-full bg-gradient-to-r from-primary to-accent rounded-full transition-all duration-500"
                            style={{ width: `${dayTotal > 0 ? (dayCompleted / dayTotal) * 100 : 0}%` }} />
                        </div>
                        <span className="text-xs text-muted-foreground min-w-[40px]">{dayCompleted}/{dayTotal}</span>
                      </div>
                    ) : isOff ? (
                      <span className="text-xs font-semibold px-2 py-1 rounded bg-emerald-500/10 text-emerald-600 border border-emerald-500/20">🌴 {dayLeave?.leave_type_name_ar || dayLeave?.leave_type_name_en || 'Off'}</span>
                    ) : dayLeave && isHourlyLeave ? (
                      <span className="text-xs font-semibold px-2 py-1 rounded bg-amber-500/10 text-amber-600 border border-amber-500/20">⏳ {dayLeave.leave_type_name_ar || dayLeave.leave_type_name_en || 'زمنية'}</span>
                    ) : (
                      <span className="text-xs text-muted-foreground/60">No tasks</span>
                    )}
                    <ArrowRight className={`w-4 h-4 text-muted-foreground/60 transition-transform duration-200 ${isExpanded ? 'rotate-90' : ''}`} />
                  </div>
                </button>

                {/* Task List */}
                {isExpanded && dayTotal > 0 && (
                  <CardContent className="pt-0 pb-4 px-5">
                    <div className="space-y-2 border-t border-border pt-4">
                      {dayTasks.map((task) => (
                        <TaskRow key={task.assignment_id} task={task} statusConfig={statusConfig}
                          formatDuration={formatDuration} onStart={(id) => startTask.mutate(id)}
                          onComplete={setCompletingTask} startPending={startTask.isPending} />
                      ))}
                    </div>
                  </CardContent>
                )}

                {/* Empty day / Off day */}
                {isExpanded && dayTotal === 0 && (
                  <CardContent className="pt-0 pb-4 px-5">
                    <div className="text-center py-6 border-t border-border">
                      {isOff ? (
                        <div className="animate-in fade-in zoom-in duration-300">
                          <div className="text-4xl mb-3">🌴</div>
                          <p className="text-base font-bold text-emerald-600">{dayLeave?.leave_type_name_ar || dayLeave?.leave_type_name_en || 'Off / Happy Holiday!'}</p>
                          <p className="text-sm text-muted-foreground mt-1">Enjoy your time off.</p>
                        </div>
                      ) : dayLeave && isHourlyLeave ? (
                        <div className="animate-in fade-in zoom-in duration-300">
                          <div className="text-4xl mb-3">⏳</div>
                          <p className="text-base font-bold text-amber-600">{dayLeave.leave_type_name_ar || dayLeave.leave_type_name_en || 'زمنية'}</p>
                          <p className="text-sm text-muted-foreground mt-1">Short leave approved for this day.</p>
                        </div>
                      ) : (
                        <p className="text-sm text-muted-foreground/60">No tasks assigned for this day</p>
                      )}
                    </div>
                  </CardContent>
                )}
              </Card>
            );
          })}
        </div>
      )}
    </div>
  );
};

// ───────────────────────────────────────────────────────────
// Stat Card
// ───────────────────────────────────────────────────────────

const StatCard = ({ label, value, color, icon }: { label: string; value: string | number; color: string; icon: React.ReactNode }) => (
  <Card>
    <CardContent className="p-4 flex items-center gap-2 sm:gap-3">
      <div className="p-2 rounded-xl bg-muted/60">{icon}</div>
      <div>
        <p className="text-xs text-muted-foreground uppercase tracking-wider">{label}</p>
        <p className={`text-xl font-bold ${color}`}>{value}</p>
      </div>
    </CardContent>
  </Card>
);

// ───────────────────────────────────────────────────────────
// Task Row (shared)
// ───────────────────────────────────────────────────────────

const TaskRow = ({
  task, statusConfig, formatDuration, onStart, onComplete, startPending,
}: {
  task: MyTask;
  statusConfig: Record<string, { icon: React.ReactNode; color: string; bg: string; label: string }>;
  formatDuration: (s: string, e?: string | null) => string;
  onStart: (id: string) => void;
  onComplete: (t: MyTask) => void;
  startPending: boolean;
}) => {
  const cfg = statusConfig[task.status];
  return (
    <div className={`flex items-center gap-4 px-4 py-3 rounded-xl border transition-all duration-200 ${cfg.bg}`}>
      <div className={`flex-shrink-0 ${cfg.color}`}>{cfg.icon}</div>
      <div className="flex-1 min-w-0">
        <div className="flex items-center gap-2 mb-0.5">
          <span className="font-medium text-foreground truncate">{task.task_title}</span>
          {task.board_name && (
            <span className="px-1.5 py-0.5 rounded text-[10px] font-medium bg-primary/10 text-primary border border-primary/20 flex-shrink-0">{task.board_name}</span>
          )}
        </div>
        <div className="flex items-center gap-3 text-xs flex-wrap">
          {task.shift_name && (
            <span className="flex items-center gap-1">
              <span className="w-2 h-2 rounded-full" style={{ backgroundColor: task.shift_color || '#0CCCCC' }} />
              <span className="text-muted-foreground">{task.shift_name}</span>
            </span>
          )}
          {task.started_at && (
            <span className="text-muted-foreground flex items-center gap-1">
              <Timer className="w-3 h-3" />{formatDuration(task.started_at, task.completed_at)}
            </span>
          )}
          {task.completed_at && <span className="text-emerald-500/70">Done at {format(new Date(task.completed_at), 'hh:mm a')}</span>}
          {task.completion_type === 'without_issue' && <span className="px-1.5 py-0.5 rounded text-[10px] font-medium bg-emerald-500/10 text-emerald-600 border border-emerald-500/20">✓ No Issues</span>}
          {task.completion_type === 'with_issue' && <span className="px-1.5 py-0.5 rounded text-[10px] font-medium bg-amber-500/10 text-amber-600 border border-amber-500/20">⚠ Has Issues</span>}
        </div>
        {task.notes && task.status === 'completed' && (
          <div className="mt-2 pt-2 border-t border-border/50 text-xs text-muted-foreground italic jodit-content max-w-full overflow-hidden" dangerouslySetInnerHTML={{ __html: task.notes }} />
        )}
      </div>
      {task.execution_id && task.status === 'pending' && (
        <Button size="sm" onClick={() => onStart(task.execution_id!)} disabled={startPending}
          className="bg-amber-500 hover:bg-amber-400 text-white gap-1.5 text-xs h-8">
          <Play className="w-3.5 h-3.5" /> Start
        </Button>
      )}
      {task.execution_id && task.status === 'in_progress' && (
        <Button size="sm" onClick={() => onComplete(task)}
          className="bg-emerald-500 hover:bg-emerald-400 text-white gap-1.5 text-xs h-8">
          <CheckCircle2 className="w-3.5 h-3.5" /> Complete
        </Button>
      )}
      {task.status === 'completed' && (
        <span className="text-emerald-500 text-xs font-medium px-2.5 py-1 rounded-lg bg-emerald-500/5">✓ Done</span>
      )}
    </div>
  );
};

// ───────────────────────────────────────────────────────────
// Board View — groups tasks by board name
// ───────────────────────────────────────────────────────────

const BoardView = ({
  tasks, statusConfig, formatDuration, onStart, onComplete, startPending,
}: {
  tasks: MyTask[];
  statusConfig: Record<string, { icon: React.ReactNode; color: string; bg: string; label: string }>;
  formatDuration: (s: string, e?: string | null) => string;
  onStart: (id: string) => void;
  onComplete: (t: MyTask) => void;
  startPending: boolean;
}) => {
  // Group by board_name (null => 'Unassigned')
  const grouped: Record<string, MyTask[]> = {};
  tasks.forEach((t) => {
    const key = t.board_name || 'No Board';
    if (!grouped[key]) grouped[key] = [];
    grouped[key].push(t);
  });

  if (Object.keys(grouped).length === 0) {
    return (
      <div className="py-16 text-center border border-border border-dashed rounded-xl bg-muted/10">
        <ClipboardList className="w-16 h-16 mx-auto mb-4 text-muted-foreground/30" />
        <h3 className="text-xl font-semibold text-muted-foreground mb-2">No board tasks this week</h3>
        <p className="text-muted-foreground">Your team leader hasn't assigned any board tasks yet.</p>
      </div>
    );
  }

  return (
    <div className="space-y-4">
      {Object.entries(grouped).map(([boardName, boardTasks]) => {
        const done = boardTasks.filter((t) => t.status === 'completed').length;
        const pct = Math.round((done / boardTasks.length) * 100);
        return (
          <Card key={boardName}>
            <CardHeader className="pb-3 border-b border-border">
              <CardTitle className="text-base flex items-center justify-between">
                <span className="flex items-center gap-2">
                  <ClipboardList className="w-4 h-4 text-primary" />
                  {boardName}
                </span>
                <div className="flex items-center gap-3">
                  <div className="w-28 h-2 bg-muted rounded-full overflow-hidden">
                    <div className="h-full bg-gradient-to-r from-primary to-accent rounded-full transition-all duration-500" style={{ width: `${pct}%` }} />
                  </div>
                  <span className="text-sm text-primary font-bold min-w-[36px] text-right">{pct}%</span>
                </div>
              </CardTitle>
            </CardHeader>
            <CardContent className="pt-4 space-y-2">
              {boardTasks.sort((a, b) => a.assigned_date.localeCompare(b.assigned_date)).map((task) => (
                <div key={task.assignment_id}>
                  <div className="text-[10px] text-muted-foreground/60 mb-1 ml-1">
                    {format(new Date(task.assigned_date.split('T')[0] + 'T00:00:00'), 'EEE, MMM d')}
                  </div>
                  <TaskRow task={task} statusConfig={statusConfig} formatDuration={formatDuration}
                    onStart={onStart} onComplete={onComplete} startPending={startPending} />
                </div>
              ))}
            </CardContent>
          </Card>
        );
      })}
    </div>
  );
};
