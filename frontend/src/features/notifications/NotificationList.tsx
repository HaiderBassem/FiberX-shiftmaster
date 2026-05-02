import { useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import api from '@/lib/api';
import { Card, CardContent } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Bell, CheckCircle2, Loader2, XCircle, AlertTriangle } from 'lucide-react';
import { format } from 'date-fns';
import { useAuthStore } from '@/store/authStore';

export const NotificationList = () => {
  const queryClient = useQueryClient();
  const { user } = useAuthStore();
  const [processingId, setProcessingId] = useState<string | null>(null);
  const [actionMessage, setActionMessage] = useState<{ type: 'success' | 'error'; text: string } | null>(null);

  const showMessage = (type: 'success' | 'error', text: string) => {
    setActionMessage({ type, text });
    setTimeout(() => setActionMessage(null), 5000);
  };

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
      showMessage('success', accept ? 'Swap accepted successfully!' : 'Swap declined.');
    } catch (err: any) {
      const msg = err?.response?.data?.error || err?.message || 'Action failed';
      showMessage('error', `Failed: ${msg}`);
      console.error('[SwapRespond]', err);
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
      showMessage('success', approve ? 'Swap approved! Employees notified.' : 'Swap rejected. Employees notified.');
    } catch (err: any) {
      const msg = err?.response?.data?.error || err?.message || 'Action failed';
      showMessage('error', `Failed: ${msg}`);
      console.error('[ManagerSwap]', err);
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
      showMessage('success', approve ? 'Leave approved! Employee notified.' : 'Leave rejected. Employee notified.');
    } catch (err: any) {
      const msg = err?.response?.data?.error || err?.message || 'Action failed';
      showMessage('error', `Failed: ${msg}`);
      console.error('[LeaveRespond]', err);
    } finally {
      setProcessingId(null);
    }
  };

  const unreadCount = notifications?.filter((n: any) => !n.is_read).length || 0;

  // --- Smart notification classification ---
  // Uses related_entity_type + title/message to determine the exact action needed.
  // Only unread notifications show action buttons.
  const classifyNotification = (n: any) => {
    const title = (n.title || '').toLowerCase();
    const message = (n.message || '').toLowerCase();
    const entityType = (n.related_entity_type || '').toLowerCase();
    const isUnread = !n.is_read;
    const isTLOrManager = user?.role === 'team_leader' || user?.role === 'manager' || user?.role === 'admin';

    // Only show action buttons for unread notifications with a related entity
    if (!isUnread || !n.related_entity_id) {
      return { type: 'info' as const };
    }

    // --- SWAP notifications ---
    if (entityType === 'swap') {
      // Swap request to target employee: "wants to swap" + "accept"
      if (title.includes('swap request') || (message.includes('wants to swap') && message.includes('accept'))) {
        return { type: 'employee_swap_request' as const };
      }
      // Swap approval needed by team leader
      if (isTLOrManager && (title.includes('swap approval') || title.includes('approval required'))) {
        return { type: 'manager_swap_approval' as const };
      }
      // Informational swap notifications (approved, rejected, accepted)
      return { type: 'info' as const };
    }

    // --- LEAVE notifications ---
    if (entityType === 'leave') {
      // New leave request needing TL/Manager approval
      if (isTLOrManager && (title.includes('new leave request') || title.includes('leave request awaiting'))) {
        return { type: 'leave_approval' as const };
      }
      // Informational leave notifications (approved, rejected, partial approval)
      return { type: 'info' as const };
    }

    return { type: 'info' as const };
  };

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

      {/* Action feedback message */}
      {actionMessage && (
        <div className={`flex items-center gap-2 p-3 rounded-lg text-sm font-medium ${
          actionMessage.type === 'success'
            ? 'bg-green-500/10 text-green-600 border border-green-500/20'
            : 'bg-destructive/10 text-destructive border border-destructive/20'
        }`}>
          {actionMessage.type === 'success'
            ? <CheckCircle2 className="w-4 h-4 shrink-0" />
            : <AlertTriangle className="w-4 h-4 shrink-0" />
          }
          {actionMessage.text}
        </div>
      )}

      {isLoading ? (
        <div className="flex justify-center py-16">
          <Loader2 className="w-8 h-8 animate-spin text-muted-foreground" />
        </div>
      ) : notifications?.length > 0 ? (
        <div className="space-y-3">
          {notifications.map((notification: any) => {
            const isUnread = !notification.is_read;
            const classification = classifyNotification(notification);

            const showActionButtons = classification.type !== 'info';

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
                  {/* 1. Employee Swap Request — Accept/Decline */}
                  {classification.type === 'employee_swap_request' && (
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

                  {/* 2. Manager/TL Swap Approval — Approve/Reject */}
                  {classification.type === 'manager_swap_approval' && (
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

                  {/* 3. Leave Approval — Approve/Reject */}
                  {classification.type === 'leave_approval' && (
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
