-- New table owned by this app.
--
-- One row per background job run — scheduled today, queue jobs later (see
-- `type`). Not a merge of activity_logs: that table is HTTP-request shaped
-- (method/path/status_code/ip), which a background run has none of.
CREATE TABLE job_logs (
    id UUID NOT NULL PRIMARY KEY,
    type VARCHAR(20) NOT NULL,
    name VARCHAR(100) NOT NULL,
    status VARCHAR(20) NOT NULL,
    message TEXT NULL,
    started_at TIMESTAMPTZ NOT NULL,
    finished_at TIMESTAMPTZ NULL,
    created_at TIMESTAMPTZ NULL,
    updated_at TIMESTAMPTZ NULL
);

CREATE INDEX job_logs_type_name_started_at_index ON job_logs (type, name, started_at);
