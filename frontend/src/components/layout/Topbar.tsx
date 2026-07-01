import { useEffect, useState } from 'react';
import { useAuthStore } from '@/store/authStore';
import { useTheme } from '@/components/ThemeProvider';
import { Button } from '@/components/ui/button';
import { LogOut, Sun, Moon, User, Menu, Key, Bell } from 'lucide-react';
import { useQuery } from '@tanstack/react-query';
import api from '@/lib/api';
import { useNavigate } from 'react-router-dom';
import { ChangePasswordModal } from '@/features/auth/ChangePasswordModal';
import { useNotification } from '@/providers/NotificationProvider';

export const Topbar = ({ onMenuClick, sidebarOpen }: { onMenuClick?: () => void; sidebarOpen?: boolean }) => {
  const { requestPermission, permission } = useNotification();
  const {
    user,
    logout,
    adminSelectedDepartmentId,
    setAdminSelectedDepartmentId,
    managerSelectedDepartmentId,
    setManagerSelectedDepartmentId,
  } = useAuthStore();
  const { theme, toggleTheme } = useTheme();
  const navigate = useNavigate();
  const [showChangePassword, setShowChangePassword] = useState(false);

  // ── Admin: all departments ──────────────────────────────────
  const { data: allDepartments } = useQuery({
    queryKey: ['departments'],
    queryFn: async () => {
      const res = await api.get('/departments');
      return res.data?.data || [];
    },
    enabled: user?.role === 'admin',
  });

  // ── Manager: only their managed departments ─────────────────
  const { data: managedDepartments } = useQuery({
    queryKey: ['my-managed-departments'],
    queryFn: async () => {
      const res = await api.get('/departments/my-managed');
      return res.data?.data || [];
    },
    enabled: user?.role === 'manager',
  });

  // ── Notifications ───────────────────────────────────────────
  const { data: notifications } = useQuery({
    queryKey: ['notifications'],
    queryFn: async () => {
      const res = await api.get('/notifications');
      return res.data?.data || [];
    },
    enabled: !!user,
  });
  
  const unreadCount = notifications?.filter((n: any) => !n.is_read).length || 0;

  // Auto-select first managed department for managers on first load
  useEffect(() => {
    if (
      user?.role === 'manager' &&
      managedDepartments &&
      managedDepartments.length > 0 &&
      !managerSelectedDepartmentId
    ) {
      setManagerSelectedDepartmentId(managedDepartments[0].id);
    }
  }, [managedDepartments, managerSelectedDepartmentId, user?.role, setManagerSelectedDepartmentId]);

  const handleLogout = () => {
    logout();
    navigate('/login');
  };

  return (
    <header className="h-14 sm:h-16 border-b border-border bg-card/80 backdrop-blur-sm px-3 sm:px-6 flex items-center justify-between sticky top-0 z-30">
      {/* Left: Menu + context */}
      <div className="flex items-center gap-2 sm:gap-3">
        {/* Hamburger menu — all screen sizes */}
        <Button
          variant="ghost"
          size="icon"
          onClick={onMenuClick}
          className="text-muted-foreground hover:text-foreground"
          title={sidebarOpen ? 'Close menu' : 'Open menu'}
        >
          <Menu className="w-5 h-5" />
        </Button>
        <div className="w-2 h-2 rounded-full bg-primary animate-pulse hidden sm:block" />
        <span className="text-xs font-medium text-muted-foreground uppercase tracking-wider">
          {user?.role?.replace('_', ' ')}
        </span>

        {/* ── Admin department selector ── */}
        {user?.role === 'admin' && allDepartments && allDepartments.length > 0 && (
          <div className="ml-4 hidden md:flex items-center">
            <select
              className="bg-transparent border border-border text-sm rounded-md px-2 py-1 text-foreground focus:outline-none focus:ring-1 focus:ring-primary"
              value={adminSelectedDepartmentId || ''}
              onChange={(e) => {
                setAdminSelectedDepartmentId(e.target.value || null);
                window.location.reload();
              }}
            >
              <option value="">All Departments</option>
              {allDepartments.map((d: any) => (
                <option key={d.id} value={d.id}>
                  {d.name}
                </option>
              ))}
            </select>
          </div>
        )}

        {/* ── Manager department selector ── */}
        {user?.role === 'manager' && managedDepartments && managedDepartments.length > 0 && (
          <div className="ml-4 hidden md:flex items-center">
            <select
              className="bg-transparent border border-border text-sm rounded-md px-2 py-1 text-foreground focus:outline-none focus:ring-1 focus:ring-primary"
              value={managerSelectedDepartmentId || ''}
              onChange={(e) => {
                setManagerSelectedDepartmentId(e.target.value || null);
                window.location.reload();
              }}
            >
              {managedDepartments.map((d: any) => (
                <option key={d.id} value={d.id}>
                  {d.name}
                </option>
              ))}
            </select>
          </div>
        )}
      </div>

      {/* Right: Actions */}
      <div className="flex items-center gap-1 sm:gap-2">
        {/* Theme toggle */}
        <Button
          variant="ghost"
          size="icon"
          onClick={toggleTheme}
          className="text-muted-foreground hover:text-foreground hover:bg-muted rounded-lg w-8 h-8 sm:w-9 sm:h-9"
          title={theme === 'dark' ? 'Switch to Light Mode' : 'Switch to Dark Mode'}
        >
          {theme === 'dark' ? (
            <Sun className="w-4 h-4 sm:w-[18px] sm:h-[18px]" />
          ) : (
            <Moon className="w-4 h-4 sm:w-[18px] sm:h-[18px]" />
          )}
        </Button>

        {/* User info */}
        <div 
          onClick={() => navigate('/profile')}
          className="flex items-center gap-1.5 sm:gap-2 px-2 sm:px-3 py-1.5 rounded-lg bg-muted/50 cursor-pointer hover:bg-muted/80 transition-colors"
          title="My Profile"
        >
          <div className="w-6 h-6 sm:w-7 sm:h-7 rounded-full bg-primary/15 flex items-center justify-center overflow-hidden">
            {user?.profile_image ? (
              <img 
                src={`${import.meta.env.VITE_API_URL ? import.meta.env.VITE_API_URL.replace('/api', '') : (import.meta.env.DEV ? 'http://localhost:8080' : '')}${user.profile_image.startsWith('/api') ? user.profile_image : '/api' + user.profile_image}`} 
                alt="Profile" 
                className="w-full h-full object-cover" 
              />
            ) : (
              <User className="w-3 h-3 sm:w-4 sm:h-4 text-primary" />
            )}
          </div>
          <span className="text-xs sm:text-sm font-medium text-foreground hidden xs:inline">
            {user?.first_name}
          </span>
        </div>

        {/* Notifications */}
        {permission === 'default' && (
          <Button
            variant="ghost"
            size="sm"
            onClick={requestPermission}
            className="flex text-primary bg-primary/10 hover:bg-primary/20 rounded-lg h-8 sm:h-9 text-xs sm:text-sm animate-pulse px-2 sm:px-3"
            title="Enable Push Notifications"
          >
            Enable Notifications
          </Button>
        )}
        <Button
          variant="ghost"
          size="icon"
          onClick={() => navigate('/notifications')}
          className="relative text-muted-foreground hover:text-primary hover:bg-primary/10 rounded-lg w-8 h-8 sm:w-9 sm:h-9"
          title="Notifications"
        >
          <Bell className="w-4 h-4 sm:w-[18px] sm:h-[18px]" />
          {unreadCount > 0 && (
            <span className="absolute -top-1 -right-1 flex items-center justify-center min-w-[16px] h-4 sm:min-w-[18px] sm:h-[18px] px-1 text-[9px] sm:text-[10px] font-bold text-white bg-destructive rounded-full border border-card shadow-sm">
              {unreadCount > 99 ? '99+' : unreadCount}
            </span>
          )}
        </Button>

        {/* Change Password */}
        <Button
          variant="ghost"
          size="icon"
          onClick={() => setShowChangePassword(true)}
          className="text-muted-foreground hover:text-primary hover:bg-primary/10 rounded-lg w-8 h-8 sm:w-9 sm:h-9"
          title="Change Password"
        >
          <Key className="w-4 h-4 sm:w-[18px] sm:h-[18px]" />
        </Button>

        {/* Logout */}
        <Button
          variant="ghost"
          size="icon"
          onClick={handleLogout}
          className="text-muted-foreground hover:text-destructive hover:bg-destructive/10 rounded-lg w-8 h-8 sm:w-9 sm:h-9"
          title="Logout"
        >
          <LogOut className="w-4 h-4 sm:w-[18px] sm:h-[18px]" />
        </Button>
      </div>

      {user && (
        <ChangePasswordModal
          isOpen={showChangePassword}
          onClose={() => setShowChangePassword(false)}
          employeeId={user.id}
          requireOldPassword={true}
        />
      )}
    </header>
  );
};
