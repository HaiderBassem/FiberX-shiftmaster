-- 054_knowledge_search_index.sql
--
-- Ranked full-text search over the knowledge base, for the assistant.
--
-- The problem this solves: knowledge search was a single ILIKE over the whole
-- query string. That works when a person types a keyword into a search box and
-- fails completely when someone asks the assistant a question — "شلون أطلب
-- إجازة إذا عندي شفت ليلي؟" is not a substring of any document ever written,
-- so the assistant retrieved nothing and had to say it did not know, about
-- policies that were sitting right there in the Info Bank.
--
-- The fix is ordinary PostgreSQL: index each document as a tsvector and rank
-- matches by how many of the question's terms appear and how densely. No
-- extension is required (the 'simple' configuration tokenises on whitespace and
-- punctuation, which is what Arabic needs — there is no Arabic stemmer here and
-- stemming Arabic badly would be worse than not stemming it), no external
-- embedding service is involved, and nothing leaves the database.
--
-- Weighting: a term in the title (weight A) outranks the same term buried in a
-- long document (weight B), which is what makes "leave policy" find the
-- document called "Leave policy" rather than the one that mentions leave once
-- in passing.
--
-- Vector search was deliberately not used. At this corpus size term ranking
-- answers the questions people actually ask, and an embedding model would add
-- a second model to load, a second thing to keep in memory, and a copy of every
-- document in a form nobody can audit.

CREATE INDEX IF NOT EXISTS idx_help_documents_fts
    ON help_documents
    USING GIN ((
        setweight(to_tsvector('simple'::regconfig, coalesce(title, '')), 'A') ||
        setweight(to_tsvector('simple'::regconfig, coalesce(content, '')), 'B')
    ));

CREATE INDEX IF NOT EXISTS idx_fiberx_data_fts
    ON fiberx_data
    USING GIN ((
        setweight(to_tsvector('simple'::regconfig, coalesce(title, '')), 'A') ||
        setweight(to_tsvector('simple'::regconfig, coalesce(content, '')), 'B')
    ));
