-- Modul 3 FR-1. The PRD specifies a Postgres tsvector; this codebase is MySQL,
-- so the equivalent is a plain text column plus InnoDB FULLTEXT indexes.
--
-- content is JSON (a TipTap node tree) and MySQL cannot index it as text, so
-- search_body carries the flattened plain text. It is filled in Go by
-- document.PlainText alongside Excerpt, in the same Save — which is what the
-- NFR's "ter-update dalam transaksi yang sama" asks for, without a trigger.
--
-- Two indexes rather than one combined column: relevance weights title above
-- body (PRD: setweight 'A' vs 'C'), and MySQL scores one index at a time, so
-- the weighting is MATCH(title)*3 + MATCH(search_body) at query time.
ALTER TABLE documents ADD COLUMN search_body MEDIUMTEXT NULL AFTER excerpt;

-- Backfill so existing documents are searchable on day one instead of only
-- after someone re-saves them. '$**.text' collects every text node in the tree
-- at any depth; the REPLACEs turn the resulting JSON array into text.
--
-- Every structural character is replaced with a space rather than the exact
-- `", "` separator being matched: MySQL renders a JSON array with its own
-- spacing, and a backfill that depends on that spacing silently leaves the
-- punctuation in the index when it changes. A full-text index tokenises on
-- whitespace, so extra spaces cost nothing and a missed quote costs a word.
--
-- Approximate on purpose: escape sequences (\n, \") stay encoded here. Every
-- document gets exact text from document.PlainText on its next save, and a
-- stray backslash costs at most one token.
UPDATE documents
SET search_body = NULLIF(TRIM(
    REPLACE(REPLACE(REPLACE(REPLACE(
        COALESCE(JSON_EXTRACT(content, '$**.text'), '[]'),
    '"', ' '), '[', ' '), ']', ' '), ',', ' ')), '')
WHERE search_body IS NULL;

ALTER TABLE documents ADD FULLTEXT KEY documents_title_fulltext (title);
ALTER TABLE documents ADD FULLTEXT KEY documents_search_body_fulltext (search_body);
