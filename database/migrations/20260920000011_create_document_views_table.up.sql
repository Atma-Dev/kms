-- One row per counted read, written beside the documents.view_count increment
-- and under the same once-per-hour throttle — the two never disagree about
-- what a view is.
--
-- view_count stays on documents and is what "terpopuler" sorts on: a COUNT
-- over this table on every search would scan every read ever recorded to
-- order one page. This table is the detail behind that number — who read what
-- and when — which the analytics module reads later.
--
-- user_id cascades on delete: a deleted user's reads are their personal
-- history, and the aggregate that survives them is already in view_count.
CREATE TABLE document_views (
    id CHAR(36) NOT NULL PRIMARY KEY,
    document_id CHAR(36) NOT NULL,
    user_id CHAR(36) NOT NULL,
    viewed_at TIMESTAMP NOT NULL,
    KEY document_views_document_viewed_index (document_id, viewed_at),
    KEY document_views_user_viewed_index (user_id, viewed_at),
    CONSTRAINT document_views_document_id_foreign FOREIGN KEY (document_id) REFERENCES documents (id) ON DELETE CASCADE,
    CONSTRAINT document_views_user_id_foreign FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE
);
