import { useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import api from '@/lib/api';
import { Card, CardContent } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Bell, CheckCircle2, Loader2, XCircle } from 'lucide-react';
import { format } from 'date-fns';
import { useAuthStore } from '@/store/authStore';

export const NotificationList = () => {
  const queryClient = useQueryClient();
  const { user } = useAuthStore();
  const [processingId, setProcessingId] = useState<string | null>(null);

  const { data: notifications, isLoading } = useQuery({
    queryKey: ['notifications'],
    queryFn: async () => { const res = await api.get('/notifications'); return res.data?.data || []; },
  });

  const markRead = useMutation({
    mutationFn: async (id: string) => { await api.post(`/notifications/${id}/read`); },
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ['notifications'] }),
  });

  const markAllRead = useMutation({
    mutationFn: async () => { await api.post('/notifications/read-all'); },
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ['notifications'] }),
  });

  // --- Inline Action Handlers ---
  const handleEmployeeSwapRespond = async (notifId: string, swapId: string, accept: boolean) => {
    setProcessingId(notifId);
    try {
      await api.post(`/swaps/${swapId}/respond`, { accept });
      await api.post(`/notifications/${notifId}/read`);
      queryClient.invalidateQueries({ queryKey: ['notifications'] });
      queryClient.invalidateQueries({ queryKey: ['swaps'] });
    } catch (err) {
      console.error(err);
    } finally {
      setProcessingId(null);
    }
  };

  const handleManagerSwapRespond = async (notifId: string, swapId: string, approve: boolean) => {
    setProcessingId(notifId);
    try {
      if (approve) {
        await api.post(`/swaps/${swapId}/approve`);
      } else {
        await api.post(`/swaps/${swapId}/reject`, { reason: 'Rejected from notifications' });
      }
      await api.post(`/notifications/${notifId}/read`);
      queryClient.invalidateQueries({ queryKey: ['notifications'] });
      queryClient.invalidateQueries({ queryKey: ['swaps'] });
    } catch (err) {
      console.error(err);
    } finally {
      setProcessingId(null);
    }
  };

  const handleLeaveRespond = async (notifId: string, leaveId: string, approve: boolean) => {
    setProcessingId(notifId);
    try {
      if (approve) {
        const endpoint = user?.role === 'team_leader' ? `/leaves/${leaveId}/approve/team-leader` : `/leaves/${leaveId}/approve/manager`;
        await api.post(endpoint);
      } else {
        await api.post(`/leaves/${leaveId}/reject`, { reason: 'Rejected from notifications' });
      }
      await api.post(`/notifications/${notifId}/read`);
      queryClient.invalidateQueries({ queryKey: ['notifications'] });
      queryClient.invalidateQueries({ queryKey: ['leaves'] });
    } catch (err) {
      console.error(err);
    } finally {
      setProcessingId(null);
    }
  };

  const unreadCount = notifications?.filter((n: any) => !n.read_at).length || 0;

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h2 className="text-2xl sm:text-3xl font-bold tracking-tight text-foreground mb-2 flex items-center gap-2 sm:gap-3">
            <Bell className="w-6 h-6 sm:w-8 sm:h-8 text-primary" />
            Notifications
          </h2>
          <p className="text-muted-foreground">
            {unreadCount > 0 ? `You have ${unreadCount} unread notification${unreadCount > 1 ? 's' : ''}.` : 'All caught up!'}
          </p>
        </div>
        {unreadCount > 0 && (
          <Button variant="outline" className="gap-2" onClick={() => markAllRead.mutate()} disabled={markAllRead.isPending}>
            <CheckCircle2 className="w-4 h-4" /> Mark All Read
          </Button>
        )}
      </div>

      {isLoading ? (
        <div className="flex justify-center py-16">
          <Loader2 className="w-8 h-8 animate-spin text-muted-foreground" />
        </div>
      ) : notifications?.length > 0 ? (
        <div className="space-y-3">
          {notifications.map((notification: any) => {
            const isUnread = !notification.read_at;
            const isEmployeeSwapReq = notification.title === 'Shift Swap Request' && notification.related_entity_type === 'swap';
            const isManagerSwapReq = notification.title === 'Shift Swap Awaiting Your Approval' && notification.related_entity_type === 'swap';
            const isLeaveReq = (notification.title === 'New Leave Request' || notification.title === 'Leave Request Awaiting Approval') && notification.related_entity_type === 'leave';
            
            const showActionButtons = isUnread && (isEmployeeSwapReq || isManagerSwapReq || isLeaveReq);

            return (
            <Card key={notification.id} className={`transition-all ${isUnread ? 'border-primary/20' : ''}`}>
              <CardContent className="p-4 flex flex-col sm:flex-row sm:items-start justify-between gap-4">
                <div className="flex items-start gap-4 min-w-0">
                  <div className={`mt-1 h-10 w-10 rounded-full flex items-center justify-center shrink-0 ${
                    isUnread ? 'bg-primary/10' : 'bg-muted'
                  }`}>
                    <Bell className={`w-5 h-5 ${isUnread ? 'text-primary' : 'text-muted-foreground'}`} />
                  </div>
                  <div className="min-w-0">
                    <p className={`font-medium truncate ${isUnread ? 'text-foreground' : 'text-muted-foreground'}`}>
                      {notification.title || notification.type || 'Notification'}
                    </p>
                    <p className="text-sm text-muted-foreground mt-0.5">{notification.message}</p>
                    {notification.created_at && (
                      <p className="text-xs text-muted-foreground/60 mt-1">
                        {format(new Date(notification.created_at), 'MMM d, yyyy · h:mm a')}
                      </p>
                    )}
                  </div>
                </div>
                
                <div className="flex flex-row sm:flex-col gap-2 shrink-0 items-end mt-2 sm:mt-0">
                  {/* 1. Employee Swap Request */}
                  {isEmployeeSwapReq && (
                    <div className="flex gap-2">
                      <Button size="sm" variant="outline" className="text-destructive border-destructive/30 hover:bg-destructive/10 h-8 px-2"
                        onClick={() => handleEmployeeSwapRespond(notification.id, notification.related_entity_id, false)}
                        disabled={processingId === notification.id}>
                        <XCircle className="w-4 h-4 mr-1" /> Decline
                      </Button>
                      <Button size="sm" className="h-8 px-2"
                        onClick={() => handleEmployeeSwapRespond(notification.id, notification.related_entity_id, true)}
                        disabled={processingId === notification.id}>
                        <CheckCircle2 className="w-4 h-4 mr-1" /> Accept
                      </Button>
                    </div>
                  )}

                  {/* 2. Manager/TL Swap Approval */}
                  {isManagerSwapReq && (
                    <div className="flex gap-2">
                      <Button size="sm" variant="outline" className="text-destructive border-destructive/30 hover:bg-destructive/10 h-8 px-2"
                        onClick={() => handleManagerSwapRespond(notification.id, notification.related_entity_id, false)}
                        disabled={processingId === notification.id}>
                        <XCircle className="w-4 h-4 mr-1" /> Reject
                      </Button>
                      <Button size="sm" className="h-8 px-2"
                        onClick={() => handleManagerSwapRespond(notification.id, notification.related_entity_id, true)}
                        disabled={processingId === notification.id}>
                        <CheckCircle2 className="w-4 h-4 mr-1" /> Approve
                      </Button>
                    </div>
                  )}

                  {/* 3. Leave Approval */}
                  {isLeaveReq && (
                    <div className="flex gap-2">
                      <Button size="sm" variant="outline" className="text-destructive border-destructive/30 hover:bg-destructive/10 h-8 px-2"
                        onClick={() => handleLeaveRespond(notification.id, notification.related_entity_id, false)}
                        disabled={processingId === notification.id}>
                        <XCircle className="w-4 h-4 mr-1" /> Reject
                      </Button>
                      <Button size="sm" className="h-8 px-2"
                        onClick={() => handleLeaveRespond(notification.id, notification.related_entity_id, true)}
                        disabled={processingId === notification.id}>
                        <CheckCircle2 className="w-4 h-4 mr-1" /> Approve
                      </Button>
                    </div>
                  )}

                  {/* Default Mark Read if no action buttons */}
                  {isUnread && !showActionButtons && (
                    <Button size="sm" variant="ghost" className="text-muted-foreground hover:text-foreground h-8 px-2"
                      onClick={() => markRead.mutate(notification.id)} disabled={markRead.isPending}>
                      Mark read
                    </Button>
                  )}
                </div>
              </CardContent>
            </Card>
            );
          })}
        </div>
      ) : (
        <Card className="border-dashed">
          <CardContent className="flex flex-col items-center justify-center p-16 text-muted-foreground">
            <Bell className="w-16 h-16 mb-4 opacity-20" />
            <h3 className="text-lg font-semibold mb-1">No notifications</h3>
            <p className="text-sm">You're all caught up! Check back later.</p>
          </CardContent>
        </Card>
      )}
    </div>
  );
};
