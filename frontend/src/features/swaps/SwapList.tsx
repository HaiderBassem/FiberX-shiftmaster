import { useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import api from '@/lib/api';
import { useAuthStore } from '@/store/authStore';
import { Card, CardContent, CardHeader, CardTitle, CardFooter } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { ArrowLeftRight, Send, Clock, CheckCircle2, XCircle, UserCircle } from 'lucide-react';
import { format } from 'date-fns';

export const SwapList = () => {
  const queryClient = useQueryClient();
  const { user } = useAuthStore();
  const [targetEmployeeId, setTargetEmployeeId] = useState('');
  const [shiftDate, setShiftDate] = useState(format(new Date(), 'yyyy-MM-dd'));
  const [reason, setReason] = useState('');
  const [error, setError] = useState<string | null>(null);

  const { data: mySwaps, isLoading } = useQuery({
    queryKey: ['swaps', 'me'],
    queryFn: async () => { const res = await api.get('/swaps/me'); return res.data?.data || []; },
  });

  const { data: pendingForMe } = useQuery({
    queryKey: ['swaps', 'pending', 'for-me'],
    queryFn: async () => { const res = await api.get('/swaps/pending/for-me'); return res.data?.data || []; },
  });

  // Fetch all employees, then filter client-side:
  // - Same department as current user
  // - Only role=employee (exclude team_leader, manager, admin)
  // - Exclude self
  const { data: employees } = useQuery({
    queryKey: ['employees', 'active'],
    queryFn: async () => { const res = await api.get('/employees?active=true'); return res.data?.data || []; },
  });

  const swapEligible = (employees || []).filter((emp: any) => {
    if (emp.id === user?.id) return false; // exclude self
    if (emp.role !== 'employee') return false; // only regular employees
    if (user?.department_id && emp.department_id !== user.department_id) return false; // same department
    return true;
  });

  const createSwap = useMutation({
    mutationFn: async () => {
      setError(null);
      await api.post('/swaps', { target_employee_id: targetEmployeeId, shift_date: shiftDate, reason });
    },
    onSuccess: () => { queryClient.invalidateQueries({ queryKey: ['swaps'] }); setReason(''); setTargetEmployeeId(''); },
    onError: (err: any) => setError(err?.response?.data?.error || err?.message || 'Failed to create swap'),
  });

  const respondSwap = useMutation({
    mutationFn: async ({ swapId, accept }: { swapId: string; accept: boolean }) => {
      await api.post(`/swaps/${swapId}/respond`, { accept });
    },
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ['swaps'] }),
  });

  const getStatusIcon = (status: string) => {
    if (status === 'approved' || status === 'completed') return <CheckCircle2 className="w-5 h-5 text-emerald-500" />;
    if (status === 'rejected') return <XCircle className="w-5 h-5 text-destructive" />;
    return <Clock className="w-5 h-5 text-amber-500" />;
  };

  const selectClass = "w-full h-10 px-3 py-2 rounded-lg bg-background border border-input text-foreground focus:outline-none focus:ring-2 focus:ring-ring transition-colors";

  return (
    <div className="space-y-6">
      <div>
        <h2 className="text-2xl sm:text-3xl font-bold tracking-tight text-foreground mb-1 flex items-center gap-2 sm:gap-3">
          <ArrowLeftRight className="w-6 h-6 sm:w-8 sm:h-8 text-primary" />
          Shift Swaps
        </h2>
        <p className="text-sm sm:text-base text-muted-foreground">Request shift swaps with teammates in your department.</p>
      </div>

      {error && (
        <div className="p-3 rounded-xl bg-destructive/10 border border-destructive/20 text-destructive text-sm">{error}</div>
      )}

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
              <Button className="w-full gap-2" onClick={() => createSwap.mutate()}
                disabled={createSwap.isPending || !targetEmployeeId || !reason}>
                <Send className="w-4 h-4" /> Send Swap Request
              </Button>
            </CardFooter>
          </Card>
        </div>

        {/* ── Lists ── */}
        <div className="lg:col-span-2 space-y-6">
          {/* Pending for me */}
          {pendingForMe?.length > 0 && (
            <div className="space-y-3">
              <h3 className="text-lg font-semibold text-foreground flex items-center gap-2">
                <UserCircle className="w-5 h-5 text-amber-500" />
                Requests For You
              </h3>
              {pendingForMe.map((swap: any) => (
                <Card key={swap.id}>
                  <CardContent className="p-4">
                    <div className="flex flex-col sm:flex-row items-start sm:items-center justify-between gap-3">
                      <div>
                        <p className="text-foreground font-medium">
                          {swap.requester_name || `Employee #${swap.requester_id?.slice(0, 8)}`}
                        </p>
                        <p className="text-sm text-muted-foreground mt-0.5">
                          {format(new Date(swap.shift_date), 'MMM d, yyyy')}
                          {swap.reason && ` · "${swap.reason}"`}
                        </p>
                      </div>
                      <div className="flex gap-2">
                        <Button size="sm" variant="outline" className="text-destructive border-destructive/30 hover:bg-destructive/10 gap-1"
                          onClick={() => respondSwap.mutate({ swapId: swap.id, accept: false })} disabled={respondSwap.isPending}>
                          <XCircle className="w-4 h-4" /> Decline
                        </Button>
                        <Button size="sm" className="gap-1"
                          onClick={() => respondSwap.mutate({ swapId: swap.id, accept: true })} disabled={respondSwap.isPending}>
                          <CheckCircle2 className="w-4 h-4" /> Accept
                        </Button>
                      </div>
                    </div>
                  </CardContent>
                </Card>
              ))}
            </div>
          )}

          {/* My swaps */}
          <div className="space-y-3">
            <h3 className="text-lg font-semibold text-foreground">Your Swap History</h3>
            {isLoading ? (
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
                          {format(new Date(swap.shift_date), 'MMM d, yyyy')}
                        </p>
                      </div>
                    </div>
                    <span className={`px-2 py-1 rounded-md text-xs font-medium uppercase tracking-wider ${
                      swap.status === 'approved' || swap.status === 'completed' ? 'bg-emerald-500/10 text-emerald-500 border border-emerald-500/20' :
                      swap.status === 'rejected' ? 'bg-destructive/10 text-destructive border border-destructive/20' :
                      'bg-amber-500/10 text-amber-500 border border-amber-500/20'
                    }`}>
                      {swap.status}
                    </span>
                  </CardContent>
                </Card>
              ))
            ) : (
              <Card className="border-dashed">
                <CardContent className="flex flex-col items-center justify-center p-12 text-muted-foreground">
                  <ArrowLeftRight className="w-12 h-12 mb-4 opacity-20" />
                  <p>No swap requests yet.</p>
                </CardContent>
              </Card>
            )}
          </div>
        </div>
      </div>
    </div>
  );
};
