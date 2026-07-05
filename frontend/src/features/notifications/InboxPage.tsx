import { useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import api from '@/lib/api';
import { announcementService } from '@/services/announcementService';
import type { Announcement } from '@/services/announcementService';
import { fmtDateTime } from '@/lib/dateUtils';
import {
  Bell, Megaphone, CheckCircle2, Loader2,
  Info, AlertTriangle, AlertCircle, Zap,
  BellOff, ArrowRightLeft, CalendarClock, ClipboardCheck, RefreshCw,
  CheckCheck, Filter, X
} from 'lucide-react';

// ─── Helper ───────────────────────────────────────────────────────────────────
const getImageUrl = (url: string) => {
  if (url.startsWith('http')) return url;
  const base = import.meta.env.VITE_API_URL
    ? import.meta.env.VITE_API_URL.replace('/api', '')
    : (import.meta.env.DEV ? 'http://localhost:8080' : '');
  return `${base}${url.startsWith('/api') ? url : '/api' + url}`;
};

// ─── Types ───────────────────────────────────────────────────────────────────
interface Notification {
  id: string;
  title: string;
  message: string;
  type: string;
  is_read: boolean;
  related_entity_type?: string;
  created_at: string;
}

// ─── Priority badge config ────────────────────────────────────────────────────
const PRIORITY_CONFIG = {
  critical: { label: 'Critical', bg: 'bg-red-500/15 border-red-500/40', text: 'text-red-400', icon: Zap, dot: 'bg-red-500' },
  important: { label: 'Important', bg: 'bg-amber-500/15 border-amber-500/40', text: 'text-amber-400', icon: AlertTriangle, dot: 'bg-amber-500' },
  info: { label: 'Info', bg: 'bg-blue-500/15 border-blue-500/40', text: 'text-blue-400', icon: Info, dot: 'bg-blue-500' },
  normal: { label: 'Normal', bg: 'bg-white/5 border-white/10', text: 'text-gray-300', icon: Megaphone, dot: 'bg-gray-400' },
};

const NOTIF_TYPE_CONFIG: Record<string, { color: string; bg: string; Icon: any }> = {
  swap: { color: 'text-purple-400', bg: 'bg-purple-500/15 border-purple-500/30', Icon: ArrowRightLeft },
  leave: { color: 'text-amber-400', bg: 'bg-amber-500/15 border-amber-500/30', Icon: CalendarClock },
  task: { color: 'text-cyan-400', bg: 'bg-cyan-500/15 border-cyan-500/30', Icon: ClipboardCheck },
  approved: { color: 'text-emerald-400', bg: 'bg-emerald-500/15 border-emerald-500/30', Icon: CheckCircle2 },
  rejected: { color: 'text-red-400', bg: 'bg-red-500/15 border-red-500/30', Icon: AlertCircle },
  default: { color: 'text-blue-400', bg: 'bg-blue-500/15 border-blue-500/30', Icon: Bell },
};

const getNotifStyle = (n: Notification) => {
  const entity = (n.related_entity_type || '').toLowerCase();
  const title = (n.title || '').toLowerCase();
  if (entity === 'swap') return NOTIF_TYPE_CONFIG.swap;
  if (entity === 'leave') return NOTIF_TYPE_CONFIG.leave;
  if (entity === 'task') return NOTIF_TYPE_CONFIG.task;
  if (title.includes('approved') || title.includes('accepted')) return NOTIF_TYPE_CONFIG.approved;
  if (title.includes('rejected') || title.includes('declined')) return NOTIF_TYPE_CONFIG.rejected;
  return NOTIF_TYPE_CONFIG.default;
};

// ─── Image Lightbox ───────────────────────────────────────────────────────────
const ImageLightbox = ({ src, onClose }: { src: string; onClose: () => void }) => (
  <div
    className="fixed inset-0 z-[60] flex items-center justify-center bg-black/80 p-4 animate-in fade-in duration-200"
    onClick={onClose}
  >
    <button
      className="absolute top-4 right-4 w-10 h-10 rounded-full bg-white/10 hover:bg-white/20 flex items-center justify-center text-white transition-colors z-10"
      onClick={onClose}
    >
      <X className="w-5 h-5" />
    </button>
    <img
      src={src}
      alt="Full size"
      className="max-w-full max-h-[85vh] rounded-xl shadow-2xl object-contain animate-in zoom-in-95 duration-200"
      onClick={(e) => e.stopPropagation()}
    />
  </div>
);

// ─── Announcement Card ────────────────────────────────────────────────────────
const AnnouncementCard = ({ a, onImageClick }: { a: Announcement; onImageClick: (url: string) => void }) => {
  const [expanded, setExpanded] = useState(false);
  const cfg = PRIORITY_CONFIG[a.priority] || PRIORITY_CONFIG.normal;
  const PriorityIcon = cfg.icon;

  return (
    <div
      className={`relative rounded-2xl border ${cfg.bg} p-4 sm:p-5 transition-all duration-300 hover:border-primary/20 cursor-pointer`}
      onClick={() => setExpanded(e => !e)}
    >
      {/* Active badge */}
      {a.is_active && (
        <span className="absolute top-3 right-3 flex items-center gap-1.5 text-xs font-semibold text-emerald-500 dark:text-emerald-400 bg-emerald-500/10 border border-emerald-500/30 rounded-full px-2 sm:px-2.5 py-0.5">
          <span className="w-1.5 h-1.5 rounded-full bg-emerald-500 dark:bg-emerald-400 animate-pulse" />
          Active
        </span>
      )}

      <div className="flex items-start gap-3 sm:gap-4 pr-12 sm:pr-16">
        {/* Icon */}
        <div className={`w-9 h-9 sm:w-10 sm:h-10 rounded-xl flex items-center justify-center flex-shrink-0 ${cfg.bg} border`}>
          <PriorityIcon className={`w-4 h-4 sm:w-5 sm:h-5 ${cfg.text}`} />
        </div>

        {/* Content */}
        <div className="flex-1 min-w-0">
          <div className="flex items-center gap-2 mb-1 flex-wrap">
            <span className={`text-[10px] sm:text-xs font-bold uppercase tracking-wider ${cfg.text}`}>
              {cfg.label}
            </span>
            <span className={`w-1 h-1 rounded-full ${cfg.dot}`} />
            <span className="text-[10px] sm:text-xs text-muted-foreground">{fmtDateTime(a.created_at)}</span>
          </div>
          <h3 className="font-semibold text-foreground text-sm sm:text-[15px] leading-snug">{a.title}</h3>
          {a.creator_name && (
            <p className="text-[10px] sm:text-xs text-muted-foreground mt-0.5">By {a.creator_name}</p>
          )}
          {expanded && (
            <p className="text-xs sm:text-sm text-foreground/80 mt-3 leading-relaxed border-t border-border pt-3">
              {a.message}
            </p>
          )}
          {!expanded && (
            <p className="text-xs sm:text-sm text-muted-foreground mt-2 line-clamp-2">{a.message}</p>
          )}

          {/* Images */}
          {a.images && a.images.length > 0 && (
            <div
              className="flex gap-2 mt-3 overflow-x-auto pb-1 -mx-1 px-1"
              onClick={(e) => e.stopPropagation()}
            >
              {a.images.map((img, idx) => (
                <button
                  key={idx}
                  onClick={() => onImageClick(getImageUrl(img))}
                  className="relative flex-shrink-0 group rounded-lg overflow-hidden border border-border hover:border-primary/50 transition-colors"
                >
                  <img
                    src={getImageUrl(img)}
                    alt={`Attachment ${idx + 1}`}
                    className="w-16 h-16 sm:w-20 sm:h-20 object-cover transition-transform group-hover:scale-105"
                  />
                </button>
              ))}
            </div>
          )}

          <button className={`mt-2 text-[10px] sm:text-xs font-medium ${cfg.text} hover:opacity-70 transition-opacity`}>
            {expanded ? 'Show less ↑' : 'Read more ↓'}
          </button>
        </div>
      </div>
    </div>
  );
};

// ─── Notification Card ────────────────────────────────────────────────────────
const NotifCard = ({ n, onMarkRead }: { n: Notification; onMarkRead: (id: string) => void }) => {
  const style = getNotifStyle(n);
  const { Icon } = style;

  return (
    <div className={`relative rounded-2xl border transition-all duration-300 hover:border-primary/20 p-4 sm:p-5 ${
      n.is_read ? 'bg-card border-border' : `${style.bg} border-border`
    }`}>
      {!n.is_read && (
        <span className="absolute top-3 sm:top-4 right-3 sm:right-4 w-2 h-2 rounded-full bg-primary shadow-[0_0_6px_2px_rgba(12,204,204,0.4)]" />
      )}
      <div className="flex items-start gap-3 sm:gap-4 pr-6 sm:pr-8">
        <div className={`w-9 h-9 sm:w-10 sm:h-10 rounded-xl flex items-center justify-center flex-shrink-0 ${
          n.is_read ? 'bg-muted border border-border' : `${style.bg} border border-border`
        }`}>
          <Icon className={`w-4 h-4 sm:w-5 sm:h-5 ${n.is_read ? 'text-muted-foreground' : style.color}`} />
        </div>
        <div className="flex-1 min-w-0">
          <p className={`font-medium text-sm sm:text-[15px] leading-snug ${n.is_read ? 'text-muted-foreground' : 'text-foreground'}`}>
            {n.title || n.type || 'Notification'}
          </p>
          <p className={`text-xs sm:text-sm mt-1 leading-relaxed ${n.is_read ? 'text-muted-foreground/80' : 'text-foreground/80'}`}>
            {n.message}
          </p>
          <div className="flex items-center justify-between mt-2 sm:mt-3 flex-wrap gap-2">
            <span className="text-[10px] sm:text-xs text-muted-foreground">{fmtDateTime(n.created_at)}</span>
            {!n.is_read && (
              <button
                onClick={() => onMarkRead(n.id)}
                className="flex items-center gap-1 sm:gap-1.5 text-[10px] sm:text-xs font-medium text-primary hover:text-primary/80 transition-colors bg-primary/10 hover:bg-primary/20 border border-primary/20 rounded-lg px-2 sm:px-2.5 py-0.5 sm:py-1"
              >
                <CheckCircle2 className="w-3 h-3" />
                Mark read
              </button>
            )}
          </div>
        </div>
      </div>
    </div>
  );
};

// ─── Main InboxPage ───────────────────────────────────────────────────────────
export const InboxPage = () => {
  const queryClient = useQueryClient();
  const [activeTab, setActiveTab] = useState<'notifications' | 'announcements'>('notifications');
  const [filter, setFilter] = useState<'all' | 'unread'>('all');
  const [priorityFilter, setPriorityFilter] = useState<string>('all');
  const [lightboxSrc, setLightboxSrc] = useState<string | null>(null);

  // ── Notifications ──
  const { data: notifications = [], isLoading: loadingNotifs } = useQuery<Notification[]>({
    queryKey: ['notifications'],
    queryFn: async () => {
      try {
        const res = await api.get('/notifications');
        return res.data?.data || [];
      } catch { return []; }
    },
    retry: false,
  });

  const markRead = useMutation({
    mutationFn: (id: string) => api.post(`/notifications/${id}/read`),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ['notifications'] }),
  });

  const markAllRead = useMutation({
    mutationFn: () => api.post('/notifications/read-all'),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ['notifications'] }),
  });

  // ── Announcements ──
  const { data: announcements = [], isLoading: loadingAnnouncements } = useQuery<Announcement[]>({
    queryKey: ['announcements-inbox'],
    queryFn: async () => {
      try {
        return await announcementService.getAll();
      } catch { return []; }
    },
    retry: false,
  });

  // ── Filtered data ──
  const filteredNotifs = notifications.filter(n =>
    filter === 'unread' ? !n.is_read : true
  );

  const filteredAnnouncements = announcements.filter(a =>
    priorityFilter === 'all' ? true : a.priority === priorityFilter
  ).sort((a, b) => {
    const order = { critical: 0, important: 1, normal: 2, info: 3 };
    return (order[a.priority] ?? 3) - (order[b.priority] ?? 3);
  });

  const unreadCount = notifications.filter(n => !n.is_read).length;
  const isLoading = activeTab === 'notifications' ? loadingNotifs : loadingAnnouncements;

  return (
    <div className="min-h-screen bg-background text-foreground p-4 sm:p-6 md:p-8">
      {/* ── Header ── */}
      <div className="mb-5 sm:mb-8">
        <div className="flex items-center gap-3 mb-2">
          <div className="w-9 h-9 sm:w-10 sm:h-10 rounded-xl bg-primary/20 border border-primary/20 flex items-center justify-center">
            <Bell className="w-4 h-4 sm:w-5 sm:h-5 text-primary" />
          </div>
          <div>
            <h1 className="text-xl sm:text-2xl font-bold text-foreground">Inbox</h1>
            <p className="text-xs sm:text-sm text-muted-foreground">
              {unreadCount > 0
                ? `${unreadCount} unread notification${unreadCount > 1 ? 's' : ''}`
                : 'All caught up!'}
            </p>
          </div>
        </div>
      </div>

      {/* ── Tabs ── */}
      <div className="flex items-center gap-1.5 sm:gap-2 mb-4 sm:mb-6 bg-card border border-border rounded-xl sm:rounded-2xl p-1 sm:p-1.5 w-full sm:w-fit overflow-x-auto">
        {[
          { id: 'notifications', label: 'Notifications', shortLabel: 'Notifs', icon: Bell, count: unreadCount },
          { id: 'announcements', label: 'Announcements', shortLabel: 'Announce', icon: Megaphone, count: announcements.length },
        ].map(tab => (
          <button
            key={tab.id}
            onClick={() => setActiveTab(tab.id as any)}
            className={`flex items-center gap-1.5 sm:gap-2 px-3 sm:px-5 py-2 sm:py-2.5 rounded-lg sm:rounded-xl text-xs sm:text-sm font-medium transition-all duration-200 flex-1 sm:flex-initial justify-center sm:justify-start whitespace-nowrap ${
              activeTab === tab.id
                ? 'bg-primary text-primary-foreground shadow-lg shadow-primary/20'
                : 'text-muted-foreground hover:text-foreground hover:bg-accent/50'
            }`}
          >
            <tab.icon className="w-3.5 h-3.5 sm:w-4 sm:h-4" />
            <span className="hidden sm:inline">{tab.label}</span>
            <span className="sm:hidden">{tab.shortLabel}</span>
            {tab.count > 0 && (
              <span className={`text-[10px] sm:text-xs font-bold rounded-full w-4 h-4 sm:w-5 sm:h-5 flex items-center justify-center ${
                activeTab === tab.id ? 'bg-background/20 text-primary-foreground' : 'bg-primary/10 text-primary'
              }`}>
                {tab.count > 9 ? '9+' : tab.count}
              </span>
            )}
          </button>
        ))}
      </div>

      {/* ── Toolbar ── */}
      <div className="flex items-center justify-between mb-4 sm:mb-5 flex-wrap gap-2 sm:gap-3">
        {activeTab === 'notifications' ? (
          <div className="flex items-center gap-1.5 sm:gap-2">
            <Filter className="w-3.5 h-3.5 sm:w-4 sm:h-4 text-muted-foreground" />
            {(['all', 'unread'] as const).map(f => (
              <button
                key={f}
                onClick={() => setFilter(f)}
                className={`px-2.5 sm:px-3 py-1 sm:py-1.5 rounded-lg text-[10px] sm:text-xs font-medium transition-all ${
                  filter === f
                    ? 'bg-primary/20 text-primary border border-primary/30'
                    : 'text-muted-foreground hover:text-foreground bg-card border border-border'
                }`}
              >
                {f === 'all' ? 'All' : `Unread (${unreadCount})`}
              </button>
            ))}
          </div>
        ) : (
          <div className="flex items-center gap-1.5 sm:gap-2 overflow-x-auto pb-1">
            <Filter className="w-3.5 h-3.5 sm:w-4 sm:h-4 text-muted-foreground flex-shrink-0" />
            {(['all', 'critical', 'important', 'normal', 'info'] as const).map(p => (
              <button
                key={p}
                onClick={() => setPriorityFilter(p)}
                className={`px-2.5 sm:px-3 py-1 sm:py-1.5 rounded-lg text-[10px] sm:text-xs font-medium capitalize transition-all whitespace-nowrap flex-shrink-0 ${
                  priorityFilter === p
                    ? 'bg-primary/20 text-primary border border-primary/30'
                    : 'text-muted-foreground hover:text-foreground bg-card border border-border'
                }`}
              >
                {p === 'all' ? 'All' : p}
              </button>
            ))}
          </div>
        )}

        {activeTab === 'notifications' && unreadCount > 0 && (
          <button
            onClick={() => markAllRead.mutate()}
            disabled={markAllRead.isPending}
            className="flex items-center gap-1.5 sm:gap-2 px-3 sm:px-4 py-1.5 sm:py-2 rounded-lg sm:rounded-xl bg-card hover:bg-accent border border-border text-xs sm:text-sm text-muted-foreground hover:text-foreground transition-all"
          >
            {markAllRead.isPending
              ? <RefreshCw className="w-3.5 h-3.5 sm:w-4 sm:h-4 animate-spin" />
              : <CheckCheck className="w-3.5 h-3.5 sm:w-4 sm:h-4" />
            }
            <span className="hidden xs:inline">Mark all read</span>
            <span className="xs:hidden">All read</span>
          </button>
        )}
      </div>

      {/* ── Content ── */}
      {isLoading ? (
        <div className="flex flex-col items-center justify-center py-16 sm:py-24 text-muted-foreground">
          <Loader2 className="w-8 h-8 sm:w-10 sm:h-10 animate-spin mb-4 text-primary" />
          <p className="text-xs sm:text-sm">Loading...</p>
        </div>
      ) : activeTab === 'notifications' ? (
        filteredNotifs.length === 0 ? (
          <div className="flex flex-col items-center justify-center py-16 sm:py-24 text-muted-foreground">
            <div className="w-14 h-14 sm:w-16 sm:h-16 rounded-2xl bg-card border border-border flex items-center justify-center mb-4">
              <BellOff className="w-7 h-7 sm:w-8 sm:h-8 opacity-40" />
            </div>
            <h3 className="text-base sm:text-lg font-semibold text-foreground mb-1">
              {filter === 'unread' ? 'No unread notifications' : 'No notifications yet'}
            </h3>
            <p className="text-xs sm:text-sm text-muted-foreground text-center px-4">
              {filter === 'unread' ? "You're all caught up!" : "We'll notify you when something happens."}
            </p>
          </div>
        ) : (
          <div className="space-y-2 sm:space-y-3 max-w-3xl">
            {filteredNotifs.map(n => (
              <NotifCard key={n.id} n={n} onMarkRead={(id) => markRead.mutate(id)} />
            ))}
          </div>
        )
      ) : (
        filteredAnnouncements.length === 0 ? (
          <div className="flex flex-col items-center justify-center py-16 sm:py-24 text-muted-foreground">
            <div className="w-14 h-14 sm:w-16 sm:h-16 rounded-2xl bg-card border border-border flex items-center justify-center mb-4">
              <Megaphone className="w-7 h-7 sm:w-8 sm:h-8 opacity-40" />
            </div>
            <h3 className="text-base sm:text-lg font-semibold text-foreground mb-1">No announcements</h3>
            <p className="text-xs sm:text-sm text-muted-foreground text-center px-4">Your department has no announcements yet.</p>
          </div>
        ) : (
          <div className="space-y-3 sm:space-y-4 max-w-3xl">
            {filteredAnnouncements.map(a => (
              <AnnouncementCard key={a.id} a={a} onImageClick={(url) => setLightboxSrc(url)} />
            ))}
          </div>
        )
      )}

      {/* ── Lightbox ── */}
      {lightboxSrc && <ImageLightbox src={lightboxSrc} onClose={() => setLightboxSrc(null)} />}
    </div>
  );
};
