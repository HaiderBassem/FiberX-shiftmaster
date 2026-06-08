import { Link, useLocation } from 'react-router-dom';
import { useAuthStore } from '@/store/authStore';
import { useMyLinks } from '@/hooks/useModuleAccess';
import {
  LayoutDashboard, Users, Calendar, CheckSquare,
  ShieldCheck, ClipboardList, Building2,
  Clock, X, MapPin, ExternalLink, Ticket, Table, BookOpen, Inbox, Megaphone, CalendarDays, User, Link as LinkIcon
} from 'lucide-react';

const navItems = [
  { to: '/', label: 'Dashboard', icon: LayoutDashboard, roles: ['employee', 'team_leader', 'manager', 'admin'] },
  { to: '/profile', label: 'My Profile', icon: User, roles: ['employee', 'team_leader', 'manager', 'admin'] },
  { to: '/tasks', label: 'My Tasks', icon: CheckSquare, roles: ['employee', 'team_leader', 'manager', 'admin'] },
  { to: '/handovers', label: 'Handovers', icon: ClipboardList, roles: ['employee', 'team_leader', 'manager', 'admin'] },
  { to: '/calendar', label: 'Calendar', icon: CalendarDays, roles: ['employee', 'team_leader', 'manager', 'admin'] },
  { to: '/requests', label: 'My Requests', icon: Inbox, roles: ['employee', 'team_leader', 'manager', 'admin'] },
  { to: '/approvals', label: 'Approvals', icon: ShieldCheck, roles: ['team_leader', 'manager', 'admin'] },
  { to: '/shifts', label: 'Schedules', icon: Calendar, roles: ['team_leader', 'manager', 'admin'] },
  { to: '/task-management', label: 'Task Center', icon: ClipboardList, roles: ['team_leader', 'manager', 'admin'] },
  { to: '/employees', label: 'Employees', icon: Users, roles: ['admin', 'manager', 'team_leader'] },
  { to: '/departments', label: 'Departments', icon: Building2, roles: ['admin'] },
  { to: '/leave-config', label: 'Leave Config', icon: CalendarDays, roles: ['admin'] },
  { to: '/info-tables', label: 'References', icon: Table, roles: ['employee', 'team_leader', 'manager', 'admin'] },
  { to: '/help', label: 'Info Bank', icon: BookOpen, roles: ['employee', 'team_leader', 'manager', 'admin'] },
  { to: '/announcements/manage', label: 'Announcements', icon: Megaphone, roles: ['manager', 'admin'], permission: 'can_post_announcements' },
  { to: '/module-settings', label: 'External Modules', icon: ShieldCheck, roles: ['team_leader', 'manager', 'admin'] },
];

export const Sidebar = ({ onClose }: { onClose?: () => void }) => {
  const { user } = useAuthStore();
  const location = useLocation();
  const { data: myModules } = useMyLinks();

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

      {/* ── External Tools (dynamic, access-controlled) ── */}
      {myModules && myModules.length > 0 && (
        <div className="px-3 pb-3">
          <div className="p-2 bg-white/5 border border-white/5 rounded-xl space-y-1">
            <h3 className="px-2 text-xs font-semibold text-gray-400 uppercase tracking-wider mb-2">External Tools</h3>
            {myModules.map(link => {
              const isLiveMap = link.title === 'Live Map' || link.url.includes('maps.shift-master.org');
              
              const Icon = (() => {
                switch (link.icon_name) {
                  case 'map-pin': return MapPin;
                  case 'ticket': return Ticket;
                  case 'external-link': return ExternalLink;
                  case 'calendar': return Calendar;
                  case 'users': return Users;
                  case 'book-open': return BookOpen;
                  default: return LinkIcon;
                }
              })();

              const handleClick = () => {
                if (isLiveMap) {
                  // Auto-login logic: admin/manager/team_leader get sys credentials, employees get emp credentials
                  const sysRoles = ['admin', 'manager', 'team_leader'];
                  const isSys = user && sysRoles.includes(user.role);
                  const u = isSys ? 'sys@fiberx.iq' : 'emp@fiberx.iq';
                  const p = isSys ? 'fibersysX' : 'empfiberX';
                  const url = `https://maps.shift-master.org/autologin?u=${encodeURIComponent(u)}&p=${encodeURIComponent(p)}`;
                  window.open(url, '_blank', 'noopener,noreferrer');
                } else {
                  window.open(link.url, '_blank', 'noopener,noreferrer');
                }
              };

              // Live Map gets a premium style
              if (isLiveMap) {
                return (
                  <button
                    key={link.id}
                    onClick={handleClick}
                    className="group relative flex items-center gap-3 w-full overflow-hidden rounded-xl px-3 py-3 transition-all duration-300 hover:scale-[1.02] cursor-pointer"
                    style={{
                      background: 'linear-gradient(135deg, rgba(12,204,204,0.18) 0%, rgba(1,163,163,0.10) 100%)',
                      border: '1px solid rgba(12,204,204,0.30)',
                      boxShadow: '0 0 18px rgba(12,204,204,0.10), inset 0 1px 0 rgba(255,255,255,0.06)',
                    }}
                  >
                    <div className="relative flex-shrink-0">
                      <span
                        className="absolute inset-0 rounded-lg animate-ping"
                        style={{ background: 'rgba(12,204,204,0.20)', animationDuration: '2.5s' }}
                      />
                      <div
                        className="relative w-8 h-8 rounded-lg flex items-center justify-center"
                        style={{
                          background: 'linear-gradient(135deg, #0CCCCC 0%, #01A3A3 100%)',
                          boxShadow: '0 0 14px rgba(12,204,204,0.45)',
                          clipPath: 'polygon(25% 0%, 75% 0%, 100% 25%, 100% 75%, 75% 100%, 25% 100%, 0% 75%, 0% 25%)',
                        }}
                      >
                        <MapPin className="w-3.5 h-3.5 text-black" />
                      </div>
                    </div>
                    <div className="flex-1 min-w-0 text-left">
                      <p className="text-sm font-semibold leading-tight" style={{ color: '#0CCCCC' }}>
                        {link.title}
                      </p>
                      <p className="text-[10px] text-muted-foreground truncate">
                        maps.shift-master.org
                      </p>
                    </div>
                    <ExternalLink
                      className="w-3.5 h-3.5 flex-shrink-0 transition-all duration-300 group-hover:translate-x-0.5 group-hover:-translate-y-0.5"
                      style={{ color: 'rgba(12,204,204,0.70)' }}
                    />
                  </button>
                );
              }

              // Regular links
              return (
                <button
                  key={link.id}
                  onClick={handleClick}
                  className="w-full flex items-center gap-3 px-2 py-2 rounded-lg text-sm text-gray-300 hover:text-white hover:bg-white/10 transition-colors group text-left"
                >
                  <Icon className="w-4 h-4 text-gray-400 group-hover:text-purple-400 transition-colors" />
                  <span className="flex-1 truncate">{link.title}</span>
                  <ExternalLink className="w-3 h-3 text-gray-500 opacity-0 group-hover:opacity-100 transition-opacity" />
                </button>
              );
            })}
          </div>
        </div>
      )}
      {/* ── Footer ── */}
      <div className="px-5 py-4 border-t border-white/5">
        <p className="text-[10px] text-muted-foreground/50 text-center">
          © {new Date().getFullYear()} Shiftmaster
        </p>
      </div>
    </aside>
  );
};
