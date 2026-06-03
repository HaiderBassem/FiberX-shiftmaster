import { useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import api from '@/lib/api';
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { ClipboardList, Plus, CheckSquare, Trash2, ToggleLeft, ToggleRight, X } from 'lucide-react';

const DAYS = ['Sunday', 'Monday', 'Tuesday', 'Wednesday', 'Thursday', 'Friday', 'Saturday'];

export const TaskManagement = () => {
  const queryClient = useQueryClient();

  // ── UI State ──
  const [showModal, setShowModal] = useState(false);

  // ── Form State ──
  const [title, setTitle] = useState('');
  const [description, setDescription] = useState('');
  const [scheduleType] = useState('daily_task');
  const [recurrence, setRecurrence] = useState('daily');
  const [recurrenceDays, setRecurrenceDays] = useState<number[]>([]);
  const [maxAssignees, setMaxAssignees] = useState(1);
  const [selectedShiftId, setSelectedShiftId] = useState('');
  const [selectedBoardId, setSelectedBoardId] = useState('');

  // ── Queries ──
  const { data: taskSchedules, isLoading: schedulesLoading } = useQuery({
    queryKey: ['tasks', 'schedules'],
    queryFn: async () => { const res = await api.get('/tasks/schedules'); return res.data?.data || []; },
  });

  const { data: shifts } = useQuery({
    queryKey: ['shifts'],
    queryFn: async () => { const res = await api.get('/shifts'); return res.data?.data || []; },
  });

  const { data: boards } = useQuery({
    queryKey: ['tasks', 'boards'],
    queryFn: async () => { const res = await api.get('/tasks/boards'); return res.data?.data || []; },
  });

  // ── Mutations ──
  const createSchedule = useMutation({
    mutationFn: async () => {
      await api.post('/tasks/schedules', {
        title, description: description || null, schedule_type: scheduleType,
        board_id: selectedBoardId || null, shift_id: selectedShiftId || null,
        recurrence, recurrence_days: recurrenceDays, max_assignees: maxAssignees, is_active: true,
      });
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['tasks'] });
      setTitle(''); setDescription(''); setRecurrenceDays([]); setSelectedBoardId('');
      setShowModal(false);
    },
  });

  const toggleActive = useMutation({
    mutationFn: async ({ id, active }: { id: string; active: boolean }) => {
      await api.patch(`/tasks/schedules/${id}/toggle`, { is_active: active });
    },
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ['tasks'] }),
  });

  const deleteSchedule = useMutation({
    mutationFn: async (id: string) => { await api.delete(`/tasks/schedules/${id}`); },
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ['tasks'] }),
  });

  const toggleDay = (day: number) => {
    setRecurrenceDays(prev => prev.includes(day) ? prev.filter(d => d !== day) : [...prev, day]);
  };

  const selectClass = "w-full h-10 px-3 py-2 rounded-lg bg-background border border-input text-foreground focus:outline-none focus:ring-2 focus:ring-ring transition-colors";

  return (
    <div className="space-y-4 h-full flex flex-col">
      <div className="flex items-center justify-between">
        <h3 className="text-xl font-semibold text-foreground flex items-center gap-2">
          <ClipboardList className="w-5 h-5 text-primary" />
          Task Schedules
        </h3>
        <Button onClick={() => setShowModal(true)} size="sm" className="gap-2">
          <Plus className="w-4 h-4" /> New
        </Button>
      </div>

      {schedulesLoading ? (
        <div className="space-y-4 flex-1">
          {[1,2,3].map(i => <Card key={i} className="animate-pulse h-24" />)}
        </div>
      ) : (
        <div className="space-y-4 flex-1 overflow-y-auto max-h-[600px] pr-2">
          {taskSchedules?.map((ts: any) => (
            <Card key={ts.id} className={`transition-all ${ts.is_active ? '' : 'opacity-60'}`}>
              <CardHeader className="p-4 pb-2">
                <div className="flex items-start justify-between">
                  <CardTitle className="text-sm font-semibold flex items-center gap-2">
                    <CheckSquare className="w-4 h-4 text-primary" />
                    {ts.title}
                  </CardTitle>
                  <div className="flex gap-1">
                    <button onClick={() => toggleActive.mutate({ id: ts.id, active: !ts.is_active })}
                      className="p-1 rounded hover:bg-muted transition-colors"
                      title={ts.is_active ? 'Deactivate' : 'Activate'}>
                      {ts.is_active ? <ToggleRight className="w-4 h-4 text-emerald-500" /> : <ToggleLeft className="w-4 h-4 text-muted-foreground" />}
                    </button>
                    <button onClick={() => deleteSchedule.mutate(ts.id)}
                      className="p-1 rounded hover:bg-destructive/10 transition-colors" title="Delete">
                      <Trash2 className="w-4 h-4 text-destructive" />
                    </button>
                  </div>
                </div>
                <CardDescription className="text-xs">
                  {ts.recurrence === 'daily' ? 'Every day' : 
                   ts.recurrence === 'weekly' ? `Days: ${ts.recurrence_days?.map((d: number) => DAYS[d]?.slice(0, 3)).join(', ')}` :
                   'One-time'}
                  {ts.max_assignees > 1 && ` · Up to ${ts.max_assignees} assignees`}
                </CardDescription>
              </CardHeader>
              {ts.description && (
                <CardContent className="p-4 pt-0">
                  <p className="text-xs text-muted-foreground line-clamp-2">{ts.description}</p>
                </CardContent>
              )}
            </Card>
          ))}
          {(!taskSchedules || taskSchedules.length === 0) && (
            <div className="p-8 text-center text-muted-foreground border border-border border-dashed rounded-xl bg-muted/10">
              <ClipboardList className="w-8 h-8 mx-auto mb-2 opacity-20" />
              <p className="text-sm">No schedules found</p>
            </div>
          )}
        </div>
      )}

      {/* ── Create Modal ── */}
      {showModal && (
        <div className="fixed inset-0 z-50 bg-background/80 backdrop-blur-sm flex items-center justify-center p-4">
          <div className="bg-card w-full max-w-lg rounded-2xl shadow-xl border border-border animate-in zoom-in-95 duration-200">
            <div className="flex items-center justify-between p-6 border-b border-border">
              <h3 className="text-lg font-semibold flex items-center gap-2">
                <Plus className="w-5 h-5 text-primary" /> Create Task Schedule
              </h3>
              <button onClick={() => setShowModal(false)} className="p-2 hover:bg-muted rounded-full transition-colors">
                <X className="w-5 h-5 text-muted-foreground" />
              </button>
            </div>
            
            <div className="p-6 space-y-4 max-h-[70vh] overflow-y-auto">
              <div className="space-y-2">
                <Label>Task Title</Label>
                <Input value={title} onChange={(e: any) => setTitle(e.target.value)} placeholder="e.g. Node check" />
              </div>
              <div className="space-y-2">
                <Label>Description</Label>
                <textarea
                  className="w-full h-20 px-3 py-2 rounded-lg bg-background border border-input text-foreground focus:outline-none focus:ring-2 focus:ring-ring resize-none text-sm transition-colors"
                  placeholder="Details about the task..."
                  value={description}
                  onChange={(e) => setDescription(e.target.value)}
                />
              </div>
              <div className="grid grid-cols-2 gap-4">
                <div className="space-y-2">
                  <Label>Board</Label>
                  <select className={selectClass} value={selectedBoardId} onChange={(e) => setSelectedBoardId(e.target.value)}>
                    <option value="">No board (optional)</option>
                    {boards?.map((b: any) => <option key={b.id} value={b.id}>{b.name}</option>)}
                  </select>
                </div>
                <div className="space-y-2">
                  <Label>Shift</Label>
                  <select className={selectClass} value={selectedShiftId} onChange={(e) => setSelectedShiftId(e.target.value)}>
                    <option value="">Any Shift</option>
                    {shifts?.map((s: any) => <option key={s.id} value={s.id}>{s.name} ({s.shift_code})</option>)}
                  </select>
                </div>
              </div>
              
              <div className="grid grid-cols-2 gap-4">
                <div className="space-y-2">
                  <Label>Recurrence</Label>
                  <select className={selectClass} value={recurrence} onChange={(e) => setRecurrence(e.target.value)}>
                    <option value="daily">Daily</option>
                    <option value="weekly">Specific Days</option>
                    <option value="once">One-Time</option>
                  </select>
                </div>
                <div className="space-y-2">
                  <Label>Max Assignees</Label>
                  <Input type="number" min={1} value={maxAssignees} onChange={(e: any) => setMaxAssignees(parseInt(e.target.value) || 1)} />
                </div>
              </div>

              {recurrence === 'weekly' && (
                <div className="space-y-2 pt-2">
                  <Label>Select Days</Label>
                  <div className="flex flex-wrap gap-2">
                    {DAYS.map((day, i) => (
                      <button key={i} onClick={() => toggleDay(i)}
                        className={`px-3 py-1.5 rounded-md text-xs font-medium transition-all ${
                          recurrenceDays.includes(i) ? 'bg-primary text-primary-foreground shadow-lg' : 'bg-muted text-muted-foreground hover:bg-muted/80'
                        }`}
                      >{day.slice(0, 3)}</button>
                    ))}
                  </div>
                </div>
              )}
            </div>
            
            <div className="p-6 border-t border-border bg-muted/20 rounded-b-2xl">
              <Button className="w-full gap-2" onClick={() => createSchedule.mutate()} disabled={createSchedule.isPending || !title}>
                <Plus className="w-4 h-4" /> Create Task Schedule
              </Button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
};
