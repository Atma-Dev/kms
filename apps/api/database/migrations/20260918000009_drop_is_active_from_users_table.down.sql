-- Restores the column, not its values: they were never read, and the job that
-- maintained them is gone.
ALTER TABLE users ADD COLUMN is_active TINYINT(1) NOT NULL DEFAULT 0 AFTER phone_number;
