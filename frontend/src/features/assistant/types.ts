// Wire types for the assistant endpoints. These mirror the backend's event
// and pending-action shapes; everything shown in structured cards is
// server-built — model-authored content only ever arrives as `text`.

export interface AssistantEvent {
  type: 'status' | 'text' | 'tool' | 'approval' | 'done' | 'error';
  text?: string;
  tool?: string;
  ok?: boolean;
  data?: unknown;
  message?: string;
  action?: PendingActionCard;
  conversation_id?: string;
}

export interface SummaryField {
  label_ar: string;
  label_en: string;
  value: string;
}

export interface ActionSummary {
  title_ar: string;
  title_en: string;
  fields: SummaryField[];
}

/** The staged card payload carried on an `approval` event. */
export interface PendingActionCard {
  action_id: string;
  type: string;
  summary: ActionSummary;
  expires_at: string;
}

export type ActionState =
  | 'pending'
  | 'approving'
  | 'executed'
  | 'failed'
  | 'rejected'
  | 'expired'
  | 'superseded';

export type Entry =
  | { kind: 'user'; id: string; text: string }
  | { kind: 'assistant_text'; id: string; text: string }
  | { kind: 'tool'; id: string; tool: string; ok: boolean; data?: unknown; message?: string }
  | {
      kind: 'approval';
      id: string;
      card: PendingActionCard;
      state: ActionState;
      failureReason?: string;
    }
  | { kind: 'error'; id: string; code: string };

/**
 * GET /assistant/status. `state` is a coarse lifecycle value; the assistant is
 * rendered for every one of them except `disabled`, so a model that is still
 * loading shows as loading rather than as a feature that vanished.
 */
export type AssistantState = 'disabled' | 'starting' | 'ready' | 'degraded' | 'unavailable';

export interface AssistantStatus {
  state: AssistantState;
  detail?: string;
  ready: boolean;
  /** Kept for compatibility with the previous status shape. */
  enabled: boolean;
}

/** Server row shape returned by the decide/get endpoints. */
export interface PendingActionRow {
  id: string;
  status: string;
  error?: string | null;
}
