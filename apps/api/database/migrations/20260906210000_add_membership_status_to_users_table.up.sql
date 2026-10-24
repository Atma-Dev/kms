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
-- distinct in a Postgres unique index, so rows with no sso_id yet are
-- unaffected.
--
-- A CHECK rather than a Postgres ENUM type: adding a status to an enum type
-- is its own ALTER TYPE that cannot run inside a transaction on older
-- servers, and a constraint is one line to replace.
DROP INDEX users_sso_id_index;
ALTER TABLE users
    ADD COLUMN membership_status VARCHAR(8) NOT NULL DEFAULT 'ACTIVE'
        CONSTRAINT users_membership_status_check CHECK (membership_status IN ('ACTIVE', 'INACTIVE')),
    ADD COLUMN deactivated_at TIMESTAMPTZ NULL,
    ADD COLUMN synced_at TIMESTAMPTZ NULL,
    ADD CONSTRAINT users_sso_id_unique UNIQUE (sso_id);
CREATE INDEX users_membership_status_index ON users (membership_status);
