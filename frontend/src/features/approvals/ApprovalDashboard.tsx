import { useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import api from '@/lib/api';
import { useAuthStore } from '@/store/authStore';
import { Card, CardContent, CardHeader, CardTitle, CardDescription, CardFooter } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { ShieldCheck, CalendarOff, ArrowLeftRight, Users, AlertTriangle, CheckCircle2, XCircle, UserPlus } from 'lucide-react';
import { format } from 'date-fns';

// ─── Coverage Preview Widget ───────────────────────────────────────────────────
const CoveragePreview = ({ shiftId, date }: { shiftId: string; date: string }) => {
  const { data: coverage, isLoading } = useQuery({
    queryKey: ['coverage', shiftId, date],
    queryFn: async () => {
      const res = await api.get(`/leaves/coverage-preview?shift_id=${shiftId}&date=${date}`);
      return res.data?.data;
    },
    enabled: !!shiftId && !!date,
  });

  if (isLoading) return <div className="animate-pulse h-16 bg-zinc-800/50 rounded-lg" />;
  if (!coverage) return null;

  const isLow = coverage.total_working <= 2;

  return (
    <div className={`p-3 rounded-lg border ${isLow ? 'bg-red-950/30 border-red-500/30' : 'bg-emerald-950/20 border-emerald-500/20'}`}>
      <div className="flex items-center gap-2 mb-2">
        {isLow ? <AlertTriangle className="w-4 h-4 text-red-400" /> : <Users className="w-4 h-4 text-emerald-400" />}
        <span className={`text-xs font-semibold uppercase tracking-wider ${isLow ? 'text-red-400' : 'text-emerald-400'}`}>
          Shift Coverage
        </span>
      </div>
      <div className="grid grid-cols-4 gap-2 text-center">
        <div>
          <div className="text-lg font-bold text-white">{coverage.total_assigned}</div>
          <div className="text-[10px] text-zinc-500 uppercase">Total</div>
        </div>
        <div>
          <div className="text-lg font-bold text-emerald-400">{coverage.total_working}</div>
          <div className="text-[10px] text-zinc-500 uppercase">Working</div>
        </div>
        <div>
          <div className="text-lg font-bold text-amber-400">{coverage.total_off}</div>
          <div className="text-[10px] text-zinc-500 uppercase">Off</div>
        </div>
        <div>
          <div className="text-lg font-bold text-red-400">{coverage.total_on_leave}</div>
          <div className="text-[10px] text-zinc-500 uppercase">On Leave</div>
        </div>
      </div>
      {isLow && (
        <p className="text-xs text-red-300 mt-2 flex items-center gap-1">
          <AlertTriangle className="w-3 h-3" /> Warning: Low staffing! Approving may cause shortages.
        </p>
      )}
    </div>
  );
};

// ─── Quick Replacement Select ──────────────────────────────────────────────────
const QuickReplacementSelect = ({ date, onSelect }: { date: string; onSelect: (empId: string) => void }) => {
  const { data: replacements, isLoading } = useQuery({
    queryKey: ['replacements', date],
    queryFn: async () => {
      const res = await api.get(`/schedules/replacements?date=${date}`);
      return res.data?.data || [];
    },
    enabled: !!date,
  });

  return (
    <div className="space-y-2">
      <Label className="text-emerald-200 flex items-center gap-1">
        <UserPlus className="w-4 h-4" /> Quick Replacement
      </Label>
      <select
        className="w-full h-10 px-3 py-2 rounded-md bg-zinc-950/50 border border-emerald-500/30 text-white focus:outline-none focus:ring-2 focus:ring-emerald-500"
        onChange={(e) => onSelect(e.target.value)}
        disabled={isLoading}
      >
        <option value="">Select a replacement (optional)</option>
        {replacements?.map((emp: any) => (
          <option key={emp.id} value={emp.id}>
            {emp.first_name} {emp.last_name} — {emp.employee_code}
          </option>
        ))}
      </select>
      {replacements?.length === 0 && !isLoading && (
        <p className="text-xs text-zinc-500">No available replacements for this date.</p>
      )}
    </div>
  );
};

// ─── Main Approval Dashboard ───────────────────────────────────────────────────
export const ApprovalDashboard = () => {
  const { user } = useAuthStore();
  const queryClient = useQueryClient();
  const [expandedLeave, setExpandedLeave] = useState<string | null>(null);
  const [rejectReason, setRejectReason] = useState('');
  const [swapError, setSwapError] = useState<string | null>(null);

  // Pending Leaves
  const { data: pendingLeaves, isLoading: leavesLoading } = useQuery({
    queryKey: ['leaves', 'pending'],
    queryFn: async () => {
      const res = await api.get('/leaves/pending');
      return res.data?.data || [];
    },
  });

  // Pending Swaps
  const { data: pendingSwaps, isLoading: swapsLoading } = useQuery({
    queryKey: ['swaps', 'pending', 'manager'],
    queryFn: async () => {
      const res = await api.get('/swaps/pending/manager');
      return res.data?.data || [];
    },
    // Phase 3: swap approvals are team_leader only (skip manager).
    enabled: user?.role === 'team_leader',
  });

  // Leave approval
  const approveLeave = useMutation({
    mutationFn: async (leaveId: string) => {
      const endpoint = user?.role === 'team_leader'
        ? `/leaves/${leaveId}/approve/team-leader`
        : `/leaves/${leaveId}/approve/manager`;
      await api.post(endpoint);
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['leaves'] });
      setExpandedLeave(null);
    },
  });

  const rejectLeave = useMutation({
    mutationFn: async (leaveId: string) => {
      await api.post(`/leaves/${leaveId}/reject`, { reason: rejectReason });
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['leaves'] });
      setRejectReason('');
      setExpandedLeave(null);
    },
  });

  // Swap approval
  const approveSwap = useMutation({
    mutationFn: async (swapId: string) => {
      setSwapError(null);
      await api.post(`/swaps/${swapId}/approve`);
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['swaps'] });
      setSwapError(null);
    },
    onError: (err: any) => {
      setSwapError(err?.response?.data?.error || err?.message || 'Failed to approve swap');
    },
  });

  const rejectSwap = useMutation({
    mutationFn: async (swapId: string) => {
      setSwapError(null);
      await api.post(`/swaps/${swapId}/reject`, { reason: '' });
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['swaps'] });
      setSwapError(null);
    },
    onError: (err: any) => {
      setSwapError(err?.response?.data?.error || err?.message || 'Failed to reject swap');
    },
  });

  return (
    <div className="space-y-8">
      <div>
        <h2 className="text-3xl font-bold tracking-tight text-white mb-2 flex items-center gap-3">
          <ShieldCheck className="w-8 h-8 text-blue-400" />
          Approval Center
        </h2>
        <p className="text-zinc-400">Review and manage pending leave and swap requests.</p>
      </div>
      {swapError && (
        <div className="p-3 rounded-xl bg-red-500/10 border border-red-500/20 text-red-300 text-sm">
          {swapError}
        </div>
      )}

      <div className="grid lg:grid-cols-2 gap-8">
        {/* ── Leave Requests ─────────────────────────────────────────── */}
        <div className="space-y-4">
          <h3 className="text-xl font-semibold text-white flex items-center gap-2">
            <CalendarOff className="w-5 h-5 text-rose-400" />
            Leave Requests
            {pendingLeaves?.length > 0 && (
              <span className="ml-2 px-2 py-0.5 rounded-full text-xs font-bold bg-rose-500/20 text-rose-400 border border-rose-500/30">
                {pendingLeaves.length}
              </span>
            )}
          </h3>

          {leavesLoading ? (
            <div className="space-y-3">
              {[1,2].map(i => <Card key={i} className="animate-pulse bg-zinc-900/50 h-32" />)}
            </div>
          ) : pendingLeaves?.length > 0 ? (
            pendingLeaves.map((leave: any) => (
              <Card key={leave.id} className="bg-zinc-900/60 border-zinc-800/60 overflow-hidden">
                <CardHeader className="pb-2">
                  <CardTitle className="text-base text-rose-200">
                    {leave.leave_type?.charAt(0).toUpperCase() + leave.leave_type?.slice(1)} Leave
                  </CardTitle>
                  <CardDescription className="text-zinc-400">
                    Employee #{leave.employee_id?.slice(0, 8)} · {format(new Date(leave.start_date), 'MMM d')} → {format(new Date(leave.end_date), 'MMM d, yyyy')}
                  </CardDescription>
                </CardHeader>
                <CardContent className="space-y-3">
                  {leave.reason && <p className="text-sm text-zinc-300 italic">"{leave.reason}"</p>}

                  {/* Coverage Preview - always visible */}
                  {leave.shift_id && (
                    <CoveragePreview shiftId={leave.shift_id} date={leave.start_date?.split('T')[0]} />
                  )}

                  {/* Expanded: Reject reason + Quick replacement */}
                  {expandedLeave === leave.id && (
                    <div className="space-y-3 pt-2 border-t border-zinc-800/60">
                      <QuickReplacementSelect
                        date={leave.start_date?.split('T')[0]}
                        onSelect={(empId) => console.log('Replacement selected:', empId)}
                      />
                      <div className="space-y-2">
                        <Label className="text-red-300">Rejection Reason (if rejecting)</Label>
                        <Input
                          value={rejectReason}
                          onChange={(e: React.ChangeEvent<HTMLInputElement>) => setRejectReason(e.target.value)}
                          placeholder="Enter reason for rejection..."
                          className="bg-black/20 border-red-500/20"
                        />
                      </div>
                    </div>
                  )}
                </CardContent>
                <CardFooter className="flex justify-end gap-3 pt-0">
                  {expandedLeave !== leave.id ? (
                    <Button
                      variant="outline"
                      className="border-blue-500/30 text-blue-400 hover:bg-blue-500/10"
                      onClick={() => setExpandedLeave(leave.id)}
                    >
                      Review Details
                    </Button>
                  ) : (
                    <>
                      <Button
                        variant="outline"
                        className="border-red-500/30 text-red-400 hover:bg-red-500/10 gap-1"
                        onClick={() => rejectLeave.mutate(leave.id)}
                        disabled={!rejectReason || rejectLeave.isPending}
                      >
                        <XCircle className="w-4 h-4" /> Reject
                      </Button>
                      <Button
                        className="bg-emerald-600 hover:bg-emerald-500 text-white gap-1"
                        onClick={() => approveLeave.mutate(leave.id)}
                        disabled={approveLeave.isPending}
                      >
                        <CheckCircle2 className="w-4 h-4" /> Approve
                      </Button>
                    </>
                  )}
                </CardFooter>
              </Card>
            ))
          ) : (
            <Card className="bg-zinc-900/20 border-zinc-800/60 border-dashed">
              <CardContent className="flex flex-col items-center justify-center p-10 text-zinc-500">
                <CalendarOff className="w-10 h-10 mb-3 opacity-20" />
                <p>No pending leave requests.</p>
              </CardContent>
            </Card>
          )}
        </div>

        {/* ── Swap Requests ─────────────────────────────────────────── */}
        <div className="space-y-4">
          <h3 className="text-xl font-semibold text-white flex items-center gap-2">
            <ArrowLeftRight className="w-5 h-5 text-cyan-400" />
            Swap Requests
            {pendingSwaps?.length > 0 && (
              <span className="ml-2 px-2 py-0.5 rounded-full text-xs font-bold bg-cyan-500/20 text-cyan-400 border border-cyan-500/30">
                {pendingSwaps.length}
              </span>
            )}
          </h3>

          {swapsLoading ? (
            <div className="space-y-3">
              {[1,2].map(i => <Card key={i} className="animate-pulse bg-zinc-900/50 h-24" />)}
            </div>
          ) : pendingSwaps?.length > 0 ? (
            pendingSwaps.map((swap: any) => (
              <Card key={swap.id} className="bg-zinc-900/60 border-zinc-800/60">
                <CardHeader className="pb-2">
                  <CardTitle className="text-base text-cyan-200">
                    Shift Swap · {format(new Date(swap.shift_date), 'MMM d, yyyy')}
                  </CardTitle>
                  <CardDescription className="text-zinc-400">
                    Requester {swap.requester_name || `#${swap.requester_id?.slice(0, 8)}`} ↔ Target {swap.target_employee_name || `#${swap.target_employee_id?.slice(0, 8)}`}
                  </CardDescription>
                </CardHeader>
                <CardContent>
                  {swap.reason && <p className="text-sm text-zinc-300 italic">"{swap.reason}"</p>}
                  {swap.shift_id && (
                    <div className="mt-3">
                      <CoveragePreview shiftId={swap.shift_id} date={swap.shift_date?.split('T')[0]} />
                    </div>
                  )}
                </CardContent>
                <CardFooter className="flex justify-end gap-3 pt-0">
                  <Button
                    variant="outline"
                    className="border-red-500/30 text-red-400 hover:bg-red-500/10 gap-1"
                    onClick={() => rejectSwap.mutate(swap.id)}
                    disabled={rejectSwap.isPending}
                  >
                    <XCircle className="w-4 h-4" /> Reject
                  </Button>
                  <Button
                    className="bg-emerald-600 hover:bg-emerald-500 text-white gap-1"
                    onClick={() => approveSwap.mutate(swap.id)}
                    disabled={approveSwap.isPending}
                  >
                    <CheckCircle2 className="w-4 h-4" /> Approve Swap
                  </Button>
                </CardFooter>
              </Card>
            ))
          ) : (
            <Card className="bg-zinc-900/20 border-zinc-800/60 border-dashed">
              <CardContent className="flex flex-col items-center justify-center p-10 text-zinc-500">
                <ArrowLeftRight className="w-10 h-10 mb-3 opacity-20" />
                <p>No pending swap requests.</p>
              </CardContent>
            </Card>
          )}
        </div>
      </div>
    </div>
  );
};
