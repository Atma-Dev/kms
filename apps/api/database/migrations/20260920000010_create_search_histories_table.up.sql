-- Modul 3 FR-5. One row per user per distinct query, not one per search:
-- the list is "what did I look for recently", and the same query typed twice
-- is one entry that moved to the top — which is also what gives
-- DELETE /search/history/{id} a stable row to point at.
--
-- The unique key is what makes that an upsert (ON DUPLICATE KEY UPDATE
-- updated_at) rather than a read-then-write two requests can both win.
-- updated_at is "last searched"; created_at stays "first searched".
--
-- Trimmed nightly by app/search's PruneHistoriesJob to
-- SEARCH_HISTORY_KEEP_PER_USER rows per account: the list only ever reads the
-- newest ten, and nothing else removes these rows — they belong to a user
-- rather than to a document, so no cascade reaches them while the account
-- lives.
CREATE TABLE search_histories (
    id CHAR(36) NOT NULL PRIMARY KEY,
    user_id CHAR(36) NOT NULL,
    query VARCHAR(255) NOT NULL,
    created_at TIMESTAMP NULL,
    updated_at TIMESTAMP NULL,
    UNIQUE KEY search_histories_user_query_unique (user_id, query),
    KEY search_histories_user_updated_index (user_id, updated_at),
    CONSTRAINT search_histories_user_id_foreign FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE
);
