import { useState } from 'react';
import { Columns3, History, ClipboardList } from 'lucide-react';
import { TaskManagement } from './TaskManagement';
import { TaskBoards } from './TaskBoards';
import { TaskHistory } from './TaskHistory';

type Tab = 'boards' | 'schedules' | 'history';

export const TaskHub = () => {
  const [activeTab, setActiveTab] = useState<Tab>('boards');

  return (
    <div className="space-y-6">
      {/* ── Page Header & Tabs ── */}
      <div className="flex flex-col md:flex-row md:items-end justify-between gap-4 border-b border-border pb-4">
        <div>
          <h2 className="text-2xl sm:text-3xl font-bold tracking-tight text-foreground flex items-center gap-2 sm:gap-3">
            <ClipboardList className="w-6 h-6 sm:w-8 sm:h-8 text-primary" />
            Task Center
          </h2>
          <p className="text-muted-foreground mt-1">
            Manage task boards, recurring schedules, and view completion history.
          </p>
        </div>

        <div className="flex items-center gap-1 bg-muted/40 p-1 rounded-xl border border-border">
          <button
            onClick={() => setActiveTab('boards')}
            className={`flex items-center gap-2 px-4 py-2 rounded-lg text-sm font-medium transition-all ${
              activeTab === 'boards'
                ? 'bg-background text-foreground shadow-sm'
                : 'text-muted-foreground hover:text-foreground hover:bg-muted/50'
            }`}
          >
            <Columns3 className="w-4 h-4" />
            Boards
          </button>
          <button
            onClick={() => setActiveTab('schedules')}
            className={`flex items-center gap-2 px-4 py-2 rounded-lg text-sm font-medium transition-all ${
              activeTab === 'schedules'
                ? 'bg-background text-foreground shadow-sm'
                : 'text-muted-foreground hover:text-foreground hover:bg-muted/50'
            }`}
          >
            <ClipboardList className="w-4 h-4" />
            Schedules
          </button>
          <button
            onClick={() => setActiveTab('history')}
            className={`flex items-center gap-2 px-4 py-2 rounded-lg text-sm font-medium transition-all ${
              activeTab === 'history'
                ? 'bg-background text-foreground shadow-sm'
                : 'text-muted-foreground hover:text-foreground hover:bg-muted/50'
            }`}
          >
            <History className="w-4 h-4" />
            History
          </button>
        </div>
      </div>

      {/* ── Tab Content ── */}
      <div className="animate-in fade-in slide-in-from-bottom-4 duration-500">
        {activeTab === 'boards' && <TaskBoards />}
        {activeTab === 'schedules' && <TaskManagement />}
        {activeTab === 'history' && <TaskHistory />}
      </div>
    </div>
  );
};
