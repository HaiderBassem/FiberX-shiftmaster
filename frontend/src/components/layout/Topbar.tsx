import { useAuthStore } from '@/store/authStore';
import { useTheme } from '@/components/ThemeProvider';
import { Button } from '@/components/ui/button';
import { LogOut, Sun, Moon, User } from 'lucide-react';
import { useNavigate } from 'react-router-dom';

export const Topbar = () => {
  const { user, logout } = useAuthStore();
  const { theme, toggleTheme } = useTheme();
  const navigate = useNavigate();

  const handleLogout = () => {
    logout();
    navigate('/login');
  };

  return (
    <header className="h-16 border-b border-border bg-card/80 backdrop-blur-sm px-6 flex items-center justify-between sticky top-0 z-30">
      {/* Left: Page context */}
      <div className="flex items-center gap-3">
        <div className="w-2 h-2 rounded-full bg-primary animate-pulse" />
        <span className="text-xs font-medium text-muted-foreground uppercase tracking-wider">
          {user?.role?.replace('_', ' ')}
        </span>
      </div>

      {/* Right: Actions */}
      <div className="flex items-center gap-2">
        {/* Theme toggle */}
        <Button
          variant="ghost"
          size="icon"
          onClick={toggleTheme}
          className="text-muted-foreground hover:text-foreground hover:bg-muted rounded-lg"
          title={theme === 'dark' ? 'Switch to Light Mode' : 'Switch to Dark Mode'}
        >
          {theme === 'dark' ? (
            <Sun className="w-[18px] h-[18px]" />
          ) : (
            <Moon className="w-[18px] h-[18px]" />
          )}
        </Button>

        {/* User info */}
        <div className="flex items-center gap-2 px-3 py-1.5 rounded-lg bg-muted/50">
          <div className="w-7 h-7 rounded-full bg-primary/15 flex items-center justify-center">
            <User className="w-4 h-4 text-primary" />
          </div>
          <span className="text-sm font-medium text-foreground">
            {user?.first_name} {user?.last_name}
          </span>
        </div>

        {/* Logout */}
        <Button
          variant="ghost"
          size="icon"
          onClick={handleLogout}
          className="text-muted-foreground hover:text-destructive hover:bg-destructive/10 rounded-lg"
          title="Logout"
        >
          <LogOut className="w-[18px] h-[18px]" />
        </Button>
      </div>
    </header>
  );
};
