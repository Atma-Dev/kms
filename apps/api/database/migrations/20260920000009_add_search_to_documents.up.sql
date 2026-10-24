-- Modul 3 FR-1: the PRD's tsvector, setweight and GIN index.
--
-- content is JSON (a TipTap node tree), so search_body carries the flattened
-- plain text. It is filled in Go by document.PlainText alongside Excerpt, in
-- the same Save — which is what the NFR's "ter-update dalam transaksi yang
-- sama" asks for. search_body also feeds search.Snippet, which needs the
-- original words rather than the lexemes a tsvector keeps.
--
-- search_vector is generated from title (weight A) and search_body (weight
-- C), so Postgres keeps it in step with every write and no code path can
-- forget it. The 'simple' configuration: the content is mostly Indonesian,
-- which Postgres has no stemmer for, and an English stemmer would mangle it.
-- 'simple' lowercases and keeps every word, short ones included.
ALTER TABLE documents ADD COLUMN search_body TEXT NULL;

-- Backfill so existing documents are searchable on day one instead of only
-- after someone re-saves them: every "text" node in the tree, at any depth.
-- Every document gets exact text from document.PlainText on its next save.
UPDATE documents
SET search_body = NULLIF(TRIM((
    SELECT string_agg(node #>> '{}', ' ')
    FROM jsonb_path_query(content, 'strict $.**.text') AS node
)), '')
WHERE search_body IS NULL;

ALTER TABLE documents ADD COLUMN search_vector TSVECTOR GENERATED ALWAYS AS (
    setweight(to_tsvector('simple', COALESCE(title, '')), 'A') ||
    setweight(to_tsvector('simple', COALESCE(search_body, '')), 'C')
) STORED;

CREATE INDEX documents_search_vector_index ON documents USING GIN (search_vector);
