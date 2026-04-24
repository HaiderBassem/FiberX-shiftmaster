import { useAuthStore } from '@/store/authStore';
import { useTheme } from '@/components/ThemeProvider';
import { Button } from '@/components/ui/button';
import { LogOut, Sun, Moon, User, Menu } from 'lucide-react';
import { useNavigate } from 'react-router-dom';

export const Topbar = ({ onMenuClick, sidebarOpen }: { onMenuClick?: () => void; sidebarOpen?: boolean }) => {
  const { user, logout } = useAuthStore();
  const { theme, toggleTheme } = useTheme();
  const navigate = useNavigate();

  const handleLogout = () => {
    logout();
    navigate('/login');
  };

  return (
    <header className="h-14 sm:h-16 border-b border-border bg-card/80 backdrop-blur-sm px-3 sm:px-6 flex items-center justify-between sticky top-0 z-30">
      {/* Left: Menu + context */}
      <div className="flex items-center gap-2 sm:gap-3">
        {/* Hamburger menu - mobile only */}
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
        <div className="flex items-center gap-1.5 sm:gap-2 px-2 sm:px-3 py-1.5 rounded-lg bg-muted/50">
          <div className="w-6 h-6 sm:w-7 sm:h-7 rounded-full bg-primary/15 flex items-center justify-center">
            <User className="w-3 h-3 sm:w-4 sm:h-4 text-primary" />
          </div>
          <span className="text-xs sm:text-sm font-medium text-foreground hidden xs:inline">
            {user?.first_name}
          </span>
          <span className="text-xs sm:text-sm font-medium text-foreground hidden sm:inline">
            {user?.last_name}
          </span>
        </div>

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
    </header>
  );
};
