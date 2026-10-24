-- Categories are a tree inside one space, at most 3 levels deep (FR-2). The
-- depth cap is enforced in app/category, not here: a CHECK cannot look at
-- other rows, so "the parent's parent has no parent" is not a constraint.
--
-- "order" is the position among siblings, renumbered 0..n-1 in one transaction
-- whenever the list is reordered — half-renumbered rows leave two categories
-- sharing a position. It is a reserved word, so every raw query quotes it.
CREATE TABLE categories (
    id UUID NOT NULL PRIMARY KEY,
    space_id UUID NOT NULL,
    parent_id UUID NULL,
    name VARCHAR(255) NOT NULL,
    slug VARCHAR(255) NOT NULL,
    "order" INT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NULL,
    updated_at TIMESTAMPTZ NULL,
    CONSTRAINT categories_space_slug_unique UNIQUE (space_id, slug),
    CONSTRAINT categories_space_id_foreign FOREIGN KEY (space_id) REFERENCES spaces (id) ON DELETE CASCADE,
    CONSTRAINT categories_parent_id_foreign FOREIGN KEY (parent_id) REFERENCES categories (id) ON DELETE CASCADE
);

CREATE INDEX categories_space_parent_order_index ON categories (space_id, parent_id, "order");
CREATE INDEX categories_parent_id_index ON categories (parent_id);
