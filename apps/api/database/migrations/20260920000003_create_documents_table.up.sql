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
    id UUID NOT NULL PRIMARY KEY,
    space_id UUID NOT NULL,
    category_id UUID NULL,
    title VARCHAR(255) NOT NULL,
    slug VARCHAR(255) NOT NULL,
    content JSONB NOT NULL,
    excerpt VARCHAR(500) NULL,
    status VARCHAR(16) NOT NULL DEFAULT 'draft',
    current_version_id UUID NULL,
    author_id UUID NOT NULL,
    last_editor_id UUID NULL,
    published_at TIMESTAMPTZ NULL,
    view_count INT NOT NULL DEFAULT 0 CHECK (view_count >= 0),
    is_pinned BOOLEAN NOT NULL DEFAULT FALSE,
    deleted_at TIMESTAMPTZ NULL,
    created_at TIMESTAMPTZ NULL,
    updated_at TIMESTAMPTZ NULL,
    CONSTRAINT documents_space_slug_unique UNIQUE (space_id, slug),
    CONSTRAINT documents_space_id_foreign FOREIGN KEY (space_id) REFERENCES spaces (id),
    CONSTRAINT documents_category_id_foreign FOREIGN KEY (category_id) REFERENCES categories (id) ON DELETE SET NULL,
    CONSTRAINT documents_author_id_foreign FOREIGN KEY (author_id) REFERENCES users (id),
    CONSTRAINT documents_last_editor_id_foreign FOREIGN KEY (last_editor_id) REFERENCES users (id)
);

CREATE INDEX documents_space_status_index ON documents (space_id, status);
CREATE INDEX documents_category_index ON documents (category_id);
CREATE INDEX documents_author_index ON documents (author_id);
CREATE INDEX documents_deleted_at_index ON documents (deleted_at);
