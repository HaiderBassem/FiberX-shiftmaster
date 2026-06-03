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
      {/* Header */}
      <motion.div variants={itemVariants}>
        <h2 className="text-2xl sm:text-2xl sm:text-3xl font-bold tracking-tight text-foreground mb-1">
          Welcome back, {user?.first_name} 👋
        </h2>
        <p className="text-sm sm:text-base text-muted-foreground">{format(new Date(), 'EEEE, MMMM d, yyyy')}</p>
      </motion.div>

      <motion.div variants={itemVariants}>
        <AnnouncementBanner />
      </motion.div>

      {/* Stats */}
      <motion.div variants={itemVariants} className="grid grid-cols-2 md:grid-cols-4 gap-3 sm:gap-4">
        <StatCard icon={<CheckSquare className="w-6 h-6 text-primary" />} label="This Week" value={totalTasks} color="text-foreground" glowColor="rgba(12,204,204,0.15)" />
        <StatCard icon={<CheckCircle2 className="w-6 h-6 text-emerald-500" />} label="Completed" value={completedTasks} color="text-emerald-500" glowColor="rgba(16,185,129,0.15)" />
        <StatCard icon={<TrendingUp className="w-6 h-6 text-primary" />} label="Progress" value={`${completionPct}%`} color="text-primary" glowColor="rgba(12,204,204,0.15)" />
        <StatCard icon={<AlertCircle className="w-6 h-6 text-amber-500" />} label="Notifications" value={notifications?.length || 0} color="text-amber-500" glowColor="rgba(245,158,11,0.15)" />
      </motion.div>

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
                      
                      <span className={`text-xs font-bold px-2.5 py-1 rounded-lg uppercase tracking-wider shrink-0 ${
                        task.status === 'completed' ? 'text-emerald-500 bg-emerald-500/20' :
                        task.status === 'in_progress' ? 'text-amber-500 bg-amber-500/20' :
                        'text-muted-foreground bg-muted'
                      }`}>
                        {task.status === 'completed' ? 'Done' : task.status === 'in_progress' ? 'Active' : 'Pending'}
                      </span>
                    </motion.div>
                  ))}
                </div>
              )}
            </CardContent>
          </Card>
        </motion.div>
      </div>

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
        <h2 className="text-2xl sm:text-2xl sm:text-3xl font-bold tracking-tight text-foreground mb-1 flex items-center gap-2 sm:gap-3">
          <Shield className="w-6 h-6 sm:w-8 sm:h-8 text-primary" />
          Analytics Dashboard
        </h2>
        <p className="text-sm sm:text-base text-muted-foreground">
          {format(new Date(), 'EEEE, MMMM d, yyyy')} · Welcome, {user?.first_name}
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

      {/* Activity History */}
      <motion.div variants={itemVariants}>
        <Card className="border-border/50 shadow-lg bg-background/50 backdrop-blur-sm">
          <CardHeader>
            <CardTitle className="flex items-center gap-2 text-xl">
              <Play className="w-5 h-5 text-primary" />
              Activity History
            </CardTitle>
            <CardDescription>Recent system events</CardDescription>
          </CardHeader>
          <CardContent>
            {activity?.length ? (
              <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-3 sm:gap-4">
                {activity.slice(0, 9).map((log: any) => (
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
              <p className="text-muted-foreground text-sm py-6 text-center">No activity yet.</p>
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
