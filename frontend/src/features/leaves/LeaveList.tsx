import { useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import api from '@/lib/api';
import { Card, CardContent } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { CalendarOff, Clock, CheckCircle2, XCircle, Plus } from 'lucide-react';
import { format } from 'date-fns';
import { fmtDate } from '@/lib/dateUtils';
import { useAuthStore } from '@/store/authStore';
import { LeaveTypeManager } from './LeaveTypeManager';
import { LeaveRequestModal } from './LeaveRequestModal';
import { motion } from 'framer-motion';

export const LeaveList = () => {
  const { user } = useAuthStore();
  const isAdmin = user?.role === 'admin';
  const [activeTab, setActiveTab] = useState<'requests' | 'types'>('requests');
  
  // Modal State
  const [isModalOpen, setIsModalOpen] = useState(false);
  
  // Form State
  const [leaveTypeId, setLeaveTypeId] = useState('');
  const [startDate, setStartDate] = useState(format(new Date(), 'yyyy-MM-dd'));
  const [endDate, setEndDate] = useState(format(new Date(), 'yyyy-MM-dd'));
  const [reason, setReason] = useState('');
  const [startTime, setStartTime] = useState('');
  const [endTime, setEndTime] = useState('');
  const [error, setError] = useState<string | null>(null);
  
  const queryClient = useQueryClient();

  const { data: leaves, isLoading } = useQuery({
    queryKey: ['leaves', 'me'],
    queryFn: async () => { const response = await api.get('/leaves/me'); return response.data?.data || []; },
  });

  const { data: leaveTypes, isLoading: isLoadingTypes } = useQuery({
    queryKey: ['leave-types', 'active'],
    queryFn: async () => { const response = await api.get('/leave-types?active=true'); return response.data?.data || []; },
  });

  const { data: balances } = useQuery({
    queryKey: ['leave-balances', user?.id],
    queryFn: async () => {
      if (!user?.id) return [];
      const res = await api.get(`/leaves/my-balances?year=${new Date(startDate).getFullYear()}`);
      return res.data?.data || [];
    },
    enabled: !!user?.id && !!startDate,
  });

  const submitMutation = useMutation({
    mutationFn: async () => {
      setError(null);
      const payload: any = { leave_type_id: leaveTypeId || (leaveTypes?.[0]?.id), start_date: startDate, end_date: endDate, reason };
      if (isHourly) {
        payload.start_time = startTime;
        payload.end_time = endTime;
      }
      await api.post('/leaves', payload);
    },
    onSuccess: () => { 
      queryClient.invalidateQueries({ queryKey: ['leaves'] }); 
      setReason(''); setStartTime(''); setEndTime(''); 
      setIsModalOpen(false);
    },
    onError: (err: any) => setError(err?.response?.data?.error || err?.message || 'Failed to submit leave'),
  });

  const cancelLeaveMutation = useMutation({
    mutationFn: async (id: string) => { await api.post(`/leaves/${id}/cancel`); },
    onSuccess: () => { queryClient.invalidateQueries({ queryKey: ['leaves'] }); },
  });

  const selectedType = leaveTypes?.find((t: any) => t.id === (leaveTypeId || leaveTypes?.[0]?.id));
  const isHourly = selectedType?.unit === 'hours';

  // Find applicable balance
  let remainingText = '';
  let hasEnoughBalance = true;
  if (selectedType && balances) {
    const month = selectedType.reset_cycle === 'monthly' ? new Date(startDate).getMonth() + 1 : 0;
    const balance = balances.find((b: any) => b.leave_type_id === selectedType.id && b.month === month);
    
    let pendingAmount = 0;
    if (leaves) {
      leaves.forEach((l: any) => {
        if ((l.status === 'pending' || l.status === 'approved_by_team_leader') && l.leave_type_id === selectedType.id) {
          const lMonth = selectedType.reset_cycle === 'monthly' ? new Date(l.start_date).getMonth() + 1 : 0;
          if (lMonth === month) {
            if (isHourly && l.start_time && l.end_time) {
              const [h1, m1] = l.start_time.split(':').map(Number);
              const [h2, m2] = l.end_time.split(':').map(Number);
              pendingAmount += (h2 - h1) + (m2 - m1) / 60;
            } else if (!isHourly) {
              const d1 = new Date(l.start_date);
              const d2 = new Date(l.end_date);
              const diffTime = Math.abs(d2.getTime() - d1.getTime());
              pendingAmount += Math.ceil(diffTime / (1000 * 60 * 60 * 24)) + 1;
            }
          }
        }
      });
    }

    if (balance) {
      const remainingAmount = balance.allocated_amount - balance.used_amount - pendingAmount;
      remainingText = `${remainingAmount} ${selectedType.unit} remaining`;
      if (selectedType.reset_cycle === 'monthly') {
        remainingText += ` this month`;
      } else {
        remainingText += ` this year`;
      }
      if (remainingAmount <= 0) {
        hasEnoughBalance = false;
        remainingText += " (Insufficient Balance)";
      }
    } else {
      // If no balance record exists for this month/year, they effectively have 0
      hasEnoughBalance = false;
      remainingText = `0 ${selectedType.unit} remaining`;
      if (selectedType.reset_cycle === 'monthly') {
        remainingText += ` this month (Insufficient Balance)`;
      } else {
        remainingText += ` this year (Insufficient Balance)`;
      }
    }
  }

  const canSubmit = !!reason && (isHourly ? !!(startTime && endTime) : true) && hasEnoughBalance;

  const getStatusIcon = (status: string) => {
    switch(status) {
      case 'approved_by_manager': return <CheckCircle2 className="w-6 h-6 text-emerald-500" />;
      case 'approved_by_team_leader': return <CheckCircle2 className="w-6 h-6 text-blue-500" />;
      case 'rejected': return <XCircle className="w-6 h-6 text-destructive" />;
      case 'cancelled': return <XCircle className="w-6 h-6 text-gray-500" />;
      default: return <Clock className="w-6 h-6 text-amber-500" />;
    }
  };

  const getStatusLabel = (status: string) => {
    switch(status) {
      case 'approved_by_manager': return 'Approved';
      case 'approved_by_team_leader': return 'TL Approved';
      case 'rejected': return 'Rejected';
      case 'cancelled': return 'Cancelled';
      case 'pending': return 'Pending';
      default: return status || 'Pending';
    }
  };

  const containerVariants: any = {
    hidden: { opacity: 0 },
    show: {
      opacity: 1,
      transition: { staggerChildren: 0.1 }
    }
  };

  const itemVariants: any = {
    hidden: { opacity: 0, y: 20 },
    show: { opacity: 1, y: 0, transition: { type: "spring", stiffness: 300, damping: 24 } }
  };

  return (
    <div className="space-y-6 pb-24">
      {isAdmin && (
        <div className="flex gap-4 border-b border-border/50 pb-2">
          <button 
            className={`font-medium text-sm transition-colors ${activeTab === 'requests' ? 'text-primary border-b-2 border-primary pb-2 -mb-[10px]' : 'text-muted-foreground hover:text-foreground pb-2 -mb-[10px]'}`} 
            onClick={() => setActiveTab('requests')}
          >
            Leave Requests
          </button>
          <button 
            className={`font-medium text-sm transition-colors ${activeTab === 'types' ? 'text-primary border-b-2 border-primary pb-2 -mb-[10px]' : 'text-muted-foreground hover:text-foreground pb-2 -mb-[10px]'}`} 
            onClick={() => setActiveTab('types')}
          >
            Leave Types
          </button>
        </div>
      )}

      {activeTab === 'types' && isAdmin ? (
        <LeaveTypeManager />
      ) : (
        <>
          <div className="flex items-center justify-between">
            <h3 className="text-2xl font-bold text-foreground tracking-tight">Your Requests</h3>
            <Button 
              onClick={() => setIsModalOpen(true)} 
              className="gap-2 shadow-lg shadow-primary/20 hover:scale-105 transition-transform"
            >
              <Plus className="w-5 h-5" /> Request Leave
            </Button>
          </div>

          {isLoading ? (
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
              {leaves?.map((leave: any) => (
                <motion.div key={leave.id} variants={itemVariants}>
                  <Card className="rounded-2xl border-white/5 hover:border-primary/20 hover:shadow-xl hover:shadow-primary/5 transition-all bg-card/50 backdrop-blur-sm overflow-hidden">
                    <CardContent className="p-5 flex flex-col sm:flex-row items-start sm:items-center justify-between gap-4">
                      <div className="flex gap-4 items-center">
                        <div className={`h-14 w-14 rounded-2xl flex items-center justify-center shrink-0 shadow-inner ${
                          leave.status === 'approved_by_manager' ? 'bg-emerald-500/10' :
                          leave.status === 'approved_by_team_leader' ? 'bg-blue-500/10' :
                          leave.status === 'rejected' ? 'bg-destructive/10' :
                          leave.status === 'cancelled' ? 'bg-gray-500/10' :
                          'bg-amber-500/10'
                        }`}>
                          {getStatusIcon(leave.status)}
                        </div>
                        <div>
                          <div className="font-semibold text-lg text-foreground capitalize tracking-tight">{leave.leave_type_name_en || 'Leave'}</div>
                          <div className="text-sm text-muted-foreground mt-1 flex items-center gap-2">
                            <Clock className="w-4 h-4 opacity-70" />
                            {leave.leave_type_name_en?.toLowerCase() === 'hourly' ? (
                              <>
                                {fmtDate(leave.start_date)}
                                {leave.start_time && leave.end_time && (
                                  <span className="font-medium">({leave.start_time} - {leave.end_time})</span>
                                )}
                              </>
                            ) : (
                              <>
                                {fmtDate(leave.start_date)} <span className="opacity-50">to</span> {fmtDate(leave.end_date)}
                              </>
                            )}
                          </div>
                        </div>
                      </div>
                      
                      <div className="flex items-center self-end sm:self-auto gap-3">
                        {(leave.status === 'pending' || leave.status === 'approved_by_team_leader') && (
                          <Button 
                            variant="outline" 
                            size="sm" 
                            className="text-destructive border-destructive/30 hover:bg-destructive/10 h-8 text-xs px-2"
                            onClick={() => {
                              if (confirm("Are you sure you want to cancel this leave request?")) {
                                cancelLeaveMutation.mutate(leave.id);
                              }
                            }}
                            disabled={cancelLeaveMutation.isPending}
                          >
                            <XCircle className="w-3 h-3 mr-1" /> Cancel Request
                          </Button>
                        )}
                        <span className={`inline-flex px-3 py-1.5 rounded-full text-xs font-bold uppercase tracking-widest shadow-sm ${
                          leave.status === 'approved_by_manager' ? 'bg-emerald-500/15 text-emerald-600 border border-emerald-500/30' :
                          leave.status === 'approved_by_team_leader' ? 'bg-blue-500/15 text-blue-600 border border-blue-500/30' :
                          leave.status === 'rejected' ? 'bg-destructive/15 text-destructive border border-destructive/30' :
                          leave.status === 'cancelled' ? 'bg-gray-500/15 text-gray-500 border border-gray-500/30' :
                          'bg-amber-500/15 text-amber-600 border border-amber-500/30'
                        }`}>
                          {getStatusLabel(leave.status)}
                        </span>
                      </div>
                    </CardContent>
                  </Card>
                </motion.div>
              ))}
              
              {(!leaves || leaves.length === 0) && (
                <motion.div variants={itemVariants}>
                  <Card className="border-dashed border-2 bg-transparent rounded-3xl">
                    <CardContent className="flex flex-col items-center justify-center py-20 text-muted-foreground">
                      <CalendarOff className="w-16 h-16 mb-6 opacity-20" />
                      <p className="text-lg font-medium text-foreground/70">No leave requests found</p>
                      <p className="text-sm opacity-60">You haven't requested any leaves yet.</p>
                    </CardContent>
                  </Card>
                </motion.div>
              )}
            </motion.div>
          )}

          <LeaveRequestModal 
            isOpen={isModalOpen}
            onClose={() => { setIsModalOpen(false); setError(null); }}
            leaveTypes={leaveTypes || []}
            isLoadingTypes={isLoadingTypes}
            leaveTypeId={leaveTypeId}
            setLeaveTypeId={setLeaveTypeId}
            startDate={startDate}
            setStartDate={setStartDate}
            endDate={endDate}
            setEndDate={setEndDate}
            startTime={startTime}
            setStartTime={setStartTime}
            endTime={endTime}
            setEndTime={setEndTime}
            reason={reason}
            setReason={setReason}
            onSubmit={() => submitMutation.mutate()}
            isPending={submitMutation.isPending}
            canSubmit={canSubmit}
            isHourly={isHourly}
            error={error}
            remainingText={remainingText}
          />
        </>
      )}
    </div>
  );
};
