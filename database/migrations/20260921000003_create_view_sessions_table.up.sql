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
    id CHAR(36) NOT NULL PRIMARY KEY,
    document_id CHAR(36) NOT NULL,
    created_by CHAR(36) NOT NULL,
    label VARCHAR(255) NULL,
    token_hash CHAR(64) NOT NULL,
    expires_at TIMESTAMP NOT NULL,
    created_at TIMESTAMP NULL,
    UNIQUE KEY view_sessions_token_hash_unique (token_hash),
    KEY view_sessions_document_index (document_id),
    KEY view_sessions_expires_at_index (expires_at),
    CONSTRAINT view_sessions_document_id_foreign FOREIGN KEY (document_id) REFERENCES documents (id) ON DELETE CASCADE,
    CONSTRAINT view_sessions_created_by_foreign FOREIGN KEY (created_by) REFERENCES users (id) ON DELETE CASCADE
);
