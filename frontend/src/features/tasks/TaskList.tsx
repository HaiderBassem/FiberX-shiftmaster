import { useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import api from '@/lib/api';
import { Card, CardContent } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import {
  CheckSquare, ChevronLeft, ChevronRight, Calendar, Play,
  CheckCircle2, Clock, Loader2, ArrowRight, Timer
} from 'lucide-react';
import { format, startOfWeek, addDays, subDays, isToday, isBefore } from 'date-fns';

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
  started_at: string | null;
  completed_at: string | null;
  notes: string | null;
}

// ───────────────────────────────────────────────────────────
// Component
// ───────────────────────────────────────────────────────────

export const MyTasksWeekly = () => {
  const queryClient = useQueryClient();

  const [weekStart, setWeekStart] = useState(() => startOfWeek(new Date(), { weekStartsOn: 0 }));
  const [expandedDay, setExpandedDay] = useState<number | null>(() => new Date().getDay());

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

  // ── Mutations ──
  const startTask = useMutation({
    mutationFn: async (executionId: string) => { await api.post(`/tasks/executions/${executionId}/start`); },
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ['my-weekly-tasks'] }),
  });

  const completeTask = useMutation({
    mutationFn: async ({ executionId, notes }: { executionId: string; notes?: string }) => {
      await api.post(`/tasks/executions/${executionId}/complete`, { notes: notes || null });
    },
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ['my-weekly-tasks'] }),
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
      {/* ── Header ── */}
      <div className="flex items-center justify-between">
        <div>
          <h2 className="text-3xl font-bold tracking-tight text-foreground mb-1 flex items-center gap-3">
            <CheckSquare className="w-8 h-8 text-primary" />
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

      {/* ── Loading ── */}
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
      ) : (
        /* ── Day Cards ── */
        <div className="space-y-3">
          {weekDates.map((date, dayIdx) => {
            const dateKey = format(date, 'yyyy-MM-dd');
            const dayTasks = tasksByDay[dateKey] || [];
            const dayCompleted = dayTasks.filter((t) => t.status === 'completed').length;
            const dayTotal = dayTasks.length;
            const isExpanded = expandedDay === dayIdx;
            const today = isToday(date);
            const isPast = isBefore(date, new Date()) && !today;

            return (
              <Card
                key={dayIdx}
                className={`transition-all duration-300 overflow-hidden ${
                  today ? 'border-primary/30 shadow-[0_0_30px_rgba(12,204,204,0.06)]' : ''
                }`}
              >
                {/* Day Header */}
                <button
                  onClick={() => setExpandedDay(isExpanded ? null : dayIdx)}
                  className="w-full text-left px-5 py-4 flex items-center justify-between group"
                >
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
                        <span className={`font-semibold ${today ? 'text-primary' : 'text-foreground'}`}>
                          {format(date, 'EEEE')}
                        </span>
                        {today && (
                          <span className="px-2 py-0.5 rounded-full text-[10px] font-bold bg-primary/15 text-primary border border-primary/20 uppercase tracking-wider">
                            Today
                          </span>
                        )}
                      </div>
                      <span className="text-xs text-muted-foreground">{format(date, 'MMMM d, yyyy')}</span>
                    </div>
                  </div>
                  <div className="flex items-center gap-3">
                    {dayTotal > 0 ? (
                      <>
                        <div className="flex items-center gap-2">
                          <div className="w-24 h-2 bg-muted rounded-full overflow-hidden">
                            <div
                              className="h-full bg-gradient-to-r from-primary to-accent rounded-full transition-all duration-500"
                              style={{ width: `${dayTotal > 0 ? (dayCompleted / dayTotal) * 100 : 0}%` }}
                            />
                          </div>
                          <span className="text-xs text-muted-foreground min-w-[40px]">{dayCompleted}/{dayTotal}</span>
                        </div>
                      </>
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
                      {dayTasks.map((task) => {
                        const cfg = statusConfig[task.status];
                        return (
                          <div key={task.assignment_id}
                            className={`flex items-center gap-4 px-4 py-3 rounded-xl border transition-all duration-200 ${cfg.bg}`}>
                            <div className={`flex-shrink-0 ${cfg.color}`}>{cfg.icon}</div>
                            <div className="flex-1 min-w-0">
                              <div className="flex items-center gap-2 mb-0.5">
                                <span className="font-medium text-foreground truncate">{task.task_title}</span>
                                {task.board_name && (
                                  <span className="px-1.5 py-0.5 rounded text-[10px] font-medium bg-primary/10 text-primary border border-primary/20 flex-shrink-0">
                                    {task.board_name}
                                  </span>
                                )}
                              </div>
                              <div className="flex items-center gap-3 text-xs">
                                {task.shift_name && (
                                  <span className="flex items-center gap-1">
                                    <span className="w-2 h-2 rounded-full" style={{ backgroundColor: task.shift_color || '#0CCCCC' }} />
                                    <span className="text-muted-foreground">{task.shift_name}</span>
                                  </span>
                                )}
                                {task.started_at && (
                                  <span className="text-muted-foreground flex items-center gap-1">
                                    <Timer className="w-3 h-3" />
                                    {formatDuration(task.started_at, task.completed_at)}
                                  </span>
                                )}
                                {task.completed_at && (
                                  <span className="text-emerald-500/70">
                                    Done at {format(new Date(task.completed_at), 'hh:mm a')}
                                  </span>
                                )}
                              </div>
                            </div>

                            {/* Actions */}
                            {task.execution_id && task.status === 'pending' && (
                              <Button size="sm" onClick={() => startTask.mutate(task.execution_id!)}
                                disabled={startTask.isPending}
                                className="bg-amber-500 hover:bg-amber-400 text-white gap-1.5 text-xs h-8">
                                <Play className="w-3.5 h-3.5" /> Start
                              </Button>
                            )}
                            {task.execution_id && task.status === 'in_progress' && (
                              <Button size="sm" onClick={() => completeTask.mutate({ executionId: task.execution_id! })}
                                disabled={completeTask.isPending}
                                className="bg-emerald-500 hover:bg-emerald-400 text-white gap-1.5 text-xs h-8">
                                <CheckCircle2 className="w-3.5 h-3.5" /> Complete
                              </Button>
                            )}
                            {task.status === 'completed' && (
                              <span className="text-emerald-500 text-xs font-medium px-2.5 py-1 rounded-lg bg-emerald-500/5">
                                ✓ Done
                              </span>
                            )}
                          </div>
                        );
                      })}
                    </div>
                  </CardContent>
                )}

                {/* Collapsed empty day */}
                {isExpanded && dayTotal === 0 && (
                  <CardContent className="pt-0 pb-4 px-5">
                    <div className="text-center py-6 border-t border-border">
                      <p className="text-sm text-muted-foreground/60">No tasks assigned for this day</p>
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
    <CardContent className="p-4 flex items-center gap-3">
      <div className="p-2 rounded-xl bg-muted/60">{icon}</div>
      <div>
        <p className="text-xs text-muted-foreground uppercase tracking-wider">{label}</p>
        <p className={`text-xl font-bold ${color}`}>{value}</p>
      </div>
    </CardContent>
  </Card>
);
