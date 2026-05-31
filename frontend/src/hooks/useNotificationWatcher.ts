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

// ── Web Audio "ding" sound (no external file needed) ────────────────────────
function playNotificationSound() {
  try {
    const ctx = new (window.AudioContext || (window as any).webkitAudioContext)();

    const notes = [
      { freq: 880, start: 0,    duration: 0.18, gain: 0.35 },
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

    // Clean up context after sound finishes
    setTimeout(() => ctx.close(), 1200);
  } catch {
    // Silently ignore if Web Audio API not available
  }
}

// ── Hook ─────────────────────────────────────────────────────────────────────
export function useNotificationWatcher(
  onNewNotifications: (notifications: IncomingNotification[]) => void
) {
  const { user } = useAuthStore();
  const queryClient = useQueryClient();
  const seenIdsRef = useRef<Set<string>>(new Set());
  const initializedRef = useRef(false);

  const { data: notifications } = useQuery({
    queryKey: ['notifications', 'watcher'],
    queryFn: async () => {
      const res = await api.get('/notifications');
      return (res.data?.data || []) as IncomingNotification[];
    },
    // Poll every 20 seconds
    refetchInterval: 20_000,
    // Don't poll when the tab is hidden
    refetchIntervalInBackground: false,
    enabled: !!user,
  });

  // Keep the global notifications cache in sync
  const syncGlobal = useCallback(() => {
    queryClient.invalidateQueries({ queryKey: ['notifications'] });
  }, [queryClient]);

  useEffect(() => {
    if (!notifications) return;

    const incoming: IncomingNotification[] = [];

    notifications.forEach((n) => {
      if (!seenIdsRef.current.has(n.id)) {
        seenIdsRef.current.add(n.id);
        // Only fire toast for unread notifications after first load
        if (initializedRef.current) {
          incoming.push(n);
        }
      }
    });

    if (!initializedRef.current) {
      initializedRef.current = true;
      return;
    }

    if (incoming.length > 0) {
      playNotificationSound();
      onNewNotifications(incoming);
      syncGlobal();
    }
  }, [notifications, onNewNotifications, syncGlobal]);
}
