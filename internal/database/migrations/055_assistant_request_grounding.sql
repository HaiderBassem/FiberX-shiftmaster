-- 055_assistant_request_grounding.sql
--
-- Two columns that make the assistant's most important property observable in
-- production rather than only in the test suite.
--
-- tool_free records that the model answered a turn having explicitly decided it
-- needed no data from ShiftMaster. That is correct for a greeting and wrong for
-- "what's my shift" — and the difference is invisible in a transcript, because
-- an invented answer and a looked-up one read identically. A rising tool_free
-- rate is the earliest signal that the model has started answering from its
-- priors, which is the failure that would quietly make the assistant untrue.
--
-- rounds is how many model↔tool iterations a turn took. Turns that consistently
-- hit the round limit mean the tool catalogue is missing something the model
-- keeps reaching for.
--
-- Neither column contains message content. The request log deliberately never
-- has: transcripts live in assistant_messages under their owner's access.

ALTER TABLE assistant_requests
    ADD COLUMN IF NOT EXISTS tool_free BOOLEAN NOT NULL DEFAULT false;

ALTER TABLE assistant_requests
    ADD COLUMN IF NOT EXISTS rounds INTEGER NOT NULL DEFAULT 0;
