-- Mirrors ng-store-operation's design exactly (2026_08_12_000000_add_
-- membership_status_to_users_table): membership_status/deactivated_at/
-- synced_at is sso's authorization decision (see app/membership), kept
-- separate from is_active, which is this app's own admin-controlled
-- operational toggle (gates deposit actions) and was already doing that
-- unrelated job.
--
-- Existing rows default ACTIVE — they can already log in today, making them
-- INACTIVE would lock out everyone sso has never touched yet.
--
-- UNIQUE on sso_id: without it, two concurrent membership pushes for the same
-- person could each miss the other's insert and create two rows. NULLs are
-- exempt from MySQL's unique check, so rows with no sso_id yet are
-- unaffected.
ALTER TABLE users DROP INDEX users_sso_id_index;
ALTER TABLE users
    ADD COLUMN membership_status ENUM('ACTIVE', 'INACTIVE') NOT NULL DEFAULT 'ACTIVE' AFTER sso_id,
    ADD COLUMN deactivated_at TIMESTAMP NULL AFTER membership_status,
    ADD COLUMN synced_at TIMESTAMP NULL AFTER deactivated_at,
    ADD UNIQUE KEY users_sso_id_unique (sso_id),
    ADD INDEX users_membership_status_index (membership_status);
