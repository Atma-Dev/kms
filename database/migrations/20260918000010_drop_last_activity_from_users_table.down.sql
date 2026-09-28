-- Restores the column, not its values: nothing was reading them.
ALTER TABLE users ADD COLUMN last_activity TIMESTAMP NULL AFTER phone_number;
