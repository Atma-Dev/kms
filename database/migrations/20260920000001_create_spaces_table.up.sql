-- A space is the top-level container for knowledge: one per team, product or
-- domain. Everything else in the content module hangs off one.
--
-- is_archived rather than a delete: a space holds documents, and removing the
-- row would orphan or cascade-destroy every one of them. Archiving hides it
-- from the pickers while leaving its documents readable.
CREATE TABLE spaces (
    id CHAR(36) NOT NULL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    slug VARCHAR(255) NOT NULL,
    description TEXT NULL,
    icon VARCHAR(64) NULL,
    is_archived TINYINT(1) NOT NULL DEFAULT 0,
    created_by CHAR(36) NOT NULL,
    created_at TIMESTAMP NULL,
    updated_at TIMESTAMP NULL,
    UNIQUE KEY spaces_slug_unique (slug),
    KEY spaces_is_archived_index (is_archived),
    CONSTRAINT spaces_created_by_foreign FOREIGN KEY (created_by) REFERENCES users (id)
);
