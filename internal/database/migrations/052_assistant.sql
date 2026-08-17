-- 052_assistant.sql
--
-- Storage for the AI workforce assistant. Three concerns, three tables, plus a
-- request log:
--
--   assistant_conversations / assistant_messages
--       The dialogue itself. Stored server-side so a follow-up turn cannot be
--       fed a fabricated history by the client, and so the same conversation
--       works against any API replica. The role and department captured at
--       creation let the service refuse to continue a conversation after the
--       employee's authority changed — old tool results in the transcript may
--       describe data the employee can no longer see.
--
--   assistant_pending_actions
--       The approval security boundary. A state-changing request is validated,
--       frozen here as exact parameters, and shown to the human. Approval
--       executes precisely these frozen parameters after revalidation — never
--       anything recomputed from model output. status transitions are made
--       with compare-and-set updates, so two replicas (or a double click)
--       cannot both execute one action.
--
--   assistant_requests
--       Observability: one row per chat turn with latencies, tool counts and
--       outcome. Content is deliberately NOT stored here — transcripts live in
--       assistant_messages under the owner's access only.

CREATE TABLE IF NOT EXISTS assistant_conversations (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    employee_id   UUID NOT NULL REFERENCES employees(id) ON DELETE CASCADE,
    -- Authority snapshot at creation; a change forces a fresh conversation.
    role          VARCHAR(20) NOT NULL,
    department_id UUID,
    message_count INTEGER NOT NULL DEFAULT 0,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_assistant_conversations_employee
    ON assistant_conversations (employee_id, updated_at DESC);

CREATE TABLE IF NOT EXISTS assistant_messages (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    conversation_id UUID NOT NULL REFERENCES assistant_conversations(id) ON DELETE CASCADE,
    seq             INTEGER NOT NULL,
    role            VARCHAR(10) NOT NULL CHECK (role IN ('user', 'assistant')),
    -- content is the provider-format block array (text / tool_use /
    -- tool_result). Tool results are stored because follow-up turns need them
    -- to make sense of the dialogue; they are only ever replayed to the model
    -- on behalf of the same employee, and never after an authority change.
    content         JSONB NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (conversation_id, seq)
);

CREATE TABLE IF NOT EXISTS assistant_pending_actions (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    employee_id     UUID NOT NULL REFERENCES employees(id) ON DELETE CASCADE,
    conversation_id UUID REFERENCES assistant_conversations(id) ON DELETE SET NULL,
    action_type     VARCHAR(40) NOT NULL,
    -- The exact, validated operation. Execution reads ONLY this.
    params          JSONB NOT NULL,
    -- What the approval card displayed, kept for audit alongside the params.
    summary         JSONB NOT NULL,
    status          VARCHAR(12) NOT NULL DEFAULT 'pending'
        CHECK (status IN ('pending', 'approved', 'rejected', 'expired', 'superseded', 'executed', 'failed')),
    result          JSONB,
    error           TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    expires_at      TIMESTAMPTZ NOT NULL,
    decided_at      TIMESTAMPTZ,
    CHECK (expires_at > created_at)
);

CREATE INDEX IF NOT EXISTS idx_assistant_pending_actions_employee
    ON assistant_pending_actions (employee_id, status, created_at DESC);

CREATE TABLE IF NOT EXISTS assistant_requests (
    id              UUID PRIMARY KEY,
    employee_id     UUID NOT NULL REFERENCES employees(id) ON DELETE CASCADE,
    conversation_id UUID,
    model           VARCHAR(80),
    tool_calls      INTEGER NOT NULL DEFAULT 0,
    input_chars     INTEGER NOT NULL DEFAULT 0,
    model_ms        INTEGER NOT NULL DEFAULT 0,
    tools_ms        INTEGER NOT NULL DEFAULT 0,
    total_ms        INTEGER NOT NULL DEFAULT 0,
    input_tokens    INTEGER NOT NULL DEFAULT 0,
    output_tokens   INTEGER NOT NULL DEFAULT 0,
    status          VARCHAR(20) NOT NULL,
    error_kind      VARCHAR(40),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_assistant_requests_employee
    ON assistant_requests (employee_id, created_at DESC);
