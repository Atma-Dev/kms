-- Files attached to a document (FR-7). file_path is relative to the storage
-- root and always starts with the space id, so a space's files can be found,
-- backed up or removed as one unit without consulting the database.
--
-- `kind` separates the two upload paths that share this table: `attachment` is
-- a listed download, `inline` is an image embedded in the content (FR-8).
-- Inline images have no place in the attachment list — they are already
-- visible in the document — but they need the same row to be cleaned up.
CREATE TABLE attachments (
    id UUID NOT NULL PRIMARY KEY,
    document_id UUID NULL,
    kind VARCHAR(16) NOT NULL DEFAULT 'attachment',
    file_path VARCHAR(512) NOT NULL,
    file_name VARCHAR(255) NOT NULL,
    file_type VARCHAR(128) NOT NULL,
    file_size BIGINT NOT NULL CHECK (file_size >= 0),
    uploaded_by UUID NOT NULL,
    created_at TIMESTAMPTZ NULL,
    updated_at TIMESTAMPTZ NULL,
    CONSTRAINT attachments_document_id_foreign FOREIGN KEY (document_id) REFERENCES documents (id) ON DELETE CASCADE,
    CONSTRAINT attachments_uploaded_by_foreign FOREIGN KEY (uploaded_by) REFERENCES users (id)
);

CREATE INDEX attachments_document_kind_index ON attachments (document_id, kind);
