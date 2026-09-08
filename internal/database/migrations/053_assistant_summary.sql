-- 053_assistant_summary.sql
--
-- Conversation memory that survives the model's context window.
--
-- The assistant replays a bounded window of recent turns to the model. That
-- window is bounded twice: by message count, and — new here — by an estimated
-- token budget, because a locally hosted model has a fixed context (8k by
-- default) and three large tool results can exhaust it on their own. Whatever
-- falls out of that window would otherwise be forgotten mid-dialogue: "the
-- same shift I told you about" stops resolving.
--
-- summary holds a short, model-written recap of the turns that have scrolled
-- out, regenerated in the background after a turn that dropped history, and
-- injected into the system prompt on subsequent turns. summarized_seq records
-- how far the recap covers, so it is extended rather than rewritten from
-- scratch, and so a stale recap is never mistaken for a current one.
--
-- Nothing here is authoritative: the recap is conversational memory only.
-- Facts are always re-fetched through tools, and a pending approval is
-- re-injected from assistant_pending_actions rather than from the recap, so a
-- summarisation defect can never cause the wrong action to be approved.

ALTER TABLE assistant_conversations
    ADD COLUMN IF NOT EXISTS summary TEXT NOT NULL DEFAULT '';

ALTER TABLE assistant_conversations
    ADD COLUMN IF NOT EXISTS summarized_seq INTEGER NOT NULL DEFAULT 0;
