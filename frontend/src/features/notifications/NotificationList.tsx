import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import api from '@/lib/api';
import { Card, CardContent } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Bell, Check, BellRing, Clock } from 'lucide-react';
import { formatDistanceToNow } from 'date-fns';

interface Notification {
  id: string;
  title: string;
  message: string | null;
  is_read: boolean;
  created_at: string;
}

export const NotificationList = () => {
  const queryClient = useQueryClient();

  const { data: notifications, isLoading } = useQuery<Notification[]>({
    queryKey: ['notifications'],
    queryFn: async () => {
      const response = await api.get('/notifications');
      return response.data?.data || [];
    },
  });

  const markReadMutation = useMutation({
    mutationFn: async (id: string) => {
      await api.post(`/notifications/${id}/read`);
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['notifications'] });
    },
  });

  const markAllReadMutation = useMutation({
    mutationFn: async () => {
      await api.post('/notifications/read-all');
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['notifications'] });
    },
  });

  const unreadCount = notifications?.filter(n => !n.is_read).length || 0;

  return (
    <div className="space-y-6 max-w-4xl mx-auto">
      <div className="flex justify-between items-end">
        <div>
          <h2 className="text-3xl font-bold tracking-tight text-white flex items-center gap-3">
            <Bell className="w-8 h-8 text-blue-400" />
            Notifications
            {unreadCount > 0 && (
              <span className="bg-blue-500 text-white text-sm px-2.5 py-0.5 rounded-full font-medium">
                {unreadCount} new
              </span>
            )}
          </h2>
          <p className="text-zinc-400 mt-2">Stay updated with important alerts and updates.</p>
        </div>
        {unreadCount > 0 && (
          <Button 
            variant="outline" 
            size="sm" 
            onClick={() => markAllReadMutation.mutate()}
            disabled={markAllReadMutation.isPending}
            className="text-zinc-300 border-zinc-700"
          >
            Mark all as read
          </Button>
        )}
      </div>

      {isLoading ? (
        <div className="space-y-4">
          {[1, 2, 3].map((i) => (
            <Card key={i} className="animate-pulse bg-zinc-900/50 h-24" />
          ))}
        </div>
      ) : (
        <div className="space-y-4">
          {notifications?.map((notification) => (
            <Card 
              key={notification.id} 
              className={`transition-all ${!notification.is_read ? 'bg-blue-950/20 border-blue-900/50 shadow-[0_0_15px_rgba(59,130,246,0.05)]' : 'bg-zinc-900/30 border-zinc-800/50 hover:bg-zinc-900/50'}`}
            >
              <CardContent className="p-4 sm:p-6 flex gap-4">
                <div className={`mt-1 h-2 w-2 rounded-full shrink-0 ${!notification.is_read ? 'bg-blue-500 shadow-[0_0_8px_rgba(59,130,246,0.8)]' : 'bg-transparent'}`} />
                <div className="flex-1 space-y-1">
                  <p className={`text-sm sm:text-base ${!notification.is_read ? 'text-zinc-100 font-medium' : 'text-zinc-400'}`}>
                    {notification.message ?? notification.title}
                  </p>
                  <p className="text-xs text-zinc-500 flex items-center gap-1">
                    <Clock className="w-3 h-3" />
                    {formatDistanceToNow(new Date(notification.created_at), { addSuffix: true })}
                  </p>
                </div>
                {!notification.is_read && (
                  <Button 
                    variant="ghost" 
                    size="icon" 
                    onClick={() => markReadMutation.mutate(notification.id)}
                    className="text-blue-400 hover:text-blue-300 hover:bg-blue-500/10 shrink-0 self-center"
                    title="Mark as read"
                  >
                    <Check className="w-5 h-5" />
                  </Button>
                )}
              </CardContent>
            </Card>
          ))}
          {(!notifications || notifications.length === 0) && (
            <Card className="bg-zinc-900/20 border-zinc-800/60 border-dashed">
              <CardContent className="flex flex-col items-center justify-center p-12 text-zinc-500">
                <BellRing className="w-12 h-12 mb-4 opacity-20" />
                <p>You're all caught up!</p>
              </CardContent>
            </Card>
          )}
        </div>
      )}
    </div>
  );
};
