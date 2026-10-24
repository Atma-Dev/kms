-- The document itself. content is JSON (TipTap/ProseMirror), never raw HTML:
-- a node tree diffs per-node between versions and can be re-rendered by any
-- editor, which a blob of markup cannot.
--
-- slug is unique per space, not globally — two spaces may each have an
-- "onboarding" document, and a global unique key would make the second one's
-- slug depend on which space happened to be written first.
--
-- deleted_at is a soft delete (FR-3): the row stays so the audit trail keeps
-- pointing at something real.
--
-- current_version_id carries no foreign key on purpose. documents and
-- document_versions reference each other, and a FK both ways cannot be
-- satisfied by either insert order.
CREATE TABLE documents (
    id CHAR(36) NOT NULL PRIMARY KEY,
    space_id CHAR(36) NOT NULL,
    category_id CHAR(36) NULL,
    title VARCHAR(255) NOT NULL,
    slug VARCHAR(255) NOT NULL,
    content JSON NOT NULL,
    excerpt VARCHAR(500) NULL,
    status VARCHAR(16) NOT NULL DEFAULT 'draft',
    current_version_id CHAR(36) NULL,
    author_id CHAR(36) NOT NULL,
    last_editor_id CHAR(36) NULL,
    published_at TIMESTAMP NULL,
    view_count INT UNSIGNED NOT NULL DEFAULT 0,
    is_pinned TINYINT(1) NOT NULL DEFAULT 0,
    deleted_at TIMESTAMP NULL,
    created_at TIMESTAMP NULL,
    updated_at TIMESTAMP NULL,
    UNIQUE KEY documents_space_slug_unique (space_id, slug),
    KEY documents_space_status_index (space_id, status),
    KEY documents_category_index (category_id),
    KEY documents_author_index (author_id),
    KEY documents_deleted_at_index (deleted_at),
    CONSTRAINT documents_space_id_foreign FOREIGN KEY (space_id) REFERENCES spaces (id),
    CONSTRAINT documents_category_id_foreign FOREIGN KEY (category_id) REFERENCES categories (id) ON DELETE SET NULL,
    CONSTRAINT documents_author_id_foreign FOREIGN KEY (author_id) REFERENCES users (id),
    CONSTRAINT documents_last_editor_id_foreign FOREIGN KEY (last_editor_id) REFERENCES users (id)
);
