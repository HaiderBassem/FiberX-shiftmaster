import { useState, useMemo } from 'react';
import { useQuery } from '@tanstack/react-query';
import { motion, AnimatePresence } from 'framer-motion';
import { 
  startOfMonth, endOfMonth, startOfWeek, endOfWeek, 
  eachDayOfInterval, format, isSameMonth, isToday, 
  addMonths, subMonths, isWithinInterval, parseISO 
} from 'date-fns';
import { ChevronLeft, ChevronRight, Calendar as CalendarIcon, User, Loader2 } from 'lucide-react';
import api from '@/lib/api';
import { useAuthStore } from '@/store/authStore';
import { Card, CardContent, CardHeader } from '@/components/ui/card';
import { Button } from '@/components/ui/button';

export const InteractiveCalendar = () => {
  const { user } = useAuthStore();
  const [currentDate, setCurrentDate] = useState(new Date());
  const [selectedEmployeeId, setSelectedEmployeeId] = useState<string>(user?.id || '');

  const isSupervisor = user?.role === 'admin' || user?.role === 'manager' || user?.role === 'team_leader';

  const monthStart = startOfMonth(currentDate);
  const monthEnd = endOfMonth(currentDate);
  const calendarStart = startOfWeek(monthStart, { weekStartsOn: 0 }); // Sunday
  const calendarEnd = endOfWeek(monthEnd, { weekStartsOn: 0 });
  const calendarDays = eachDayOfInterval({ start: calendarStart, end: calendarEnd });

  const startDateStr = format(calendarStart, 'yyyy-MM-dd');
  const endDateStr = format(calendarEnd, 'yyyy-MM-dd');

  // Fetch employees for supervisor dropdown
  const { data: employees } = useQuery({
    queryKey: ['employees', 'all'],
    queryFn: async () => {
      const res = await api.get('/employees');
      return res.data?.data || [];
    },
    enabled: isSupervisor,
  });

  // Fetch Employee Shifts
  const { data: shifts, isLoading: shiftsLoading } = useQuery({
    queryKey: ['schedules', 'employee', selectedEmployeeId, startDateStr, endDateStr],
    queryFn: async () => {
      if (!selectedEmployeeId) return [];
      const res = await api.get(`/schedules/employee/${selectedEmployeeId}?from=${startDateStr}&to=${endDateStr}`);
      return res.data?.data || [];
    },
    enabled: !!selectedEmployeeId,
  });

  // Fetch Leaves
  const { data: leaves, isLoading: leavesLoading } = useQuery({
    queryKey: ['leaves', selectedEmployeeId === user?.id ? 'me' : 'history', startDateStr, endDateStr],
    queryFn: async () => {
      const endpoint = selectedEmployeeId === user?.id ? '/leaves/me' : '/leaves/history';
      const res = await api.get(endpoint);
      const allLeaves = res.data?.data || [];
      // Filter for this employee if using history
      if (selectedEmployeeId !== user?.id) {
        return allLeaves.filter((l: any) => l.employee_id === selectedEmployeeId);
      }
      return allLeaves;
    },
    enabled: !!selectedEmployeeId,
  });

  // Fetch Swaps
  const { data: swaps, isLoading: swapsLoading } = useQuery({
    queryKey: ['swaps', selectedEmployeeId === user?.id ? 'me' : 'history', startDateStr, endDateStr],
    queryFn: async () => {
      const endpoint = selectedEmployeeId === user?.id ? '/swaps/me' : '/swaps/history';
      const res = await api.get(endpoint);
      const allSwaps = res.data?.data || [];
      // Filter for this employee if using history
      if (selectedEmployeeId !== user?.id) {
        return allSwaps.filter((s: any) => s.requester_id === selectedEmployeeId || s.target_employee_id === selectedEmployeeId);
      }
      return allSwaps;
    },
    enabled: !!selectedEmployeeId,
  });

  const isLoading = shiftsLoading || leavesLoading || swapsLoading;

  // Next/Prev Month
  const nextMonth = () => setCurrentDate(addMonths(currentDate, 1));
  const prevMonth = () => setCurrentDate(subMonths(currentDate, 1));
  const goToToday = () => setCurrentDate(new Date());

  // Group data by date
  const eventsByDate = useMemo(() => {
    const map: Record<string, { shift?: any, leaves: any[], swaps: any[] }> = {};
    calendarDays.forEach(day => {
      map[format(day, 'yyyy-MM-dd')] = { leaves: [], swaps: [] };
    });

    (shifts || []).forEach((s: any) => {
      const dateKey = s.shift_date?.split('T')[0];
      if (map[dateKey]) map[dateKey].shift = s;
    });

    (leaves || []).forEach((l: any) => {
      const start = parseISO(l.start_date);
      const end = parseISO(l.end_date);
      calendarDays.forEach(day => {
        if (isWithinInterval(day, { start, end })) {
          map[format(day, 'yyyy-MM-dd')]?.leaves.push(l);
        }
      });
    });

    (swaps || []).forEach((s: any) => {
      const dateKey = s.shift_date?.split('T')[0];
      if (map[dateKey]) map[dateKey].swaps.push(s);
    });

    return map;
  }, [shifts, leaves, swaps, calendarDays]);

  const selectClass = "h-10 px-3 py-2 rounded-xl bg-background border border-input text-foreground focus:outline-none focus:ring-2 focus:ring-primary/50 transition-colors cursor-pointer shadow-sm";

  return (
    <div className="space-y-6 sm:space-y-8 max-w-[1600px] mx-auto pb-12">
      {/* Header */}
      <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
        <div>
          <h2 className="text-2xl sm:text-3xl font-bold tracking-tight text-foreground flex items-center gap-3">
            <CalendarIcon className="w-8 h-8 text-primary" />
            Interactive Calendar
          </h2>
          <p className="text-muted-foreground mt-1 text-sm sm:text-base">
            Your comprehensive monthly view of shifts, leaves, and swaps.
          </p>
        </div>

        {isSupervisor && (
          <div className="flex items-center gap-2">
            <User className="w-5 h-5 text-muted-foreground" />
            <select 
              className={selectClass + " w-full sm:w-64"}
              value={selectedEmployeeId}
              onChange={(e) => setSelectedEmployeeId(e.target.value)}
            >
              {(employees || []).filter((e: any) => e.status === 'active').map((e: any) => (
                <option key={e.id} value={e.id}>
                  {e.id === user?.id ? "My Calendar (Me)" : `${e.first_name} ${e.last_name}`}
                </option>
              ))}
            </select>
          </div>
        )}
      </div>

      <Card className="border-border/50 shadow-lg overflow-hidden bg-background/50 backdrop-blur-sm">
        <CardHeader className="border-b border-border/50 bg-muted/20 py-4 px-4 sm:px-6">
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-2">
              <Button variant="outline" size="icon" onClick={prevMonth} className="rounded-full w-8 h-8 sm:w-10 sm:h-10 border-border/50 hover:bg-primary/10 hover:text-primary">
                <ChevronLeft className="w-4 h-4 sm:w-5 sm:h-5" />
              </Button>
              <h3 className="text-lg sm:text-2xl font-bold min-w-[120px] sm:min-w-[160px] text-center text-foreground tracking-tight">
                {format(currentDate, 'MMMM yyyy')}
              </h3>
              <Button variant="outline" size="icon" onClick={nextMonth} className="rounded-full w-8 h-8 sm:w-10 sm:h-10 border-border/50 hover:bg-primary/10 hover:text-primary">
                <ChevronRight className="w-4 h-4 sm:w-5 sm:h-5" />
              </Button>
            </div>
            <Button variant="secondary" size="sm" onClick={goToToday} className="rounded-full px-4 sm:px-6 shadow-sm">
              Today
            </Button>
          </div>
        </CardHeader>

        <CardContent className="p-0 sm:p-2">
          {isLoading && !shifts ? (
            <div className="flex justify-center items-center h-[400px]">
              <Loader2 className="w-8 h-8 animate-spin text-primary/50" />
            </div>
          ) : (
            <div className="w-full overflow-x-auto pb-2">
              <div className="min-w-[700px]">
                {/* Weekday Headers */}
                <div className="grid grid-cols-7 gap-1 sm:gap-2 p-2">
                  {['Sun', 'Mon', 'Tue', 'Wed', 'Thu', 'Fri', 'Sat'].map(day => (
                    <div key={day} className="text-center font-bold text-xs sm:text-sm uppercase tracking-wider text-muted-foreground/70 py-2">
                      {day}
                    </div>
                  ))}
                </div>

                {/* Calendar Grid */}
                <AnimatePresence mode="wait">
                  <motion.div 
                    key={currentDate.toISOString()}
                    initial={{ opacity: 0, y: 10 }}
                    animate={{ opacity: 1, y: 0 }}
                    exit={{ opacity: 0, y: -10 }}
                    transition={{ duration: 0.2 }}
                    className="grid grid-cols-7 gap-1 sm:gap-2 p-2 pt-0"
                  >
                    {calendarDays.map((day) => {
                      const dateKey = format(day, 'yyyy-MM-dd');
                      const data = eventsByDate[dateKey];
                      const isCurrentMonth = isSameMonth(day, currentDate);
                      const isDayToday = isToday(day);

                      return (
                        <div 
                          key={dateKey} 
                          className={`
                            min-h-[100px] sm:min-h-[120px] rounded-xl sm:rounded-2xl border transition-all duration-200 p-1 sm:p-2 flex flex-col gap-1
                            ${isCurrentMonth ? 'bg-card border-border/60 shadow-sm' : 'bg-muted/10 border-border/20 opacity-50'}
                            ${isDayToday ? 'ring-2 ring-primary/50 ring-offset-1 ring-offset-background' : ''}
                            hover:border-primary/30 hover:shadow-md
                          `}
                        >
                          <div className={`text-right text-xs sm:text-sm font-semibold mb-1 ${isDayToday ? 'text-primary' : 'text-foreground/70'}`}>
                            {format(day, 'd')}
                          </div>
                          
                          <div className="flex flex-col gap-1 overflow-y-auto max-h-[80px] sm:max-h-[100px] no-scrollbar">
                            {/* Shift Badge */}
                            {data?.shift && (
                              <ShiftBadge shift={data.shift} />
                            )}

                            {/* Default to OFF if no shift and is in past or today (just a placeholder logic, usually we just show nothing if no shift) */}
                            {!data?.shift && isCurrentMonth && (
                               <div className="text-[10px] sm:text-xs px-1.5 py-0.5 rounded-md bg-muted/30 text-muted-foreground border border-border/50 truncate text-center">
                                 No Shift
                               </div>
                            )}

                            {/* Leaves */}
                            {data?.leaves.map((l: any, i: number) => (
                              <LeaveBadge key={i} leave={l} />
                            ))}

                            {/* Swaps */}
                            {data?.swaps.map((s: any, i: number) => (
                              <SwapBadge key={i} swap={s} currentUserId={selectedEmployeeId} />
                            ))}
                          </div>
                        </div>
                      );
                    })}
                  </motion.div>
                </AnimatePresence>
              </div>
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  );
};

// ─── Subcomponents ────────────────────────────────────────────────────────────

const ShiftBadge = ({ shift }: { shift: any }) => {
  const status = (shift.shift_status || 'working').toLowerCase();
  
  let styles = "bg-muted text-foreground border-border";
  if (status === 'working') styles = "bg-emerald-500/10 text-emerald-600 dark:text-emerald-400 border-emerald-500/20";
  if (status === 'off') styles = "bg-amber-500/10 text-amber-600 dark:text-amber-400 border-amber-500/20";
  if (status === 'leave') styles = "bg-rose-500/10 text-rose-600 dark:text-rose-400 border-rose-500/20";
  if (status === 'vacation') styles = "bg-blue-500/10 text-blue-600 dark:text-blue-400 border-blue-500/20";

  return (
    <div className={`text-[10px] sm:text-xs font-semibold px-1.5 py-1 rounded-md border truncate text-center ${styles}`}>
      {status === 'working' ? 'Working' : status.toUpperCase()}
    </div>
  );
};

const LeaveBadge = ({ leave }: { leave: any }) => {
  const isApproved = leave.status === 'approved_by_manager' || leave.status === 'approved_by_team_leader';
  const isRejected = leave.status === 'rejected';
  
  let styles = "bg-amber-500/10 text-amber-600 border-amber-500/20"; // pending
  if (isApproved) styles = "bg-purple-500/10 text-purple-600 dark:text-purple-400 border-purple-500/20";
  if (isRejected) styles = "bg-destructive/10 text-destructive border-destructive/20 line-through opacity-70";

  return (
    <div className={`text-[10px] sm:text-[11px] font-medium px-1.5 py-0.5 rounded-md border truncate flex justify-between items-center ${styles}`} title={leave.leave_type_name}>
      <span>🌴 {leave.leave_type_name || 'Leave'}</span>
      {!isApproved && !isRejected && <span className="opacity-70 text-[9px]">(Pend)</span>}
    </div>
  );
};

const SwapBadge = ({ swap, currentUserId }: { swap: any, currentUserId: string }) => {
  const isRequester = swap.requester_id === currentUserId;
  const isApproved = swap.status === 'approved';
  
  let styles = "bg-indigo-500/10 text-indigo-600 dark:text-indigo-400 border-indigo-500/20";
  
  return (
    <div className={`text-[10px] sm:text-[11px] font-medium px-1.5 py-0.5 rounded-md border truncate flex gap-1 items-center ${styles}`} title={`Swap ${isRequester ? 'Out' : 'In'}`}>
      <span>🔄 {isRequester ? 'Swap Out' : 'Swap In'}</span>
      {!isApproved && swap.status !== 'rejected' && <span className="opacity-70 text-[9px]">(Pend)</span>}
    </div>
  );
};

export default InteractiveCalendar;
