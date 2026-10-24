DROP INDEX IF EXISTS users_membership_status_index;
ALTER TABLE users
    DROP CONSTRAINT users_sso_id_unique,
    DROP COLUMN synced_at,
    DROP COLUMN deactivated_at,
    DROP COLUMN membership_status;
CREATE INDEX users_sso_id_index ON users (sso_id);
