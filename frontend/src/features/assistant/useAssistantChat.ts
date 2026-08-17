import { useCallback, useRef, useState } from 'react';
import api from '@/lib/api';
import { useAuthStore } from '@/store/authStore';
import type { AssistantEvent, Entry, PendingActionCard, ActionState } from './types';

// The chat endpoint streams Server-Sent Events over a POST. axios cannot
// consume a stream in the browser, so this hook speaks fetch directly and
// mirrors the api client's auth behaviour: bearer token from the store, one
// refresh-and-retry on 401.

const API_BASE = import.meta.env.VITE_API_URL || '/api';

let entrySeq = 0;
const nextId = () => `e${++entrySeq}`;

async function refreshToken(): Promise<string | null> {
  const { refreshToken: rt } = useAuthStore.getState();
  if (!rt) return null;
  try {
    const res = await api.post('/auth/refresh', { refresh_token: rt });
    const data = res.data?.data;
    if (data?.access_token) {
      useAuthStore.getState().setTokens(data.access_token, data.refresh_token ?? rt);
      return data.access_token as string;
    }
  } catch {
    /* fall through — the caller reports the failure */
  }
  return null;
}

export interface AssistantChat {
  entries: Entry[];
  busy: boolean;
  /** transient phase for the indicator: 'thinking' or a tool name */
  phase: string | null;
  conversationId: string | null;
  send: (text: string) => Promise<void>;
  decide: (entryId: string, actionId: string, approve: boolean) => Promise<void>;
  reset: () => void;
}

export function useAssistantChat(): AssistantChat {
  const [entries, setEntries] = useState<Entry[]>([]);
  const [busy, setBusy] = useState(false);
  const [phase, setPhase] = useState<string | null>(null);
  const [conversationId, setConversationId] = useState<string | null>(null);
  const abortRef = useRef<AbortController | null>(null);

  const append = useCallback((entry: Entry) => {
    setEntries(prev => [...prev, entry]);
  }, []);

  const applyEvent = useCallback(
    (ev: AssistantEvent) => {
      switch (ev.type) {
        case 'status':
          setPhase(ev.text ?? null);
          break;
        case 'text':
          if (ev.text) append({ kind: 'assistant_text', id: nextId(), text: ev.text });
          break;
        case 'tool':
          append({
            kind: 'tool',
            id: nextId(),
            tool: ev.tool ?? '',
            ok: !!ev.ok,
            data: ev.data,
            message: ev.message,
          });
          break;
        case 'approval':
          if (ev.action) {
            const card = ev.action as PendingActionCard;
            // A new card supersedes any earlier still-pending one, matching
            // the server's supersession rule.
            setEntries(prev =>
              prev
                .map(e =>
                  e.kind === 'approval' && e.state === 'pending'
                    ? { ...e, state: 'superseded' as ActionState }
                    : e,
                )
                .concat([{ kind: 'approval', id: nextId(), card, state: 'pending' }]),
            );
          }
          break;
        case 'error':
          append({ kind: 'error', id: nextId(), code: ev.message ?? 'assistant_error' });
          break;
        case 'done':
          if (ev.conversation_id) setConversationId(ev.conversation_id);
          break;
      }
    },
    [append],
  );

  const streamOnce = useCallback(
    async (token: string, body: string, signal: AbortSignal): Promise<Response> => {
      return fetch(`${API_BASE}/assistant/chat`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${token}`,
        },
        body,
        signal,
        credentials: 'include',
      });
    },
    [],
  );

  const send = useCallback(
    async (text: string) => {
      const message = text.trim();
      if (!message || busy) return;

      setBusy(true);
      setPhase('thinking');
      append({ kind: 'user', id: nextId(), text: message });

      const controller = new AbortController();
      abortRef.current = controller;
      const body = JSON.stringify({
        message,
        conversation_id: conversationId || undefined,
      });

      try {
        const token = useAuthStore.getState().token || '';
        let res = await streamOnce(token, body, controller.signal);
        if (res.status === 401) {
          const fresh = await refreshToken();
          if (!fresh) throw new Error('unauthorized');
          res = await streamOnce(fresh, body, controller.signal);
        }
        if (!res.ok) {
          let code = 'assistant_error';
          if (res.status === 429) code = 'rate_limited';
          else if (res.status === 409) code = 'busy';
          else if (res.status === 503) code = 'assistant_unavailable';
          else {
            try {
              const parsed = await res.json();
              if (typeof parsed?.error === 'string' && parsed.error.includes('conversation')) {
                code = 'conversation_unavailable';
              }
            } catch {
              /* keep generic */
            }
          }
          append({ kind: 'error', id: nextId(), code });
          return;
        }

        // Parse the SSE stream: events are "data:<json>" lines separated by
        // blank lines.
        const reader = res.body?.getReader();
        if (!reader) throw new Error('no stream');
        const decoder = new TextDecoder();
        let buffer = '';
        for (;;) {
          const { done, value } = await reader.read();
          if (done) break;
          buffer += decoder.decode(value, { stream: true });
          let sep: number;
          while ((sep = buffer.indexOf('\n\n')) >= 0) {
            const chunk = buffer.slice(0, sep);
            buffer = buffer.slice(sep + 2);
            for (const line of chunk.split('\n')) {
              if (!line.startsWith('data:')) continue;
              try {
                applyEvent(JSON.parse(line.slice(5)) as AssistantEvent);
              } catch {
                /* ignore malformed frames */
              }
            }
          }
        }
      } catch (err) {
        if (!(err instanceof DOMException && err.name === 'AbortError')) {
          append({ kind: 'error', id: nextId(), code: 'network' });
        }
      } finally {
        setBusy(false);
        setPhase(null);
        abortRef.current = null;
      }
    },
    [append, applyEvent, busy, conversationId, streamOnce],
  );

  const decide = useCallback(async (entryId: string, actionId: string, approve: boolean) => {
    setEntries(prev =>
      prev.map(e => (e.kind === 'approval' && e.id === entryId ? { ...e, state: 'approving' } : e)),
    );
    const setState = (state: ActionState, failureReason?: string) =>
      setEntries(prev =>
        prev.map(e =>
          e.kind === 'approval' && e.id === entryId ? { ...e, state, failureReason } : e,
        ),
      );
    try {
      const res = await api.post(`/assistant/actions/${actionId}/${approve ? 'approve' : 'reject'}`);
      const data = res.data?.data;
      const status: string = data?.action?.status ?? '';
      if (status === 'executed') setState('executed');
      else if (status === 'rejected') setState('rejected');
      else if (status === 'failed') setState('failed', data?.failure_reason || data?.action?.error || '');
      else setState((status as ActionState) || 'expired');
    } catch (err: unknown) {
      // 409: the action is no longer pending — reflect its actual state.
      const conflictAction = (
        err as { response?: { data?: { data?: { action?: { status?: string; error?: string } } } } }
      )?.response?.data?.data?.action;
      const status = conflictAction?.status;
      if (status === 'executed' || status === 'rejected' || status === 'superseded') {
        setState(status);
      } else if (status === 'failed') {
        setState('failed', conflictAction?.error || '');
      } else {
        setState('expired');
      }
    }
  }, []);

  const reset = useCallback(() => {
    abortRef.current?.abort();
    setEntries([]);
    setConversationId(null);
    setBusy(false);
    setPhase(null);
  }, []);

  return { entries, busy, phase, conversationId, send, decide, reset };
}
