import { useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import api from '@/lib/api';
import { Card, CardContent, CardHeader, CardTitle, CardFooter, CardDescription } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { ArrowLeftRight, Clock, Send } from 'lucide-react';
import { format } from 'date-fns';

export const SwapList = () => {
  const [targetEmployeeId, setTargetEmployeeId] = useState('');
  const [shiftDate, setShiftDate] = useState(format(new Date(), 'yyyy-MM-dd'));
  const [reason, setReason] = useState('');
  const [error, setError] = useState<string | null>(null);
  
  const queryClient = useQueryClient();

  // Fetch my requests
  const { data: mySwaps, isLoading: mySwapsLoading } = useQuery({
    queryKey: ['swaps', 'me'],
    queryFn: async () => {
      const response = await api.get('/swaps/me');
      return response.data?.data || [];
    },
  });

  // Fetch eligible targets
  const { data: eligibleTargets, isLoading: targetsLoading } = useQuery({
    queryKey: ['swaps', 'eligibleTargets', shiftDate],
    queryFn: async () => {
      const response = await api.get(`/swaps/eligible-targets?date=${shiftDate}`);
      return response.data?.data || [];
    },
    enabled: !!shiftDate,
  });

  // Fetch requests pending my answer
  const { data: pendingSwaps, isLoading: pendingLoading } = useQuery({
    queryKey: ['swaps', 'pending'],
    queryFn: async () => {
      const response = await api.get('/swaps/pending');
      return response.data?.data || [];
    },
  });

  const requestSwapMutation = useMutation({
    mutationFn: async () => {
      setError(null);
      await api.post('/swaps', {
        target_employee_id: targetEmployeeId,
        shift_date: shiftDate,
        reason: reason || null
      });
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['swaps'] });
      setReason('');
      setTargetEmployeeId('');
    },
    onError: (err: any) => {
      setError(err?.response?.data?.error || err?.message || 'Failed to request swap');
    },
  });

  const respondMutation = useMutation({
    mutationFn: async ({ id, accepted }: { id: string, accepted: boolean }) => {
      await api.post(`/swaps/${id}/respond`, { accept: accepted });
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['swaps'] });
    },
  });

  const getStatusBadge = (status: string) => {
    switch(status) {
      case 'approved': 
        return <span className="inline-flex px-2 py-1 rounded-md text-xs font-medium uppercase tracking-wider bg-emerald-500/10 text-emerald-400 border border-emerald-500/20">Approved</span>;
      case 'rejected': 
      case 'declined':
        return <span className="inline-flex px-2 py-1 rounded-md text-xs font-medium uppercase tracking-wider bg-red-500/10 text-red-400 border border-red-500/20">{status}</span>;
      case 'employee_accepted':
        return <span className="inline-flex px-2 py-1 rounded-md text-xs font-medium uppercase tracking-wider bg-blue-500/10 text-blue-400 border border-blue-500/20">Awaiting Team Leader</span>;
      default: 
        return <span className="inline-flex px-2 py-1 rounded-md text-xs font-medium uppercase tracking-wider bg-amber-500/10 text-amber-400 border border-amber-500/20">{status || 'Pending'}</span>;
    }
  };

  const personName = (swap: any, key: 'requester' | 'target') => {
    const fromApi = key === 'requester' ? swap.requester_name : swap.target_employee_name;
    if (fromApi) return fromApi;
    return key === 'requester' ? 'Requester' : 'Target';
  };

  return (
    <div className="space-y-8">
      <div>
        <h2 className="text-3xl font-bold tracking-tight text-white mb-2">Shift Swaps</h2>
        <p className="text-zinc-400">Request shift changes with colleagues or respond to requests.</p>
      </div>
      {error && (
        <div className="p-3 rounded-xl bg-red-500/10 border border-red-500/20 text-red-300 text-sm">
          {error}
        </div>
      )}

      <div className="grid lg:grid-cols-3 gap-6">
        {/* Left Column: Request Form */}
        <div className="lg:col-span-1 space-y-6">
          <Card className="bg-zinc-900/80 border-cyan-500/20 shadow-[0_0_30px_rgba(6,182,212,0.05)] sticky top-24">
            <CardHeader>
              <CardTitle className="flex items-center gap-2 text-cyan-100">
                <ArrowLeftRight className="w-5 h-5 text-cyan-400" />
                Request a Swap
              </CardTitle>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="space-y-2">
                <Label className="text-cyan-200">Target Employee</Label>
                <select 
                  className="w-full h-10 px-3 py-2 rounded-md bg-zinc-950/50 border border-cyan-500/30 text-white focus:outline-none focus:ring-2 focus:ring-cyan-500"
                  value={targetEmployeeId}
                  onChange={(e) => setTargetEmployeeId(e.target.value)}
                  disabled={targetsLoading}
                >
                  <option value="" disabled>Select an employee</option>
                  {eligibleTargets?.map((emp: any) => (
                    <option key={emp.id} value={emp.id}>
                      {emp.first_name} {emp.last_name} {emp.is_off ? '(Day Off)' : '(Working)'}
                    </option>
                  ))}
                </select>
                {eligibleTargets?.length === 0 && !targetsLoading && (
                  <p className="text-xs text-red-400 mt-1">No eligible employees found for this date.</p>
                )}
              </div>
              
              <div className="space-y-2">
                <Label>Shift Date</Label>
                <Input 
                  type="date" 
                  value={shiftDate} 
                  onChange={(e: React.ChangeEvent<HTMLInputElement>) => setShiftDate(e.target.value)}
                  className="bg-black/20 border-cyan-500/20"
                />
              </div>

              <div className="space-y-2">
                <Label>Reason</Label>
                <textarea 
                  className="w-full h-20 px-3 py-2 rounded-md bg-zinc-950/50 border border-cyan-500/30 text-white focus:outline-none focus:ring-2 focus:ring-cyan-500 resize-none"
                  placeholder="Reason for swapping..."
                  value={reason}
                  onChange={(e) => setReason(e.target.value)}
                />
              </div>
            </CardContent>
            <CardFooter>
              <Button 
                className="w-full bg-cyan-600 hover:bg-cyan-500 text-white gap-2"
                onClick={() => requestSwapMutation.mutate()}
                disabled={requestSwapMutation.isPending || !targetEmployeeId}
              >
                <Send className="w-4 h-4" />
                Send Request
              </Button>
            </CardFooter>
          </Card>
        </div>

        {/* Right Column: Swap Lists */}
        <div className="lg:col-span-2 space-y-8">
          {/* Pending Action from Me */}
          <div className="space-y-4">
            <h3 className="text-xl font-semibold text-white flex items-center gap-2">
              <span className="relative flex h-3 w-3">
                <span className="animate-ping absolute inline-flex h-full w-full rounded-full bg-amber-400 opacity-75"></span>
                <span className="relative inline-flex rounded-full h-3 w-3 bg-amber-500"></span>
              </span>
              Action Required
            </h3>
            
            {pendingLoading ? (
               <Card className="animate-pulse bg-zinc-900/50 h-24" />
            ) : pendingSwaps?.length > 0 ? (
              pendingSwaps.map((swap: any) => (
                <Card key={swap.id} className="bg-amber-950/20 border-amber-900/50">
                  <CardHeader className="pb-2">
                    <CardTitle className="text-base text-amber-200">
                      Swap request for {format(new Date(swap.shift_date), 'MMM d, yyyy')}
                    </CardTitle>
                    <CardDescription className="text-amber-200/60">From {personName(swap, 'requester')}</CardDescription>
                  </CardHeader>
                  <CardContent>
                    <p className="text-sm text-zinc-300">"{swap.reason}"</p>
                  </CardContent>
                  <CardFooter className="flex justify-end gap-3 pt-0">
                    <Button 
                      variant="outline" 
                      onClick={() => respondMutation.mutate({ id: swap.id, accepted: false })}
                      className="border-red-500/30 text-red-400 hover:bg-red-500/10"
                    >
                      Decline
                    </Button>
                    <Button 
                      onClick={() => respondMutation.mutate({ id: swap.id, accepted: true })}
                      className="bg-emerald-600 hover:bg-emerald-500 text-white"
                    >
                      Accept Swap
                    </Button>
                  </CardFooter>
                </Card>
              ))
            ) : (
              <Card className="bg-zinc-900/20 border-zinc-800/60 border-dashed">
                <CardContent className="flex items-center justify-center p-8 text-zinc-500">
                  No pending swap requests requiring your action.
                </CardContent>
              </Card>
            )}
          </div>

          <div className="h-px bg-zinc-800/60" />

          {/* My Outgoing Requests */}
          <div className="space-y-4">
            <h3 className="text-xl font-semibold text-white">My Requests</h3>
            
            {mySwapsLoading ? (
               <Card className="animate-pulse bg-zinc-900/50 h-24" />
            ) : mySwaps?.length > 0 ? (
              mySwaps.map((swap: any) => (
                <Card key={swap.id} className="bg-zinc-900/40 border-zinc-800/60 flex flex-col sm:flex-row sm:items-center justify-between">
                  <div className="p-4 flex-1">
                    <div className="font-medium text-white">To: {personName(swap, 'target')}</div>
                    <div className="text-sm text-zinc-400 mt-1 flex items-center gap-2">
                      <Clock className="w-4 h-4" /> 
                      {format(new Date(swap.shift_date), 'MMM d, yyyy')}
                    </div>
                  </div>
                  <div className="p-4 sm:border-l border-zinc-800/60 flex items-center justify-end sm:justify-center min-w-[160px]">
                    {getStatusBadge(swap.status)}
                  </div>
                </Card>
              ))
            ) : (
              <Card className="bg-zinc-900/20 border-zinc-800/60 border-dashed">
                <CardContent className="flex flex-col items-center justify-center p-8 text-zinc-500">
                  <ArrowLeftRight className="w-12 h-12 mb-4 opacity-20" />
                  <p>You haven't requested any shift swaps.</p>
                </CardContent>
              </Card>
            )}
          </div>
        </div>
      </div>
    </div>
  );
};
