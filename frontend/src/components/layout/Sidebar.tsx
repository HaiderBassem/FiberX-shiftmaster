import { Link, useLocation, useNavigate } from 'react-router-dom';
import { useAuthStore } from '@/store/authStore';
import { useQuery } from '@tanstack/react-query';
import { useTranslation } from 'react-i18next';
import api from '@/lib/api';
import {
  LayoutDashboard, Users, Calendar, CheckSquare,
  ShieldCheck, ClipboardList, Building2, Database,
  Clock, X, Table, BookOpen, Inbox, Megaphone, CalendarDays, Link as LinkIcon, LogOut, Ticket
} from 'lucide-react';

const navItems = [
  { to: '/', labelKey: 'dashboard', icon: LayoutDashboard, roles: ['employee', 'team_leader', 'manager', 'admin'], core: true },
  { to: '/tasks', labelKey: 'my_tasks', icon: CheckSquare, roles: ['employee', 'team_leader', 'manager', 'admin'], moduleId: 'tasks' },
  { to: '/handovers', labelKey: 'handovers', icon: ClipboardList, roles: ['employee', 'team_leader', 'manager', 'admin'], moduleId: 'handovers' },
  { to: '/calendar', labelKey: 'calendar', icon: CalendarDays, roles: ['employee', 'team_leader', 'manager', 'admin'], moduleId: 'calendar' },
  { to: '/requests', labelKey: 'my_requests', icon: Inbox, roles: ['employee', 'team_leader', 'manager', 'admin'], core: true },
  { to: '/approvals', labelKey: 'approvals', icon: ShieldCheck, roles: ['team_leader', 'manager', 'admin'], core: true },
  { to: '/shifts', labelKey: 'shifts', icon: Calendar, roles: ['team_leader', 'manager', 'admin'], core: true },
  { to: '/task-management', labelKey: 'task_center', icon: ClipboardList, roles: ['team_leader', 'manager', 'admin'], moduleId: 'task_center' },
  { to: '/employees', labelKey: 'manage_employees', icon: Users, roles: ['admin', 'manager', 'team_leader'], core: true },
  { to: '/departments', labelKey: 'departments', icon: Building2, roles: ['admin'], core: true },
  { to: '/leave-config', labelKey: 'leave_config', icon: CalendarDays, roles: ['admin'], core: true },
  { to: '/info-tables', labelKey: 'references', icon: Table, roles: ['employee', 'team_leader', 'manager', 'admin'], moduleId: 'references' },
  { to: '/help', labelKey: 'info_bank', icon: BookOpen, roles: ['employee', 'team_leader', 'manager', 'admin'], moduleId: 'info_bank' },
  { to: '/fiberx-data', labelKey: 'fiber_data', icon: Database, roles: ['employee', 'team_leader', 'manager', 'admin'], requiresFiberx: true, moduleId: 'fiberx_data' },
  { to: '/external-tools', labelKey: 'external_tools', icon: LinkIcon, roles: ['employee', 'team_leader', 'manager', 'admin'], core: true },
  { to: '/tickets', labelKey: 'tickets', icon: Ticket, roles: ['employee', 'team_leader', 'manager', 'admin'], moduleId: 'tickets' },
  { to: '/announcements/manage', labelKey: 'announcements', icon: Megaphone, roles: ['manager', 'admin'], permission: 'can_post_announcements', core: true },
  { to: '/module-settings', labelKey: 'external_modules', icon: ShieldCheck, roles: ['admin', 'manager', 'team_leader'], core: true },
];

export const Sidebar = ({ onClose }: { onClose?: () => void }) => {
  const { user, logout } = useAuthStore();
  const location = useLocation();
  const navigate = useNavigate();
  const { t } = useTranslation();

  const handleLogout = () => {
    logout();
    navigate('/login');
  };

  // Fetch the user's department to check fiberx_enabled and active_modules
  const { data: userDepartment, isLoading: deptLoading } = useQuery({
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

    // Additional permission checks (e.g. can_post_announcements)
    if (!hasAccess && (item as any).permission) {
      if ((item as any).permission === 'can_post_announcements' && (user as any).can_post_announcements) {
        hasAccess = true;
      }
    }

    if (!hasAccess) return false;

    // While department is still loading, show all items the user has role access to
    // so the sidebar doesn't flicker/disappear on page load
    if (deptLoading || !userDepartment) return true;

    // FiberX: hide for non-leadership if not enabled for department
    if ((item as any).requiresFiberx) {
      const isLeadership = user.role === 'admin' || user.role === 'manager' || user.role === 'team_leader';
      if (!isLeadership && !userDepartment?.fiberx_enabled) {
        return false;
      }
    }

    // Module toggling: only hide if active_modules explicitly excludes this module
    if ((item as any).moduleId) {
      const activeModules: string[] = userDepartment?.active_modules || [];
      // If active_modules is an empty array, that means ALL modules are disabled (intentional)
      // If it's missing/null (shouldn't happen after migration fix), show everything
      if (activeModules.length > 0 && !activeModules.includes((item as any).moduleId)) {
        return false;
      }
    }

    return true;
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
              {t(`navigation.${item.labelKey}`)}
              {isActive && (
                <div className="ml-auto w-1.5 h-1.5 rounded-full bg-primary animate-pulse" />
              )}
            </Link>
          );
        })}
      </nav>

      {/* Logout Button */}
      <div className="px-3 pb-3">
        <button
          onClick={handleLogout}
          className="w-full flex items-center gap-3 px-3 py-2.5 rounded-lg text-sm font-medium text-destructive/80 hover:text-destructive hover:bg-destructive/10 transition-all duration-200 group"
        >
          <LogOut className="w-[18px] h-[18px] flex-shrink-0 transition-colors group-hover:text-destructive" />
          {t('navigation.logout')}
        </button>
      </div>

      {/* Footer */}
      <div className="px-5 py-4 border-t border-white/5">
        <p className="text-[10px] text-muted-foreground/50 text-center">
          © {new Date().getFullYear()} Shiftmaster
        </p>
      </div>
    </aside>
  );
};
