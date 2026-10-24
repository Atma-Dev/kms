-- Kept so the version sequence matches the MySQL history. On MySQL this
-- widened job_logs to TIMESTAMP(6); Postgres TIMESTAMPTZ already stores
-- microseconds, so there is nothing left to change.
SELECT 1;
