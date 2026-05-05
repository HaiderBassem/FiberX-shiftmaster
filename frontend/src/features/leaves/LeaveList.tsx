import { useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import api from '@/lib/api';
import { Card, CardContent, CardHeader, CardTitle, CardFooter } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { CalendarOff, Send, Clock, CheckCircle2, XCircle } from 'lucide-react';
import { format } from 'date-fns';
import { fmtDate } from '@/lib/dateUtils';

export const LeaveList = () => {
  const [leaveType, setLeaveType] = useState('annual');
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

  const submitMutation = useMutation({
    mutationFn: async () => {
      setError(null);
      const payload: any = { leave_type: leaveType, start_date: startDate, end_date: endDate, reason };
      if (leaveType === 'hourly') {
        payload.start_time = startTime;
        payload.end_time = endTime;
      }
      await api.post('/leaves', payload);
    },
    onSuccess: () => { queryClient.invalidateQueries({ queryKey: ['leaves'] }); setReason(''); setStartTime(''); setEndTime(''); },
    onError: (err: any) => setError(err?.response?.data?.error || err?.message || 'Failed to submit leave'),
  });

  const isHourly = leaveType === 'hourly';
  const canSubmit = reason && (isHourly ? (startTime && endTime) : true);

  const getStatusIcon = (status: string) => {
    switch(status) {
      case 'approved_by_manager': return <CheckCircle2 className="w-5 h-5 text-emerald-500" />;
      case 'approved_by_team_leader': return <CheckCircle2 className="w-5 h-5 text-blue-500" />;
      case 'rejected': return <XCircle className="w-5 h-5 text-destructive" />;
      default: return <Clock className="w-5 h-5 text-amber-500" />;
    }
  };

  const getStatusLabel = (status: string) => {
    switch(status) {
      case 'approved_by_manager': return 'Approved';
      case 'approved_by_team_leader': return 'TL Approved';
      case 'rejected': return 'Rejected';
      case 'pending': return 'Pending';
      default: return status || 'Pending';
    }
  };

  const selectClass = "w-full h-10 px-3 py-2 rounded-lg bg-background border border-input text-foreground focus:outline-none focus:ring-2 focus:ring-ring transition-colors";

  return (
    <div className="space-y-8">
      <div>
        <h2 className="text-2xl sm:text-3xl font-bold tracking-tight text-foreground mb-2 flex items-center gap-2 sm:gap-3">
          <CalendarOff className="w-6 h-6 sm:w-8 sm:h-8 text-primary" />
          Leave Management
        </h2>
        <p className="text-muted-foreground">Request time off and track your leave balances.</p>
      </div>

      {error && (
        <div className="p-3 rounded-xl bg-destructive/10 border border-destructive/20 text-destructive text-sm">{error}</div>
      )}

      <div className="grid lg:grid-cols-3 gap-6">
        <div className="lg:col-span-1">
          <Card className="sticky top-24">
            <CardHeader>
              <CardTitle className="flex items-center gap-2">
                <CalendarOff className="w-5 h-5 text-primary" />
                New Request
              </CardTitle>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="space-y-2">
                <Label>Leave Type</Label>
                <select className={selectClass} value={leaveType} onChange={(e) => setLeaveType(e.target.value)}>
                  <option value="annual">Annual Leave</option>
                  <option value="sick">Sick Leave</option>
                  <option value="unpaid">Unpaid Leave</option>
                  <option value="hourly">Hourly Leave</option>
                </select>
              </div>
              <div className="grid grid-cols-2 gap-4">
                <div className="space-y-2">
                  <Label>{isHourly ? 'Date' : 'Start Date'}</Label>
                  <Input type="date" value={startDate} onChange={(e: React.ChangeEvent<HTMLInputElement>) => {
                    setStartDate(e.target.value);
                    if (isHourly) setEndDate(e.target.value);
                  }} />
                </div>
                {!isHourly && (
                  <div className="space-y-2">
                    <Label>End Date</Label>
                    <Input type="date" value={endDate} onChange={(e: React.ChangeEvent<HTMLInputElement>) => setEndDate(e.target.value)} />
                  </div>
                )}
              </div>
              {isHourly && (
                <div className="grid grid-cols-2 gap-4">
                  <div className="space-y-2">
                    <Label>From Time</Label>
                    <Input type="time" value={startTime} onChange={(e: React.ChangeEvent<HTMLInputElement>) => setStartTime(e.target.value)} />
                  </div>
                  <div className="space-y-2">
                    <Label>To Time</Label>
                    <Input type="time" value={endTime} onChange={(e: React.ChangeEvent<HTMLInputElement>) => setEndTime(e.target.value)} />
                  </div>
                </div>
              )}
              <div className="space-y-2">
                <Label>Reason</Label>
                <textarea 
                  className="w-full h-24 px-3 py-2 rounded-lg bg-background border border-input text-foreground focus:outline-none focus:ring-2 focus:ring-ring resize-none text-sm transition-colors"
                  placeholder="Please provide a reason..."
                  value={reason}
                  onChange={(e) => setReason(e.target.value)}
                />
              </div>
            </CardContent>
            <CardFooter>
              <Button className="w-full gap-2" onClick={() => submitMutation.mutate()} disabled={submitMutation.isPending || !canSubmit}>
                <Send className="w-4 h-4" /> Submit Request
              </Button>
            </CardFooter>
          </Card>
        </div>

        <div className="lg:col-span-2 space-y-4">
          <h3 className="text-xl font-semibold text-foreground mb-4">Your Request History</h3>
          
          {isLoading ? (
            <div className="space-y-4">
              {[1, 2].map((i) => <Card key={i} className="animate-pulse h-24" />)}
            </div>
          ) : (
            <div className="space-y-4">
              {leaves?.map((leave: any) => (
                <Card key={leave.id}>
                  <CardContent className="p-4 flex items-center justify-between">
                    <div className="flex gap-4 items-center">
                      <div className="h-10 w-10 rounded-full bg-muted flex items-center justify-center">
                        {getStatusIcon(leave.status)}
                      </div>
                      <div>
                        <div className="font-medium text-foreground capitalize">{leave.leave_type} Leave</div>
                        <div className="text-sm text-muted-foreground mt-1">
                          {leave.leave_type === 'hourly' ? (
                            <>
                              {fmtDate(leave.start_date)}
                              {leave.start_time && leave.end_time && (
                                <span className="ml-1">· {leave.start_time} → {leave.end_time}</span>
                              )}
                            </>
                          ) : (
                            <>
                              {fmtDate(leave.start_date)} - {fmtDate(leave.end_date)}
                            </>
                          )}
                        </div>
                      </div>
                    </div>
                    <div className="text-right flex flex-col items-end">
                      <span className={`inline-flex px-2 py-1 rounded-md text-xs font-medium uppercase tracking-wider ${
                        leave.status === 'approved_by_manager' ? 'bg-emerald-500/10 text-emerald-500 border border-emerald-500/20' :
                        leave.status === 'approved_by_team_leader' ? 'bg-blue-500/10 text-blue-500 border border-blue-500/20' :
                        leave.status === 'rejected' ? 'bg-destructive/10 text-destructive border border-destructive/20' :
                        'bg-amber-500/10 text-amber-500 border border-amber-500/20'
                      }`}>
                        {getStatusLabel(leave.status)}
                      </span>
                    </div>
                  </CardContent>
                </Card>
              ))}
              
              {(!leaves || leaves.length === 0) && (
                <Card className="border-dashed">
                  <CardContent className="flex flex-col items-center justify-center p-12 text-muted-foreground">
                    <CalendarOff className="w-12 h-12 mb-4 opacity-20" />
                    <p>No leave requests found.</p>
                  </CardContent>
                </Card>
              )}
            </div>
          )}
        </div>
      </div>
    </div>
  );
};
