import { useState, useCallback, useEffect, useRef } from 'react';
import { Bell, X, CheckCircle2, ArrowLeftRight, ClipboardList, AlertCircle, Info } from 'lucide-react';
import { useNotificationWatcher, type IncomingNotification } from '@/hooks/useNotificationWatcher';

// ── Duration each toast stays on screen (ms) ────────────────────────────────
const TOAST_DURATION = 5500;

interface ToastItem extends IncomingNotification {
  toastId: number;
  progress: number; // 0–100
}

// ── Icon per notification type ───────────────────────────────────────────────
function NotifIcon({ type, entityType }: { type: string; entityType: string | null }) {
  const cls = 'w-5 h-5 shrink-0';
  if (entityType === 'swap')  return <ArrowLeftRight className={`${cls} text-primary`} />;
  if (entityType === 'leave') return <Bell className={`${cls} text-amber-400`} />;
  if (type === 'approval')    return <CheckCircle2 className={`${cls} text-emerald-400`} />;
  if (type === 'task')        return <ClipboardList className={`${cls} text-blue-400`} />;
  if (type === 'warning')     return <AlertCircle className={`${cls} text-rose-400`} />;
  return <Info className={`${cls} text-primary`} />;
}

// ── Priority accent colour ───────────────────────────────────────────────────
function accentColor(priority: string) {
  switch (priority) {
    case 'high':   return 'border-l-rose-500';
    case 'medium': return 'border-l-amber-400';
    default:       return 'border-l-primary';
  }
}

// ── Single toast card ────────────────────────────────────────────────────────
function ToastCard({
  toast,
  onClose,
}: {
  toast: ToastItem;
  onClose: (id: number) => void;
}) {
  return (
    <div
      className={`
        relative flex items-start gap-3
        w-[340px] max-w-[calc(100vw-2rem)]
        rounded-2xl border-l-4 ${accentColor(toast.priority)}
        bg-card/95 backdrop-blur-xl
        border border-border/60
        shadow-2xl shadow-black/30
        px-4 py-3.5
        animate-toast-in
        overflow-hidden
      `}
    >
      {/* Icon */}
      <div className="mt-0.5 w-9 h-9 rounded-xl bg-muted/60 flex items-center justify-center shrink-0">
        <NotifIcon type={toast.type} entityType={toast.related_entity_type} />
      </div>

      {/* Text */}
      <div className="flex-1 min-w-0 pr-6">
        <p className="text-sm font-semibold text-foreground leading-tight truncate">
          {toast.title || 'New Notification'}
        </p>
        {toast.message && (
          <p className="text-xs text-muted-foreground mt-0.5 line-clamp-2 leading-relaxed">
            {toast.message}
          </p>
        )}
      </div>

      {/* Close */}
      <button
        onClick={() => onClose(toast.toastId)}
        className="absolute top-2.5 right-2.5 w-6 h-6 rounded-lg flex items-center justify-center
                   text-muted-foreground hover:text-foreground hover:bg-muted/60 transition-colors"
      >
        <X className="w-3.5 h-3.5" />
      </button>

      {/* Progress bar */}
      <div
        className="absolute bottom-0 left-0 h-[3px] bg-primary/60 rounded-b-2xl transition-none"
        style={{ width: `${toast.progress}%`, transition: 'width 100ms linear' }}
      />
    </div>
  );
}

// ── Container ────────────────────────────────────────────────────────────────
export function NotificationToastContainer() {
  const [toasts, setToasts] = useState<ToastItem[]>([]);
  const nextId = useRef(0);

  const dismiss = useCallback((toastId: number) => {
    setToasts((prev) => prev.filter((t) => t.toastId !== toastId));
  }, []);

  const addToasts = useCallback((notifications: IncomingNotification[]) => {
    const newToasts: ToastItem[] = notifications.slice(0, 4).map((n) => ({
      ...n,
      toastId: ++nextId.current,
      progress: 100,
    }));
    setToasts((prev) => [...newToasts, ...prev].slice(0, 6));
  }, []);

  // Countdown each toast's progress bar, dismiss when reaches 0
  useEffect(() => {
    if (toasts.length === 0) return;

    const TICK = 100; // ms
    const interval = setInterval(() => {
      setToasts((prev) => {
        const next: ToastItem[] = [];
        for (const t of prev) {
          const newProgress = t.progress - (100 / (TOAST_DURATION / TICK));
          if (newProgress <= 0) continue; // expired → remove
          next.push({ ...t, progress: newProgress });
        }
        return next;
      });
    }, TICK);

    return () => clearInterval(interval);
  }, [toasts.length]);

  useNotificationWatcher(addToasts);

  if (toasts.length === 0) return null;

  return (
    <div
      className="fixed top-20 right-4 z-[100] flex flex-col gap-3 pointer-events-none"
      aria-live="polite"
      aria-atomic="false"
    >
      {toasts.map((toast) => (
        <div key={toast.toastId} className="pointer-events-auto">
          <ToastCard toast={toast} onClose={dismiss} />
        </div>
      ))}
    </div>
  );
}
