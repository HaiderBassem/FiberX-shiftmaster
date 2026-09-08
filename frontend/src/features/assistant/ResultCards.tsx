import { useTranslation } from 'react-i18next';
import {
  CalendarDays,
  Clock,
  ListChecks,
  Users,
  Wifi,
  Wallet,
} from 'lucide-react';

// Structured renderers for tool results. Every value here came from the
// backend's tool layer (never from model text); anything we don't have a
// renderer for simply doesn't get a card — the assistant's text carries it.

const asRecord = (v: unknown): Record<string, unknown> | null =>
  v && typeof v === 'object' && !Array.isArray(v) ? (v as Record<string, unknown>) : null;
const asArray = (v: unknown): Record<string, unknown>[] =>
  Array.isArray(v) ? (v.filter(x => x && typeof x === 'object') as Record<string, unknown>[]) : [];
const str = (v: unknown): string => (typeof v === 'string' ? v : '');
const num = (v: unknown): number | null => (typeof v === 'number' ? v : null);

function CardShell({
  icon,
  title,
  children,
}: {
  icon: React.ReactNode;
  title: string;
  children: React.ReactNode;
}) {
  return (
    <div className="rounded-xl border bg-card/60 p-3 text-sm space-y-2">
      <div className="flex items-center gap-2 text-muted-foreground text-xs font-medium uppercase tracking-wide">
        {icon}
        {title}
      </div>
      {children}
    </div>
  );
}

function ShiftLine({ shift }: { shift: Record<string, unknown> }) {
  const { t } = useTranslation();
  const status = str(shift.status);
  return (
    <div className="flex flex-wrap items-center gap-x-3 gap-y-1">
      <span className="font-semibold text-foreground">
        {str(shift.shift_name) || t(`assistant.status_${status}`, status)}
      </span>
      {str(shift.start_time) && (
        <span className="tabular-nums" dir="ltr">
          {str(shift.start_time)} → {str(shift.end_time)}
        </span>
      )}
      {shift.crosses_midnight === true && (
        <span className="text-xs rounded-full bg-primary/10 text-primary px-2 py-0.5">
          {t('assistant.overnight')}
        </span>
      )}
      <span className="text-muted-foreground tabular-nums" dir="ltr">
        {str(shift.date)}
      </span>
    </div>
  );
}

export function ToolResultCard({ tool, data }: { tool: string; data: unknown }) {
  const { t, i18n } = useTranslation();
  const record = asRecord(data);
  if (!record) return null;
  const ar = i18n.language?.startsWith('ar');

  switch (tool) {
    case 'get_current_shift': {
      const active = asRecord(record.active);
      const upcoming = asRecord(record.upcoming);
      return (
        <CardShell icon={<Clock className="h-3.5 w-3.5" />} title={t('assistant.card_current_shift')}>
          {active ? (
            <>
              <ShiftLine shift={active} />
              {str(record.remaining_until_end) && (
                <div className="text-muted-foreground">
                  {t('assistant.remaining')}: <span dir="ltr">{str(record.remaining_until_end)}</span>
                </div>
              )}
            </>
          ) : upcoming ? (
            <>
              <div className="text-muted-foreground">{t('assistant.no_active_shift')}</div>
              <ShiftLine shift={upcoming} />
            </>
          ) : (
            <div className="text-muted-foreground">{t('assistant.no_shift')}</div>
          )}
        </CardShell>
      );
    }

    case 'get_my_schedule': {
      const days = asArray(record.days);
      if (days.length === 0) return null;
      return (
        <CardShell icon={<CalendarDays className="h-3.5 w-3.5" />} title={t('assistant.card_schedule')}>
          <div className="space-y-1">
            {days.map((day, i) => (
              <div key={i} className="flex items-center gap-2">
                <span className="w-24 shrink-0 tabular-nums text-muted-foreground" dir="ltr">
                  {str(day.date)}
                </span>
                {str(day.start_time) ? (
                  <span className="tabular-nums" dir="ltr">
                    {str(day.start_time)}–{str(day.end_time)}
                  </span>
                ) : (
                  <span className="text-muted-foreground">
                    {t(`assistant.status_${str(day.status)}`, str(day.status))}
                  </span>
                )}
              </div>
            ))}
          </div>
        </CardShell>
      );
    }

    case 'get_my_tasks': {
      const tasks = asArray(record.tasks);
      const counts = asRecord(record.counts) ?? {};
      return (
        <CardShell icon={<ListChecks className="h-3.5 w-3.5" />} title={t('assistant.card_tasks')}>
          {tasks.length === 0 ? (
            <div className="text-muted-foreground">{t('assistant.no_tasks')}</div>
          ) : (
            <>
              <div className="flex gap-3 text-xs text-muted-foreground">
                {Object.entries(counts).map(([k, v]) => (
                  <span key={k}>
                    {t(`assistant.task_${k}`, k)}: {String(v)}
                  </span>
                ))}
              </div>
              <div className="space-y-1">
                {tasks.slice(0, 6).map((task, i) => (
                  <div key={i} className="flex items-center gap-2">
                    <span
                      className={`h-1.5 w-1.5 rounded-full shrink-0 ${
                        str(task.status) === 'completed'
                          ? 'bg-emerald-500'
                          : str(task.status) === 'in_progress'
                            ? 'bg-amber-500'
                            : 'bg-muted-foreground/50'
                      }`}
                    />
                    <span className="truncate">{str(task.title)}</span>
                    <span className="ms-auto text-xs text-muted-foreground tabular-nums" dir="ltr">
                      {str(task.date)}
                    </span>
                  </div>
                ))}
              </div>
            </>
          )}
        </CardShell>
      );
    }

    case 'get_my_leave_balance': {
      const balances = asArray(record.balances);
      if (balances.length === 0) return null;
      return (
        <CardShell icon={<Wallet className="h-3.5 w-3.5" />} title={t('assistant.card_balances')}>
          <div className="space-y-1">
            {balances.slice(0, 6).map((b, i) => (
              <div key={i} className="flex items-center gap-2">
                <span className="truncate">{str(b.leave_type)}</span>
                <span className="ms-auto tabular-nums" dir="ltr">
                  {num(b.remaining) ?? '—'} / {num(b.allocated) ?? '—'}
                </span>
              </div>
            ))}
          </div>
        </CardShell>
      );
    }

    case 'search_service_plans': {
      const plans = asArray(record.plans);
      if (plans.length === 0) return null;
      return (
        <CardShell icon={<Wifi className="h-3.5 w-3.5" />} title={t('assistant.card_plans')}>
          <div className="overflow-x-auto">
            <table className="w-full text-xs">
              <thead className="text-muted-foreground">
                <tr className="text-start">
                  <th className="text-start font-medium py-1">{t('assistant.plan')}</th>
                  <th className="text-start font-medium py-1">{t('assistant.speed')}</th>
                  <th className="text-end font-medium py-1">{t('assistant.price')}</th>
                </tr>
              </thead>
              <tbody>
                {plans.slice(0, 8).map((p, i) => (
                  <tr key={i} className="border-t border-border/50">
                    <td className="py-1 pe-2">
                      {str(p.name)}
                      <span className="text-muted-foreground"> · {str(p.province)}</span>
                    </td>
                    <td className="py-1 pe-2" dir="ltr">
                      {str(p.speed) || '—'}
                    </td>
                    <td className="py-1 text-end tabular-nums" dir="ltr">
                      {num(p.price_iqd)?.toLocaleString(ar ? 'ar-IQ' : 'en-US') ?? '—'}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </CardShell>
      );
    }

    case 'get_team_status': {
      const counts = asRecord(record.status_counts) ?? {};
      return (
        <CardShell icon={<Users className="h-3.5 w-3.5" />} title={str(record.department) || t('assistant.card_team')}>
          <div className="flex flex-wrap gap-2">
            <span className="rounded-full bg-primary/10 text-primary px-2 py-0.5 text-xs">
              {t('assistant.on_shift_now')}: {num(record.on_shift_now) ?? 0}
            </span>
            <span className="rounded-full bg-emerald-500/10 text-emerald-500 px-2 py-0.5 text-xs">
              {t('assistant.checked_in')}: {num(record.checked_in_now) ?? 0}
            </span>
            {Object.entries(counts).map(([k, v]) => (
              <span key={k} className="rounded-full bg-muted px-2 py-0.5 text-xs text-muted-foreground">
                {t(`assistant.status_${k}`, k)}: {String(v)}
              </span>
            ))}
          </div>
        </CardShell>
      );
    }

    default:
      return null;
  }
}
