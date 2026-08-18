import { useEffect, useMemo, useRef, useState } from 'react';
import { useQuery } from '@tanstack/react-query';
import { useTranslation } from 'react-i18next';
import { Loader2, RotateCcw, Send, Sparkles, Wrench } from 'lucide-react';
import api from '@/lib/api';
import { useAuthStore } from '@/store/authStore';
import { Button } from '@/components/ui/button';
import { useAssistantChat } from './useAssistantChat';
import { ToolResultCard } from './ResultCards';
import { ApprovalCard } from './ApprovalCard';
import type { Entry } from './types';

// The Home-page assistant. Renders nothing at all when the backend reports
// the feature unconfigured, so the dashboard is byte-identical to before in
// that case.

function EntryView({
  entry,
  onDecide,
}: {
  entry: Entry;
  onDecide: (entryId: string, actionId: string, approve: boolean) => void;
}) {
  const { t } = useTranslation();

  switch (entry.kind) {
    case 'user':
      return (
        <div className="flex justify-end">
          <div
            className="max-w-[85%] rounded-2xl rounded-br-sm bg-primary text-primary-foreground px-3.5 py-2 text-sm whitespace-pre-wrap break-words"
            dir="auto"
          >
            {entry.text}
          </div>
        </div>
      );
    case 'assistant_text':
      return (
        <div className="flex">
          <div
            className="max-w-[85%] rounded-2xl rounded-bl-sm bg-muted px-3.5 py-2 text-sm text-foreground whitespace-pre-wrap break-words"
            dir="auto"
          >
            {entry.text}
          </div>
        </div>
      );
    case 'tool': {
      if (!entry.ok) {
        // The server's failure detail is internal English diagnostics; users
        // get the translated generic line and the model's own follow-up text.
        return (
          <div className="flex items-center gap-1.5 text-xs text-muted-foreground ps-1">
            <Wrench className="h-3 w-3" />
            <span dir="auto">{t('assistant.tool_failed')}</span>
          </div>
        );
      }
      const card = <ToolResultCard tool={entry.tool} data={entry.data} />;
      return card ? <div className="max-w-[95%]">{card}</div> : null;
    }
    case 'approval':
      return (
        <div className="max-w-[95%]">
          <ApprovalCard
            entry={entry}
            onDecide={approve => onDecide(entry.id, entry.card.action_id, approve)}
          />
        </div>
      );
    case 'error':
      return (
        <div className="text-sm text-destructive/90 bg-destructive/10 rounded-lg px-3 py-2">
          {t(`assistant.err_${entry.code}`, t('assistant.err_assistant_error'))}
        </div>
      );
    default:
      return null;
  }
}

export default function AssistantPanel() {
  const { t } = useTranslation();
  const user = useAuthStore(s => s.user);
  const { entries, busy, phase, send, decide, reset } = useAssistantChat();
  const [input, setInput] = useState('');
  const scrollRef = useRef<HTMLDivElement>(null);

  const { data: status } = useQuery({
    queryKey: ['assistant-status'],
    queryFn: async () => {
      const res = await api.get('/assistant/status');
      return res.data?.data as { enabled: boolean } | undefined;
    },
    staleTime: 5 * 60 * 1000,
    // A transient failure here makes the whole assistant vanish with no way
    // back (the panel renders null and nothing remounts it), so keep trying
    // quietly until an answer arrives.
    retry: 3,
    refetchOnWindowFocus: true,
    refetchInterval: query => (query.state.data === undefined ? 60 * 1000 : false),
  });

  const suggestions = useMemo(() => {
    const role = user?.role ?? 'employee';
    const key =
      role === 'manager' || role === 'admin'
        ? 'assistant.suggestions_manager'
        : role === 'team_leader'
          ? 'assistant.suggestions_team_leader'
          : 'assistant.suggestions_employee';
    const list = t(key, { returnObjects: true });
    return Array.isArray(list) ? (list as string[]) : [];
  }, [t, user?.role]);

  useEffect(() => {
    scrollRef.current?.scrollTo({ top: scrollRef.current.scrollHeight, behavior: 'smooth' });
  }, [entries, phase]);

  if (!status?.enabled) return null;

  const submit = () => {
    const text = input.trim();
    if (!text || busy) return;
    setInput('');
    void send(text);
  };

  const phaseLabel =
    phase && phase.startsWith('tool:')
      ? t(`assistant.tool_${phase.slice(5)}`, t('assistant.working'))
      : t('assistant.thinking');

  return (
    <div className="rounded-2xl border bg-card overflow-hidden">
      <div className="flex items-center gap-2 px-4 py-3 border-b bg-card/80">
        <span className="h-8 w-8 rounded-lg bg-primary/15 text-primary flex items-center justify-center">
          <Sparkles className="h-4 w-4" />
        </span>
        <div className="min-w-0">
          <div className="font-semibold text-foreground leading-tight">{t('assistant.title')}</div>
          <div className="text-xs text-muted-foreground truncate">{t('assistant.subtitle')}</div>
        </div>
        {entries.length > 0 && (
          <Button
            variant="ghost"
            size="sm"
            className="ms-auto text-muted-foreground"
            onClick={reset}
            title={t('assistant.new_chat')}
          >
            <RotateCcw className="h-4 w-4" />
          </Button>
        )}
      </div>

      <div
        ref={scrollRef}
        className="px-4 py-3 space-y-3 overflow-y-auto"
        style={{ maxHeight: entries.length > 0 ? '26rem' : undefined }}
      >
        {entries.length === 0 && (
          <div className="py-2 space-y-3">
            <p className="text-sm text-muted-foreground" dir="auto">
              {t('assistant.greeting', { name: user?.first_name?.split(' ')[0] ?? '' })}
            </p>
            <div className="flex flex-wrap gap-2">
              {suggestions.map(sample => (
                <button
                  key={sample}
                  type="button"
                  dir="auto"
                  className="text-xs rounded-full border border-border bg-muted/50 px-3 py-1.5 text-foreground hover:border-primary/50 hover:text-primary transition-colors"
                  onClick={() => void send(sample)}
                >
                  {sample}
                </button>
              ))}
            </div>
          </div>
        )}

        {entries.map(entry => (
          <EntryView key={entry.id} entry={entry} onDecide={decide} />
        ))}

        {busy && (
          <div className="flex items-center gap-2 text-xs text-muted-foreground">
            <Loader2 className="h-3.5 w-3.5 animate-spin" />
            {phaseLabel}
          </div>
        )}
      </div>

      <div className="px-3 pb-3 pt-1">
        <div className="flex items-end gap-2 rounded-xl border bg-background px-3 py-2 focus-within:border-primary/60 transition-colors">
          <textarea
            value={input}
            onChange={e => setInput(e.target.value)}
            onKeyDown={e => {
              if (e.key === 'Enter' && !e.shiftKey) {
                e.preventDefault();
                submit();
              }
            }}
            rows={1}
            dir="auto"
            maxLength={4000}
            placeholder={t('assistant.placeholder')}
            className="flex-1 resize-none bg-transparent text-sm text-foreground placeholder:text-muted-foreground focus:outline-none max-h-28"
          />
          <Button size="icon" className="h-8 w-8 shrink-0" disabled={busy || !input.trim()} onClick={submit}>
            <Send className="h-4 w-4 rtl:-scale-x-100" />
          </Button>
        </div>
      </div>
    </div>
  );
}
