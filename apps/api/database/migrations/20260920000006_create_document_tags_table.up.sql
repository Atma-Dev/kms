-- The join. Composite primary key rather than a surrogate id: the pair is the
-- identity, and it makes tagging the same document twice impossible instead of
-- merely unusual.
CREATE TABLE document_tags (
    document_id UUID NOT NULL,
    tag_id UUID NOT NULL,
    PRIMARY KEY (document_id, tag_id),
    CONSTRAINT document_tags_document_id_foreign FOREIGN KEY (document_id) REFERENCES documents (id) ON DELETE CASCADE,
    CONSTRAINT document_tags_tag_id_foreign FOREIGN KEY (tag_id) REFERENCES tags (id) ON DELETE CASCADE
);

CREATE INDEX document_tags_tag_index ON document_tags (tag_id);
