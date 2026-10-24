-- A space is the top-level container for knowledge: one per team, product or
-- domain. Everything else in the content module hangs off one.
--
-- is_archived rather than a delete: a space holds documents, and removing the
-- row would orphan or cascade-destroy every one of them. Archiving hides it
-- from the pickers while leaving its documents readable.
CREATE TABLE spaces (
    id UUID NOT NULL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    slug VARCHAR(255) NOT NULL,
    description TEXT NULL,
    icon VARCHAR(64) NULL,
    is_archived BOOLEAN NOT NULL DEFAULT FALSE,
    created_by UUID NOT NULL,
    created_at TIMESTAMPTZ NULL,
    updated_at TIMESTAMPTZ NULL,
    CONSTRAINT spaces_slug_unique UNIQUE (slug),
    CONSTRAINT spaces_created_by_foreign FOREIGN KEY (created_by) REFERENCES users (id)
);

CREATE INDEX spaces_is_archived_index ON spaces (is_archived);
