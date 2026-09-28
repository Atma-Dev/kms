-- Every reach for a document's content, recorded for forensics (Modul 4 FR-4).
--
-- Deliberately NOT the same table as document_views (Modul 3). That one is the
-- detail behind documents.view_count and is written under a once-per-hour
-- throttle, so the number and the rows agree. This one is throttled by nothing:
-- an investigation asks "who opened this, how often, from where", and a table
-- that drops 59 minutes of every hour cannot answer it.
--
-- user_id is ON DELETE SET NULL, not CASCADE as in document_views. A view
-- count survives its reader because the aggregate already holds it; an access
-- log survives its reader because the row IS the evidence, and removing an
-- account must not be a way to erase the trail behind it.
--
-- Nothing in the application updates or deletes a row here: there are no
-- write routes but the append, which is what "append-only" means in FR-4.
-- MySQL is not asked to enforce it — a trigger that refuses UPDATE still
-- yields to anyone holding the credentials that could drop the trigger.
--
-- ponytail: no partitioning and no retention job; one row per document open.
-- Ceiling is roughly a million rows a year per thousand daily readers, which
-- InnoDB handles on the two indexes below. Add monthly partitions or a
-- summarise-then-prune job (app/scheduler already has the shape) when the
-- table outgrows the admin queries that read it.
CREATE TABLE document_access_logs (
    id CHAR(36) NOT NULL PRIMARY KEY,
    document_id CHAR(36) NOT NULL,
    user_id CHAR(36) NULL,
    action VARCHAR(16) NOT NULL,
    ip_address VARCHAR(45) NULL,
    user_agent VARCHAR(512) NULL,
    created_at TIMESTAMP NOT NULL,
    KEY document_access_logs_document_created_index (document_id, created_at),
    KEY document_access_logs_user_created_index (user_id, created_at),
    CONSTRAINT document_access_logs_document_id_foreign FOREIGN KEY (document_id) REFERENCES documents (id) ON DELETE CASCADE,
    CONSTRAINT document_access_logs_user_id_foreign FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE SET NULL
);
