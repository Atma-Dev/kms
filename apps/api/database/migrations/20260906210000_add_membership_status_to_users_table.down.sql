ALTER TABLE users
    DROP INDEX users_membership_status_index,
    DROP INDEX users_sso_id_unique,
    DROP COLUMN synced_at,
    DROP COLUMN deactivated_at,
    DROP COLUMN membership_status;
ALTER TABLE users ADD INDEX users_sso_id_index (sso_id);
