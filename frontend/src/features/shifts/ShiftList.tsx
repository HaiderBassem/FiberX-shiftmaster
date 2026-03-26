import { useMemo, useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import api from '@/lib/api';
import { useAuthStore } from '@/store/authStore';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Clock, Edit3, Plus, Save, Trash2, X } from 'lucide-react';
import { format } from 'date-fns';

interface Shift {
  id: string;
  shift_code: string;
  name: string;
  name_en?: string | null;
  start_time: string; // API returns stringified time.Time
  end_time: string;
  color_code?: string | null;
  requires_vehicle?: boolean;
  min_rest_hours?: number;
}

export const ShiftList = () => {
  const queryClient = useQueryClient();
  const { user } = useAuthStore();
  const canManage = user?.role === 'admin' || user?.role === 'manager' || user?.role === 'team_leader';

  const [showCreate, setShowCreate] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const [createShiftCode, setCreateShiftCode] = useState('');
  const [createName, setCreateName] = useState('');
  const [createStart, setCreateStart] = useState('08:00');
  const [createEnd, setCreateEnd] = useState('16:00');
  const [createColor, setCreateColor] = useState('#3b82f6');
  const [createMinRest, setCreateMinRest] = useState(0);
  const [createRequiresVehicle, setCreateRequiresVehicle] = useState(false);

  const [editId, setEditId] = useState<string | null>(null);
  const [editShiftCode, setEditShiftCode] = useState('');
  const [editName, setEditName] = useState('');
  const [editStart, setEditStart] = useState('08:00');
  const [editEnd, setEditEnd] = useState('16:00');
  const [editColor, setEditColor] = useState('#3b82f6');
  const [editMinRest, setEditMinRest] = useState(0);
  const [editRequiresVehicle, setEditRequiresVehicle] = useState(false);

  const { data: shifts, isLoading, isError } = useQuery<Shift[]>({
    queryKey: ['shifts'],
    queryFn: async () => {
      const response = await api.get('/shifts');
      return response.data?.data || [];
    },
  });

  const shiftsForUi = useMemo(() => (shifts || []).map((s) => ({
    ...s,
    start_hm: toHHMM(s.start_time),
    end_hm: toHHMM(s.end_time),
    color: s.color_code || '#3b82f6',
    requires_vehicle: !!s.requires_vehicle,
    min_rest_hours: s.min_rest_hours ?? 0,
  })), [shifts]);

  const createMutation = useMutation({
    mutationFn: async () => {
      setError(null);
      await api.post('/shifts', {
        shift_code: createShiftCode,
        name: createName,
        name_en: null,
        start_time: createStart,
        end_time: createEnd,
        color_code: createColor || null,
        requires_vehicle: createRequiresVehicle,
        min_rest_hours: createMinRest,
      });
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['shifts'] });
      setShowCreate(false);
      setCreateShiftCode('');
      setCreateName('');
      setCreateStart('08:00');
      setCreateEnd('16:00');
      setCreateColor('#3b82f6');
      setCreateMinRest(0);
      setCreateRequiresVehicle(false);
    },
    onError: (err: any) => setError(err?.response?.data?.error || err?.message || 'Failed to create shift'),
  });

  const updateMutation = useMutation({
    mutationFn: async () => {
      if (!editId) return;
      setError(null);
      await api.put(`/shifts/${editId}`, {
        shift_code: editShiftCode,
        name: editName,
        name_en: null,
        start_time: editStart,
        end_time: editEnd,
        color_code: editColor || null,
        requires_vehicle: editRequiresVehicle,
        min_rest_hours: editMinRest,
      });
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['shifts'] });
      setEditId(null);
    },
    onError: (err: any) => setError(err?.response?.data?.error || err?.message || 'Failed to update shift'),
  });

  const deleteMutation = useMutation({
    mutationFn: async (id: string) => {
      setError(null);
      await api.delete(`/shifts/${id}`);
    },
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ['shifts'] }),
    onError: (err: any) => setError(err?.response?.data?.error || err?.message || 'Failed to delete shift'),
  });

  const startEdit = (s: any) => {
    setEditId(s.id);
    setEditShiftCode(s.shift_code);
    setEditName(s.name);
    setEditStart(s.start_hm);
    setEditEnd(s.end_hm);
    setEditColor(s.color);
    setEditMinRest(s.min_rest_hours);
    setEditRequiresVehicle(!!s.requires_vehicle);
  };

  return (
    <div className="space-y-6">
      <div className="flex justify-between items-end">
        <div>
          <h2 className="text-3xl font-bold tracking-tight text-white">Shifts Configuration</h2>
          <p className="text-zinc-400">
            {canManage ? 'Create and edit shift times and rules.' : 'View shift times and rules.'}
          </p>
        </div>
        {canManage && (
          <Button
            onClick={() => { setShowCreate((v) => !v); setError(null); }}
            className="bg-emerald-600 hover:bg-emerald-500 text-white gap-2"
          >
            {showCreate ? <X className="w-4 h-4" /> : <Plus className="w-4 h-4" />}
            {showCreate ? 'Close' : 'New Shift'}
          </Button>
        )}
      </div>

      {error && (
        <div className="p-3 rounded-xl bg-red-500/10 border border-red-500/20 text-red-300 text-sm">
          {error}
        </div>
      )}

      {canManage && showCreate && (
        <Card className="bg-zinc-900/50 border-zinc-800/60">
          <CardHeader>
            <CardTitle className="text-white flex items-center gap-2">
              <Plus className="w-5 h-5 text-emerald-400" />
              Create Shift
            </CardTitle>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="grid md:grid-cols-6 gap-4">
              <div className="space-y-2 md:col-span-2">
                <Label className="text-zinc-300">Shift code</Label>
                <Input value={createShiftCode} onChange={(e) => setCreateShiftCode(e.target.value)} className="bg-zinc-950/50 border-zinc-700" />
              </div>
              <div className="space-y-2 md:col-span-2">
                <Label className="text-zinc-300">Name</Label>
                <Input value={createName} onChange={(e) => setCreateName(e.target.value)} className="bg-zinc-950/50 border-zinc-700" />
              </div>
              <div className="space-y-2">
                <Label className="text-zinc-300">Start</Label>
                <Input type="time" value={createStart} onChange={(e) => setCreateStart(e.target.value)} className="bg-zinc-950/50 border-zinc-700" />
              </div>
              <div className="space-y-2">
                <Label className="text-zinc-300">End</Label>
                <Input type="time" value={createEnd} onChange={(e) => setCreateEnd(e.target.value)} className="bg-zinc-950/50 border-zinc-700" />
              </div>
              <div className="space-y-2">
                <Label className="text-zinc-300">Color</Label>
                <Input type="color" value={createColor} onChange={(e) => setCreateColor(e.target.value)} className="bg-zinc-950/50 border-zinc-700 h-10 p-1" />
              </div>
              <div className="space-y-2">
                <Label className="text-zinc-300">Min rest (hrs)</Label>
                <Input type="number" min={0} value={createMinRest} onChange={(e) => setCreateMinRest(parseInt(e.target.value) || 0)} className="bg-zinc-950/50 border-zinc-700" />
              </div>
              <div className="flex items-center gap-2 md:col-span-2">
                <input id="veh" type="checkbox" checked={createRequiresVehicle} onChange={(e) => setCreateRequiresVehicle(e.target.checked)} />
                <Label htmlFor="veh" className="text-zinc-300">Requires vehicle</Label>
              </div>
            </div>
            <div className="flex justify-end">
              <Button
                onClick={() => createMutation.mutate()}
                disabled={createMutation.isPending || !createShiftCode || !createName || !createStart || !createEnd}
                className="bg-emerald-600 hover:bg-emerald-500 text-white gap-2"
              >
                <Save className="w-4 h-4" />
                {createMutation.isPending ? 'Saving…' : 'Create'}
              </Button>
            </div>
          </CardContent>
        </Card>
      )}

      {isLoading ? (
        <div className="grid gap-6 md:grid-cols-3">
          {[1, 2, 3].map((i) => (
            <Card key={i} className="animate-pulse bg-zinc-900/50">
              <CardHeader className="h-20 bg-zinc-800/50 rounded-t-xl" />
            </Card>
          ))}
        </div>
      ) : isError ? (
        <div className="p-4 rounded-xl bg-red-500/10 border border-red-500/20 text-red-400">
          Failed to load shifts. Please try again.
        </div>
      ) : (
        <div className="grid gap-6 md:grid-cols-2 lg:grid-cols-3">
          {shiftsForUi?.map((shift: any) => (
            <Card key={shift.id} className="bg-zinc-900/40 border-zinc-800/60 transition-all hover:bg-zinc-800/60 overflow-hidden relative group">
              <div
                className="absolute top-0 w-full h-1.5 transition-all group-hover:h-2"
                style={{ backgroundColor: shift.color }}
              />
              <CardHeader className="flex flex-row items-center justify-between gap-4 pb-2">
                <div className="flex items-center gap-4">
                  <div
                    className="h-10 w-10 flex shrink-0 items-center justify-center rounded-xl shadow-inner"
                    style={{ backgroundColor: `${shift.color}20`, color: shift.color }}
                  >
                    <Clock className="w-5 h-5" />
                  </div>
                  <div>
                    <CardTitle className="text-lg font-semibold text-white">{shift.name}</CardTitle>
                    <p className="text-xs font-mono text-zinc-500">{shift.shift_code}</p>
                  </div>
                </div>

                {canManage && (
                  <div className="flex items-center gap-2 opacity-0 group-hover:opacity-100 transition-opacity">
                    <Button
                      size="icon"
                      variant="outline"
                      className="h-8 w-8 border-zinc-700 text-zinc-300 hover:bg-zinc-800"
                      onClick={() => startEdit(shift)}
                      title="Edit"
                    >
                      <Edit3 className="w-4 h-4" />
                    </Button>
                    <Button
                      size="icon"
                      variant="outline"
                      className="h-8 w-8 border-red-500/30 text-red-400 hover:bg-red-500/10"
                      onClick={() => { if (confirm(`Delete shift "${shift.name}"?`)) deleteMutation.mutate(shift.id); }}
                      title="Delete"
                    >
                      <Trash2 className="w-4 h-4" />
                    </Button>
                  </div>
                )}
              </CardHeader>

              <CardContent className="space-y-3">
                {editId === shift.id ? (
                  <div className="space-y-3 pt-3 border-t border-zinc-800">
                    <div className="grid grid-cols-2 gap-3">
                      <div className="space-y-2">
                        <Label className="text-zinc-300">Code</Label>
                        <Input value={editShiftCode} onChange={(e) => setEditShiftCode(e.target.value)} className="bg-zinc-950/50 border-zinc-700" />
                      </div>
                      <div className="space-y-2">
                        <Label className="text-zinc-300">Name</Label>
                        <Input value={editName} onChange={(e) => setEditName(e.target.value)} className="bg-zinc-950/50 border-zinc-700" />
                      </div>
                      <div className="space-y-2">
                        <Label className="text-zinc-300">Start</Label>
                        <Input type="time" value={editStart} onChange={(e) => setEditStart(e.target.value)} className="bg-zinc-950/50 border-zinc-700" />
                      </div>
                      <div className="space-y-2">
                        <Label className="text-zinc-300">End</Label>
                        <Input type="time" value={editEnd} onChange={(e) => setEditEnd(e.target.value)} className="bg-zinc-950/50 border-zinc-700" />
                      </div>
                      <div className="space-y-2">
                        <Label className="text-zinc-300">Color</Label>
                        <Input type="color" value={editColor} onChange={(e) => setEditColor(e.target.value)} className="bg-zinc-950/50 border-zinc-700 h-10 p-1" />
                      </div>
                      <div className="space-y-2">
                        <Label className="text-zinc-300">Min rest (hrs)</Label>
                        <Input type="number" min={0} value={editMinRest} onChange={(e) => setEditMinRest(parseInt(e.target.value) || 0)} className="bg-zinc-950/50 border-zinc-700" />
                      </div>
                      <div className="flex items-center gap-2 col-span-2">
                        <input id={`veh-${shift.id}`} type="checkbox" checked={editRequiresVehicle} onChange={(e) => setEditRequiresVehicle(e.target.checked)} />
                        <Label htmlFor={`veh-${shift.id}`} className="text-zinc-300">Requires vehicle</Label>
                      </div>
                    </div>

                    <div className="flex justify-end gap-2">
                      <Button
                        variant="outline"
                        className="border-zinc-700 text-zinc-300"
                        onClick={() => setEditId(null)}
                      >
                        Cancel
                      </Button>
                      <Button
                        className="bg-emerald-600 hover:bg-emerald-500 text-white gap-2"
                        onClick={() => updateMutation.mutate()}
                        disabled={updateMutation.isPending || !editShiftCode || !editName || !editStart || !editEnd}
                      >
                        <Save className="w-4 h-4" />
                        {updateMutation.isPending ? 'Saving…' : 'Save'}
                      </Button>
                    </div>
                  </div>
                ) : (
                  <div className="flex items-center gap-4 text-sm mt-3 pt-3 border-t border-zinc-800">
                    <div className="flex flex-col">
                      <span className="text-zinc-500 text-xs uppercase tracking-wider">Start</span>
                      <span className="text-white font-mono">{shift.start_hm}</span>
                    </div>
                    <div className="h-6 w-px bg-zinc-800" />
                    <div className="flex flex-col">
                      <span className="text-zinc-500 text-xs uppercase tracking-wider">End</span>
                      <span className="text-white font-mono">{shift.end_hm}</span>
                    </div>
                  </div>
                )}
              </CardContent>
            </Card>
          ))}
          {shifts?.length === 0 && (
            <div className="col-span-full p-8 text-center text-zinc-500 border border-zinc-800/60 border-dashed rounded-xl bg-zinc-900/20">
              No shifts configured.
            </div>
          )}
        </div>
      )}
    </div>
  );
};

function toHHMM(raw: string): string {
  if (!raw) return '';
  // If ISO string, try to parse.
  if (raw.includes('T')) {
    const d = new Date(raw);
    if (!Number.isNaN(d.getTime())) return format(d, 'HH:mm');
  }
  // If "HH:MM:SS" or "HH:MM"
  if (raw.length >= 5 && raw[2] === ':') return raw.slice(0, 5);
  return raw;
}
