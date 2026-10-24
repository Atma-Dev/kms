-- New table owned by this app.
--
-- queue_jobs is the durable source of truth for a job's *current* state.
-- Every attempt is additionally written to job_logs with type='queue' (see
-- 20260907000001_create_job_logs_table), so the run-history page shows both
-- scheduled and queue runs from one table while this one only ever holds
-- what a worker still needs to act on.
--
-- TIMESTAMP(6): plain TIMESTAMP rounds to whole seconds, which breaks the
-- `available_at <= NOW()` claim query for jobs dispatched mid-second and
-- makes durations always compute as multiples of 1000ms — queue attempts
-- routinely finish in well under a second.
--
-- Requires MySQL 8.0+: Repository.Claim uses SELECT ... FOR UPDATE SKIP
-- LOCKED so concurrent workers never fight over the same row.
CREATE TABLE queue_jobs (
    id CHAR(36) NOT NULL PRIMARY KEY,
    queue VARCHAR(50) NOT NULL DEFAULT 'default',
    name VARCHAR(100) NOT NULL,
    payload JSON NULL,
    status VARCHAR(20) NOT NULL,
    attempts INT NOT NULL DEFAULT 0,
    max_attempts INT NOT NULL DEFAULT 3,
    available_at TIMESTAMP(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    reserved_at TIMESTAMP(6) NULL,
    started_at TIMESTAMP(6) NULL,
    finished_at TIMESTAMP(6) NULL,
    last_error TEXT NULL,
    dispatched_by CHAR(36) NULL,
    created_at TIMESTAMP(6) NULL,
    updated_at TIMESTAMP(6) NULL,
    INDEX queue_jobs_claim_index (status, queue, available_at),
    INDEX queue_jobs_status_created_at_index (status, created_at),
    INDEX queue_jobs_name_index (name)
);
