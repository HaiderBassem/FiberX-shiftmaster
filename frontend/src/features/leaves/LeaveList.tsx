import { useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import api from '@/lib/api';
import { Card, CardContent, CardHeader, CardTitle, CardFooter } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { CalendarOff, Send, Clock, CheckCircle2, XCircle } from 'lucide-react';
import { format } from 'date-fns';

export const LeaveList = () => {
  const [leaveType, setLeaveType] = useState('annual');
  const [startDate, setStartDate] = useState(format(new Date(), 'yyyy-MM-dd'));
  const [endDate, setEndDate] = useState(format(new Date(), 'yyyy-MM-dd'));
  const [reason, setReason] = useState('');
  
  const queryClient = useQueryClient();

  const { data: leaves, isLoading } = useQuery({
    queryKey: ['leaves', 'me'],
    queryFn: async () => {
      const response = await api.get('/leaves/me');
      return response.data?.data || [];
    },
  });

  const submitMutation = useMutation({
    mutationFn: async () => {
      await api.post('/leaves', {
        leave_type: leaveType,
        start_date: startDate,
        end_date: endDate,
        reason
      });
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['leaves'] });
      setReason('');
    },
  });

  const getStatusIcon = (status: string) => {
    switch(status) {
      case 'approved': return <CheckCircle2 className="w-5 h-5 text-emerald-400" />;
      case 'rejected': return <XCircle className="w-5 h-5 text-red-400" />;
      default: return <Clock className="w-5 h-5 text-amber-400" />;
    }
  };

  return (
    <div className="space-y-8">
      <div>
        <h2 className="text-3xl font-bold tracking-tight text-white mb-2">Leave Management</h2>
        <p className="text-zinc-400">Request time off and track your leave balances.</p>
      </div>

      <div className="grid lg:grid-cols-3 gap-6">
        <div className="lg:col-span-1">
          <Card className="bg-zinc-900/80 border-rose-500/20 shadow-[0_0_30px_rgba(225,29,72,0.05)] sticky top-24">
            <CardHeader>
              <CardTitle className="flex items-center gap-2 text-rose-100">
                <CalendarOff className="w-5 h-5 text-rose-400" />
                New Request
              </CardTitle>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="space-y-2">
                <Label className="text-rose-200">Leave Type</Label>
                <select 
                  className="w-full h-10 px-3 py-2 rounded-md bg-zinc-950/50 border border-rose-500/30 text-white focus:outline-none focus:ring-2 focus:ring-rose-500"
                  value={leaveType}
                  onChange={(e) => setLeaveType(e.target.value)}
                >
                  <option value="annual">Annual Leave</option>
                  <option value="sick">Sick Leave</option>
                  <option value="unpaid">Unpaid Leave</option>
                </select>
              </div>
              
              <div className="grid grid-cols-2 gap-4">
                <div className="space-y-2">
                  <Label>Start Date</Label>
                  <Input 
                    type="date" 
                    value={startDate} 
                    onChange={(e: React.ChangeEvent<HTMLInputElement>) => setStartDate(e.target.value)}
                    className="bg-black/20 border-rose-500/20"
                  />
                </div>
                <div className="space-y-2">
                  <Label>End Date</Label>
                  <Input 
                    type="date" 
                    value={endDate} 
                    onChange={(e: React.ChangeEvent<HTMLInputElement>) => setEndDate(e.target.value)}
                    className="bg-black/20 border-rose-500/20"
                  />
                </div>
              </div>

              <div className="space-y-2">
                <Label>Reason</Label>
                <textarea 
                  className="w-full h-24 px-3 py-2 rounded-md bg-zinc-950/50 border border-rose-500/30 text-white focus:outline-none focus:ring-2 focus:ring-rose-500 resize-none"
                  placeholder="Please provide a reason..."
                  value={reason}
                  onChange={(e) => setReason(e.target.value)}
                />
              </div>
            </CardContent>
            <CardFooter>
              <Button 
                className="w-full bg-rose-600 hover:bg-rose-500 text-white gap-2"
                onClick={() => submitMutation.mutate()}
                disabled={submitMutation.isPending || !reason}
              >
                <Send className="w-4 h-4" />
                Submit Request
              </Button>
            </CardFooter>
          </Card>
        </div>

        <div className="lg:col-span-2 space-y-4">
          <h3 className="text-xl font-semibold text-white mb-4">Your Request History</h3>
          
          {isLoading ? (
            <div className="space-y-4">
              {[1, 2].map((i) => (
                <Card key={i} className="animate-pulse bg-zinc-900/50 h-24" />
              ))}
            </div>
          ) : (
            <div className="space-y-4">
              {leaves?.map((leave: any) => (
                <Card key={leave.id} className="bg-zinc-900/40 border-zinc-800/60">
                  <CardContent className="p-4 flex items-center justify-between">
                    <div className="flex gap-4 items-center">
                      <div className="h-10 w-10 rounded-full bg-zinc-800 flex items-center justify-center">
                        {getStatusIcon(leave.status)}
                      </div>
                      <div>
                        <div className="font-medium text-white capitalize">{leave.leave_type} Leave</div>
                        <div className="text-sm text-zinc-400 mt-1">
                          {format(new Date(leave.start_date), 'MMM d, yyyy')} - {format(new Date(leave.end_date), 'MMM d, yyyy')}
                        </div>
                      </div>
                    </div>
                    <div className="text-right flex flex-col items-end">
                      <span className={`inline-flex px-2 py-1 rounded-md text-xs font-medium uppercase tracking-wider ${
                        leave.status === 'approved' ? 'bg-emerald-500/10 text-emerald-400 border border-emerald-500/20' :
                        leave.status === 'rejected' ? 'bg-red-500/10 text-red-400 border border-red-500/20' :
                        'bg-amber-500/10 text-amber-400 border border-amber-500/20'
                      }`}>
                        {leave.status || 'Pending'}
                      </span>
                    </div>
                  </CardContent>
                </Card>
              ))}
              
              {(!leaves || leaves.length === 0) && (
                <Card className="bg-zinc-900/20 border-zinc-800/60 border-dashed">
                  <CardContent className="flex flex-col items-center justify-center p-12 text-zinc-500">
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
