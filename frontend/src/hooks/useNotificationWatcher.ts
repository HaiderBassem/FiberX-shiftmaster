import { useEffect, useRef, useCallback } from 'react';
import { useQuery, useQueryClient } from '@tanstack/react-query';
import api from '@/lib/api';
import { useAuthStore } from '@/store/authStore';

export interface IncomingNotification {
  id: string;
  title: string;
  message: string | null;
  type: string;
  priority: string;
  related_entity_type: string | null;
  created_at: string;
}

/**
 * The application's only notification sound.
 *
 * There used to be a second implementation in NotificationProvider, so a
 * notification arriving over more than one channel played two different chimes
 * on top of each other.
 */
export function playNotificationSound() {
  try {
    const AudioCtor = window.AudioContext || (window as unknown as { webkitAudioContext?: typeof AudioContext }).webkitAudioContext;
    if (!AudioCtor) return;

    const ctx = new AudioCtor();

    const notes = [
      { freq: 880, start: 0, duration: 0.18, gain: 0.35 },
      { freq: 1100, start: 0.12, duration: 0.18, gain: 0.28 },
      { freq: 1320, start: 0.24, duration: 0.32, gain: 0.22 },
    ];

    notes.forEach(({ freq, start, duration, gain }) => {
      const osc = ctx.createOscillator();
      const gainNode = ctx.createGain();

      osc.connect(gainNode);
      gainNode.connect(ctx.destination);

      osc.type = 'sine';
      osc.frequency.setValueAtTime(freq, ctx.currentTime + start);

      gainNode.gain.setValueAtTime(0, ctx.currentTime + start);
      gainNode.gain.linearRampToValueAtTime(gain, ctx.currentTime + start + 0.02);
      gainNode.gain.exponentialRampToValueAtTime(0.001, ctx.currentTime + start + duration);

      osc.start(ctx.currentTime + start);
      osc.stop(ctx.currentTime + start + duration + 0.05);
    });

    setTimeout(() => void ctx.close(), 1200);
  } catch {
    // Web Audio is unavailable or blocked until the first user gesture.
  }
}

/** Bounds the seen-id set so a long session cannot grow it without limit. */
const MAX_SEEN_IDS = 500;

/**
 * The single place that decides whether a notification is new.
 *
 * Every delivery channel — the WebSocket, web push, and this poll — converges
 * here: the socket and push handlers in NotificationProvider only invalidate the
 * query, and the resulting refetch runs through the id diff below. Because the
 * decision is made once, against stable server-assigned ids, the same
 * notification arriving over two channels within the same second still produces
 * exactly one toast and one sound.
 *
 * The poll remains as a reconciliation path: it covers the gap while the socket
 * is reconnecting, and any notification created while the tab was closed.
 */
export function useNotificationWatcher(
  onNewNotifications: (notifications: IncomingNotification[]) => void
) {
  const userId = useAuthStore((s) => s.user?.id ?? null);
  const queryClient = useQueryClient();

  const seenIdsRef = useRef<Set<string>>(new Set());
  // Which user the seen-id set belongs to. Without this, signing in as a second
  // user reused the first user's "already initialised" state and every one of
  // the new user's existing notifications was announced as if it had just
  // arrived.
  const initializedForRef = useRef<string | null>(null);

  const { data: notifications } = useQuery({
    queryKey: ['notifications', 'watcher'],
    queryFn: async () => {
      const res = await api.get('/notifications');
      return (res.data?.data ?? []) as IncomingNotification[];
    },
    refetchInterval: 20_000,
    refetchIntervalInBackground: false,
    enabled: !!userId,
  });

  // Refresh the inbox and the unread badge, but never this hook's own query:
  // invalidating that from inside its own effect would trigger an immediate
  // redundant refetch on every delivery.
  const syncOtherViews = useCallback(() => {
    queryClient.invalidateQueries({
      predicate: (query) => {
        const [root, scope] = query.queryKey as [unknown, unknown];
        if (root === 'notifications-unread') return true;
        return root === 'notifications' && scope !== 'watcher';
      },
    });
  }, [queryClient]);

  useEffect(() => {
    if (!userId) {
      seenIdsRef.current.clear();
      initializedForRef.current = null;
      return;
    }
    if (!notifications) return;

    // First load for this user establishes the baseline silently; announcing
    // the entire existing inbox on sign-in would be noise, not news.
    const isFirstLoadForUser = initializedForRef.current !== userId;
    if (isFirstLoadForUser) {
      seenIdsRef.current = new Set(notifications.map((n) => n.id));
      initializedForRef.current = userId;
      return;
    }

    const incoming = notifications.filter((n) => !seenIdsRef.current.has(n.id));
    if (incoming.length === 0) return;

    incoming.forEach((n) => seenIdsRef.current.add(n.id));

    // Keep only the most recent ids; the server returns newest first.
    if (seenIdsRef.current.size > MAX_SEEN_IDS) {
      seenIdsRef.current = new Set(notifications.slice(0, MAX_SEEN_IDS).map((n) => n.id));
    }

    playNotificationSound();
    onNewNotifications(incoming);
    syncOtherViews();
  }, [notifications, userId, onNewNotifications, syncOtherViews]);
}
