import { useQuery } from '@tanstack/react-query';
import api from '@/lib/api';
import { useAuthStore } from '@/store/authStore';
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from '@/components/ui/card';
import {
  Users, CheckCircle2, Clock, Play, CalendarDays, Shield, BarChart3,
  CheckSquare, TrendingUp, Briefcase, AlertCircle
} from 'lucide-react';
import { format, startOfWeek } from 'date-fns';
import { fmtDateTime } from '@/lib/dateUtils';
import { AnnouncementBanner } from '../announcements/AnnouncementBanner';

// ───────────────────────────────────────────────────────────
// Dashboard
// ───────────────────────────────────────────────────────────

export const Dashboard = () => {
  const { user } = useAuthStore();
  const isEmployee = user?.role === 'employee';

  if (isEmployee) return <EmployeeDashboard />;
  return <LeaderDashboard />;
};

// ═══════════════════════════════════════════════════════════
// Employee Dashboard
// ═══════════════════════════════════════════════════════════

const EmployeeDashboard = () => {
  const { user } = useAuthStore();
  const today = format(new Date(), 'yyyy-MM-dd');
  const weekStart = format(startOfWeek(new Date(), { weekStartsOn: 0 }), 'yyyy-MM-dd');

  const { data: weeklyTasks } = useQuery({
    queryKey: ['my-weekly-tasks', weekStart],
    queryFn: async () => {
      const res = await api.get(`/tasks/my-week?week_start=${weekStart}`);
      return (res.data?.data || []) as any[];
    },
  });

  const { data: notifications } = useQuery({
    queryKey: ['notifications-unread'],
    queryFn: async () => {
      const res = await api.get('/notifications/unread');
      return (res.data?.data || []) as any[];
    },
  });

  const { data: activity } = useQuery({
    queryKey: ['activity'],
    queryFn: async () => {
      const res = await api.get('/activity');
      return (res.data?.data || []) as any[];
    },
  });

  const totalTasks = weeklyTasks?.length || 0;
  const completedTasks = weeklyTasks?.filter((t: any) => t.status === 'completed').length || 0;
  const inProgressTasks = weeklyTasks?.filter((t: any) => t.status === 'in_progress').length || 0;
  const pendingTasks = totalTasks - completedTasks - inProgressTasks;
  const completionPct = totalTasks > 0 ? Math.round((completedTasks / totalTasks) * 100) : 0;

  const todayTasks = weeklyTasks?.filter((t: any) => t.assigned_date?.startsWith(today)) || [];

  return (
    <div className="space-y-6">
      {/* Header */}
      <div>
        <h2 className="text-2xl sm:text-2xl sm:text-3xl font-bold tracking-tight text-foreground mb-1">
          Welcome back, {user?.first_name} 👋
        </h2>
        <p className="text-sm sm:text-base text-muted-foreground">{format(new Date(), 'EEEE, MMMM d, yyyy')}</p>
      </div>

      <AnnouncementBanner />

      {/* Stats */}
      <div className="grid grid-cols-2 gap-3 sm:gap-4">
        <StatCard icon={<CheckSquare className="w-5 h-5 text-primary" />} label="This Week" value={totalTasks} color="text-foreground" />
        <StatCard icon={<CheckCircle2 className="w-5 h-5 text-emerald-500" />} label="Completed" value={completedTasks} color="text-emerald-500" />
        <StatCard icon={<TrendingUp className="w-5 h-5 text-primary" />} label="Progress" value={`${completionPct}%`} color="text-primary" />
        <StatCard icon={<AlertCircle className="w-5 h-5 text-amber-500" />} label="Notifications" value={notifications?.length || 0} color="text-amber-500" />
      </div>

      {/* Today's Tasks */}
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <CalendarDays className="w-5 h-5 text-primary" />
            Today's Tasks
          </CardTitle>
          <CardDescription>Your assigned tasks for today</CardDescription>
        </CardHeader>
        <CardContent>
          {todayTasks.length === 0 ? (
            <div className="text-center py-8 text-muted-foreground">
              <CheckCircle2 className="w-12 h-12 mx-auto mb-3 opacity-20" />
              <p className="font-medium">No tasks for today!</p>
              <p className="text-sm">Enjoy your day or check your weekly schedule.</p>
            </div>
          ) : (
            <div className="space-y-2">
              {todayTasks.map((task: any) => (
                <div key={task.assignment_id} className={`flex items-start sm:items-center gap-3 px-3 sm:px-4 py-3 rounded-xl border transition-all ${
                  task.status === 'completed' ? 'bg-emerald-500/5 border-emerald-500/20' :
                  task.status === 'in_progress' ? 'bg-amber-500/5 border-amber-500/20' :
                  'bg-muted/30 border-border'
                }`}>
                  {task.status === 'completed' ? <CheckCircle2 className="w-5 h-5 text-emerald-500" /> :
                   task.status === 'in_progress' ? <Play className="w-5 h-5 text-amber-500" /> :
                   <Clock className="w-5 h-5 text-muted-foreground" />}
                  <div className="flex-1">
                    <span className="font-medium text-foreground">{task.task_title}</span>
                    {task.board_name && (
                      <span className="ml-2 px-1.5 py-0.5 rounded text-[10px] font-medium bg-primary/10 text-primary border border-primary/20">
                        {task.board_name}
                      </span>
                    )}
                  </div>
                  <span className={`text-xs font-medium px-2 py-1 rounded-lg ${
                    task.status === 'completed' ? 'text-emerald-500 bg-emerald-500/10' :
                    task.status === 'in_progress' ? 'text-amber-500 bg-amber-500/10' :
                    'text-muted-foreground bg-muted'
                  }`}>
                    {task.status === 'completed' ? '✓ Done' : task.status === 'in_progress' ? 'Working...' : 'Pending'}
                  </span>
                </div>
              ))}
            </div>
          )}
        </CardContent>
      </Card>

      {/* Weekly Progress */}
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <BarChart3 className="w-5 h-5 text-primary" />
            Weekly Progress
          </CardTitle>
        </CardHeader>
        <CardContent>
          <div className="flex items-center gap-4 mb-4">
            <div className="flex-1">
              <div className="w-full h-4 bg-muted rounded-full overflow-hidden">
                <div className="h-full bg-gradient-to-r from-primary to-accent rounded-full transition-all duration-1000"
                  style={{ width: `${completionPct}%` }} />
              </div>
            </div>
            <span className="text-2xl font-bold text-primary min-w-[60px] text-right">{completionPct}%</span>
          </div>
          <div className="grid grid-cols-3 gap-3 text-center">
            <div className="p-3 rounded-xl bg-muted/40 border border-border">
              <p className="text-2xl font-bold text-emerald-500">{completedTasks}</p>
              <p className="text-xs text-muted-foreground">Completed</p>
            </div>
            <div className="p-3 rounded-xl bg-muted/40 border border-border">
              <p className="text-2xl font-bold text-amber-500">{inProgressTasks}</p>
              <p className="text-xs text-muted-foreground">In Progress</p>
            </div>
            <div className="p-3 rounded-xl bg-muted/40 border border-border">
              <p className="text-2xl font-bold text-muted-foreground">{pendingTasks}</p>
              <p className="text-xs text-muted-foreground">Pending</p>
            </div>
          </div>
        </CardContent>
      </Card>

      {/* Activity History */}
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <Play className="w-5 h-5 text-primary" />
            Activity History
          </CardTitle>
          <CardDescription>Recent account/workflow activity</CardDescription>
        </CardHeader>
        <CardContent>
          {activity?.length ? (
            <div className="space-y-3">
              {activity.slice(0, 6).map((log: any) => (
                <div
                  key={log.id}
                  className="flex items-start justify-between gap-4 p-3 rounded-xl bg-muted/30 border border-border"
                >
                  <div className="min-w-0">
                    <p className="text-sm font-medium text-foreground truncate">
                      {log.action} <span className="text-muted-foreground font-normal">({log.table_name})</span>
                    </p>
                    <p className="text-xs text-muted-foreground mt-1">
                      {log.created_at ? fmtDateTime(log.created_at, 'MMM d, HH:mm') : ''}
                    </p>
                  </div>
                </div>
              ))}
            </div>
          ) : (
            <p className="text-muted-foreground text-sm">No activity yet.</p>
          )}
        </CardContent>
      </Card>
    </div>
  );
};

// ═══════════════════════════════════════════════════════════
// Leader / Manager Dashboard
// ═══════════════════════════════════════════════════════════

const LeaderDashboard = () => {
  const { user } = useAuthStore();

  const { data: employees } = useQuery({
    queryKey: ['all-employees-active'],
    queryFn: async () => {
      const res = await api.get('/employees?active=true');
      return (res.data?.data || []) as any[];
    },
  });

  const { data: boardStats } = useQuery({
    queryKey: ['board-stats'],
    queryFn: async () => {
      const res = await api.get('/tasks/boards/stats');
      return (res.data?.data || []) as any[];
    },
  });

  const { data: shifts } = useQuery({
    queryKey: ['shifts'],
    queryFn: async () => {
      const res = await api.get('/shifts');
      return (res.data?.data || []) as any[];
    },
  });

  const { data: notifications } = useQuery({
    queryKey: ['notifications-unread'],
    queryFn: async () => {
      const res = await api.get('/notifications/unread');
      return (res.data?.data || []) as any[];
    },
  });

  const { data: activity } = useQuery({
    queryKey: ['activity'],
    queryFn: async () => {
      const res = await api.get('/activity');
      return (res.data?.data || []) as any[];
    },
  });

  const totalEmployees = employees?.filter((e: any) => e.role === 'employee').length || 0;

  // Group employees by shift
  const employeesByShift: Record<string, { name: string; color: string; count: number }> = {};
  shifts?.forEach((s: any) => {
    const count = employees?.filter((e: any) => e.default_shift_id === s.id && e.role === 'employee').length || 0;
    employeesByShift[s.id] = { name: s.name, color: s.color_code || '#0CCCCC', count };
  });

  const totalBoardTasks = boardStats?.reduce((sum: number, b: any) => sum + b.total_assigned, 0) || 0;
  const totalBoardDone = boardStats?.reduce((sum: number, b: any) => sum + b.total_completed, 0) || 0;
  const overallPct = totalBoardTasks > 0 ? Math.round((totalBoardDone / totalBoardTasks) * 100) : 0;

  return (
    <div className="space-y-6">
      {/* Header */}
      <div>
        <h2 className="text-2xl sm:text-2xl sm:text-3xl font-bold tracking-tight text-foreground mb-1 flex items-center gap-2 sm:gap-3">
          <Shield className="w-6 h-6 sm:w-8 sm:h-8 text-primary" />
          Dashboard
        </h2>
        <p className="text-sm sm:text-base text-muted-foreground">
          {format(new Date(), 'EEEE, MMMM d, yyyy')} · Welcome, {user?.first_name}
        </p>
      </div>

      <AnnouncementBanner />

      {/* Stats */}
      <div className="grid grid-cols-2 gap-3 sm:gap-4">
        <StatCard icon={<Users className="w-5 h-5 text-primary" />} label="Active Employees" value={totalEmployees} color="text-primary" />
        <StatCard icon={<CheckSquare className="w-5 h-5 text-primary" />} label="Total Tasks" value={totalBoardTasks} color="text-primary" />
        <StatCard icon={<TrendingUp className="w-5 h-5 text-emerald-500" />} label="Overall Progress" value={`${overallPct}%`} color="text-emerald-500" />
        <StatCard icon={<AlertCircle className="w-5 h-5 text-amber-500" />} label="Pending Alerts" value={notifications?.length || 0} color="text-amber-500" />
      </div>

      {/* Employees by Shift */}
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <Briefcase className="w-5 h-5 text-primary" />
            Team by Shift
          </CardTitle>
          <CardDescription>Employee distribution across shifts</CardDescription>
        </CardHeader>
        <CardContent>
          {Object.keys(employeesByShift).length === 0 ? (
            <p className="text-muted-foreground text-center py-4">No shifts configured</p>
          ) : (
            <div className="grid grid-cols-1 sm:grid-cols-2 md:grid-cols-3 gap-3 sm:gap-4">
              {Object.entries(employeesByShift).map(([shiftId, shift]) => (
                <div key={shiftId} className="p-4 rounded-xl bg-muted/30 border border-border flex items-center gap-4">
                  <div className="w-3 h-12 rounded-full" style={{ backgroundColor: shift.color }} />
                  <div>
                    <p className="font-semibold text-foreground">{shift.name}</p>
                    <p className="text-2xl font-bold text-foreground">{shift.count}</p>
                    <p className="text-xs text-muted-foreground">employees</p>
                  </div>
                </div>
              ))}
            </div>
          )}
        </CardContent>
      </Card>

      {/* Board Completion Overview */}
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <BarChart3 className="w-5 h-5 text-primary" />
            Board Completion Tracker
          </CardTitle>
          <CardDescription>Real-time task completion across all boards</CardDescription>
        </CardHeader>
        <CardContent>
          {!boardStats || boardStats.length === 0 ? (
            <p className="text-muted-foreground text-center py-4">No boards with tasks yet</p>
          ) : (
            <div className="space-y-4">
              {boardStats.map((board: any) => (
                <div key={board.board_id} className="p-4 rounded-xl bg-muted/20 border border-border">
                  <div className="flex items-center justify-between mb-2">
                    <span className="font-semibold text-foreground">{board.board_name}</span>
                    <span className="text-sm font-bold text-primary">{board.completion_pct}%</span>
                  </div>
                  <div className="w-full h-3 bg-muted rounded-full overflow-hidden mb-2">
                    <div className="h-full bg-gradient-to-r from-primary to-accent rounded-full transition-all duration-500"
                      style={{ width: `${board.completion_pct}%` }} />
                  </div>
                  <div className="flex items-center gap-3 sm:gap-4 text-xs flex-wrap">
                    <span className="text-emerald-500">✓ {board.total_completed} done</span>
                    <span className="text-amber-500">▶ {board.total_in_progress} active</span>
                    <span className="text-muted-foreground">◌ {board.total_pending} pending</span>
                    <span className="text-muted-foreground/60 ml-auto">{board.total_assigned} total</span>
                  </div>
                </div>
              ))}
            </div>
          )}
        </CardContent>
      </Card>

      {/* Activity History */}
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <Play className="w-5 h-5 text-primary" />
            Activity History
          </CardTitle>
          <CardDescription>Recent account/workflow activity</CardDescription>
        </CardHeader>
        <CardContent>
          {activity?.length ? (
            <div className="space-y-3">
              {activity.slice(0, 6).map((log: any) => (
                <div
                  key={log.id}
                  className="flex items-start justify-between gap-4 p-3 rounded-xl bg-muted/30 border border-border"
                >
                  <div className="min-w-0">
                    <p className="text-sm font-medium text-foreground truncate">
                      {log.action} <span className="text-muted-foreground font-normal">({log.table_name})</span>
                    </p>
                    <p className="text-xs text-muted-foreground mt-1">
                      {log.created_at ? fmtDateTime(log.created_at, 'MMM d, HH:mm') : ''}
                    </p>
                  </div>
                </div>
              ))}
            </div>
          ) : (
            <p className="text-muted-foreground text-sm">No activity yet.</p>
          )}
        </CardContent>
      </Card>
    </div>
  );
};

// ───────────────────────────────────────────────────────────
// Stat Card
// ───────────────────────────────────────────────────────────

const StatCard = ({ label, value, color, icon }: { label: string; value: string | number; color: string; icon: React.ReactNode }) => (
  <Card>
    <CardContent className="p-3 sm:p-4 flex items-center gap-2 sm:gap-3">
      <div className="p-2 sm:p-2.5 rounded-xl bg-muted/60">{icon}</div>
      <div>
        <p className="text-[10px] sm:text-xs text-muted-foreground uppercase tracking-wider">{label}</p>
        <p className={`text-lg sm:text-xl font-bold ${color}`}>{value}</p>
      </div>
    </CardContent>
  </Card>
);
