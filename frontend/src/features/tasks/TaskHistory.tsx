import { useState } from 'react';
import { useQuery } from '@tanstack/react-query';
import api from '@/lib/api';
import { Card, CardContent } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import {
  History, ChevronLeft, ChevronRight, Calendar, Loader2,
  CheckCircle2, AlertTriangle, Clock, Play, Timer, User
} from 'lucide-react';
import { format, addDays, subDays } from 'date-fns';

// ───────────────────────────────────────────────────────────
// Types
// ───────────────────────────────────────────────────────────

interface TaskHistoryRow {
  assignment_id: string;
  execution_id: string;
  assigned_date: string;
  task_title: string;
  task_description: string | null;
  board_name: string | null;
  employee_id: string;
  employee_name: string;
  employee_code: string;
  status: string;
  completion_type: string | null;
  started_at: string | null;
  completed_at: string | null;
  notes: string | null;
}

interface TaskBoard {
  id: string;
  name: string;
}

// ───────────────────────────────────────────────────────────
// Component
// ───────────────────────────────────────────────────────────

export const TaskHistory = () => {
  const [selectedDate, setSelectedDate] = useState(() => new Date());
  const [selectedBoard, setSelectedBoard] = useState<string>('');
  const dateStr = format(selectedDate, 'yyyy-MM-dd');

  // Fetch boards for filter
  const { data: boards } = useQuery({
    queryKey: ['boards-list'],
    queryFn: async () => {
      const res = await api.get('/tasks/boards');
      return (res.data?.data || []) as TaskBoard[];
    },
  });

  // Fetch history
  const { data: history, isLoading } = useQuery({
    queryKey: ['task-history', dateStr, selectedBoard],
    queryFn: async () => {
      let url = `/tasks/history?date=${dateStr}`;
      if (selectedBoard) url += `&board_id=${selectedBoard}`;
      const res = await api.get(url);
      return (res.data?.data || []) as TaskHistoryRow[];
    },
  });

  // Stats
  const total = history?.length || 0;
  const completed = history?.filter((t) => t.status === 'completed').length || 0;
  const withoutIssue = history?.filter((t) => t.completion_type === 'without_issue').length || 0;
  const withIssue = history?.filter((t) => t.completion_type === 'with_issue').length || 0;
  const pending = history?.filter((t) => t.status === 'pending').length || 0;
  const inProgress = history?.filter((t) => t.status === 'in_progress').length || 0;

  // Group by employee
  const byEmployee: Record<string, { name: string; code: string; tasks: TaskHistoryRow[] }> = {};
  history?.forEach((row) => {
    if (!byEmployee[row.employee_id]) {
      byEmployee[row.employee_id] = { name: row.employee_name, code: row.employee_code, tasks: [] };
    }
    byEmployee[row.employee_id].tasks.push(row);
  });

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

  const statusBadge = (status: string, completionType: string | null) => {
    if (status === 'completed' && completionType === 'without_issue') {
      return { icon: <CheckCircle2 className="w-4 h-4" />, text: 'Completed ✓', cls: 'bg-emerald-500/10 text-emerald-600 border-emerald-500/20' };
    }
    if (status === 'completed' && completionType === 'with_issue') {
      return { icon: <AlertTriangle className="w-4 h-4" />, text: 'Completed ⚠', cls: 'bg-amber-500/10 text-amber-600 border-amber-500/20' };
    }
    if (status === 'completed') {
      return { icon: <CheckCircle2 className="w-4 h-4" />, text: 'Completed', cls: 'bg-emerald-500/10 text-emerald-600 border-emerald-500/20' };
    }
    if (status === 'in_progress') {
      return { icon: <Play className="w-4 h-4" />, text: 'In Progress', cls: 'bg-amber-500/10 text-amber-500 border-amber-500/20' };
    }
    return { icon: <Clock className="w-4 h-4" />, text: 'Pending', cls: 'bg-muted/30 text-muted-foreground border-border' };
  };

  return (
    <div className="space-y-6">
      {/* ── Header ── */}
      <div>
        <h2 className="text-2xl sm:text-3xl font-bold tracking-tight text-foreground mb-1 flex items-center gap-2 sm:gap-3">
          <History className="w-6 h-6 sm:w-8 sm:h-8 text-primary" />
          Task History
        </h2>
        <p className="text-muted-foreground">View all employee task completions by date</p>
      </div>

      {/* ── Stats ── */}
      <div className="grid grid-cols-2 md:grid-cols-5 gap-3">
        <StatCard label="Total Tasks" value={total} color="text-foreground" icon={<History className="w-5 h-5 text-primary" />} />
        <StatCard label="Completed" value={completed} color="text-emerald-500" icon={<CheckCircle2 className="w-5 h-5 text-emerald-500" />} />
        <StatCard label="No Issues" value={withoutIssue} color="text-emerald-500" icon={<CheckCircle2 className="w-5 h-5 text-emerald-400" />} />
        <StatCard label="With Issues" value={withIssue} color="text-amber-500" icon={<AlertTriangle className="w-5 h-5 text-amber-500" />} />
        <StatCard label="Pending" value={pending + inProgress} color="text-muted-foreground" icon={<Clock className="w-5 h-5 text-muted-foreground" />} />
      </div>

      {/* ── Filters ── */}
      <div className="flex items-center justify-between bg-muted/30 rounded-xl p-3 border border-border gap-4 flex-wrap">
        <div className="flex items-center gap-2">
          <Button variant="outline" size="sm" onClick={() => setSelectedDate((prev) => subDays(prev, 1))} className="gap-1">
            <ChevronLeft className="w-4 h-4" />
          </Button>
          <div className="flex items-center gap-2 px-3 py-1.5 bg-background rounded-lg border border-input">
            <Calendar className="w-4 h-4 text-muted-foreground" />
            <input
              type="date"
              value={dateStr}
              onChange={(e) => setSelectedDate(new Date(e.target.value + 'T00:00:00'))}
              className="bg-transparent text-foreground text-sm border-none focus:outline-none"
            />
          </div>
          <Button variant="outline" size="sm" onClick={() => setSelectedDate((prev) => addDays(prev, 1))} className="gap-1">
            <ChevronRight className="w-4 h-4" />
          </Button>
          <Button variant="ghost" size="sm" onClick={() => setSelectedDate(new Date())} className="text-primary text-xs">
            Today
          </Button>
        </div>

        <div className="flex items-center gap-2">
          <span className="text-xs text-muted-foreground">Board:</span>
          <select
            value={selectedBoard}
            onChange={(e) => setSelectedBoard(e.target.value)}
            className="h-8 px-3 rounded-lg bg-background border border-input text-foreground text-sm"
          >
            <option value="">All Boards</option>
            {boards?.map((b) => <option key={b.id} value={b.id}>{b.name}</option>)}
          </select>
        </div>
      </div>

      {/* ── Content ── */}
      {isLoading ? (
        <div className="flex justify-center py-16">
          <Loader2 className="w-8 h-8 animate-spin text-muted-foreground" />
        </div>
      ) : total === 0 ? (
        <div className="py-16 text-center border border-border border-dashed rounded-xl bg-muted/10">
          <History className="w-16 h-16 mx-auto mb-4 text-muted-foreground/30" />
          <h3 className="text-xl font-semibold text-muted-foreground mb-2">No tasks for this date</h3>
          <p className="text-muted-foreground">Try selecting a different date or board.</p>
        </div>
      ) : (
        <div className="space-y-4">
          {Object.entries(byEmployee).map(([empId, emp]) => (
            <Card key={empId}>
              {/* Employee Header */}
              <div className="px-5 py-3 border-b border-border flex items-center justify-between">
                <div className="flex items-center gap-2 sm:gap-3">
                  <div className="w-9 h-9 rounded-xl bg-primary/10 flex items-center justify-center">
                    <User className="w-4 h-4 text-primary" />
                  </div>
                  <div>
                    <span className="font-semibold text-foreground">{emp.name}</span>
                    <span className="text-xs text-muted-foreground ml-2">({emp.code})</span>
                  </div>
                </div>
                <div className="flex items-center gap-2 text-xs text-muted-foreground">
                  <span>{emp.tasks.filter((t) => t.status === 'completed').length}/{emp.tasks.length} completed</span>
                  {emp.tasks.some((t) => t.completion_type === 'with_issue') && (
                    <span className="px-1.5 py-0.5 rounded bg-amber-500/10 text-amber-600 border border-amber-500/20 text-[10px] font-medium">
                      ⚠ Issues
                    </span>
                  )}
                </div>
              </div>

              {/* Tasks */}
              <CardContent className="p-4">
                <div className="space-y-2">
                  {emp.tasks.map((task) => {
                    const badge = statusBadge(task.status, task.completion_type);
                    return (
                      <div key={task.execution_id}
                        className={`flex items-start gap-3 px-4 py-3 rounded-xl border ${badge.cls}`}>
                        <div className="flex-shrink-0 mt-0.5">{badge.icon}</div>
                        <div className="flex-1 min-w-0">
                          <div className="flex items-center gap-2 mb-0.5">
                            <span className="font-medium text-foreground">{task.task_title}</span>
                            {task.board_name && (
                              <span className="px-1.5 py-0.5 rounded text-[10px] font-medium bg-primary/10 text-primary border border-primary/20">
                                {task.board_name}
                              </span>
                            )}
                          </div>
                          <div className="flex items-center gap-3 text-xs flex-wrap">
                            <span className="font-medium">{badge.text}</span>
                            {task.started_at && (
                              <span className="text-muted-foreground flex items-center gap-1">
                                <Timer className="w-3 h-3" />
                                {formatDuration(task.started_at, task.completed_at)}
                              </span>
                            )}
                            {task.completed_at && (
                              <span className="text-muted-foreground">
                                at {format(new Date(task.completed_at), 'hh:mm a')}
                              </span>
                            )}
                          </div>
                          {task.notes && (
                            <p className="text-xs text-muted-foreground mt-1.5 italic bg-background/50 px-2 py-1 rounded-lg">
                              📝 {task.notes}
                            </p>
                          )}
                        </div>
                      </div>
                    );
                  })}
                </div>
              </CardContent>
            </Card>
          ))}
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
    <CardContent className="p-3 flex items-center gap-2 sm:gap-3">
      <div className="p-2 rounded-xl bg-muted/60">{icon}</div>
      <div>
        <p className="text-[10px] text-muted-foreground uppercase tracking-wider">{label}</p>
        <p className={`text-lg font-bold ${color}`}>{value}</p>
      </div>
    </CardContent>
  </Card>
);
