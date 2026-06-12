import { useState } from 'react';
import { Outlet } from 'react-router-dom';
import { Sidebar } from './Sidebar';
import { Topbar } from './Topbar';
import { AnnouncementTicker } from '@/features/announcements/AnnouncementTicker';
import { NotificationToastContainer } from '@/components/NotificationToast';

export const DashboardLayout = () => {
  const [sidebarOpen, setSidebarOpen] = useState(false);

  return (
    <div className="flex min-h-screen bg-background">
      {/* Overlay — all screen sizes */}
      {sidebarOpen && (
        <div
          className="fixed inset-0 bg-black/40 backdrop-blur-sm z-40"
          onClick={() => setSidebarOpen(false)}
        />
      )}

      {/* Sidebar — always hidden by default, slides in on toggle */}
      <div className={`
        fixed inset-y-0 left-0 z-50
        transform transition-all duration-300 ease-in-out
        ${sidebarOpen ? 'translate-x-0' : '-translate-x-full'}
      `}>
        <Sidebar onClose={() => setSidebarOpen(false)} />
      </div>

      {/* Main content — always full width */}
      <main className="flex-1 flex flex-col min-w-0 w-full">
        <Topbar onMenuClick={() => setSidebarOpen(!sidebarOpen)} sidebarOpen={sidebarOpen} />
        <AnnouncementTicker />
        <div className="flex-1 p-3 sm:p-4 md:p-6 overflow-auto">
          <Outlet />
        </div>
      </main>

      {/* Global notification toasts — rendered above everything */}
      <NotificationToastContainer />
    </div>
  );
};
