-- Categories are a tree inside one space, at most 3 levels deep (FR-2). The
-- depth cap is enforced in app/category, not here: MySQL cannot express "the
-- parent's parent has no parent" as a constraint.
--
-- `order` is the position among siblings, renumbered 0..n-1 in one transaction
-- whenever the list is reordered — half-renumbered rows leave two categories
-- sharing a position.
CREATE TABLE categories (
    id CHAR(36) NOT NULL PRIMARY KEY,
    space_id CHAR(36) NOT NULL,
    parent_id CHAR(36) NULL,
    name VARCHAR(255) NOT NULL,
    slug VARCHAR(255) NOT NULL,
    `order` INT NOT NULL DEFAULT 0,
    created_at TIMESTAMP NULL,
    updated_at TIMESTAMP NULL,
    UNIQUE KEY categories_space_slug_unique (space_id, slug),
    KEY categories_space_parent_order_index (space_id, parent_id, `order`),
    CONSTRAINT categories_space_id_foreign FOREIGN KEY (space_id) REFERENCES spaces (id) ON DELETE CASCADE,
    CONSTRAINT categories_parent_id_foreign FOREIGN KEY (parent_id) REFERENCES categories (id) ON DELETE CASCADE
);
