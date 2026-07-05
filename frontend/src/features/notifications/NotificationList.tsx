import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import api from '@/lib/api';
import { Card, CardContent } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Bell, CheckCircle2, Loader2 } from 'lucide-react';
import { fmtDateTime } from '@/lib/dateUtils';

export const NotificationList = () => {
  const queryClient = useQueryClient();

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

  const unreadCount = notifications?.filter((n: any) => !n.is_read).length || 0;

  // Icon based on notification type
  const getNotifColor = (n: any) => {
    const title = (n.title || '').toLowerCase();
    const entityType = (n.related_entity_type || '').toLowerCase();
    if (entityType === 'swap') return 'text-primary bg-primary/10';
    if (entityType === 'leave') return 'text-amber-500 bg-amber-500/10';
    if (title.includes('approved') || title.includes('accepted')) return 'text-emerald-500 bg-emerald-500/10';
    if (title.includes('rejected') || title.includes('declined')) return 'text-destructive bg-destructive/10';
    return 'text-primary bg-primary/10';
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

      {isLoading ? (
        <div className="flex justify-center py-16">
          <Loader2 className="w-8 h-8 animate-spin text-muted-foreground" />
        </div>
      ) : unreadCount > 0 ? (
        <div className="space-y-3">
          {notifications.filter((n: any) => !n.is_read).map((notification: any) => {
            const isUnread = !notification.is_read;
            const colorClass = getNotifColor(notification);

            return (
              <Card
                key={notification.id}
                className={`transition-all ${isUnread ? 'border-primary/20' : ''}`}
              >
                <CardContent className="p-4 flex flex-col sm:flex-row sm:items-center justify-between gap-4">
                  <div className="flex items-start gap-4 min-w-0">
                    <div className={`mt-0.5 h-10 w-10 rounded-full flex items-center justify-center shrink-0 ${isUnread ? colorClass : 'bg-muted text-muted-foreground'}`}>
                      <Bell className="w-5 h-5" />
                    </div>
                    <div className="min-w-0 flex-1">
                      <p className={`font-medium ${isUnread ? 'text-foreground' : 'text-muted-foreground'}`}>
                        {notification.title || notification.type || 'Notification'}
                      </p>
                      <p className="text-sm text-muted-foreground mt-0.5 leading-relaxed">
                        {notification.message}
                      </p>
                      {notification.created_at && (
                        <p className="text-xs text-muted-foreground/60 mt-1.5">
                          {fmtDateTime(notification.created_at)}
                        </p>
                      )}
                    </div>
                  </div>

                  {/* Mark as read only — no approve/reject buttons */}
                  {isUnread && (
                    <Button
                      size="sm"
                      variant="ghost"
                      className="text-muted-foreground hover:text-foreground h-8 px-3 shrink-0"
                      onClick={() => markRead.mutate(notification.id)}
                      disabled={markRead.isPending}
                    >
                      <CheckCircle2 className="w-3.5 h-3.5 mr-1.5" />
                      Mark read
                    </Button>
                  )}
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
