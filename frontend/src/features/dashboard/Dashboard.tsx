import { useState } from 'react';
import { useTranslation } from 'react-i18next';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import api from '@/lib/api';
import { useAuthStore } from '@/store/authStore';
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { RichTextEditor } from '@/components/RichTextEditor';
import {
  Users, CheckCircle2, Clock, Play, CalendarDays, Shield, BarChart3,
  CheckSquare, TrendingUp, Briefcase, AlertCircle, X, AlertTriangle
} from 'lucide-react';
import { format, startOfWeek } from 'date-fns';
import { fmtDateTime } from '@/lib/dateUtils';
import { AnnouncementBanner } from '../announcements/AnnouncementBanner';
import { motion } from 'framer-motion';
import { 
  ResponsiveContainer, PieChart, Pie, Cell, Tooltip, 
  BarChart, Bar, XAxis, YAxis, CartesianGrid, Legend 
} from 'recharts';

// ───────────────────────────────────────────────────────────
// Shared Animations
// ───────────────────────────────────────────────────────────
const containerVariants: any = {
  hidden: { opacity: 0 },
  show: { opacity: 1, transition: { staggerChildren: 0.1 } }
};

const itemVariants: any = {
  hidden: { opacity: 0, y: 20 },
  show: { opacity: 1, y: 0, transition: { type: "spring", stiffness: 300, damping: 24 } }
};

// ───────────────────────────────────────────────────────────
// Completion Dialog
// ───────────────────────────────────────────────────────────

const CompletionDialog = ({
  task,
  onClose,
  onComplete,
  isPending,
}: {
  task: any;
  onClose: () => void;
  onComplete: (completionType: string, notes: string) => void;
  isPending: boolean;
}) => {
  const [notes, setNotes] = useState('');

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50 backdrop-blur-sm">
      <div className="bg-card border border-border rounded-2xl shadow-2xl w-full max-w-md mx-4 overflow-hidden">
        <div className="px-6 py-4 border-b border-border flex items-center justify-between">
          <div>
            <h3 className="text-lg font-semibold text-foreground">Complete Task</h3>
            <p className="text-sm text-muted-foreground">{task.task_title}</p>
          </div>
          <button onClick={onClose} className="p-1.5 rounded-lg hover:bg-muted transition-colors">
            <X className="w-5 h-5 text-muted-foreground" />
          </button>
        </div>

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
  const { t } = useTranslation();
  const queryClient = useQueryClient();
  const today = format(new Date(), 'yyyy-MM-dd');
  const weekStart = format(startOfWeek(new Date(), { weekStartsOn: 0 }), 'yyyy-MM-dd');

  const [completingTask, setCompletingTask] = useState<any>(null);

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

  const { data: userDepartment } = useQuery({
    queryKey: ['my-department', user?.department_id],
    queryFn: async () => {
      if (!user?.department_id) return null;
      const res = await api.get(`/departments/${user?.department_id}`);
      return res.data?.data;
    },
    enabled: !!user?.department_id,
  });

  const activeModules = userDepartment?.active_modules || [];
  const tasksEnabled = !userDepartment || activeModules.includes('tasks');

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

  const totalTasks = weeklyTasks?.length || 0;
  const completedTasks = (weeklyTasks || []).filter((t: any) => t.status === 'completed').length;
  const inProgressTasks = (weeklyTasks || []).filter((t: any) => t.status === 'in_progress').length;
  const pendingTasks = totalTasks - completedTasks - inProgressTasks;
  const completionPct = totalTasks > 0 ? Math.round((completedTasks / totalTasks) * 100) : 0;

  const todayTasks = (weeklyTasks || []).filter((t: any) => t.assigned_date?.startsWith(today));

  // Chart Data
  const pieData = [
    { name: 'Completed', value: completedTasks, color: '#10b981' }, // emerald-500
    { name: 'In Progress', value: inProgressTasks, color: '#f59e0b' }, // amber-500
    { name: 'Pending', value: pendingTasks, color: '#64748b' }, // slate-500
  ].filter(d => d.value > 0);

  return (
    <motion.div 
      className="space-y-6 max-w-[1600px] mx-auto pb-8"
      variants={containerVariants}
      initial="hidden"
      animate="show"
    >
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

      {/* Header */}
      <motion.div variants={itemVariants}>
        <h2 className="text-2xl sm:text-2xl sm:text-3xl font-bold tracking-tight text-foreground mb-1">
          {t('dashboard.welcome_back')}, {user?.first_name?.split(' ')[0]} 👋
        </h2>
        <p className="text-sm sm:text-base text-muted-foreground">{format(new Date(), 'EEEE, MMMM d, yyyy')}</p>
      </motion.div>

      <motion.div variants={itemVariants}>
        <AnnouncementBanner />
      </motion.div>

      {/* Stats */}
      <motion.div variants={itemVariants} className="grid grid-cols-2 md:grid-cols-4 gap-3 sm:gap-4">
        {tasksEnabled && (
          <>
            <StatCard icon={<CheckSquare className="w-6 h-6 text-primary" />} label="This Week" value={totalTasks} color="text-foreground" glowColor="rgba(12,204,204,0.15)" />
            <StatCard icon={<CheckCircle2 className="w-6 h-6 text-emerald-500" />} label="Completed" value={completedTasks} color="text-emerald-500" glowColor="rgba(16,185,129,0.15)" />
            <StatCard icon={<TrendingUp className="w-6 h-6 text-primary" />} label="Progress" value={`${completionPct}%`} color="text-primary" glowColor="rgba(12,204,204,0.15)" />
          </>
        )}
        <StatCard icon={<AlertCircle className="w-6 h-6 text-amber-500" />} label="Notifications" value={notifications?.length || 0} color="text-amber-500" glowColor="rgba(245,158,11,0.15)" />
      </motion.div>

      {tasksEnabled && (
        <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
        {/* Weekly Progress Chart */}
        <motion.div variants={itemVariants}>
          <Card className="h-full border-border/50 shadow-lg bg-background/50 backdrop-blur-sm">
            <CardHeader>
              <CardTitle className="flex items-center gap-2 text-xl">
                <BarChart3 className="w-5 h-5 text-primary" />
                Weekly Progress
              </CardTitle>
            </CardHeader>
            <CardContent>
              {totalTasks === 0 ? (
                <div className="text-center py-12 text-muted-foreground">No tasks assigned this week</div>
              ) : (
                <div className="flex flex-col items-center justify-center h-[280px]">
                  <ResponsiveContainer width="100%" height="100%">
                    <PieChart>
                      <Pie
                        data={pieData}
                        cx="50%"
                        cy="50%"
                        innerRadius={70}
                        outerRadius={95}
                        paddingAngle={5}
                        dataKey="value"
                      >
                        {pieData.map((entry, index) => (
                          <Cell key={`cell-${index}`} fill={entry.color} stroke="transparent" />
                        ))}
                      </Pie>
                      <Tooltip 
                        contentStyle={{ backgroundColor: '#1e293b', border: 'none', borderRadius: '12px', color: '#fff' }}
                        itemStyle={{ color: '#fff' }}
                      />
                      <Legend verticalAlign="bottom" height={36} />
                    </PieChart>
                  </ResponsiveContainer>
                  <div className="absolute flex flex-col items-center justify-center pointer-events-none mt-[-36px]">
                    <span className="text-3xl font-bold text-foreground">{completionPct}%</span>
                    <span className="text-xs text-muted-foreground uppercase tracking-wider">Done</span>
                  </div>
                </div>
              )}
            </CardContent>
          </Card>
        </motion.div>

        {/* Today's Tasks */}
        <motion.div variants={itemVariants}>
          <Card className="h-full border-border/50 shadow-lg bg-background/50 backdrop-blur-sm">
            <CardHeader>
              <CardTitle className="flex items-center gap-2 text-xl">
                <CalendarDays className="w-5 h-5 text-primary" />
                Today's Tasks
              </CardTitle>
              <CardDescription>Your assigned tasks for today</CardDescription>
            </CardHeader>
            <CardContent>
              {todayTasks.length === 0 ? (
                <div className="text-center py-12 text-muted-foreground">
                  <CheckCircle2 className="w-16 h-16 mx-auto mb-4 opacity-20" />
                  <p className="font-semibold text-lg">No tasks for today!</p>
                  <p className="text-sm">Enjoy your day or check your weekly schedule.</p>
                </div>
              ) : (
                <div className="space-y-3 overflow-y-auto max-h-[280px] pr-2 no-scrollbar">
                  {todayTasks.map((task: any) => (
                    <motion.div 
                      key={task.assignment_id} 
                      whileHover={{ scale: 1.01 }}
                      className={`flex items-start sm:items-center gap-4 px-4 py-3 rounded-xl border transition-all ${
                        task.status === 'completed' ? 'bg-emerald-500/10 border-emerald-500/30 shadow-sm shadow-emerald-500/10' :
                        task.status === 'in_progress' ? 'bg-amber-500/10 border-amber-500/30 shadow-sm shadow-amber-500/10' :
                        'bg-card border-border shadow-sm'
                      }`}
                    >
                      {task.status === 'completed' ? <CheckCircle2 className="w-6 h-6 text-emerald-500 flex-shrink-0" /> :
                       task.status === 'in_progress' ? <Play className="w-6 h-6 text-amber-500 flex-shrink-0" /> :
                       <Clock className="w-6 h-6 text-muted-foreground flex-shrink-0" />}
                      
                      <div className="flex-1 min-w-0">
                        <span className="font-semibold text-foreground text-sm sm:text-base block truncate">{task.task_title}</span>
                        {task.board_name && (
                          <span className="inline-block mt-1 px-2 py-0.5 rounded-md text-[10px] font-bold tracking-wider bg-primary/15 text-primary border border-primary/20 uppercase">
                            {task.board_name}
                          </span>
                        )}
                      </div>
                      
                      <div className="flex flex-col gap-2 shrink-0">
                        <span className={`text-[10px] font-bold px-2 py-0.5 rounded uppercase tracking-wider text-center ${
                          task.status === 'completed' ? 'text-emerald-500 bg-emerald-500/10' :
                          task.status === 'in_progress' ? 'text-amber-500 bg-amber-500/10' :
                          'text-muted-foreground bg-muted'
                        }`}>
                          {task.status === 'completed' ? 'Done' : task.status === 'in_progress' ? 'Active' : 'Pending'}
                        </span>
                        
                        {task.execution_id && task.status === 'pending' && (
                          <Button size="sm" onClick={() => startTask.mutate(task.execution_id!)} disabled={startTask.isPending}
                            className="bg-amber-500 hover:bg-amber-400 text-white gap-1.5 text-xs h-7 px-2">
                            <Play className="w-3 h-3" /> Start
                          </Button>
                        )}
                        {task.execution_id && task.status === 'in_progress' && (
                          <Button size="sm" onClick={() => setCompletingTask(task)}
                            className="bg-emerald-500 hover:bg-emerald-400 text-white gap-1.5 text-xs h-7 px-2">
                            <CheckCircle2 className="w-3 h-3" /> Complete
                          </Button>
                        )}
                      </div>
                    </motion.div>
                  ))}
                </div>
              )}
            </CardContent>
          </Card>
        </motion.div>
      </div>
      )}

      {/* Activity History */}
      <motion.div variants={itemVariants}>
        <Card className="border-border/50 shadow-lg bg-background/50 backdrop-blur-sm">
          <CardHeader>
            <CardTitle className="flex items-center gap-2 text-xl">
              <Play className="w-5 h-5 text-primary" />
              Activity History
            </CardTitle>
            <CardDescription>Recent account/workflow activity</CardDescription>
          </CardHeader>
          <CardContent>
            {activity?.length ? (
              <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-3 sm:gap-4">
                {activity.slice(0, 6).map((log: any) => (
                  <div
                    key={log.id}
                    className="flex items-start justify-between gap-4 p-4 rounded-xl bg-card border border-border/60 shadow-sm hover:border-primary/40 transition-colors"
                  >
                    <div className="min-w-0">
                      <p className="text-sm font-semibold text-foreground truncate">
                        {log.action} <span className="text-muted-foreground font-normal">({log.table_name})</span>
                      </p>
                      <p className="text-xs text-muted-foreground mt-1.5 flex items-center gap-1.5">
                        <Clock className="w-3 h-3" />
                        {log.created_at ? fmtDateTime(log.created_at, 'MMM d, HH:mm') : ''}
                      </p>
                    </div>
                  </div>
                ))}
              </div>
            ) : (
              <p className="text-muted-foreground text-sm text-center py-6">No activity yet.</p>
            )}
          </CardContent>
        </Card>
      </motion.div>
    </motion.div>
  );
};

// ═══════════════════════════════════════════════════════════
// Leader / Manager Dashboard
// ═══════════════════════════════════════════════════════════

const LeaderDashboard = () => {
  const user = useAuthStore(s => s.user);
  const { t } = useTranslation();
  const [selectedShiftFilter, setSelectedShiftFilter] = useState<string>('all');

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

  const { data: todayLeaves } = useQuery({
    queryKey: ['dashboard-leaves-history'],
    queryFn: async () => {
      const res = await api.get('/leaves/history');
      return res.data?.data || [];
    },
  });

  const { data: todaySwaps } = useQuery({
    queryKey: ['dashboard-swaps-history'],
    queryFn: async () => {
      const res = await api.get('/swaps/history');
      return res.data?.data || [];
    },
  });

  const { data: todaySchedules } = useQuery({
    queryKey: ['dashboard-schedules-daily'],
    queryFn: async () => {
      const res = await api.get('/schedules/daily');
      return res.data?.data || [];
    },
  });

  const { data: notifications } = useQuery({
    queryKey: ['notifications-unread'],
    queryFn: async () => {
      const res = await api.get('/notifications/unread');
      return (res.data?.data || []) as any[];
    },
  });


  const totalEmployees = (employees || []).filter((e: any) => e.role === 'employee').length;

  // Group employees by shift for Recharts Pie
  const shiftPieData = (shifts || []).map((s: any) => {
    const count = (employees || []).filter((e: any) => e.default_shift_id === s.id && e.role === 'employee').length;
    return { name: s.name, value: count, color: s.color_code || '#0CCCCC' };
  }).filter(s => s.value > 0);

  const totalBoardTasks = boardStats?.reduce((sum: number, b: any) => sum + b.total_assigned, 0) || 0;
  const totalBoardDone = boardStats?.reduce((sum: number, b: any) => sum + b.total_completed, 0) || 0;
  const overallPct = totalBoardTasks > 0 ? Math.round((totalBoardDone / totalBoardTasks) * 100) : 0;

  // Board Stats for Bar Chart
  const boardBarData = (boardStats || []).map((b: any) => ({
    name: b.board_name,
    Completed: b.total_completed,
    Active: b.total_in_progress,
    Pending: b.total_pending,
    total: b.total_assigned
  }));

  return (
    <motion.div 
      className="space-y-6 max-w-[1600px] mx-auto pb-8"
      variants={containerVariants}
      initial="hidden"
      animate="show"
    >
      {/* Header */}
        <motion.div variants={itemVariants}>
          <h2 className="text-2xl sm:text-3xl font-bold tracking-tight text-foreground mb-1 flex items-center gap-2 sm:gap-3">
            <Shield className="w-6 h-6 sm:w-8 sm:h-8 text-primary" />
            {t('dashboard.analytics')}
          </h2>
          <p className="text-sm sm:text-base text-muted-foreground">
          {format(new Date(), 'EEEE, MMMM d, yyyy')} • {t('topbar.welcome')}, {user?.first_name?.split(' ')[0]}
          </p>
        </motion.div>

      <motion.div variants={itemVariants}>
        <AnnouncementBanner />
      </motion.div>

      {/* Stats */}
      <motion.div variants={itemVariants} className="grid grid-cols-2 md:grid-cols-4 gap-3 sm:gap-4">
        <StatCard icon={<Users className="w-6 h-6 text-primary" />} label="Active Employees" value={totalEmployees} color="text-primary" glowColor="rgba(12,204,204,0.15)" />
        <StatCard icon={<CheckSquare className="w-6 h-6 text-primary" />} label="Total Tasks" value={totalBoardTasks} color="text-primary" glowColor="rgba(12,204,204,0.15)" />
        <StatCard icon={<TrendingUp className="w-6 h-6 text-emerald-500" />} label="Overall Progress" value={`${overallPct}%`} color="text-emerald-500" glowColor="rgba(16,185,129,0.15)" />
        <StatCard icon={<AlertCircle className="w-6 h-6 text-amber-500" />} label="Pending Alerts" value={notifications?.length || 0} color="text-amber-500" glowColor="rgba(245,158,11,0.15)" />
      </motion.div>

      <div className="grid grid-cols-1 xl:grid-cols-3 gap-6">
        {/* Employees by Shift (Donut Chart) */}
        <motion.div variants={itemVariants} className="xl:col-span-1">
          <Card className="h-full border-border/50 shadow-lg bg-background/50 backdrop-blur-sm">
            <CardHeader>
              <CardTitle className="flex items-center gap-2 text-xl">
                <Briefcase className="w-5 h-5 text-primary" />
                Team Distribution
              </CardTitle>
              <CardDescription>Employees by Shift</CardDescription>
            </CardHeader>
            <CardContent>
              {shiftPieData.length === 0 ? (
                <p className="text-muted-foreground text-center py-12">No shifts configured</p>
              ) : (
                <div className="flex flex-col items-center justify-center h-[300px]">
                  <ResponsiveContainer width="100%" height="100%">
                    <PieChart>
                      <Pie
                        data={shiftPieData}
                        cx="50%"
                        cy="50%"
                        innerRadius={80}
                        outerRadius={110}
                        paddingAngle={4}
                        dataKey="value"
                      >
                        {shiftPieData.map((entry, index) => (
                          <Cell key={`cell-${index}`} fill={entry.color} stroke="transparent" />
                        ))}
                      </Pie>
                      <Tooltip 
                        contentStyle={{ backgroundColor: '#1e293b', border: 'none', borderRadius: '12px', color: '#fff' }}
                        itemStyle={{ color: '#fff' }}
                        formatter={(value, name) => [`${value} employees`, name]}
                      />
                      <Legend verticalAlign="bottom" height={36} />
                    </PieChart>
                  </ResponsiveContainer>
                  <div className="absolute flex flex-col items-center justify-center pointer-events-none mt-[-36px]">
                    <span className="text-4xl font-bold text-foreground">{totalEmployees}</span>
                    <span className="text-[10px] text-muted-foreground uppercase tracking-widest mt-1">Total</span>
                  </div>
                </div>
              )}
            </CardContent>
          </Card>
        </motion.div>

        {/* Board Completion Tracker (Bar Chart) */}
        <motion.div variants={itemVariants} className="xl:col-span-2">
          <Card className="h-full border-border/50 shadow-lg bg-background/50 backdrop-blur-sm">
            <CardHeader>
              <CardTitle className="flex items-center gap-2 text-xl">
                <BarChart3 className="w-5 h-5 text-primary" />
                Task Board Performance
              </CardTitle>
              <CardDescription>Completion vs Pending across all boards</CardDescription>
            </CardHeader>
            <CardContent>
              {!boardStats || boardStats.length === 0 ? (
                <p className="text-muted-foreground text-center py-12">No boards with tasks yet</p>
              ) : (
                <div className="h-[300px] w-full pt-4">
                  <ResponsiveContainer width="100%" height="100%">
                    <BarChart
                      data={boardBarData}
                      margin={{ top: 10, right: 10, left: -20, bottom: 0 }}
                    >
                      <CartesianGrid strokeDasharray="3 3" vertical={false} stroke="rgba(255,255,255,0.05)" />
                      <XAxis dataKey="name" axisLine={false} tickLine={false} tick={{ fill: '#64748b', fontSize: 12 }} />
                      <YAxis axisLine={false} tickLine={false} tick={{ fill: '#64748b', fontSize: 12 }} />
                      <Tooltip
                        contentStyle={{ backgroundColor: '#1e293b', border: 'none', borderRadius: '12px', color: '#fff', boxShadow: '0 10px 15px -3px rgba(0, 0, 0, 0.5)' }}
                        cursor={{ fill: 'rgba(255,255,255,0.05)' }}
                      />
                      <Legend iconType="circle" wrapperStyle={{ paddingTop: '20px' }} />
                      <Bar dataKey="Completed" stackId="a" fill="#10b981" radius={[0, 0, 4, 4]} barSize={40} />
                      <Bar dataKey="Active" stackId="a" fill="#f59e0b" />
                      <Bar dataKey="Pending" stackId="a" fill="#334155" radius={[4, 4, 0, 0]} />
                    </BarChart>
                  </ResponsiveContainer>
                </div>
              )}
            </CardContent>
          </Card>
        </motion.div>
      </div>

      {/* Active Staff Today */}
      <motion.div variants={itemVariants}>
        <Card className="border-border/50 shadow-lg bg-background/50 backdrop-blur-sm">
          <CardHeader className="flex flex-row items-start sm:items-center justify-between gap-4">
            <div>
              <CardTitle className="flex items-center gap-2 text-xl">
                <Users className="w-5 h-5 text-primary" />
                Active Staff Today
              </CardTitle>
              <CardDescription>Employees scheduled for today</CardDescription>
            </div>
            <div className="flex items-center gap-2 shrink-0">
              <select
                className="bg-muted/50 border border-border/50 text-sm rounded-lg px-3 py-1.5 focus:ring-1 focus:ring-primary outline-none"
                value={selectedShiftFilter}
                onChange={(e) => setSelectedShiftFilter(e.target.value)}
              >
                <option value="all">All Shifts</option>
                {shifts?.map((s: any) => (
                  <option key={s.id} value={s.id.toString()}>{s.name}</option>
                ))}
              </select>
            </div>
          </CardHeader>
          <CardContent>
            {employees ? (() => {
              const todayStr = new Date().toISOString().split('T')[0];
              const staff = employees.filter((e: any) => {
                if (e.role !== 'employee') return false;
                
                let effectiveShiftId = e.default_shift_id;
                
                // Check for approved Swaps today
                const todaysSwap = todaySwaps?.find((s: any) => 
                  s.shift_date?.startsWith(todayStr) && 
                  s.status === 'approved' &&
                  (s.requester_id === e.id || s.target_employee_id === e.id)
                );
                
                if (todaysSwap) {
                   if (todaysSwap.requester_id === e.id) {
                      return false; // Swapped out, not working
                   } else if (todaysSwap.target_employee_id === e.id) {
                      const requester = employees.find((req: any) => req.id === todaysSwap.requester_id);
                      if (requester) effectiveShiftId = requester.default_shift_id;
                   }
                }

                // Check for approved Leaves today
                const isOnLeave = todayLeaves?.some((l: any) => {
                  if (l.employee_id !== e.id) return false;
                  if (l.status !== 'approved' && l.status !== 'approved_by_manager' && l.status !== 'approved_by_team_leader') return false;
                  const start = new Date(l.start_date).toISOString().split('T')[0];
                  const end = new Date(l.end_date).toISOString().split('T')[0];
                  return todayStr >= start && todayStr <= end;
                });
                
                if (isOnLeave) return false;

                // Check for OFF schedules today
                const todaySched = todaySchedules?.find((s: any) => s.employee_id === e.id && s.shift_date?.startsWith(todayStr));
                if (todaySched) {
                    if (todaySched.shift_status === 'off' || todaySched.shift_status === 'leave' || todaySched.shift_status === 'vacation') {
                        return false;
                    }
                    if (todaySched.shift_id) {
                        effectiveShiftId = todaySched.shift_id;
                    }
                }

                return selectedShiftFilter === 'all' || effectiveShiftId?.toString() === selectedShiftFilter;
              });

              if (!staff.length) return <p className="text-muted-foreground text-sm py-6 text-center">No staff found for this shift.</p>;
              return (
                <div className="grid grid-cols-1 sm:grid-cols-2 md:grid-cols-3 lg:grid-cols-4 gap-3">
                  {staff.map((emp: any) => {
                    // Re-calculate effective shift for display
                    let effectiveShiftId = emp.default_shift_id;
                    const todaysSwap = todaySwaps?.find((s: any) => s.shift_date?.startsWith(todayStr) && s.status === 'approved' && s.target_employee_id === emp.id);
                    if (todaysSwap) {
                        const requester = employees.find((req: any) => req.id === todaysSwap.requester_id);
                        if (requester) effectiveShiftId = requester.default_shift_id;
                    }
                    const shift = shifts?.find((s: any) => s.id === effectiveShiftId);
                    
                    return (
                      <div key={emp.id} className="flex items-center gap-3 p-3 rounded-xl bg-card border border-border/60 shadow-sm hover:border-primary/40 transition-colors">
                        {emp.profile_image ? (
                          <img src={emp.profile_image} alt="" className="w-10 h-10 rounded-full object-cover shrink-0" />
                        ) : (
                          <div className="w-10 h-10 rounded-full bg-primary/10 flex items-center justify-center text-primary font-bold text-lg shrink-0">
                            {emp.first_name?.[0]}{emp.last_name?.[0]}
                          </div>
                        )}
                        <div className="min-w-0 flex-1">
                          <p className="text-sm font-semibold text-foreground truncate">{emp.first_name} {emp.last_name}</p>
                          <p className="text-xs text-muted-foreground truncate flex items-center gap-1.5 mt-0.5">
                            <span className="w-2 h-2 rounded-full" style={{ backgroundColor: shift?.color_code || '#ccc' }} />
                            {shift?.name || 'No Shift'}
                            {todaysSwap && <span className="ml-1 text-[9px] bg-indigo-500/20 text-indigo-500 px-1 rounded">Swapped In</span>}
                          </p>
                        </div>
                      </div>
                    );
                  })}
                </div>
              );
            })() : (
              <p className="text-muted-foreground text-sm py-6 text-center">Loading staff...</p>
            )}
          </CardContent>
        </Card>
      </motion.div>
    </motion.div>
  );
};

// ───────────────────────────────────────────────────────────
// Stat Card
// ───────────────────────────────────────────────────────────

const StatCard = ({ label, value, color, icon, glowColor }: { label: string; value: string | number; color: string; icon: React.ReactNode, glowColor?: string }) => (
  <motion.div whileHover={{ scale: 1.02, y: -2 }} transition={{ type: "spring", stiffness: 400, damping: 25 }}>
    <Card 
      className="overflow-hidden border-border/40 relative bg-background/60 backdrop-blur-md"
      style={{ boxShadow: glowColor ? `0 8px 32px -12px ${glowColor}` : 'none' }}
    >
      <div className="absolute top-0 right-0 p-4 opacity-10 pointer-events-none scale-150 transform translate-x-4 -translate-y-4">
        {icon}
      </div>
      <CardContent className="p-4 sm:p-5 flex flex-col justify-center h-full relative z-10">
        <div className="flex items-center gap-3 mb-2 sm:mb-3">
          <div className="p-2 sm:p-2.5 rounded-xl bg-muted/60 border border-white/5 backdrop-blur-xl shadow-inner">{icon}</div>
          <p className="text-[10px] sm:text-xs font-bold text-muted-foreground uppercase tracking-widest">{label}</p>
        </div>
        <div>
          <p className={`text-2xl sm:text-4xl font-extrabold tracking-tight ${color}`}>{value}</p>
        </div>
      </CardContent>
    </Card>
  </motion.div>
);
