-- One snapshot of a document's content, taken before an edit overwrites it
-- (FR-5). Only documents that have been published are versioned: a draft that
-- was never published has no reader to protect, so its edits are not snapshot.
--
-- version_number is per document and starts at 1, so "v3" means the same thing
-- in the UI as it does here. The unique key is what makes a concurrent second
-- edit fail rather than silently reuse a number.
CREATE TABLE document_versions (
    id CHAR(36) NOT NULL PRIMARY KEY,
    document_id CHAR(36) NOT NULL,
    version_number INT UNSIGNED NOT NULL,
    content JSON NOT NULL,
    editor_id CHAR(36) NULL,
    change_note VARCHAR(500) NULL,
    created_at TIMESTAMP NULL,
    UNIQUE KEY document_versions_document_number_unique (document_id, version_number),
    CONSTRAINT document_versions_document_id_foreign FOREIGN KEY (document_id) REFERENCES documents (id) ON DELETE CASCADE,
    CONSTRAINT document_versions_editor_id_foreign FOREIGN KEY (editor_id) REFERENCES users (id)
);
