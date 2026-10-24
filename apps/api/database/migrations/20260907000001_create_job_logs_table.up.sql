-- New table owned by this app.
--
-- One row per background job run — scheduled today, queue jobs later (see
-- `type`). Not a merge of activity_logs: that table is HTTP-request shaped
-- (method/path/status_code/ip), which a background run has none of.
CREATE TABLE job_logs (
    id CHAR(36) NOT NULL PRIMARY KEY,
    type VARCHAR(20) NOT NULL,
    name VARCHAR(100) NOT NULL,
    status VARCHAR(20) NOT NULL,
    message TEXT NULL,
    started_at TIMESTAMP NOT NULL,
    finished_at TIMESTAMP NULL,
    created_at TIMESTAMP NULL,
    updated_at TIMESTAMP NULL,
    INDEX job_logs_type_name_started_at_index (type, name, started_at)
);
