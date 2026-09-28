-- New table owned by this app.
--
-- Backs the in-app notification feed: one row per event delivered to one
-- user (e.g. a tagihan the user's own payment moved to "paid"). data is a
-- free-form JSON payload for whatever the frontend needs to deep-link into
-- (invoice_number, old/new status, ...) without a column per notification
-- type.
CREATE TABLE notifications (
    id CHAR(36) NOT NULL PRIMARY KEY,
    user_id CHAR(36) NOT NULL,
    type VARCHAR(50) NOT NULL,
    title VARCHAR(255) NOT NULL,
    body VARCHAR(500) NOT NULL,
    data JSON NULL,
    read_at TIMESTAMP NULL,
    created_at TIMESTAMP NULL,
    updated_at TIMESTAMP NULL,
    INDEX notifications_user_id_created_at_index (user_id, created_at),
    INDEX notifications_user_id_read_at_index (user_id, read_at),
    CONSTRAINT notifications_user_id_foreign FOREIGN KEY (user_id) REFERENCES users (id)
);
