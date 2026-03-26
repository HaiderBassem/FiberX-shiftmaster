import { NavLink } from 'react-router-dom';
import { useAuthStore } from '@/store/authStore';
import { 
  Building2, 
  Users, 
  Calendar, 
  CheckSquare, 
  CalendarOff, 
  ArrowLeftRight,
  Bell,
  ShieldCheck,
  ClipboardList,
  LayoutDashboard
} from 'lucide-react';
import { cn } from '@/lib/utils';

export const Sidebar = () => {
  const { user } = useAuthStore();
  const role = user?.role;

  const mainLinks = [
    { to: '/', icon: <LayoutDashboard className="w-5 h-5" />, label: 'Dashboard', roles: ['employee', 'team_leader', 'manager', 'admin'] },
    { to: '/tasks', icon: <CheckSquare className="w-5 h-5" />, label: 'My Tasks', roles: ['employee', 'team_leader', 'manager', 'admin'] },
    { to: '/leaves', icon: <CalendarOff className="w-5 h-5" />, label: 'My Leaves', roles: ['employee', 'team_leader', 'manager', 'admin'] },
    { to: '/swaps', icon: <ArrowLeftRight className="w-5 h-5" />, label: 'My Swaps', roles: ['employee', 'team_leader', 'manager', 'admin'] },
  ];

  const supervisorLinks = [
    { to: '/approvals', icon: <ShieldCheck className="w-5 h-5" />, label: 'Approvals', roles: ['team_leader', 'manager', 'admin'] },
    { to: '/task-management', icon: <ClipboardList className="w-5 h-5" />, label: 'Task Management', roles: ['team_leader', 'manager', 'admin'] },
    { to: '/task-boards', icon: <CheckSquare className="w-5 h-5" />, label: 'Task Boards', roles: ['team_leader', 'manager', 'admin'] },
    { to: '/shifts', icon: <Calendar className="w-5 h-5" />, label: 'Shifts & Schedules', roles: ['team_leader', 'manager', 'admin'] },
  ];

  const adminLinks = [
    { to: '/departments', icon: <Building2 className="w-5 h-5" />, label: 'Departments', roles: ['admin'] },
    { to: '/employees', icon: <Users className="w-5 h-5" />, label: 'Employees', roles: ['admin', 'manager', 'team_leader'] },
  ];

  const renderSection = (title: string, links: typeof mainLinks) => {
    const visible = links.filter(l => role && l.roles.includes(role));
    if (visible.length === 0) return null;
    return (
      <div className="space-y-1">
        <p className="px-3 text-[10px] font-semibold uppercase tracking-widest text-zinc-500 mb-2">{title}</p>
        {visible.map(link => (
          <NavLink
            key={link.to}
            to={link.to}
            end={link.to === '/'}
            className={({ isActive }) =>
              cn(
                "flex items-center gap-3 px-3 py-2.5 rounded-lg text-sm font-medium transition-all duration-200 group relative overflow-hidden",
                isActive
                  ? "text-blue-400 bg-blue-500/10 shadow-[inset_0_1px_0_rgba(255,255,255,0.1)]"
                  : "text-zinc-400 hover:text-zinc-200 hover:bg-zinc-800/50"
              )
            }
          >
            {({ isActive }) => (
              <>
                {isActive && (
                  <div className="absolute left-0 top-1/2 -translate-y-1/2 w-1 h-6 bg-blue-500 rounded-r-md shadow-[0_0_10px_rgba(59,130,246,0.5)]" />
                )}
                <span className={cn("transition-colors", isActive ? "text-blue-400" : "text-zinc-500 group-hover:text-zinc-300")}>
                  {link.icon}
                </span>
                {link.label}
              </>
            )}
          </NavLink>
        ))}
      </div>
    );
  };

  return (
    <aside className="w-64 bg-zinc-950/80 border-r border-zinc-800/60 backdrop-blur-xl h-screen sticky top-0 flex flex-col transition-all duration-300 shadow-2xl z-20 hidden md:flex">
      <div className="h-16 flex items-center px-6 border-b border-zinc-800/60 bg-zinc-900/50">
        <h1 className="text-xl font-bold bg-gradient-to-r from-blue-400 to-indigo-400 bg-clip-text text-transparent">ShiftMaster</h1>
      </div>
      <nav className="flex-1 overflow-y-auto py-4 px-3 space-y-6">
        {renderSection('Main', mainLinks)}
        {renderSection('Management', supervisorLinks)}
        {renderSection('Administration', adminLinks)}
      </nav>
      <div className="p-4 border-t border-zinc-800/60 bg-zinc-900/30">
        <NavLink
            to="/notifications"
            className={({ isActive }) =>
              cn(
                "flex items-center gap-3 px-3 py-2.5 rounded-lg text-sm font-medium transition-all duration-200",
                isActive ? "text-blue-400 bg-blue-500/10" : "text-zinc-400 hover:text-zinc-200 hover:bg-zinc-800/50"
              )
            }
          >
            <Bell className="w-5 h-5" />
            Notifications
          </NavLink>
      </div>
    </aside>
  );
};
