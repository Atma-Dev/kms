-- Tags are global, not per space (FR-6): a tag is a word, and the same word
-- meaning two things in two spaces is a tagging problem, not a schema one.
-- Users create them freely, so `name` is what they typed and `slug` is what
-- makes "Deployment" and "deployment" the same tag.
CREATE TABLE tags (
    id UUID NOT NULL PRIMARY KEY,
    name VARCHAR(64) NOT NULL,
    slug VARCHAR(64) NOT NULL,
    created_at TIMESTAMPTZ NULL,
    updated_at TIMESTAMPTZ NULL,
    CONSTRAINT tags_slug_unique UNIQUE (slug)
);
