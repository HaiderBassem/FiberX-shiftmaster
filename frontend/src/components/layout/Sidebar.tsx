import { Link, useLocation } from 'react-router-dom';
import { useAuthStore } from '@/store/authStore';
import { useQuery } from '@tanstack/react-query';
import api from '@/lib/api';
import {
  LayoutDashboard, Users, Calendar, CheckSquare,
  ShieldCheck, ClipboardList, Building2, Database,
  Clock, X, Table, BookOpen, Inbox, Megaphone, CalendarDays, Link as LinkIcon
} from 'lucide-react';
const navItems = [
  { to: '/', label: 'Dashboard', icon: LayoutDashboard, roles: ['employee', 'team_leader', 'manager', 'admin'], core: true },

  { to: '/tasks', label: 'My Tasks', icon: CheckSquare, roles: ['employee', 'team_leader', 'manager', 'admin'], moduleId: 'tasks' },
  { to: '/handovers', label: 'Handovers', icon: ClipboardList, roles: ['employee', 'team_leader', 'manager', 'admin'], moduleId: 'handovers' },
  { to: '/calendar', label: 'Calendar', icon: CalendarDays, roles: ['employee', 'team_leader', 'manager', 'admin'], moduleId: 'calendar' },
  { to: '/requests', label: 'My Requests', icon: Inbox, roles: ['employee', 'team_leader', 'manager', 'admin'], core: true },
  { to: '/approvals', label: 'Approvals', icon: ShieldCheck, roles: ['team_leader', 'manager', 'admin'], core: true },
  { to: '/shifts', label: 'Schedules', icon: Calendar, roles: ['team_leader', 'manager', 'admin'], core: true },
  { to: '/task-management', label: 'Task Center', icon: ClipboardList, roles: ['team_leader', 'manager', 'admin'], moduleId: 'task_center' },
  { to: '/employees', label: 'Employees', icon: Users, roles: ['admin', 'manager', 'team_leader'], core: true },
  { to: '/departments', label: 'Departments', icon: Building2, roles: ['admin'], core: true },
  { to: '/leave-config', label: 'Leave Config', icon: CalendarDays, roles: ['admin'], core: true },
  { to: '/info-tables', label: 'References', icon: Table, roles: ['employee', 'team_leader', 'manager', 'admin'], moduleId: 'references' },
  { to: '/help', label: 'Info Bank', icon: BookOpen, roles: ['employee', 'team_leader', 'manager', 'admin'], moduleId: 'info_bank' },
  { to: '/fiberx-data', label: 'FiberX Data', icon: Database, roles: ['employee', 'team_leader', 'manager', 'admin'], requiresFiberx: true, moduleId: 'fiberx_data' },
  { to: '/external-tools', label: 'External Tools', icon: LinkIcon, roles: ['employee', 'team_leader', 'manager', 'admin'], core: true },
  { to: '/announcements/manage', label: 'Announcements', icon: Megaphone, roles: ['manager', 'admin'], permission: 'can_post_announcements', core: true },
  { to: '/module-settings', label: 'External Modules', icon: ShieldCheck, roles: ['admin'], core: true },
];

export const Sidebar = ({ onClose }: { onClose?: () => void }) => {
  const { user } = useAuthStore();
  const location = useLocation();

  // Fetch the user's department to check fiberx_enabled
  const { data: userDepartment } = useQuery({
    queryKey: ['my-department', user?.department_id],
    queryFn: async () => {
      const res = await api.get(`/departments/${user?.department_id}`);
      return res.data?.data;
    },
    enabled: !!user?.department_id,
  });

  const visibleItems = navItems.filter((item) => {
    if (!user) return false;
    
    // Check role
    let hasAccess = item.roles.includes(user.role);
    
    // Additional permission checks
    if (!hasAccess && (item as any).permission) {
      if ((item as any).permission === 'can_post_announcements' && (user as any).can_post_announcements) {
        hasAccess = true;
      }
    }

    // FiberX Data fallback logic (if active_modules is used, this may be redundant, but kept for safety)
    if ((item as any).requiresFiberx && hasAccess) {
      const isLeadership = user.role === 'admin' || user.role === 'manager' || user.role === 'team_leader';
      if (!isLeadership && !userDepartment?.fiberx_enabled) {
        hasAccess = false;
      }
    }

    // Module toggling logic based on active_modules in department
    if (hasAccess && (item as any).moduleId && userDepartment) {
       const activeModules = userDepartment.active_modules || [];
       if (!activeModules.includes((item as any).moduleId)) {
         hasAccess = false;
       }
    }
    
    return hasAccess;
  });

  return (
    <aside className="w-64 h-screen bg-sidebar text-sidebar-foreground flex flex-col border-r border-border/30">
      {/* ── Brand ── */}
      <div className="px-5 py-6 flex items-center justify-between border-b border-white/5">
        <div className="flex items-center gap-3">
          <div className="w-10 h-10 rounded-xl bg-primary/15 flex items-center justify-center glow-teal-sm">
            <Clock className="w-5 h-5 text-primary" />
          </div>
          <div>
            <h1 className="text-xl font-bold tracking-tight text-gradient-teal">
              Shiftmaster
            </h1>
            <p className="text-[10px] font-medium text-muted-foreground uppercase tracking-widest">
              Shift Management
            </p>
          </div>
        </div>
        {/* Close button - mobile only */}
        {onClose && (
          <button
            onClick={onClose}
            className="p-1.5 rounded-lg hover:bg-white/10 transition-colors"
          >
            <X className="w-5 h-5 text-muted-foreground" />
          </button>
        )}
      </div>

      {/* ── Nav ── */}
      <nav className="flex-1 px-3 py-4 space-y-1 overflow-y-auto">
        {visibleItems.map((item) => {
          const isActive =
            item.to === '/'
              ? location.pathname === '/'
              : location.pathname.startsWith(item.to);

          return (
            <Link
              key={item.to}
              to={item.to}
              onClick={onClose}
              className={`
                flex items-center gap-3 px-3 py-2.5 rounded-lg text-sm font-medium
                transition-all duration-200 group
                ${isActive
                  ? 'bg-primary/10 text-primary shadow-sm'
                  : 'text-muted-foreground hover:text-foreground hover:bg-white/5'
                }
              `}
            >
              <item.icon className={`w-[18px] h-[18px] flex-shrink-0 transition-colors ${isActive ? 'text-primary' : 'text-muted-foreground group-hover:text-foreground'
                }`} />
              {item.label}
              {isActive && (
                <div className="ml-auto w-1.5 h-1.5 rounded-full bg-primary animate-pulse" />
              )}
            </Link>
          );
        })}
      </nav>


      {/* ── Footer ── */}
      <div className="px-5 py-4 border-t border-white/5">
        <p className="text-[10px] text-muted-foreground/50 text-center">
          © {new Date().getFullYear()} Shiftmaster
        </p>
      </div>
    </aside>
  );
};
