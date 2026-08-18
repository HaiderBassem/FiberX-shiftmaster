import React, { useEffect, useCallback, createContext, useContext } from 'react';
import { useQueryClient } from '@tanstack/react-query';
import { useAuthStore } from '@/store/authStore';
import api from '@/lib/api';

interface NotificationContextType {
  requestPermission: () => Promise<void>;
  permission: NotificationPermission;
}

const NotificationContext = createContext<NotificationContextType>({
  requestPermission: async () => {},
  permission: 'default',
});

export const useNotification = () => useContext(NotificationContext);

// Base64 to Uint8Array helper for VAPID key
function urlBase64ToUint8Array(base64String: string) {
  const padding = '='.repeat((4 - (base64String.length % 4)) % 4);
  const base64 = (base64String + padding).replace(/-/g, '+').replace(/_/g, '/');

  const rawData = window.atob(base64);
  const outputArray = new Uint8Array(rawData.length);

  for (let i = 0; i < rawData.length; ++i) {
    outputArray[i] = rawData.charCodeAt(i);
  }
  return outputArray;
}

/**
 * Owns the two push-style delivery channels: web push and the notification
 * WebSocket.
 *
 * Neither channel renders anything. Both are treated purely as a signal that the
 * server has something new, and both respond by invalidating the notifications
 * query. Refetching then feeds the one place that decides what is actually new —
 * the id diff in useNotificationWatcher — which in turn raises exactly one toast
 * and plays exactly one sound.
 *
 * Rendering from the socket payload directly, as this component used to, meant
 * the same notification arrived twice on a machine where both the socket and the
 * poll were live: once as a sonner toast from here, once as a card from the
 * watcher, with two different chimes. Treating transport as a signal rather than
 * as content removes that class of bug rather than papering over it.
 */
export const NotificationProvider: React.FC<{ children: React.ReactNode }> = ({ children }) => {
  const isAuthenticated = useAuthStore((s) => s.isAuthenticated);
  const queryClient = useQueryClient();
  const [permission, setPermission] = React.useState<NotificationPermission>(
    'Notification' in window ? Notification.permission : 'default'
  );

  // The single response to any inbound signal. React Query matches by key
  // prefix, so ['notifications'] also refreshes the watcher's own query; the
  // unread badge uses a separate root key and has to be named explicitly.
  // Announcements ride the same wire: their broadcast creates no notification
  // row, so without this a visible tab never learned a new announcement
  // existed until reload.
  const refreshNotifications = useCallback(() => {
    queryClient.invalidateQueries({ queryKey: ['notifications'] });
    queryClient.invalidateQueries({ queryKey: ['notifications-unread'] });
    queryClient.invalidateQueries({ queryKey: ['announcements'] });
    queryClient.invalidateQueries({ queryKey: ['announcements-inbox'] });
  }, [queryClient]);

  const subscribeToPush = useCallback(async () => {
    if (!('serviceWorker' in navigator) || !('PushManager' in window)) return;

    try {
      const registration = await navigator.serviceWorker.register('/sw.js');
      await navigator.serviceWorker.ready;

      const { data: keyData } = await api.get('/push/public-key');
      const publicKey = keyData?.data?.publicKey;
      if (!publicKey) return; // Push is not configured on the server.

      const applicationServerKey = urlBase64ToUint8Array(publicKey);
      let subscription = await registration.pushManager.getSubscription();

      try {
        subscription = await registration.pushManager.subscribe({
          userVisibleOnly: true,
          applicationServerKey,
        });
      } catch {
        // An existing subscription bound to a different VAPID key blocks a new
        // one; drop it and retry once.
        if (subscription) await subscription.unsubscribe();
        subscription = await registration.pushManager.subscribe({
          userVisibleOnly: true,
          applicationServerKey,
        });
      }

      await api.post('/push/subscribe', subscription);
    } catch (err) {
      console.error('Failed to subscribe to push notifications:', err);
    }
  }, []);

  const requestPermission = useCallback(async () => {
    if (!('Notification' in window)) return;

    const result = await Notification.requestPermission();
    setPermission(result);

    if (result === 'granted' && isAuthenticated) {
      await subscribeToPush();
    }
  }, [isAuthenticated, subscribeToPush]);

  useEffect(() => {
    if (isAuthenticated && permission === 'granted') {
      void subscribeToPush();
    }
  }, [isAuthenticated, permission, subscribeToPush]);

  // ── Service worker push → signal ──────────────────────────────────────────
  useEffect(() => {
    if (!('serviceWorker' in navigator)) return;

    const handleMessage = (event: MessageEvent) => {
      if (event.data?.type === 'PUSH_NOTIFICATION') {
        refreshNotifications();
      }
    };

    navigator.serviceWorker.addEventListener('message', handleMessage);
    return () => navigator.serviceWorker.removeEventListener('message', handleMessage);
  }, [refreshNotifications]);

  // ── WebSocket → signal ────────────────────────────────────────────────────
  useEffect(() => {
    if (!isAuthenticated) return;

    // `cancelled` makes the effect safe under StrictMode's double invocation in
    // development, and guarantees that a logout mid-handshake does not leave an
    // orphaned socket behind.
    let cancelled = false;
    let socket: WebSocket | null = null;
    let reconnectTimer: ReturnType<typeof setTimeout> | null = null;
    let attempt = 0;

    const scheduleReconnect = () => {
      if (cancelled) return;
      attempt += 1;
      // Exponential backoff with jitter: a fixed short delay meant every client
      // reconnected in lockstep after a server restart.
      const backoff = Math.min(30_000, 1_000 * 2 ** Math.min(attempt, 5));
      reconnectTimer = setTimeout(connect, backoff + Math.random() * 1_000);
    };

    const connect = async () => {
      if (cancelled) return;

      let ticket: string | undefined;
      try {
        // A browser cannot set an Authorization header on a WebSocket, so the
        // server issues a short-lived single-use ticket for the URL instead of
        // the access token itself. Each attempt needs its own ticket.
        const { data } = await api.post('/notifications/ws-ticket');
        ticket = data?.data?.ticket;
      } catch {
        // Includes the case where the session has ended; the auth interceptor
        // handles logout, and this effect is torn down by the state change.
        scheduleReconnect();
        return;
      }

      if (!ticket || cancelled) return;

      const scheme = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
      const url = `${scheme}//${window.location.host}/api/notifications/ws?ticket=${encodeURIComponent(ticket)}`;

      const ws = new WebSocket(url);
      socket = ws;

      ws.onopen = () => {
        attempt = 0;
      };

      ws.onmessage = (event) => {
        // The payload is not rendered. Anything other than the handshake ack is
        // treated as "the server has something new for you".
        try {
          if (JSON.parse(event.data)?.type === 'connected') return;
        } catch {
          // Unparseable frames are still a signal.
        }
        refreshNotifications();
      };

      ws.onerror = () => ws.close();

      ws.onclose = () => {
        if (cancelled || socket !== ws) return;
        scheduleReconnect();
      };
    };

    void connect();

    return () => {
      cancelled = true;
      if (reconnectTimer) clearTimeout(reconnectTimer);
      if (socket) {
        // Detach the handler first so tearing the socket down does not look like
        // an unexpected drop and schedule a reconnect we just cancelled.
        socket.onclose = null;
        socket.onerror = null;
        socket.onmessage = null;
        socket.close();
        socket = null;
      }
    };
  }, [isAuthenticated, refreshNotifications]);

  return (
    <NotificationContext.Provider value={{ requestPermission, permission }}>
      {children}
    </NotificationContext.Provider>
  );
};
