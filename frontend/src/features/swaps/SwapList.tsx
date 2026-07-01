import { useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import api from '@/lib/api';
import { useAuthStore } from '@/store/authStore';
import { Card, CardContent } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import {
  ArrowLeftRight, Clock, CheckCircle2, XCircle, UserCircle, Inbox, Plus
} from 'lucide-react';
import { format } from 'date-fns';
import { fmtDate } from '@/lib/dateUtils';
import { motion, AnimatePresence } from 'framer-motion';
import { SwapRequestModal } from './SwapRequestModal';

export const SwapList = () => {
  const queryClient = useQueryClient();
  const { user } = useAuthStore();
  
  // Modal State
  const [isModalOpen, setIsModalOpen] = useState(false);
  
  // Form State
  const [targetEmployeeId, setTargetEmployeeId] = useState('');
  const [shiftDate, setShiftDate] = useState(format(new Date(), 'yyyy-MM-dd'));
  const [reason, setReason] = useState('');
  const [error, setError] = useState<string | null>(null);
  const [success, setSuccess] = useState<string | null>(null);

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
      setIsModalOpen(false);
    },
    onError: (err: any) => setError(err?.response?.data?.error || err?.message || 'Failed to create swap'),
  });

  const respondSwap = useMutation({
    mutationFn: async ({ swapId, accept }: { swapId: string; accept: boolean }) => {
      await api.post(`/swaps/${swapId}/respond`, { accept });
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['swaps'] });
      setSuccess("Successfully responded to swap request");
      setTimeout(() => setSuccess(null), 3000);
    },
    onError: (err: any) => setError(err?.response?.data?.error || err?.message || 'Failed to respond'),
  });

  const cancelSwapMutation = useMutation({
    mutationFn: async (id: string) => { await api.post(`/swaps/${id}/cancel`); },
    onSuccess: () => { 
      queryClient.invalidateQueries({ queryKey: ['swaps'] }); 
    },
    onError: (err: any) => alert(err?.response?.data?.error || err?.message || 'Failed to cancel swap request'),
  });

  const getStatusBadge = (status: string) => {
    if (status === 'approved' || status === 'completed') {
      return <span className="inline-flex px-3 py-1.5 rounded-full text-xs font-bold uppercase tracking-widest shadow-sm bg-emerald-500/15 text-emerald-600 border border-emerald-500/30">{status}</span>;
    }
    if (status === 'rejected') {
      return <span className="inline-flex px-3 py-1.5 rounded-full text-xs font-bold uppercase tracking-widest shadow-sm bg-destructive/15 text-destructive border border-destructive/30">{status}</span>;
    }
    if (status === 'employee_accepted') {
      return <span className="inline-flex px-3 py-1.5 rounded-full text-xs font-bold uppercase tracking-widest shadow-sm bg-blue-500/15 text-blue-600 border border-blue-500/30">Awaiting TL</span>;
    }
    return <span className="inline-flex px-3 py-1.5 rounded-full text-xs font-bold uppercase tracking-widest shadow-sm bg-amber-500/15 text-amber-600 border border-amber-500/30">{status}</span>;
  };

  const getStatusIcon = (status: string) => {
    if (status === 'approved' || status === 'completed') return <CheckCircle2 className="w-6 h-6 text-emerald-500" />;
    if (status === 'rejected') return <XCircle className="w-6 h-6 text-destructive" />;
    if (status === 'employee_accepted') return <Clock className="w-6 h-6 text-blue-500" />;
    return <Clock className="w-6 h-6 text-amber-500" />;
  };

  const incomingCount = pendingForMe?.length || 0;
  const canSubmit = targetEmployeeId && shiftDate && reason;

  const containerVariants: any = {
    hidden: { opacity: 0 },
    show: { opacity: 1, transition: { staggerChildren: 0.1 } }
  };

  const itemVariants: any = {
    hidden: { opacity: 0, y: 20 },
    show: { opacity: 1, y: 0, transition: { type: "spring", stiffness: 300, damping: 24 } }
  };

  return (
    <div className="space-y-6 pb-24">
      <div className="flex items-center justify-between">
        <h3 className="text-2xl font-bold text-foreground tracking-tight">Shift Swaps</h3>
        <Button 
          onClick={() => setIsModalOpen(true)} 
          className="gap-2 shadow-lg shadow-primary/20 hover:scale-105 transition-transform"
        >
          <Plus className="w-5 h-5" /> Request Swap
        </Button>
      </div>

      <AnimatePresence>
        {incomingCount > 0 && (
          <motion.div
            initial={{ opacity: 0, height: 0, marginBottom: 0 }}
            animate={{ opacity: 1, height: 'auto', marginBottom: 24 }}
            exit={{ opacity: 0, height: 0, marginBottom: 0 }}
            className="overflow-hidden"
          >
            <div className="rounded-2xl border-2 bg-amber-500/5 border-amber-500/30 p-5 shadow-inner">
              <div className="flex items-center gap-3 mb-4">
                <Inbox className="w-6 h-6 text-amber-500" />
                <h3 className="text-lg font-bold text-foreground flex items-center gap-2">
                  Incoming Requests
                  <span className="inline-flex items-center justify-center w-6 h-6 rounded-full text-xs font-bold bg-amber-500 text-white shadow-sm">
                    {incomingCount}
                  </span>
                </h3>
              </div>
  
                {error && (
                  <div className="mb-4 p-3 rounded-xl bg-destructive/10 border border-destructive/20 text-destructive text-sm font-medium">{error}</div>
                )}
                {success && (
                  <div className="mb-4 p-3 rounded-xl bg-emerald-500/10 border border-emerald-500/20 text-emerald-500 text-sm font-medium">{success}</div>
                )}

              {pendingLoading ? (
                <div className="space-y-3">
                  {[1, 2].map(i => <div key={i} className="animate-pulse h-20 bg-muted/50 rounded-xl" />)}
                </div>
              ) : (
                <div className="space-y-3">
                  {pendingForMe!.map((swap: any) => (
                  <motion.div 
                    key={swap.id} 
                    initial={{ opacity: 0, x: -20 }}
                    animate={{ opacity: 1, x: 0 }}
                    className="flex flex-col sm:flex-row items-start sm:items-center justify-between gap-4 p-4 rounded-xl bg-background border border-border shadow-sm"
                  >
                    <div className="flex items-center gap-4">
                      <div className="h-12 w-12 rounded-xl bg-amber-500/10 flex items-center justify-center flex-shrink-0 overflow-hidden">
                        {swap.requester_profile_image ? (
                          <img src={swap.requester_profile_image} alt="" className="w-full h-full object-cover" />
                        ) : (
                          <UserCircle className="w-6 h-6 text-amber-500" />
                        )}
                      </div>
                      <div>
                        <p className="text-foreground font-semibold text-lg tracking-tight">
                          {swap.requester_name || `Employee #${swap.requester_id?.slice(0, 8)}`}
                        </p>
                        <p className="text-sm text-muted-foreground flex items-center gap-2">
                          <Clock className="w-4 h-4 opacity-70" />
                          {fmtDate(swap.shift_date)}
                        </p>
                        {swap.reason && <p className="text-sm italic text-muted-foreground mt-1">"{swap.reason}"</p>}
                      </div>
                    </div>
                    <div className="flex gap-2 w-full sm:w-auto">
                      <Button
                        size="sm"
                        variant="outline"
                        className="flex-1 sm:flex-none border-destructive/30 text-destructive hover:bg-destructive hover:text-white transition-colors rounded-xl"
                        onClick={() => respondSwap.mutate({ swapId: swap.id, accept: false })}
                        disabled={respondSwap.isPending}
                      >
                        Decline
                      </Button>
                      <Button
                        size="sm"
                        className="flex-1 sm:flex-none bg-emerald-500 hover:bg-emerald-600 text-white shadow-md shadow-emerald-500/20 rounded-xl"
                        onClick={() => respondSwap.mutate({ swapId: swap.id, accept: true })}
                        disabled={respondSwap.isPending}
                      >
                        Accept
                      </Button>
                    </div>
                  </motion.div>
                ))}
              </div>
              )}
            </div>
          </motion.div>
        )}
      </AnimatePresence>

      <div className="space-y-4">
        <h3 className="text-xl font-semibold text-foreground tracking-tight">Your Swap History</h3>
        
        {mySwapsLoading ? (
          <div className="space-y-4">
            {[1, 2, 3].map((i) => <Card key={i} className="animate-pulse h-28 rounded-2xl bg-muted/50 border-transparent" />)}
          </div>
        ) : (
          <motion.div 
            variants={containerVariants}
            initial="hidden"
            animate="show"
            className="space-y-4"
          >
            {mySwaps?.map((swap: any) => (
              <motion.div key={swap.id} variants={itemVariants}>
                <Card className="rounded-2xl border-white/5 hover:border-primary/20 hover:shadow-xl hover:shadow-primary/5 transition-all bg-card/50 backdrop-blur-sm overflow-hidden">
                  <CardContent className="p-5 flex flex-col sm:flex-row items-start sm:items-center justify-between gap-4">
                    <div className="flex gap-4 items-center">
                      <div className={`h-14 w-14 rounded-2xl flex items-center justify-center shrink-0 shadow-inner ${
                        swap.status === 'approved' || swap.status === 'completed' ? 'bg-emerald-500/10' :
                        swap.status === 'rejected' ? 'bg-destructive/10' :
                        swap.status === 'employee_accepted' ? 'bg-blue-500/10' :
                        'bg-amber-500/10'
                      }`}>
                        {getStatusIcon(swap.status)}
                      </div>
                      <div>
                        <div className="font-semibold text-lg text-foreground capitalize tracking-tight flex items-center gap-2">
                          <ArrowLeftRight className="w-4 h-4 text-primary" />
                          Swap with
                          {swap.target_profile_image && (
                            <img src={swap.target_profile_image} alt="" className="w-5 h-5 rounded-full object-cover" />
                          )}
                          {swap.target_employee_name || 'Colleague'}
                        </div>
                        <div className="text-sm text-muted-foreground mt-1 flex items-center gap-2">
                          <Clock className="w-4 h-4 opacity-70" />
                          {fmtDate(swap.shift_date)}
                        </div>
                      </div>
                    </div>
                    
                    <div className="flex items-center self-end sm:self-auto gap-3">
                        {swap.status === 'pending' && (
                          <Button 
                            variant="outline" 
                            size="sm" 
                            className="text-destructive border-destructive/30 hover:bg-destructive/10 h-8 text-xs px-2"
                            onClick={() => {
                              if (confirm("Are you sure you want to cancel this swap request?")) {
                                cancelSwapMutation.mutate(swap.id);
                              }
                            }}
                            disabled={cancelSwapMutation.isPending}
                          >
                            <XCircle className="w-3 h-3 mr-1" /> Cancel Request
                          </Button>
                        )}
                        {getStatusBadge(swap.status)}
                      </div>
                  </CardContent>
                </Card>
              </motion.div>
            ))}
            
            {(!mySwaps || mySwaps.length === 0) && (
              <motion.div variants={itemVariants}>
                <Card className="border-dashed border-2 bg-transparent rounded-3xl">
                  <CardContent className="flex flex-col items-center justify-center py-20 text-muted-foreground">
                    <ArrowLeftRight className="w-16 h-16 mb-6 opacity-20" />
                    <p className="text-lg font-medium text-foreground/70">No swap requests found</p>
                    <p className="text-sm opacity-60">You haven't requested any shift swaps yet.</p>
                  </CardContent>
                </Card>
              </motion.div>
            )}
          </motion.div>
        )}
      </div>

      <SwapRequestModal 
        isOpen={isModalOpen}
        onClose={() => setIsModalOpen(false)}
        employees={swapEligible}
        targetEmployeeId={targetEmployeeId}
        setTargetEmployeeId={setTargetEmployeeId}
        shiftDate={shiftDate}
        setShiftDate={setShiftDate}
        reason={reason}
        setReason={setReason}
        onSubmit={() => createSwap.mutate()}
        isPending={createSwap.isPending}
        canSubmit={!!canSubmit}
        error={error}
      />
    </div>
  );
};
