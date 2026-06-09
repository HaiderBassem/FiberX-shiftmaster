import { Link, useLocation } from 'react-router-dom';
import { useAuthStore } from '@/store/authStore';
import { useMyLinks } from '@/hooks/useModuleAccess';
import { useQuery } from '@tanstack/react-query';
import api from '@/lib/api';
import {
  LayoutDashboard, Users, Calendar, CheckSquare,
  ShieldCheck, ClipboardList, Building2, Database,
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
  { to: '/fiberx-data', label: 'FiberX Data', icon: Database, roles: ['employee', 'team_leader', 'manager', 'admin'], requiresFiberx: true },
  { to: '/announcements/manage', label: 'Announcements', icon: Megaphone, roles: ['manager', 'admin'], permission: 'can_post_announcements' },
  { to: '/module-settings', label: 'External Modules', icon: ShieldCheck, roles: ['admin'] },
];

export const Sidebar = ({ onClose }: { onClose?: () => void }) => {
  const { user } = useAuthStore();
  const location = useLocation();
  const { data: myModules } = useMyLinks();

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

    // FiberX Data: only show if department has fiberx_enabled or user is admin
    if ((item as any).requiresFiberx && hasAccess) {
      if (user.role !== 'admin' && !userDepartment?.fiberx_enabled) {
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

      {/* ── External Tools (dynamic, access-controlled) ── */}
      {myModules && myModules.length > 0 && (
        <div className="px-3 pb-3 space-y-2">
          <h3 className="px-1 text-[10px] font-semibold text-gray-500 uppercase tracking-widest">External Tools</h3>
          {myModules.map(link => {
            const isLiveMap = link.title === 'Live Map' || link.url.includes('maps.shift-master.org');
            
            // Color theme per icon type
            const theme = (() => {
              switch (link.icon_name) {
                case 'map-pin': return { c1: '#0CCCCC', c2: '#01A3A3', bg1: 'rgba(12,204,204,0.15)', bg2: 'rgba(1,163,163,0.08)', border: 'rgba(12,204,204,0.25)', glow: 'rgba(12,204,204,0.35)' };
                case 'ticket': return { c1: '#F59E0B', c2: '#D97706', bg1: 'rgba(245,158,11,0.15)', bg2: 'rgba(217,119,6,0.08)', border: 'rgba(245,158,11,0.25)', glow: 'rgba(245,158,11,0.35)' };
                case 'calendar': return { c1: '#8B5CF6', c2: '#7C3AED', bg1: 'rgba(139,92,246,0.15)', bg2: 'rgba(124,58,237,0.08)', border: 'rgba(139,92,246,0.25)', glow: 'rgba(139,92,246,0.35)' };
                case 'users': return { c1: '#3B82F6', c2: '#2563EB', bg1: 'rgba(59,130,246,0.15)', bg2: 'rgba(37,99,235,0.08)', border: 'rgba(59,130,246,0.25)', glow: 'rgba(59,130,246,0.35)' };
                case 'book-open': return { c1: '#10B981', c2: '#059669', bg1: 'rgba(16,185,129,0.15)', bg2: 'rgba(5,150,105,0.08)', border: 'rgba(16,185,129,0.25)', glow: 'rgba(16,185,129,0.35)' };
                case 'external-link': return { c1: '#EC4899', c2: '#DB2777', bg1: 'rgba(236,72,153,0.15)', bg2: 'rgba(219,39,119,0.08)', border: 'rgba(236,72,153,0.25)', glow: 'rgba(236,72,153,0.35)' };
                default: return { c1: '#A78BFA', c2: '#7C3AED', bg1: 'rgba(167,139,250,0.15)', bg2: 'rgba(124,58,237,0.08)', border: 'rgba(167,139,250,0.25)', glow: 'rgba(167,139,250,0.35)' };
              }
            })();

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

            // Extract domain for subtitle
            const domain = (() => {
              try { return new URL(link.url).hostname; } catch { return link.url; }
            })();

            return (
              <button
                key={link.id}
                onClick={handleClick}
                className="group relative flex items-center gap-3 w-full overflow-hidden rounded-xl px-3 py-2.5 transition-all duration-300 hover:scale-[1.02] cursor-pointer"
                style={{
                  background: `linear-gradient(135deg, ${theme.bg1} 0%, ${theme.bg2} 100%)`,
                  border: `1px solid ${theme.border}`,
                  boxShadow: `0 0 12px ${theme.bg2}, inset 0 1px 0 rgba(255,255,255,0.05)`,
                }}
              >
                <div className="relative flex-shrink-0">
                  <div
                    className="relative w-8 h-8 rounded-lg flex items-center justify-center transition-shadow duration-300 group-hover:shadow-lg"
                    style={{
                      background: `linear-gradient(135deg, ${theme.c1} 0%, ${theme.c2} 100%)`,
                      boxShadow: `0 0 10px ${theme.glow}`,
                    }}
                  >
                    <Icon className="w-3.5 h-3.5 text-white" />
                  </div>
                </div>
                <div className="flex-1 min-w-0 text-left">
                  <p className="text-sm font-semibold leading-tight transition-colors" style={{ color: theme.c1 }}>
                    {link.title}
                  </p>
                  <p className="text-[10px] text-muted-foreground truncate">
                    {domain}
                  </p>
                </div>
                <ExternalLink
                  className="w-3.5 h-3.5 flex-shrink-0 transition-all duration-300 opacity-50 group-hover:opacity-100 group-hover:translate-x-0.5 group-hover:-translate-y-0.5"
                  style={{ color: theme.c1 }}
                />
              </button>
            );
          })}
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
