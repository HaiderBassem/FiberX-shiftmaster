import { useTranslation } from 'react-i18next';
import { CheckCircle2, Loader2, ShieldQuestion, XCircle } from 'lucide-react';
import { Button } from '@/components/ui/button';
import type { Entry } from './types';

// The approval card is the human half of the action security boundary. Every
// value on it was validated and frozen by the backend at staging time; the
// buttons call the authenticated approve/reject endpoints — the model cannot
// press them, and "yes" in chat does nothing.

type ApprovalEntry = Extract<Entry, { kind: 'approval' }>;

export function ApprovalCard({
  entry,
  onDecide,
}: {
  entry: ApprovalEntry;
  onDecide: (approve: boolean) => void;
}) {
  const { t, i18n } = useTranslation();
  const ar = i18n.language?.startsWith('ar');
  const { card, state } = entry;
  const title = ar ? card.summary.title_ar : card.summary.title_en;

  return (
    <div
      className={`rounded-xl border p-4 space-y-3 transition-colors ${
        state === 'pending' || state === 'approving'
          ? 'border-primary/40 bg-primary/5'
          : 'border-border bg-card/60'
      }`}
    >
      <div className="flex items-center gap-2">
        <ShieldQuestion className="h-4 w-4 text-primary shrink-0" />
        <span className="font-semibold text-foreground">{title}</span>
        {(state === 'pending' || state === 'approving') && (
          <span className="ms-auto text-xs text-muted-foreground">
            {t('assistant.needs_approval')}
          </span>
        )}
      </div>

      <dl className="space-y-1.5 text-sm">
        {card.summary.fields.map((field, i) => (
          <div key={i} className="flex gap-3">
            <dt className="w-28 shrink-0 text-muted-foreground">
              {ar ? field.label_ar : field.label_en}
            </dt>
            <dd className="text-foreground break-words" dir="auto">
              {field.value}
            </dd>
          </div>
        ))}
      </dl>

      {state === 'pending' && (
        <div className="flex flex-wrap items-center gap-2 pt-1">
          <Button size="sm" onClick={() => onDecide(true)}>
            {t('assistant.approve')}
          </Button>
          <Button
            size="sm"
            variant="outline"
            className="text-destructive border-destructive/40 hover:bg-destructive/10"
            onClick={() => onDecide(false)}
          >
            {t('assistant.reject')}
          </Button>
          {card.expires_at && (
            <span className="ms-auto text-xs text-muted-foreground" dir="auto">
              {t('assistant.valid_until', { time: card.expires_at })}
            </span>
          )}
        </div>
      )}

      {state === 'approving' && (
        <div className="flex items-center gap-2 text-sm text-muted-foreground">
          <Loader2 className="h-4 w-4 animate-spin" />
          {t('assistant.executing')}
        </div>
      )}
      {state === 'executed' && (
        <div className="flex items-center gap-2 text-sm text-emerald-500">
          <CheckCircle2 className="h-4 w-4" />
          {t('assistant.executed')}
        </div>
      )}
      {state === 'rejected' && (
        <div className="flex items-center gap-2 text-sm text-muted-foreground">
          <XCircle className="h-4 w-4" />
          {t('assistant.rejected_by_you')}
        </div>
      )}
      {state === 'failed' && (
        <div className="flex items-start gap-2 text-sm text-destructive">
          <XCircle className="h-4 w-4 mt-0.5 shrink-0" />
          <span dir="auto">
            {t('assistant.execution_failed')}
            {entry.failureReason ? `: ${entry.failureReason}` : ''}
          </span>
        </div>
      )}
      {state === 'expired' && (
        <div className="text-sm text-muted-foreground">{t('assistant.expired')}</div>
      )}
      {state === 'superseded' && (
        <div className="text-sm text-muted-foreground">{t('assistant.superseded')}</div>
      )}
    </div>
  );
}
