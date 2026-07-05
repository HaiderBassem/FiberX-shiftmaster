import { useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import api from '@/lib/api';
import { useAuthStore } from '@/store/authStore';
import { Card, CardContent, CardHeader, CardTitle, CardDescription, CardFooter } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import {
  LayoutGrid, Plus, Trash2, Edit3, ChevronLeft, ChevronRight,
  Calendar, Users, CheckSquare, ClipboardList, ArrowLeft,
  Filter, X, Loader2, Play, CheckCircle2, Clock, BarChart3
} from 'lucide-react';
import { format, startOfWeek, addDays, subDays } from 'date-fns';
import { fmtDate } from '@/lib/dateUtils';

const DAYS = ['Sunday', 'Monday', 'Tuesday', 'Wednesday', 'Thursday', 'Friday', 'Saturday'];

// ───────────────────────────────────────────────────────────
// Types
// ───────────────────────────────────────────────────────────

interface TaskBoard {
  id: string;
  name: string;
  description: string | null;
  recurrence_type: 'daily' | 'weekly';
  is_active: boolean;
  created_by: string | null;
  created_at: string;
}

interface BoardStats {
  board_id: string;
  board_name: string;
  total_assigned: number;
  total_pending: number;
  total_in_progress: number;
  total_completed: number;
  completion_pct: number;
}

interface Shift {
  id: string;
  name: string;
  shift_code: string;
  color_code: string | null;
}

interface TaskSchedule {
  id: string;
  title: string;
  description: string | null;
  board_id: string | null;
  shift_id: string | null;
  recurrence: string;
  recurrence_days: number[] | null;
  max_assignees: number;
  is_active: boolean;
}

interface RecurringAssignment {
  id: string;
  schedule_id: string;
  employee_id: string;
  day_of_week: number;
  assigned_by: string | null;
  created_at: string;
}

interface BoardViewRow {
  employee_id: string;
  employee_name: string;
  employee_code: string;
  day_of_week: number;
  assigned_date: string | null;
  task_id: string | null;
  task_title: string | null;
  assignment_id: string | null;
  execution_id: string | null;
  status: string | null;
  started_at: string | null;
  completed_at: string | null;
}

const selectClass = "w-full h-10 px-3 py-2 rounded-lg bg-background border border-input text-foreground focus:outline-none focus:ring-2 focus:ring-ring transition-colors";

// ───────────────────────────────────────────────────────────
// Main Component
// ───────────────────────────────────────────────────────────

export const TaskBoards = () => {
  const { user } = useAuthStore();
  const isTeamLeader = user?.role === 'team_leader';
  const canEdit = isTeamLeader || user?.role === 'admin';

  const [selectedBoard, setSelectedBoard] = useState<TaskBoard | null>(null);

  if (selectedBoard) {
    return (
      <BoardDetailView
        board={selectedBoard}
        canEdit={canEdit}
        onBack={() => setSelectedBoard(null)}
      />
    );
  }

  return <BoardListView canEdit={canEdit} onSelectBoard={setSelectedBoard} />;
};

// ═══════════════════════════════════════════════════════════
// Board List View
// ═══════════════════════════════════════════════════════════

const BoardListView = ({
  canEdit,
  onSelectBoard,
}: {
  canEdit: boolean;
  onSelectBoard: (b: TaskBoard) => void;
}) => {
  const queryClient = useQueryClient();

  const [showForm, setShowForm] = useState(false);
  const [boardName, setBoardName] = useState('');
  const [boardDesc, setBoardDesc] = useState('');
  const [boardRecurrence, setBoardRecurrence] = useState<'daily' | 'weekly'>('weekly');

  const [editId, setEditId] = useState<string | null>(null);
  const [editName, setEditName] = useState('');
  const [editDesc, setEditDesc] = useState('');
  const [editRecurrence, setEditRecurrence] = useState<'daily' | 'weekly'>('weekly');

  const [error, setError] = useState<string | null>(null);

  // ── Queries ──
  const { data: boards, isLoading } = useQuery({
    queryKey: ['boards'],
    queryFn: async () => {
      const res = await api.get('/tasks/boards');
      return (res.data?.data || []) as TaskBoard[];
    },
  });

  const { data: stats } = useQuery({
    queryKey: ['board-stats'],
    queryFn: async () => {
      const res = await api.get('/tasks/boards/stats');
      return (res.data?.data || []) as BoardStats[];
    },
  });

  const statsMap: Record<string, BoardStats> = {};
  stats?.forEach((s) => { statsMap[s.board_id] = s; });

  // ── Mutations ──
  const createBoard = useMutation({
    mutationFn: async () => {
      await api.post('/tasks/boards', {
        name: boardName,
        description: boardDesc || null,
        recurrence_type: boardRecurrence,
      });
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['boards'] });
      queryClient.invalidateQueries({ queryKey: ['board-stats'] });
      setBoardName(''); setBoardDesc(''); setShowForm(false); setError(null);
    },
    onError: (err: any) => {
      setError(err?.response?.data?.error || err?.message || 'Failed to create board');
    },
  });

  const updateBoard = useMutation({
    mutationFn: async () => {
      await api.put(`/tasks/boards/${editId}`, {
        name: editName, description: editDesc || null,
        recurrence_type: editRecurrence, is_active: true,
      });
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['boards'] });
      setEditId(null); setError(null);
    },
    onError: (err: any) => {
      setError(err?.response?.data?.error || err?.message || 'Failed to update board');
    },
  });

  const deleteBoard = useMutation({
    mutationFn: async (id: string) => { await api.delete(`/tasks/boards/${id}`); },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['boards'] });
      queryClient.invalidateQueries({ queryKey: ['board-stats'] });
      setError(null);
    },
    onError: (err: any) => {
      setError(err?.response?.data?.error || err?.message || 'Failed to delete board');
    },
  });

  const startEdit = (b: TaskBoard) => {
    setEditId(b.id); setEditName(b.name); setEditDesc(b.description || ''); setEditRecurrence(b.recurrence_type);
  };

  return (
    <div className="space-y-8">
      {/* Header */}
      <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-3">
        <div />
        {canEdit && (
          <Button onClick={() => setShowForm(!showForm)} className="gap-2">
            {showForm ? <X className="w-4 h-4" /> : <Plus className="w-4 h-4" />}
            {showForm ? 'Cancel' : 'New Board'}
          </Button>
        )}
      </div>

      {error && (
        <div className="flex items-center gap-3 px-4 py-3 rounded-xl bg-destructive/10 border border-destructive/20 text-destructive text-sm">
          <span className="flex-1">⚠️ {error}</span>
          <button onClick={() => setError(null)} className="text-destructive hover:text-destructive/80 font-bold">×</button>
        </div>
      )}

      {/* Create Form */}
      {showForm && canEdit && (
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <Plus className="w-5 h-5 text-primary" />
              Create New Board
            </CardTitle>
          </CardHeader>
          <CardContent>
            <div className="grid md:grid-cols-3 gap-4">
              <div className="space-y-2">
                <Label>Board Name</Label>
                <Input value={boardName} onChange={(e: React.ChangeEvent<HTMLInputElement>) => setBoardName(e.target.value)}
                  placeholder="e.g. Node Check, Mobile Ticket..." />
              </div>
              <div className="space-y-2">
                <Label>Description</Label>
                <Input value={boardDesc} onChange={(e: React.ChangeEvent<HTMLInputElement>) => setBoardDesc(e.target.value)}
                  placeholder="Optional description..." />
              </div>
              <div className="space-y-2">
                <Label>Recurrence</Label>
                <select className={selectClass}
                  value={boardRecurrence} onChange={(e) => setBoardRecurrence(e.target.value as 'daily' | 'weekly')}>
                  <option value="daily">Daily (Repeats every day)</option>
                  <option value="weekly">Weekly (Specific days)</option>
                </select>
              </div>
            </div>
          </CardContent>
          <CardFooter>
            <Button onClick={() => createBoard.mutate()} disabled={createBoard.isPending || !boardName.trim()}
              className="gap-2">
              <Plus className="w-4 h-4" /> Create Board
            </Button>
          </CardFooter>
        </Card>
      )}

      {/* Board Cards */}
      {isLoading ? (
        <div className="grid md:grid-cols-2 lg:grid-cols-3 gap-6">
          {[1, 2, 3].map((i) => <Card key={i} className="animate-pulse h-52" />)}
        </div>
      ) : !boards || boards.length === 0 ? (
        <div className="p-16 text-center border border-border border-dashed rounded-xl bg-muted/10">
          <LayoutGrid className="w-16 h-16 mx-auto mb-4 text-muted-foreground/30" />
          <h3 className="text-xl font-semibold text-muted-foreground mb-2">No boards yet</h3>
          <p className="text-muted-foreground">
            {canEdit ? 'Create your first task board to start organizing tasks.' : 'No task boards have been created yet.'}
          </p>
        </div>
      ) : (
        <div className="grid md:grid-cols-2 lg:grid-cols-3 gap-6">
          {boards.map((board) => {
            const bs = statsMap[board.id];

            return (
              <div key={board.id}>
                {editId === board.id ? (
                  <Card>
                    <CardContent className="pt-6 space-y-3">
                      <Input value={editName} onChange={(e: React.ChangeEvent<HTMLInputElement>) => setEditName(e.target.value)} />
                      <Input value={editDesc} onChange={(e: React.ChangeEvent<HTMLInputElement>) => setEditDesc(e.target.value)}
                        placeholder="Description..." />
                      <select className={selectClass}
                        value={editRecurrence} onChange={(e) => setEditRecurrence(e.target.value as 'daily' | 'weekly')}>
                        <option value="daily">Daily</option><option value="weekly">Weekly</option>
                      </select>
                      <div className="flex gap-2">
                        <Button size="sm" onClick={() => updateBoard.mutate()} disabled={updateBoard.isPending || !editName.trim()}>Save</Button>
                        <Button size="sm" variant="outline" onClick={() => setEditId(null)}>Cancel</Button>
                      </div>
                    </CardContent>
                  </Card>
                ) : (
                  <Card className="hover:shadow-md transition-all duration-300 cursor-pointer group"
                    onClick={() => onSelectBoard(board)}>
                    <CardHeader className="pb-2">
                      <div className="flex items-start justify-between">
                        <CardTitle className="text-lg group-hover:text-primary transition-colors flex items-center gap-2">
                          <ClipboardList className="w-5 h-5 text-primary" />
                          {board.name}
                        </CardTitle>
                        {canEdit && (
                          <div className="flex gap-1 opacity-0 group-hover:opacity-100 transition-opacity">
                            <button onClick={(e) => { e.stopPropagation(); startEdit(board); }}
                              className="p-1.5 rounded-md hover:bg-muted transition-colors" title="Edit">
                              <Edit3 className="w-4 h-4 text-muted-foreground" />
                            </button>
                            <button onClick={(e) => { e.stopPropagation(); if (confirm(`Delete "${board.name}"?`)) deleteBoard.mutate(board.id); }}
                              className="p-1.5 rounded-md hover:bg-destructive/10 transition-colors" title="Delete">
                              <Trash2 className="w-4 h-4 text-destructive" />
                            </button>
                          </div>
                        )}
                      </div>
                      <CardDescription className="text-xs">{board.description || 'No description'}</CardDescription>
                    </CardHeader>
                    <CardContent className="pt-0 space-y-3">
                      {/* Stats mini-bar */}
                      {bs && bs.total_assigned > 0 && (
                        <div>
                          <div className="flex items-center justify-between text-xs mb-1.5">
                            <span className="text-muted-foreground">Completion</span>
                            <span className="text-foreground font-semibold">{bs.completion_pct}%</span>
                          </div>
                          <div className="w-full h-2 bg-muted rounded-full overflow-hidden">
                            <div className="h-full bg-gradient-to-r from-primary to-accent rounded-full transition-all duration-500"
                              style={{ width: `${bs.completion_pct}%` }} />
                          </div>
                          <div className="flex items-center gap-3 mt-2 text-[10px]">
                            <span className="text-emerald-500">✓ {bs.total_completed}</span>
                            <span className="text-amber-500">▶ {bs.total_in_progress}</span>
                            <span className="text-muted-foreground">◌ {bs.total_pending}</span>
                          </div>
                        </div>
                      )}
                      <div className="flex items-center gap-3 text-xs">
                        <span className={`px-2 py-0.5 rounded-full font-medium ${
                          board.recurrence_type === 'daily'
                            ? 'bg-amber-500/10 text-amber-500 border border-amber-500/20'
                            : 'bg-blue-500/10 text-blue-500 border border-blue-500/20'
                        }`}>
                          {board.recurrence_type === 'daily' ? '🔄 Daily' : '📅 Weekly'}
                        </span>
                        <span className="text-muted-foreground/60">Created {fmtDate(board.created_at)}</span>
                      </div>
                    </CardContent>
                  </Card>
                )}
              </div>
            );
          })}
        </div>
      )}
    </div>
  );
};

// ═══════════════════════════════════════════════════════════
// Board Detail View (Employee × Day Tracker Grid)
// ═══════════════════════════════════════════════════════════

const BoardDetailView = ({
  board,
  canEdit,
  onBack,
}: {
  board: TaskBoard;
  canEdit: boolean;
  onBack: () => void;
}) => {
  const queryClient = useQueryClient();

  const [selectedShiftId, setSelectedShiftId] = useState<string>('');
  const [weekStart, setWeekStart] = useState(() => startOfWeek(new Date(), { weekStartsOn: 0 }));

  const [showTaskForm, setShowTaskForm] = useState(false);
  const [taskTitle, setTaskTitle] = useState('');
  const [taskDesc, setTaskDesc] = useState('');
  const [taskRecurrence, setTaskRecurrence] = useState('daily');
  const [taskDays, setTaskDays] = useState<number[]>([]);
  const [taskMaxAssignees, setTaskMaxAssignees] = useState(1);

  const [assigningCell, setAssigningCell] = useState<{ empId: string; day: number } | null>(null);
  const [assignTaskId, setAssignTaskId] = useState('');
  const [error, setError] = useState<string | null>(null);

  // ── Queries ──
  const { data: shifts } = useQuery({
    queryKey: ['shifts'],
    queryFn: async () => { const res = await api.get('/shifts'); return (res.data?.data || []) as Shift[]; },
  });

  const { data: boardTasks, isLoading: tasksLoading } = useQuery({
    queryKey: ['board-tasks', board.id],
    queryFn: async () => {
      const res = await api.get(`/tasks/schedules?board_id=${board.id}`);
      return (res.data?.data || []) as TaskSchedule[];
    },
  });

  const { data: boardViewData, isLoading: viewLoading } = useQuery({
    queryKey: ['board-view', board.id, selectedShiftId, format(weekStart, 'yyyy-MM-dd')],
    queryFn: async () => {
      let url = `/tasks/boards/${board.id}/view`;
      const from = format(weekStart, 'yyyy-MM-dd');
      const to = format(addDays(weekStart, 6), 'yyyy-MM-dd');
      const params = new URLSearchParams();
      if (selectedShiftId) params.set('shift_id', selectedShiftId);
      params.set('from', from);
      params.set('to', to);
      url += `?${params.toString()}`;
      const res = await api.get(url);
      return (res.data?.data || []) as BoardViewRow[];
    },
  });

  // Fetch ALL eligible employees (role=employee only, filtered by shift)
  const { data: eligibleEmployees } = useQuery({
    queryKey: ['board-eligible-employees', selectedShiftId],
    queryFn: async () => {
      let url = '/tasks/boards/eligible-employees';
      if (selectedShiftId) url += `?shift_id=${selectedShiftId}`;
      const res = await api.get(url);
      return (res.data?.data || []) as { id: string; first_name: string; last_name: string; employee_code: string }[];
    },
  });

  // Fetch recurring assignments for this board
  const { data: recurringAssignments } = useQuery({
    queryKey: ['board-recurring', board.id],
    queryFn: async () => {
      const res = await api.get(`/tasks/boards/${board.id}/recurring`);
      return (res.data?.data || []) as RecurringAssignment[];
    },
  });

  // Build a set of recurring keys for quick lookup
  const recurringSet = new Set<string>();
  recurringAssignments?.forEach((ra) => {
    recurringSet.add(`${ra.schedule_id}:${ra.employee_id}:${ra.day_of_week}`);
  });

  // ── Mutations ──
  const createTask = useMutation({
    mutationFn: async () => {
      await api.post('/tasks/schedules', {
        title: taskTitle, description: taskDesc || null, board_id: board.id,
        shift_id: selectedShiftId || null, recurrence: taskRecurrence,
        recurrence_days: taskDays.length > 0 ? taskDays : null,
        max_assignees: taskMaxAssignees, is_active: true,
      });
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['board-tasks', board.id] });
      queryClient.invalidateQueries({ queryKey: ['board-view'] });
      setTaskTitle(''); setTaskDesc(''); setTaskDays([]); setShowTaskForm(false); setError(null);
    },
    onError: (err: any) => setError(err?.response?.data?.error || 'Failed to create task'),
  });

  const deleteTask = useMutation({
    mutationFn: async (id: string) => { await api.delete(`/tasks/schedules/${id}`); },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['board-tasks', board.id] });
      queryClient.invalidateQueries({ queryKey: ['board-view'] });
    },
  });

  const assignTask = useMutation({
    mutationFn: async ({ scheduleId, employeeId, dayOfWeek }: { scheduleId: string; employeeId: string; dayOfWeek: number }) => {
      await api.post('/tasks/recurring-assign', { schedule_id: scheduleId, employee_id: employeeId, day_of_week: dayOfWeek });
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['board-view'] });
      queryClient.invalidateQueries({ queryKey: ['board-stats'] });
      queryClient.invalidateQueries({ queryKey: ['board-recurring', board.id] });
      setAssigningCell(null); setAssignTaskId(''); setError(null);
    },
    onError: (err: any) => setError(err?.response?.data?.error || 'Failed to assign recurring task'),
  });

  const removeAssignment = useMutation({
    mutationFn: async ({ scheduleId, employeeId, dayOfWeek }: { scheduleId: string; employeeId: string; dayOfWeek: number }) => {
      await api.post('/tasks/recurring-assignments/remove', { schedule_id: scheduleId, employee_id: employeeId, day_of_week: dayOfWeek });
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['board-view'] });
      queryClient.invalidateQueries({ queryKey: ['board-stats'] });
      queryClient.invalidateQueries({ queryKey: ['board-recurring', board.id] });
    },
  });

  const toggleDay = (day: number) => {
    setTaskDays((prev) => (prev.includes(day) ? prev.filter((d) => d !== day) : [...prev, day]));
  };

  const weekDates = Array.from({ length: 7 }, (_, i) => addDays(weekStart, i));

  // ── Build grid: merge eligible employees with board view data ──
  const gridData: Record<string, {
    name: string;
    code: string;
    days: Record<number, { taskTitle: string; taskId: string; assignmentId: string; executionId: string | null; status: string }[]>;
  }> = {};

  // First: add ALL eligible employees (even if they have no assignments)
  const eligibleSet = new Set((eligibleEmployees || []).map((e) => e.id));
  eligibleEmployees?.forEach((emp) => {
    gridData[emp.id] = {
      name: `${emp.first_name} ${emp.last_name}`,
      code: emp.employee_code,
      days: {},
    };
  });

  // Then: overlay the actual assignments from board view
  boardViewData?.forEach((row) => {
    // When shift filter is active, keep rows strictly within eligible employees for that shift.
    if (selectedShiftId && !eligibleSet.has(row.employee_id)) return;

    if (!gridData[row.employee_id]) {
      gridData[row.employee_id] = { name: row.employee_name, code: row.employee_code, days: {} };
    }
    if (!row.assigned_date) return;
    const assignedKey = String(row.assigned_date).slice(0, 10);
    const dayIdx = weekDates.findIndex((d) => format(d, 'yyyy-MM-dd') === assignedKey);
    if (dayIdx < 0) return;

    if (!gridData[row.employee_id].days[dayIdx]) {
      gridData[row.employee_id].days[dayIdx] = [];
    }
    if (row.task_title && row.task_id && row.assignment_id) {
      gridData[row.employee_id].days[dayIdx].push({
        taskTitle: row.task_title,
        taskId: row.task_id,
        assignmentId: row.assignment_id,
        executionId: row.execution_id,
        status: row.status || 'pending',
      });
    }
  });

  // ── Grid stats ──
  let gridTotal = 0, gridDone = 0, gridProgress = 0;
  Object.values(gridData).forEach((emp) => {
    Object.values(emp.days).forEach((tasks) => {
      tasks.forEach((t) => {
        gridTotal++;
        if (t.status === 'completed') gridDone++;
        else if (t.status === 'in_progress') gridProgress++;
      });
    });
  });
  const gridPct = gridTotal > 0 ? Math.round((gridDone / gridTotal) * 100) : 0;

  const statusBadge = (status: string) => {
    switch (status) {
      case 'completed': return 'bg-emerald-500/15 text-emerald-500 border-emerald-500/20';
      case 'in_progress': return 'bg-amber-500/15 text-amber-500 border-amber-500/20';
      default: return 'bg-muted/40 text-muted-foreground border-border';
    }
  };

  const statusIcon = (status: string) => {
    switch (status) {
      case 'completed': return <CheckCircle2 className="w-3 h-3 text-emerald-500" />;
      case 'in_progress': return <Play className="w-3 h-3 text-amber-500" />;
      default: return <Clock className="w-3 h-3 text-muted-foreground" />;
    }
  };

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-4">
          <Button variant="outline" size="sm" onClick={onBack} className="gap-2">
            <ArrowLeft className="w-4 h-4" /> Back
          </Button>
          <div>
            <h2 className="text-2xl font-bold text-foreground flex items-center gap-2 sm:gap-3">
              <ClipboardList className="w-7 h-7 text-primary" />
              {board.name}
            </h2>
            <p className="text-sm text-muted-foreground">{board.description || 'No description'}</p>
          </div>
        </div>
        <div className="flex items-center gap-2 sm:gap-3">
          {gridTotal > 0 && (
            <div className="flex items-center gap-2 px-3 py-1.5 rounded-lg bg-muted/30 border border-border">
              <BarChart3 className="w-4 h-4 text-primary" />
              <span className="text-sm text-foreground font-semibold">{gridPct}%</span>
              <span className="text-xs text-muted-foreground">({gridDone}/{gridTotal})</span>
            </div>
          )}
          {canEdit && (
            <Button onClick={() => setShowTaskForm(!showTaskForm)} className="gap-2">
              {showTaskForm ? <X className="w-4 h-4" /> : <Plus className="w-4 h-4" />}
              {showTaskForm ? 'Cancel' : 'Add Task'}
            </Button>
          )}
        </div>
      </div>

      {error && (
        <div className="flex items-center gap-3 px-4 py-3 rounded-xl bg-destructive/10 border border-destructive/20 text-destructive text-sm">
          <span className="flex-1">⚠️ {error}</span>
          <button onClick={() => setError(null)} className="text-destructive hover:text-destructive/80 font-bold">×</button>
        </div>
      )}

      {/* Create Task Form */}
      {showTaskForm && canEdit && (
        <Card>
          <CardHeader>
            <CardTitle className="text-base flex items-center gap-2">
              <Plus className="w-5 h-5 text-primary" />
              Add New Task to "{board.name}"
            </CardTitle>
          </CardHeader>
          <CardContent>
            <div className="grid md:grid-cols-2 lg:grid-cols-4 gap-4">
              <div className="space-y-2">
                <Label>Task Title</Label>
                <Input value={taskTitle} onChange={(e: React.ChangeEvent<HTMLInputElement>) => setTaskTitle(e.target.value)}
                  placeholder="e.g. Check Node #42" />
              </div>
              <div className="space-y-2">
                <Label>Description</Label>
                <Input value={taskDesc} onChange={(e: React.ChangeEvent<HTMLInputElement>) => setTaskDesc(e.target.value)}
                  placeholder="Optional..." />
              </div>
              <div className="space-y-2">
                <Label>Recurrence</Label>
                <select className={selectClass}
                  value={taskRecurrence} onChange={(e) => setTaskRecurrence(e.target.value)}>
                  <option value="daily">Every Day</option>
                  <option value="periodic">Specific Days</option>
                </select>
              </div>
              <div className="space-y-2">
                <Label>Max Assignees</Label>
                <Input type="number" min={1} value={taskMaxAssignees}
                  onChange={(e: React.ChangeEvent<HTMLInputElement>) => setTaskMaxAssignees(parseInt(e.target.value) || 1)} />
              </div>
            </div>
            {taskRecurrence === 'periodic' && (
              <div className="mt-4 space-y-2">
                <Label>Select Days</Label>
                <div className="flex flex-wrap gap-2">
                  {DAYS.map((day, i) => (
                    <button key={i} onClick={() => toggleDay(i)} className={`px-3 py-1.5 rounded-md text-xs font-medium transition-all ${
                      taskDays.includes(i) ? 'bg-primary text-primary-foreground shadow-lg' : 'bg-muted text-muted-foreground hover:bg-muted/80'
                    }`}>{day.slice(0, 3)}</button>
                  ))}
                </div>
              </div>
            )}
          </CardContent>
          <CardFooter>
            <Button onClick={() => createTask.mutate()} disabled={createTask.isPending || !taskTitle.trim()} className="gap-2">
              <Plus className="w-4 h-4" /> Add Task
            </Button>
          </CardFooter>
        </Card>
      )}

      {/* Filters Row */}
      <div className="flex items-center gap-4 flex-wrap">
        <div className="flex items-center gap-2">
          <Filter className="w-4 h-4 text-muted-foreground" />
          <select className={selectClass + " w-auto min-w-[160px]"}
            value={selectedShiftId} onChange={(e) => setSelectedShiftId(e.target.value)}>
            <option value="">All Shifts</option>
            {shifts?.map((s) => <option key={s.id} value={s.id}>{s.name} ({s.shift_code})</option>)}
          </select>
        </div>
        <div className="flex items-center gap-2 ml-auto">
          <Button variant="outline" size="sm" onClick={() => setWeekStart((prev) => subDays(prev, 7))}
            className="h-9 w-9 p-0">
            <ChevronLeft className="w-4 h-4" />
          </Button>
          <span className="text-sm text-foreground min-w-[180px] text-center font-medium">
            <Calendar className="w-4 h-4 inline-block mr-1.5 text-muted-foreground" />
            {format(weekStart, 'MMM d')} – {format(addDays(weekStart, 6), 'MMM d, yyyy')}
          </span>
          <Button variant="outline" size="sm" onClick={() => setWeekStart((prev) => addDays(prev, 7))}
            className="h-9 w-9 p-0">
            <ChevronRight className="w-4 h-4" />
          </Button>
        </div>
      </div>

      {/* Tasks of this Board */}
      {tasksLoading ? (
        <div className="flex justify-center py-8"><Loader2 className="w-6 h-6 animate-spin text-muted-foreground" /></div>
      ) : boardTasks && boardTasks.length > 0 ? (
        <Card>
          <CardHeader className="pb-3">
            <CardTitle className="text-base flex items-center gap-2">
              <CheckSquare className="w-4 h-4 text-primary" />
              Tasks in this Board ({boardTasks.length})
            </CardTitle>
          </CardHeader>
          <CardContent>
            <div className="flex flex-wrap gap-2">
              {boardTasks.map((task) => (
                <div key={task.id} className="flex items-center gap-2 px-3 py-2 rounded-lg bg-muted/30 border border-border text-sm">
                  <span className="text-foreground font-medium">{task.title}</span>
                  <span className="text-xs text-muted-foreground">
                    {task.recurrence === 'daily' ? 'Daily' : `Days: ${task.recurrence_days?.map((d) => DAYS[d]?.slice(0, 3)).join(', ')}`}
                  </span>
                  {task.max_assignees > 1 && (
                    <span className="text-xs px-1.5 py-0.5 rounded bg-primary/10 text-primary">×{task.max_assignees}</span>
                  )}
                  {canEdit && (
                    <button onClick={() => { if (confirm(`Delete task "${task.title}"?`)) deleteTask.mutate(task.id); }}
                      className="p-0.5 rounded hover:bg-destructive/10">
                      <Trash2 className="w-3.5 h-3.5 text-destructive" />
                    </button>
                  )}
                </div>
              ))}
            </div>
          </CardContent>
        </Card>
      ) : (
        <Card className="border-dashed">
          <CardContent className="py-8 text-center text-muted-foreground">
            <CheckSquare className="w-10 h-10 mx-auto mb-3 opacity-20" />
            <p>No tasks in this board yet. {canEdit ? 'Add a task above to get started.' : ''}</p>
          </CardContent>
        </Card>
      )}

      {/* Employee × Day Grid */}
      <Card>
        <CardHeader className="pb-3 border-b border-border">
          <CardTitle className="text-base flex items-center gap-2">
            <Users className="w-4 h-4 text-primary" />
            Employee Schedule Tracker
            <span className="text-xs text-muted-foreground font-normal ml-2">
              ({Object.keys(gridData).length} employees)
            </span>
          </CardTitle>
          <CardDescription>All employees with their task assignments and completion status</CardDescription>
        </CardHeader>
        <CardContent className="p-0">
          {viewLoading ? (
            <div className="flex justify-center py-12"><Loader2 className="w-6 h-6 animate-spin text-muted-foreground" /></div>
          ) : Object.keys(gridData).length === 0 ? (
            <div className="py-12 text-center text-muted-foreground">
              <Users className="w-10 h-10 mx-auto mb-3 opacity-20" />
              <p>No employees found. Make sure employees have been created with the right shift assigned.</p>
            </div>
          ) : (
            <div className="overflow-x-auto">
              <table className="w-full text-sm">
                <thead>
                  <tr className="border-b border-border">
                    <th className="text-left px-4 py-3 text-muted-foreground font-semibold sticky left-0 bg-card min-w-[180px] z-10">Employee</th>
                    {weekDates.map((date, i) => (
                      <th key={i} className="text-center px-3 py-3 text-muted-foreground font-medium min-w-[140px]">
                        <div className="text-xs">{DAYS[i]}</div>
                        <div className="text-[10px] text-muted-foreground/60">{format(date, 'MMM d')}</div>
                      </th>
                    ))}
                  </tr>
                </thead>
                <tbody>
                  {Object.entries(gridData).map(([empId, emp]) => (
                    <tr key={empId} className="border-b border-border/50 hover:bg-muted/20 transition-colors">
                      <td className="px-4 py-3 sticky left-0 bg-card z-10">
                        <div className="font-medium text-foreground">{emp.name}</div>
                        <div className="text-xs text-muted-foreground">{emp.code}</div>
                      </td>
                      {weekDates.map((_date, dayIdx) => {
                        const dayTasks = emp.days[dayIdx] || [];
                        const isAssigning = assigningCell?.empId === empId && assigningCell?.day === dayIdx;
                        return (
                          <td key={dayIdx} className="px-2 py-2 text-center align-top">
                            <div className="space-y-1 min-h-[40px]">
                              {dayTasks.map((t, tIdx) => {
                                const isRecurring = recurringSet.has(`${t.taskId}:${empId}:${dayIdx}`);
                                return (
                                <div key={tIdx}
                                  className={`relative group/tag px-2 py-1.5 rounded-lg border text-xs flex items-center gap-1.5 ${statusBadge(t.status)}`}>
                                  {statusIcon(t.status)}
                                  <span className="truncate font-medium">{t.taskTitle}</span>
                                  {isRecurring && <span title="Recurring every week" className="text-[9px] opacity-60">🔁</span>}
                                  {canEdit && isRecurring && (
                                    <button onClick={() => removeAssignment.mutate({ scheduleId: t.taskId, employeeId: empId, dayOfWeek: dayIdx })}
                                      className="absolute -top-1.5 -right-1.5 w-4 h-4 rounded-full bg-destructive text-white text-[10px] hidden group-hover/tag:flex items-center justify-center shadow-lg">
                                      ×
                                    </button>
                                  )}
                                </div>
                                );
                              })}
                              {/* Quick Assign Button */}
                              {canEdit && boardTasks && boardTasks.length > 0 && (
                                <button
                                  onClick={() => {
                                    const next = isAssigning ? null : { empId, day: dayIdx };
                                    setAssigningCell(next);
                                    setAssignTaskId('');
                                  }}
                                  className="w-full py-1 rounded-md border border-dashed border-border text-muted-foreground hover:border-primary/40 hover:text-primary transition-colors text-[10px]"
                                >+</button>
                              )}
                              {/* Quick Assign Dropdown */}
                              {isAssigning && (
                                <div className="mt-1 p-2 bg-popover rounded-lg border border-border shadow-xl z-20 relative space-y-1.5">
                                  <select className="w-full h-7 px-2 rounded bg-background border border-input text-foreground text-[11px]"
                                    value={assignTaskId} onChange={(e) => setAssignTaskId(e.target.value)}>
                                    <option value="">Select task...</option>
                                    {boardTasks
                                      ?.filter((t) =>
                                        (t.recurrence === 'daily' || (t.recurrence_days ?? []).includes(dayIdx)) &&
                                        (!selectedShiftId || t.shift_id === selectedShiftId || t.shift_id === null)
                                      )
                                      .map((t) => <option key={t.id} value={t.id}>{t.title}</option>)}
                                  </select>
                                  <Button size="sm" disabled={!assignTaskId || assignTask.isPending}
                                    onClick={() => {
                                      if (assignTaskId) {
                                        assignTask.mutate({
                                          scheduleId: assignTaskId,
                                          employeeId: empId,
                                          dayOfWeek: dayIdx,
                                        });
                                      }
                                    }}
                                    className="w-full h-7 text-[11px]">
                                    Assign (Recurring)
                                  </Button>
                                </div>
                              )}
                            </div>
                          </td>
                        );
                      })}
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  );
};
