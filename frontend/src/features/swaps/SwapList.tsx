import { useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import api from '@/lib/api';
import { useAuthStore } from '@/store/authStore';
import { Card, CardContent, CardHeader, CardTitle, CardFooter } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import {
  ArrowLeftRight, Send, Clock, CheckCircle2, XCircle, UserCircle, Inbox,
} from 'lucide-react';
import { format } from 'date-fns';
import { fmtDate } from '@/lib/dateUtils';

export const SwapList = () => {
  const queryClient = useQueryClient();
  const { user } = useAuthStore();
  const [targetEmployeeId, setTargetEmployeeId] = useState('');
  const [shiftDate, setShiftDate] = useState(format(new Date(), 'yyyy-MM-dd'));
  const [reason, setReason] = useState('');
  const [error, setError] = useState<string | null>(null);
  const [respondError, setRespondError] = useState<string | null>(null);

  // My outgoing swap requests
  const { data: mySwaps, isLoading: mySwapsLoading } = useQuery({
    queryKey: ['swaps', 'me'],
    queryFn: async () => { const res = await api.get('/swaps/me'); return res.data?.data || []; },
  });

  // Incoming swap requests targeting me (pending my accept/decline)
  const { data: pendingForMe, isLoading: pendingLoading } = useQuery({
    queryKey: ['swaps', 'pending', 'for-me'],
    queryFn: async () => { const res = await api.get('/swaps/pending/for-me'); return res.data?.data || []; },
    refetchInterval: 30000, // Poll every 30s for new requests
  });

  // All employees for the dropdown
  const { data: employees } = useQuery({
    queryKey: ['employees', 'active'],
    queryFn: async () => { const res = await api.get('/employees?active=true'); return res.data?.data || []; },
  });

  const swapEligible = (employees || []).filter((emp: any) => {
    if (emp.id === user?.id) return false;
    if (emp.role !== 'employee') return false;
    if (user?.department_id && emp.department_id !== user.department_id) return false;
    return true;
  });

  const createSwap = useMutation({
    mutationFn: async () => {
      setError(null);
      await api.post('/swaps', { target_employee_id: targetEmployeeId, shift_date: shiftDate, reason });
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['swaps'] });
      setReason('');
      setTargetEmployeeId('');
    },
    onError: (err: any) => setError(err?.response?.data?.error || err?.message || 'Failed to create swap'),
  });

  const respondSwap = useMutation({
    mutationFn: async ({ swapId, accept }: { swapId: string; accept: boolean }) => {
      setRespondError(null);
      await api.post(`/swaps/${swapId}/respond`, { accept });
    },
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ['swaps'] }),
    onError: (err: any) => setRespondError(err?.response?.data?.error || err?.message || 'Failed to respond to swap'),
  });

  const getStatusBadge = (status: string) => {
    if (status === 'approved' || status === 'completed') {
      return <span className="px-2 py-1 rounded-md text-xs font-medium uppercase tracking-wider bg-emerald-500/10 text-emerald-500 border border-emerald-500/20">{status}</span>;
    }
    if (status === 'rejected') {
      return <span className="px-2 py-1 rounded-md text-xs font-medium uppercase tracking-wider bg-destructive/10 text-destructive border border-destructive/20">{status}</span>;
    }
    if (status === 'employee_accepted') {
      return <span className="px-2 py-1 rounded-md text-xs font-medium uppercase tracking-wider bg-blue-500/10 text-blue-500 border border-blue-500/20">Awaiting TL</span>;
    }
    return <span className="px-2 py-1 rounded-md text-xs font-medium uppercase tracking-wider bg-amber-500/10 text-amber-500 border border-amber-500/20">{status}</span>;
  };

  const getStatusIcon = (status: string) => {
    if (status === 'approved' || status === 'completed') return <CheckCircle2 className="w-5 h-5 text-emerald-500" />;
    if (status === 'rejected') return <XCircle className="w-5 h-5 text-destructive" />;
    if (status === 'employee_accepted') return <Clock className="w-5 h-5 text-blue-500" />;
    return <Clock className="w-5 h-5 text-amber-500" />;
  };

  const selectClass = "w-full h-10 px-3 py-2 rounded-lg bg-background border border-input text-foreground focus:outline-none focus:ring-2 focus:ring-ring transition-colors";

  const incomingCount = pendingForMe?.length || 0;

  return (
    <div className="space-y-6">


      {error && (
        <div className="p-3 rounded-xl bg-destructive/10 border border-destructive/20 text-destructive text-sm">{error}</div>
      )}

      {/* ── Incoming Requests Banner (always visible) ── */}
      <div className={`rounded-xl border p-4 ${incomingCount > 0
        ? 'bg-amber-500/5 border-amber-500/30'
        : 'bg-muted/30 border-border'
      }`}>
        <div className="flex items-center gap-3 mb-3">
          <Inbox className={`w-5 h-5 ${incomingCount > 0 ? 'text-amber-500' : 'text-muted-foreground'}`} />
          <h3 className={`text-base font-semibold ${incomingCount > 0 ? 'text-foreground' : 'text-muted-foreground'} flex items-center gap-2`}>
            Incoming Swap Requests
            {incomingCount > 0 && (
              <span className="inline-flex items-center justify-center w-5 h-5 rounded-full text-[11px] font-bold bg-amber-500 text-white">
                {incomingCount}
              </span>
            )}
          </h3>
        </div>

        {respondError && (
          <div className="mb-3 p-2 rounded-lg bg-destructive/10 border border-destructive/20 text-destructive text-sm">{respondError}</div>
        )}

        {pendingLoading ? (
          <div className="space-y-2">
            {[1, 2].map(i => <div key={i} className="animate-pulse h-16 bg-muted/50 rounded-lg" />)}
          </div>
        ) : incomingCount > 0 ? (
          <div className="space-y-3">
            {pendingForMe!.map((swap: any) => (
              <div key={swap.id} className="flex flex-col sm:flex-row items-start sm:items-center justify-between gap-3 p-3 rounded-lg bg-background border border-border">
                <div className="flex items-center gap-3">
                  <div className="h-10 w-10 rounded-full bg-amber-500/10 flex items-center justify-center flex-shrink-0">
                    <UserCircle className="w-5 h-5 text-amber-500" />
                  </div>
                  <div>
                    <p className="text-foreground font-medium">
                      {swap.requester_name || `Employee #${swap.requester_id?.slice(0, 8)}`}
                    </p>
                    <p className="text-sm text-muted-foreground">
                      {fmtDate(swap.shift_date)}
                      {swap.reason && <span className="italic"> · "{swap.reason}"</span>}
                    </p>
                  </div>
                </div>
                <div className="flex gap-2 flex-shrink-0">
                  <Button
                    size="sm"
                    variant="outline"
                    className="text-destructive border-destructive/30 hover:bg-destructive/10 gap-1"
                    onClick={() => respondSwap.mutate({ swapId: swap.id, accept: false })}
                    disabled={respondSwap.isPending}
                  >
                    <XCircle className="w-4 h-4" /> Decline
                  </Button>
                  <Button
                    size="sm"
                    className="gap-1 bg-emerald-600 hover:bg-emerald-700 text-white"
                    onClick={() => respondSwap.mutate({ swapId: swap.id, accept: true })}
                    disabled={respondSwap.isPending}
                  >
                    <CheckCircle2 className="w-4 h-4" /> Accept
                  </Button>
                </div>
              </div>
            ))}
          </div>
        ) : (
          <p className="text-sm text-muted-foreground text-center py-4">
            No incoming swap requests at the moment.
          </p>
        )}
      </div>

      <div className="grid lg:grid-cols-3 gap-6">
        {/* ── Create Swap ── */}
        <div className="lg:col-span-1">
          <Card className="sticky top-24">
            <CardHeader>
              <CardTitle className="flex items-center gap-2 text-base">
                <ArrowLeftRight className="w-5 h-5 text-primary" />
                New Swap Request
              </CardTitle>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="space-y-2">
                <Label>Swap With</Label>
                <select className={selectClass} value={targetEmployeeId} onChange={(e) => setTargetEmployeeId(e.target.value)}>
                  <option value="" disabled>Select a teammate</option>
                  {swapEligible.map((emp: any) => (
                    <option key={emp.id} value={emp.id}>{emp.first_name} {emp.last_name} — {emp.employee_code}</option>
                  ))}
                </select>
                {swapEligible.length === 0 && (
                  <p className="text-xs text-muted-foreground">No eligible teammates found in your department.</p>
                )}
              </div>
              <div className="space-y-2">
                <Label>Shift Date</Label>
                <Input type="date" value={shiftDate} onChange={(e: React.ChangeEvent<HTMLInputElement>) => setShiftDate(e.target.value)} />
              </div>
              <div className="space-y-2">
                <Label>Reason</Label>
                <textarea
                  className="w-full h-20 px-3 py-2 rounded-lg bg-background border border-input text-foreground focus:outline-none focus:ring-2 focus:ring-ring resize-none text-sm transition-colors"
                  placeholder="Why do you need this swap?"
                  value={reason}
                  onChange={(e) => setReason(e.target.value)}
                />
              </div>
            </CardContent>
            <CardFooter>
              <Button
                className="w-full gap-2"
                onClick={() => createSwap.mutate()}
                disabled={createSwap.isPending || !targetEmployeeId || !reason}
              >
                <Send className="w-4 h-4" /> {createSwap.isPending ? 'Sending…' : 'Send Swap Request'}
              </Button>
            </CardFooter>
          </Card>
        </div>

        {/* ── My Swap History ── */}
        <div className="lg:col-span-2 space-y-3">
          <h3 className="text-lg font-semibold text-foreground">Your Swap History</h3>
          {mySwapsLoading ? (
            <div className="space-y-3">{[1, 2].map(i => <Card key={i} className="animate-pulse h-20" />)}</div>
          ) : mySwaps?.length > 0 ? (
            mySwaps.map((swap: any) => (
              <Card key={swap.id}>
                <CardContent className="p-4 flex flex-col sm:flex-row items-start sm:items-center justify-between gap-3">
                  <div className="flex items-center gap-3">
                    <div className="h-10 w-10 rounded-full bg-muted flex items-center justify-center flex-shrink-0">
                      {getStatusIcon(swap.status)}
                    </div>
                    <div>
                      <p className="font-medium text-foreground">
                        Swap with {swap.target_employee_name || `#${swap.target_employee_id?.slice(0, 8)}`}
                      </p>
                      <p className="text-sm text-muted-foreground">
                        {fmtDate(swap.shift_date)}
                        {swap.reason && <span className="italic"> · "{swap.reason}"</span>}
                      </p>
                    </div>
                  </div>
                  {getStatusBadge(swap.status)}
                </CardContent>
              </Card>
            ))
          ) : (
            <Card className="border-dashed">
              <CardContent className="flex flex-col items-center justify-center p-12 text-muted-foreground">
                <ArrowLeftRight className="w-12 h-12 mb-4 opacity-20" />
                <p>No swap requests sent yet.</p>
              </CardContent>
            </Card>
          )}
        </div>
      </div>
    </div>
  );
};
