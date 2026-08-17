import { useState } from 'react';
import { Outlet } from 'react-router-dom';
import { Sidebar } from './Sidebar';
import { Topbar } from './Topbar';
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

      {/*
        Sidebar — hidden by default, slides in on toggle.

        Positioned with logical properties (start-0, and an RTL-aware translate)
        rather than left-0 and -translate-x-full. With the physical values the
        panel anchored to the left and slid out to the left in Arabic too, so in
        RTL it opened from the wrong edge and covered the content it was meant to
        sit beside.
      */}
      <div className={`
        fixed inset-y-0 start-0 z-50
        transform transition-all duration-300 ease-in-out
        ${sidebarOpen ? 'translate-x-0' : '-translate-x-full rtl:translate-x-full'}
      `}>
        <Sidebar onClose={() => setSidebarOpen(false)} />
      </div>

      {/* Main content — always full width */}
      <main className="flex-1 flex flex-col min-w-0 w-full">
        <Topbar onMenuClick={() => setSidebarOpen(!sidebarOpen)} sidebarOpen={sidebarOpen} />
        <div className="flex-1 p-3 sm:p-4 md:p-6 overflow-auto">
          <Outlet />
        </div>
      </main>

      {/* Global notification toasts — rendered above everything */}
      <NotificationToastContainer />
    </div>
  );
};
