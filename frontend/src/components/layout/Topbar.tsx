import { useAuthStore } from '@/store/authStore';
import { Button } from '@/components/ui/button';
import { LogOut, Menu, Bell } from 'lucide-react';
import { Link } from 'react-router-dom';

export const Topbar = () => {
  const { user, logout } = useAuthStore();

  return (
    <header className="h-16 border-b border-zinc-800/60 bg-zinc-950/80 backdrop-blur-md flex items-center justify-between px-4 sm:px-6 z-10 sticky top-0 shadow-sm">
      <div className="flex items-center gap-3">
        <button className="md:hidden p-2 text-zinc-400 hover:text-zinc-100 transition-colors">
          <Menu className="w-6 h-6" />
        </button>
        <div className="hidden sm:flex flex-col">
          <span className="text-sm font-medium text-zinc-100 capitalize">
            {user?.role.replace('_', ' ')} Dashboard
          </span>
          <span className="text-xs text-zinc-400">Welcome back, {user?.first_name}</span>
        </div>
      </div>
      
      <div className="flex items-center gap-4">
        <Link to="/notifications">
          <Button variant="ghost" size="icon" className="text-zinc-400 hover:text-blue-400 hover:bg-blue-500/10 relative">
            <Bell className="w-5 h-5" />
            <span className="absolute top-2 right-2.5 h-2 w-2 rounded-full bg-blue-500 shadow-[0_0_8px_rgba(59,130,246,0.8)]"></span>
          </Button>
        </Link>

        <div className="h-8 w-px bg-zinc-800" />

        <div className="flex items-center gap-2 px-3 py-1.5 rounded-full bg-zinc-900/80 border border-zinc-800 text-sm">
          <div className="w-6 h-6 rounded-full bg-blue-600/20 text-blue-400 flex items-center justify-center font-bold">
            {user?.first_name?.[0]}
          </div>
          <span className="font-medium text-zinc-200 hidden sm:block">{user?.first_name} {user?.last_name}</span>
        </div>
        
        <Button variant="ghost" size="icon" onClick={() => logout()} className="text-zinc-400 hover:text-red-400 hover:bg-red-500/10">
          <LogOut className="w-5 h-5" />
        </Button>
      </div>
    </header>
  );
};
