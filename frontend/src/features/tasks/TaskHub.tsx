import { ClipboardList } from 'lucide-react';
import { TaskManagement } from './TaskManagement';
import { TaskBoards } from './TaskBoards';
import { TaskHistory } from './TaskHistory';

export const TaskHub = () => {
  return (
    <div className="space-y-6">
      {/* ── Page Header ── */}
      <div className="flex flex-col gap-4 border-b border-border pb-4">
        <div>
          <h2 className="text-2xl sm:text-3xl font-bold tracking-tight text-foreground flex items-center gap-2 sm:gap-3">
            <ClipboardList className="w-6 h-6 sm:w-8 sm:h-8 text-primary" />
            Task Center
          </h2>
          <p className="text-muted-foreground mt-1">
            Manage task boards, recurring schedules, and view completion history.
          </p>
        </div>
      </div>

      {/* ── Unified Dashboard Layout ── */}
      <div className="grid grid-cols-1 xl:grid-cols-3 gap-6 animate-in fade-in slide-in-from-bottom-4 duration-500">
        
        {/* Left Column: Active Task Boards (Takes 2 columns on extra large screens) */}
        <div className="xl:col-span-2 space-y-6">
          <div className="bg-card text-card-foreground border rounded-xl p-6 shadow-sm">
            <TaskBoards />
          </div>
        </div>

        {/* Right Column: Task Schedules and History */}
        <div className="space-y-6 flex flex-col">
          <div className="bg-card text-card-foreground border rounded-xl p-6 shadow-sm flex-1">
            <TaskManagement />
          </div>
          <div className="bg-card text-card-foreground border rounded-xl p-6 shadow-sm flex-1">
            <TaskHistory />
          </div>
        </div>

      </div>
    </div>
  );
};
