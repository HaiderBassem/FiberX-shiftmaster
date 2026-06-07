import { useState } from 'react';
import { useQuery } from '@tanstack/react-query';
import api from '@/lib/api';
import { useAuthStore } from '@/store/authStore';
import { Card } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import {
  History, ChevronLeft, ChevronRight, Calendar, Loader2,
  CheckCircle2, AlertTriangle, Clock, Play
} from 'lucide-react';
import { format, addDays, subDays } from 'date-fns';
import { fmtTime } from '@/lib/dateUtils';

interface TaskHistoryRow {
  assignment_id: string;
  execution_id: string;
  assigned_date: string;
  task_title: string;
  task_description: string | null;
  board_name: string | null;
  employee_id: string;
  employee_name: string;
  employee_code: string;
  employee_profile_image: string | null;
  status: string;
  completion_type: string | null;
  started_at: string | null;
  completed_at: string | null;
  notes: string | null;
}

export const TaskHistory = () => {
  const [selectedDate, setSelectedDate] = useState(() => new Date());
  const dateStr = format(selectedDate, 'yyyy-MM-dd');

  const { user, adminSelectedDepartmentId, managerSelectedDepartmentId } = useAuthStore();
  const selectedDeptId = user?.role === 'admin' ? adminSelectedDepartmentId : (user?.role === 'manager' ? managerSelectedDepartmentId : null);

  const { data: history, isLoading } = useQuery({
    queryKey: ['task-history', dateStr, selectedDeptId],
    queryFn: async () => {
      const url = `/tasks/history?date=${dateStr}`;
      const res = await api.get(url);
      return (res.data?.data || []) as TaskHistoryRow[];
    },
  });

  const statusBadge = (status: string, completionType: string | null) => {
    if (status === 'completed' && completionType === 'without_issue') {
      return { icon: <CheckCircle2 className="w-3 h-3" />, text: 'Completed', cls: 'bg-emerald-500/10 text-emerald-600 border-emerald-500/20' };
    }
    if (status === 'completed' && completionType === 'with_issue') {
      return { icon: <AlertTriangle className="w-3 h-3" />, text: 'Issues', cls: 'bg-amber-500/10 text-amber-600 border-amber-500/20' };
    }
    if (status === 'completed') {
      return { icon: <CheckCircle2 className="w-3 h-3" />, text: 'Completed', cls: 'bg-emerald-500/10 text-emerald-600 border-emerald-500/20' };
    }
    if (status === 'in_progress') {
      return { icon: <Play className="w-3 h-3" />, text: 'Active', cls: 'bg-amber-500/10 text-amber-500 border-amber-500/20' };
    }
    return { icon: <Clock className="w-3 h-3" />, text: 'Pending', cls: 'bg-muted/30 text-muted-foreground border-border' };
  };

  return (
    <div className="space-y-4 h-full flex flex-col">
      
      {/* Header & Date Filter */}
      <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-3">
        <h3 className="text-xl font-semibold text-foreground flex items-center gap-2">
          <History className="w-5 h-5 text-primary" />
          Task History
        </h3>
        <div className="flex items-center gap-1 bg-muted/30 p-1 rounded-lg border border-border">
          <Button variant="ghost" size="icon" className="h-7 w-7" onClick={() => setSelectedDate((prev) => subDays(prev, 1))}>
            <ChevronLeft className="w-4 h-4" />
          </Button>
          <div className="flex items-center gap-1 px-2 py-1 text-sm font-medium">
            <Calendar className="w-3 h-3 text-muted-foreground" />
            <input
              type="date"
              value={dateStr}
              onChange={(e) => setSelectedDate(new Date(e.target.value + 'T00:00:00'))}
              className="bg-transparent text-foreground border-none focus:outline-none max-w-[110px]"
            />
          </div>
          <Button variant="ghost" size="icon" className="h-7 w-7" onClick={() => setSelectedDate((prev) => addDays(prev, 1))}>
            <ChevronRight className="w-4 h-4" />
          </Button>
        </div>
      </div>

      {/* Feed */}
      {isLoading ? (
        <div className="flex-1 flex items-center justify-center min-h-[200px]">
          <Loader2 className="w-6 h-6 animate-spin text-primary/50" />
        </div>
      ) : history?.length === 0 ? (
        <div className="flex-1 p-8 text-center text-muted-foreground border border-border border-dashed rounded-xl bg-muted/10 flex flex-col items-center justify-center">
          <History className="w-8 h-8 mb-2 opacity-20" />
          <p className="text-sm">No activity on this date.</p>
        </div>
      ) : (
        <div className="flex-1 overflow-y-auto max-h-[600px] pr-2 space-y-3">
          {history?.map((task) => {
            const badge = statusBadge(task.status, task.completion_type);
            return (
              <Card key={task.assignment_id} className="p-3 shadow-sm border-border hover:border-primary/20 transition-colors">
                <div className="flex items-start justify-between gap-2">
                  <div className="space-y-1">
                    <p className="text-sm font-medium leading-none">{task.task_title}</p>
                    <p className="text-xs text-muted-foreground flex flex-wrap items-center gap-1.5 pt-1">
                      {task.employee_profile_image && (
                        <img src={task.employee_profile_image} alt="" className="w-4 h-4 rounded-full object-cover" />
                      )}
                      <span className="font-medium text-foreground/80">{task.employee_name}</span>
                      {task.board_name && (
                        <>
                          <span className="opacity-50">•</span>
                          <span className="text-primary/70">{task.board_name}</span>
                        </>
                      )}
                    </p>
                  </div>
                  <div className="flex flex-col items-end gap-1">
                    <span className={`flex items-center gap-1 px-1.5 py-0.5 rounded text-[10px] font-medium border ${badge.cls}`}>
                      {badge.icon} {badge.text}
                    </span>
                    {task.completed_at && (
                      <span className="text-[10px] text-muted-foreground">
                        {fmtTime(task.completed_at)}
                      </span>
                    )}
                  </div>
                </div>
                {task.notes && (
                  <div className="mt-2 p-2 bg-muted/30 rounded text-xs text-muted-foreground border-l-2 border-primary/40">
                    <span className="font-medium text-foreground block mb-1">Notes & Attachments:</span>
                    <div className="jodit-content overflow-hidden max-w-full" dangerouslySetInnerHTML={{ __html: task.notes }} />
                  </div>
                )}
              </Card>
            );
          })}
        </div>
      )}
    </div>
  );
};
