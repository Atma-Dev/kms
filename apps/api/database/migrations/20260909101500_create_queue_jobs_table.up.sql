-- New table owned by this app.
--
-- queue_jobs is the durable source of truth for a job's *current* state.
-- Every attempt is additionally written to job_logs with type='queue' (see
-- 20260907000001_create_job_logs_table), so the run-history page shows both
-- scheduled and queue runs from one table while this one only ever holds
-- what a worker still needs to act on.
--
-- Postgres TIMESTAMPTZ already keeps microseconds, which the claim query
-- (`available_at <= NOW()`) and the sub-second durations need.
--
-- Repository.Claim uses SELECT ... FOR UPDATE SKIP LOCKED so concurrent
-- workers never fight over the same row.
CREATE TABLE queue_jobs (
    id UUID NOT NULL PRIMARY KEY,
    queue VARCHAR(50) NOT NULL DEFAULT 'default',
    name VARCHAR(100) NOT NULL,
    payload JSONB NULL,
    status VARCHAR(20) NOT NULL,
    attempts INT NOT NULL DEFAULT 0,
    max_attempts INT NOT NULL DEFAULT 3,
    available_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    reserved_at TIMESTAMPTZ NULL,
    started_at TIMESTAMPTZ NULL,
    finished_at TIMESTAMPTZ NULL,
    last_error TEXT NULL,
    dispatched_by UUID NULL,
    created_at TIMESTAMPTZ NULL,
    updated_at TIMESTAMPTZ NULL
);

CREATE INDEX queue_jobs_claim_index ON queue_jobs (status, queue, available_at);
CREATE INDEX queue_jobs_status_created_at_index ON queue_jobs (status, created_at);
CREATE INDEX queue_jobs_name_index ON queue_jobs (name);
