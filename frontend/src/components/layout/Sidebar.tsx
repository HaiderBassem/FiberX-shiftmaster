import { Link, useLocation } from 'react-router-dom';
import { useAuthStore } from '@/store/authStore';
import {
  LayoutDashboard, Users, Calendar, CheckSquare, CalendarOff,
  ArrowLeftRight, ShieldCheck, Bell, ClipboardList, Building2, Columns3,
  Clock, History, X, MapPin, ExternalLink, Ticket,
} from 'lucide-react';

const navItems = [
  { to: '/', label: 'Dashboard', icon: LayoutDashboard, roles: ['employee', 'team_leader', 'manager', 'admin'] },
  { to: '/tasks', label: 'My Tasks', icon: CheckSquare, roles: ['employee', 'team_leader', 'manager', 'admin'] },
  { to: '/leaves', label: 'Leaves', icon: CalendarOff, roles: ['employee', 'team_leader', 'manager', 'admin'] },
  { to: '/swaps', label: 'Swaps', icon: ArrowLeftRight, roles: ['employee', 'team_leader', 'manager', 'admin'] },
  { to: '/notifications', label: 'Notifications', icon: Bell, roles: ['employee', 'team_leader', 'manager', 'admin'] },
  { to: '/approvals', label: 'Approvals', icon: ShieldCheck, roles: ['team_leader', 'manager', 'admin'] },
  { to: '/shifts', label: 'Schedules', icon: Calendar, roles: ['team_leader', 'manager', 'admin'] },
  { to: '/task-management', label: 'Tasks Management', icon: ClipboardList, roles: ['team_leader', 'manager', 'admin'] },
  { to: '/task-boards', label: 'Task Boards', icon: Columns3, roles: ['team_leader', 'manager', 'admin'] },
  { to: '/task-history', label: 'Task History', icon: History, roles: ['team_leader', 'manager', 'admin'] },
  { to: '/employees', label: 'Employees', icon: Users, roles: ['admin', 'manager', 'team_leader'] },
  { to: '/departments', label: 'Departments', icon: Building2, roles: ['admin'] },
];

export const Sidebar = ({ onClose }: { onClose?: () => void }) => {
  const { user } = useAuthStore();
  const location = useLocation();

  const visibleItems = navItems.filter(
    (item) => user && item.roles.includes(user.role)
  );

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

      {/* ── Live Map Tab ── */}
      <div className="px-3 pb-3">
        <button
          id="live-map-tab"
          onClick={() => {
            // تحديد الكريدنشلز حسب الدور
            const sysRoles = ['admin', 'manager', 'team_leader'];
            const isSys = user && sysRoles.includes(user.role);
            const u = isSys ? 'sys@fiberx.iq' : 'emp@fiberx.iq';
            const p = isSys ? 'fibersysX' : 'empfiberX';
            const url = `https://maps.shift-master.org/autologin?u=${encodeURIComponent(u)}&p=${encodeURIComponent(p)}`;
            window.open(url, '_blank', 'noopener,noreferrer');
          }}
          className="group relative flex items-center gap-3 w-full overflow-hidden rounded-xl px-3 py-3 transition-all duration-300 hover:scale-[1.02] cursor-pointer"
          style={{
            background: 'linear-gradient(135deg, rgba(12,204,204,0.18) 0%, rgba(1,163,163,0.10) 100%)',
            border: '1px solid rgba(12,204,204,0.30)',
            boxShadow: '0 0 18px rgba(12,204,204,0.10), inset 0 1px 0 rgba(255,255,255,0.06)',
          }}
        >
          {/* Geometric hex background */}
          <svg
            className="absolute right-0 top-0 opacity-10 transition-opacity duration-300 group-hover:opacity-20"
            width="80" height="56" viewBox="0 0 80 56" fill="none"
            aria-hidden="true"
          >
            <polygon points="56,4 76,16 76,40 56,52 36,40 36,16" stroke="#0CCCCC" strokeWidth="1.2" fill="rgba(12,204,204,0.08)" />
            <polygon points="72,0 80,4 80,12 72,16 64,12 64,4" stroke="#0CCCCC" strokeWidth="0.8" fill="none" />
            <polygon points="40,10 52,17 52,31 40,38 28,31 28,17" stroke="#01A3A3" strokeWidth="0.6" fill="none" />
          </svg>

          {/* Icon with animated ring */}
          <div className="relative flex-shrink-0">
            {/* Pulse ring */}
            <span
              className="absolute inset-0 rounded-lg animate-ping"
              style={{ background: 'rgba(12,204,204,0.20)', animationDuration: '2.5s' }}
            />
            <div
              className="relative w-9 h-9 rounded-lg flex items-center justify-center"
              style={{
                background: 'linear-gradient(135deg, #0CCCCC 0%, #01A3A3 100%)',
                boxShadow: '0 0 14px rgba(12,204,204,0.45)',
                clipPath: 'polygon(25% 0%, 75% 0%, 100% 25%, 100% 75%, 75% 100%, 25% 100%, 0% 75%, 0% 25%)',
              }}
            >
              <MapPin className="w-4 h-4 text-black" />
            </div>
          </div>

          {/* Text */}
          <div className="flex-1 min-w-0 text-left">
            <p className="text-sm font-semibold leading-tight" style={{ color: '#0CCCCC' }}>
              Live Map
            </p>
            <p className="text-[10px] text-muted-foreground truncate">
              maps.shift-master.org
            </p>
          </div>

          {/* External link icon */}
          <ExternalLink
            className="w-3.5 h-3.5 flex-shrink-0 transition-all duration-300 group-hover:translate-x-0.5 group-hover:-translate-y-0.5"
            style={{ color: 'rgba(12,204,204,0.70)' }}
          />
        </button>
      </div>

      {/* ── Ticket System Tab ── */}
      <div className="px-3 pb-3">
        <button
          id="ticket-system-tab"
          onClick={() => window.open('https://ticket.shift-master.org/', '_blank', 'noopener,noreferrer')}
          className="group relative flex items-center gap-3 w-full overflow-hidden rounded-xl px-3 py-3 transition-all duration-300 hover:scale-[1.02] cursor-pointer"
          style={{
            background: 'linear-gradient(135deg, rgba(139,92,246,0.18) 0%, rgba(109,40,217,0.10) 100%)',
            border: '1px solid rgba(139,92,246,0.30)',
            boxShadow: '0 0 18px rgba(139,92,246,0.10), inset 0 1px 0 rgba(255,255,255,0.06)',
          }}
        >
          {/* Geometric background */}
          <svg
            className="absolute right-0 top-0 opacity-10 transition-opacity duration-300 group-hover:opacity-20"
            width="80" height="56" viewBox="0 0 80 56" fill="none"
            aria-hidden="true"
          >
            <polygon points="56,4 76,16 76,40 56,52 36,40 36,16" stroke="#8B5CF6" strokeWidth="1.2" fill="rgba(139,92,246,0.08)" />
            <polygon points="72,0 80,4 80,12 72,16 64,12 64,4" stroke="#8B5CF6" strokeWidth="0.8" fill="none" />
            <polygon points="40,10 52,17 52,31 40,38 28,31 28,17" stroke="#6D28D9" strokeWidth="0.6" fill="none" />
          </svg>

          {/* Icon */}
          <div className="relative flex-shrink-0">
            <span
              className="absolute inset-0 rounded-lg animate-ping"
              style={{ background: 'rgba(139,92,246,0.20)', animationDuration: '2.8s' }}
            />
            <div
              className="relative w-9 h-9 rounded-lg flex items-center justify-center"
              style={{
                background: 'linear-gradient(135deg, #8B5CF6 0%, #6D28D9 100%)',
                boxShadow: '0 0 14px rgba(139,92,246,0.45)',
                clipPath: 'polygon(25% 0%, 75% 0%, 100% 25%, 100% 75%, 75% 100%, 25% 100%, 0% 75%, 0% 25%)',
              }}
            >
              <Ticket className="w-4 h-4 text-white" />
            </div>
          </div>

          {/* Text */}
          <div className="flex-1 min-w-0 text-left">
            <p className="text-sm font-semibold leading-tight" style={{ color: '#A78BFA' }}>
              Ticket System
            </p>
            <p className="text-[10px] text-muted-foreground truncate">
              ticket.shift-master.org
            </p>
          </div>

          {/* External link icon */}
          <ExternalLink
            className="w-3.5 h-3.5 flex-shrink-0 transition-all duration-300 group-hover:translate-x-0.5 group-hover:-translate-y-0.5"
            style={{ color: 'rgba(139,92,246,0.70)' }}
          />
        </button>
      </div>

      {/* ── Footer ── */}
      <div className="px-5 py-4 border-t border-white/5">
        <p className="text-[10px] text-muted-foreground/50 text-center">
          © {new Date().getFullYear()} Shiftmaster
        </p>
      </div>
    </aside>
  );
};
