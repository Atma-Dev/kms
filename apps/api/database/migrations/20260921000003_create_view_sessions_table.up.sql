-- A time-limited link to one document (Modul 4 FR-5).
--
-- The ordinary viewer needs no row here: a logged-in reader is already
-- identified by their JWT, and the link they use is valid exactly as long as
-- their session is. This table is the other half of FR-5 — a link handed to
-- someone who has no account, which must therefore carry its own credential
-- and its own expiry.
--
-- token_hash, not token: the token travels in a URL, which ends up in browser
-- history, a referer header and whatever the recipient pastes it into. Storing
-- the SHA-256 means a copy of this table is not a set of working links.
--
-- created_by survives the document it points at being restricted further: it
-- is who is answerable for the share, and the access log rows the link
-- produces carry no user of their own.
CREATE TABLE view_sessions (
    id UUID NOT NULL PRIMARY KEY,
    document_id UUID NOT NULL,
    created_by UUID NOT NULL,
    label VARCHAR(255) NULL,
    token_hash CHAR(64) NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NULL,
    CONSTRAINT view_sessions_token_hash_unique UNIQUE (token_hash),
    CONSTRAINT view_sessions_document_id_foreign FOREIGN KEY (document_id) REFERENCES documents (id) ON DELETE CASCADE,
    CONSTRAINT view_sessions_created_by_foreign FOREIGN KEY (created_by) REFERENCES users (id) ON DELETE CASCADE
);

CREATE INDEX view_sessions_document_index ON view_sessions (document_id);
CREATE INDEX view_sessions_expires_at_index ON view_sessions (expires_at);
