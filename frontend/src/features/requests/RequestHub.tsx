import { useState } from 'react';
import { CalendarOff, ArrowLeftRight, Inbox } from 'lucide-react';
import { LeaveList } from '../leaves/LeaveList';
import { SwapList } from '../swaps/SwapList';

type Tab = 'leaves' | 'swaps';

export const RequestHub = () => {
  const [activeTab, setActiveTab] = useState<Tab>('leaves');

  return (
    <div className="space-y-6">
      {/* ── Page Header & Tabs ── */}
      <div className="flex flex-col md:flex-row md:items-end justify-between gap-4 border-b border-border pb-4">
        <div>
          <h2 className="text-2xl sm:text-3xl font-bold tracking-tight text-foreground flex items-center gap-2 sm:gap-3">
            <Inbox className="w-6 h-6 sm:w-8 sm:h-8 text-primary" />
            Requests Center
          </h2>
          <p className="text-muted-foreground mt-1 text-sm sm:text-base">
            Manage your time off requests and shift swaps with teammates.
          </p>
        </div>

        <div className="flex items-center gap-1 bg-muted/40 p-1 rounded-xl border border-border w-full md:w-auto overflow-x-auto snap-x">
          <button
            onClick={() => setActiveTab('leaves')}
            className={`flex-1 md:flex-none flex items-center justify-center gap-2 px-4 py-2 rounded-lg text-sm font-medium transition-all snap-center whitespace-nowrap ${
              activeTab === 'leaves'
                ? 'bg-background text-foreground shadow-sm'
                : 'text-muted-foreground hover:text-foreground hover:bg-muted/50'
            }`}
          >
            <CalendarOff className="w-4 h-4" />
            Leaves
          </button>
          <button
            onClick={() => setActiveTab('swaps')}
            className={`flex-1 md:flex-none flex items-center justify-center gap-2 px-4 py-2 rounded-lg text-sm font-medium transition-all snap-center whitespace-nowrap ${
              activeTab === 'swaps'
                ? 'bg-background text-foreground shadow-sm'
                : 'text-muted-foreground hover:text-foreground hover:bg-muted/50'
            }`}
          >
            <ArrowLeftRight className="w-4 h-4" />
            Swaps
          </button>
        </div>
      </div>

      {/* ── Tab Content ── */}
      <div className="animate-in fade-in slide-in-from-bottom-4 duration-500">
        {activeTab === 'leaves' && <LeaveList />}
        {activeTab === 'swaps' && <SwapList />}
      </div>
    </div>
  );
};
