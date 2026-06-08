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
  CheckCheck, Filter
} from 'lucide-react';

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

// ─── Announcement Card ────────────────────────────────────────────────────────
const AnnouncementCard = ({ a }: { a: Announcement }) => {
  const [expanded, setExpanded] = useState(false);
  const cfg = PRIORITY_CONFIG[a.priority] || PRIORITY_CONFIG.normal;
  const PriorityIcon = cfg.icon;

  return (
    <div
      className={`relative rounded-2xl border ${cfg.bg} p-5 transition-all duration-300 hover:border-white/20 cursor-pointer`}
      onClick={() => setExpanded(e => !e)}
    >
      {/* Active badge */}
      {a.is_active && (
        <span className="absolute top-3 right-3 flex items-center gap-1.5 text-xs font-semibold text-emerald-400 bg-emerald-500/10 border border-emerald-500/30 rounded-full px-2.5 py-0.5">
          <span className="w-1.5 h-1.5 rounded-full bg-emerald-400 animate-pulse" />
          Active
        </span>
      )}

      <div className="flex items-start gap-4 pr-16">
        {/* Icon */}
        <div className={`w-10 h-10 rounded-xl flex items-center justify-center flex-shrink-0 ${cfg.bg} border`}>
          <PriorityIcon className={`w-5 h-5 ${cfg.text}`} />
        </div>

        {/* Content */}
        <div className="flex-1 min-w-0">
          <div className="flex items-center gap-2 mb-1">
            <span className={`text-xs font-bold uppercase tracking-wider ${cfg.text}`}>
              {cfg.label}
            </span>
            <span className={`w-1 h-1 rounded-full ${cfg.dot}`} />
            <span className="text-xs text-gray-500">{fmtDateTime(a.created_at)}</span>
          </div>
          <h3 className="font-semibold text-white text-[15px] leading-snug">{a.title}</h3>
          {a.creator_name && (
            <p className="text-xs text-gray-500 mt-0.5">By {a.creator_name}</p>
          )}
          {expanded && (
            <p className="text-sm text-gray-300 mt-3 leading-relaxed border-t border-white/5 pt-3">
              {a.message}
            </p>
          )}
          {!expanded && (
            <p className="text-sm text-gray-400 mt-2 line-clamp-2">{a.message}</p>
          )}
          <button className={`mt-2 text-xs font-medium ${cfg.text} hover:opacity-70 transition-opacity`}>
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
    <div className={`relative rounded-2xl border transition-all duration-300 hover:border-white/20 p-5 ${
      n.is_read ? 'bg-white/[0.02] border-white/5' : `${style.bg}`
    }`}>
      {!n.is_read && (
        <span className="absolute top-4 right-4 w-2 h-2 rounded-full bg-blue-500 shadow-[0_0_6px_2px_rgba(59,130,246,0.4)]" />
      )}
      <div className="flex items-start gap-4 pr-8">
        <div className={`w-10 h-10 rounded-xl flex items-center justify-center flex-shrink-0 ${
          n.is_read ? 'bg-white/5 border border-white/10' : `${style.bg} border`
        }`}>
          <Icon className={`w-5 h-5 ${n.is_read ? 'text-gray-500' : style.color}`} />
        </div>
        <div className="flex-1 min-w-0">
          <p className={`font-medium text-[15px] leading-snug ${n.is_read ? 'text-gray-400' : 'text-white'}`}>
            {n.title || n.type || 'Notification'}
          </p>
          <p className={`text-sm mt-1 leading-relaxed ${n.is_read ? 'text-gray-600' : 'text-gray-300'}`}>
            {n.message}
          </p>
          <div className="flex items-center justify-between mt-3">
            <span className="text-xs text-gray-600">{fmtDateTime(n.created_at)}</span>
            {!n.is_read && (
              <button
                onClick={() => onMarkRead(n.id)}
                className="flex items-center gap-1.5 text-xs font-medium text-blue-400 hover:text-blue-300 transition-colors bg-blue-500/10 hover:bg-blue-500/20 border border-blue-500/20 rounded-lg px-2.5 py-1"
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
    <div className="min-h-screen bg-[#080C16] text-white p-6 md:p-8">
      {/* ── Header ── */}
      <div className="mb-8">
        <div className="flex items-center gap-3 mb-2">
          <div className="w-10 h-10 rounded-xl bg-gradient-to-br from-blue-500/30 to-purple-500/30 border border-blue-500/20 flex items-center justify-center">
            <Bell className="w-5 h-5 text-blue-400" />
          </div>
          <div>
            <h1 className="text-2xl font-bold text-white">Inbox</h1>
            <p className="text-sm text-gray-500">
              {unreadCount > 0
                ? `${unreadCount} unread notification${unreadCount > 1 ? 's' : ''}`
                : 'All caught up!'}
            </p>
          </div>
        </div>
      </div>

      {/* ── Tabs ── */}
      <div className="flex items-center gap-2 mb-6 bg-white/5 border border-white/10 rounded-2xl p-1.5 w-fit">
        {[
          { id: 'notifications', label: 'Notifications', icon: Bell, count: unreadCount },
          { id: 'announcements', label: 'Announcements', icon: Megaphone, count: announcements.length },
        ].map(tab => (
          <button
            key={tab.id}
            onClick={() => setActiveTab(tab.id as any)}
            className={`flex items-center gap-2 px-5 py-2.5 rounded-xl text-sm font-medium transition-all duration-200 ${
              activeTab === tab.id
                ? 'bg-gradient-to-r from-blue-600 to-purple-600 text-white shadow-lg shadow-blue-500/20'
                : 'text-gray-400 hover:text-white hover:bg-white/5'
            }`}
          >
            <tab.icon className="w-4 h-4" />
            {tab.label}
            {tab.count > 0 && (
              <span className={`text-xs font-bold rounded-full w-5 h-5 flex items-center justify-center ${
                activeTab === tab.id ? 'bg-white/20 text-white' : 'bg-white/10 text-gray-400'
              }`}>
                {tab.count > 9 ? '9+' : tab.count}
              </span>
            )}
          </button>
        ))}
      </div>

      {/* ── Toolbar ── */}
      <div className="flex items-center justify-between mb-5 flex-wrap gap-3">
        {activeTab === 'notifications' ? (
          <div className="flex items-center gap-2">
            <Filter className="w-4 h-4 text-gray-500" />
            {(['all', 'unread'] as const).map(f => (
              <button
                key={f}
                onClick={() => setFilter(f)}
                className={`px-3 py-1.5 rounded-lg text-xs font-medium transition-all ${
                  filter === f
                    ? 'bg-blue-500/20 text-blue-400 border border-blue-500/30'
                    : 'text-gray-500 hover:text-gray-300 bg-white/5 border border-white/10'
                }`}
              >
                {f === 'all' ? 'All' : `Unread (${unreadCount})`}
              </button>
            ))}
          </div>
        ) : (
          <div className="flex items-center gap-2">
            <Filter className="w-4 h-4 text-gray-500" />
            {(['all', 'critical', 'important', 'normal', 'info'] as const).map(p => (
              <button
                key={p}
                onClick={() => setPriorityFilter(p)}
                className={`px-3 py-1.5 rounded-lg text-xs font-medium capitalize transition-all ${
                  priorityFilter === p
                    ? 'bg-purple-500/20 text-purple-400 border border-purple-500/30'
                    : 'text-gray-500 hover:text-gray-300 bg-white/5 border border-white/10'
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
            className="flex items-center gap-2 px-4 py-2 rounded-xl bg-white/5 hover:bg-white/10 border border-white/10 text-sm text-gray-400 hover:text-white transition-all"
          >
            {markAllRead.isPending
              ? <RefreshCw className="w-4 h-4 animate-spin" />
              : <CheckCheck className="w-4 h-4" />
            }
            Mark all read
          </button>
        )}
      </div>

      {/* ── Content ── */}
      {isLoading ? (
        <div className="flex flex-col items-center justify-center py-24 text-gray-600">
          <Loader2 className="w-10 h-10 animate-spin mb-4" />
          <p className="text-sm">Loading...</p>
        </div>
      ) : activeTab === 'notifications' ? (
        filteredNotifs.length === 0 ? (
          <div className="flex flex-col items-center justify-center py-24 text-gray-600">
            <div className="w-16 h-16 rounded-2xl bg-white/5 border border-white/10 flex items-center justify-center mb-4">
              <BellOff className="w-8 h-8 opacity-40" />
            </div>
            <h3 className="text-lg font-semibold text-gray-400 mb-1">
              {filter === 'unread' ? 'No unread notifications' : 'No notifications yet'}
            </h3>
            <p className="text-sm text-gray-600">
              {filter === 'unread' ? "You're all caught up!" : "We'll notify you when something happens."}
            </p>
          </div>
        ) : (
          <div className="space-y-3 max-w-3xl">
            {filteredNotifs.map(n => (
              <NotifCard key={n.id} n={n} onMarkRead={(id) => markRead.mutate(id)} />
            ))}
          </div>
        )
      ) : (
        filteredAnnouncements.length === 0 ? (
          <div className="flex flex-col items-center justify-center py-24 text-gray-600">
            <div className="w-16 h-16 rounded-2xl bg-white/5 border border-white/10 flex items-center justify-center mb-4">
              <Megaphone className="w-8 h-8 opacity-40" />
            </div>
            <h3 className="text-lg font-semibold text-gray-400 mb-1">No announcements</h3>
            <p className="text-sm text-gray-600">Your department has no announcements yet.</p>
          </div>
        ) : (
          <div className="space-y-4 max-w-3xl">
            {filteredAnnouncements.map(a => (
              <AnnouncementCard key={a.id} a={a} />
            ))}
          </div>
        )
      )}
    </div>
  );
};
