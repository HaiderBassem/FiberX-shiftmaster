import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import api from '@/lib/api';
import { Card, CardContent } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Bell, CheckCircle2, Loader2 } from 'lucide-react';
import { format } from 'date-fns';
import { useNavigate } from 'react-router-dom';

export const NotificationList = () => {
  const queryClient = useQueryClient();
  const navigate = useNavigate();

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
          {notifications.map((notification: any) => (
            <Card key={notification.id} className={`transition-all ${!notification.read_at ? 'border-primary/20' : ''}`}>
              <CardContent className="p-4 flex items-start justify-between gap-4">
                <div className="flex items-start gap-4 min-w-0">
                  <div className={`mt-1 h-10 w-10 rounded-full flex items-center justify-center shrink-0 ${
                    !notification.read_at ? 'bg-primary/10' : 'bg-muted'
                  }`}>
                    <Bell className={`w-5 h-5 ${!notification.read_at ? 'text-primary' : 'text-muted-foreground'}`} />
                  </div>
                  <div className="min-w-0">
                    <p className={`font-medium truncate ${!notification.read_at ? 'text-foreground' : 'text-muted-foreground'}`}>
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
                <div className="flex flex-col gap-2 shrink-0 items-end">
                  {(notification.type === 'shift_change' && notification.related_entity_type === 'swap') && (
                    <Button size="sm" variant="outline" className="border-primary/50 text-primary hover:bg-primary/10 h-8"
                      onClick={() => navigate('/swaps')}>
                      Review Swap
                    </Button>
                  )}
                  {(notification.type === 'leave_request' || notification.type === 'approval') && (
                    <Button size="sm" variant="outline" className="border-primary/50 text-primary hover:bg-primary/10 h-8"
                      onClick={() => navigate('/approvals')}>
                      Review Approval
                    </Button>
                  )}
                  {!notification.read_at && (
                    <Button size="sm" variant="ghost" className="text-muted-foreground hover:text-foreground h-8 px-2"
                      onClick={() => markRead.mutate(notification.id)} disabled={markRead.isPending}>
                      Mark read
                    </Button>
                  )}
                </div>
              </CardContent>
            </Card>
          ))}
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
