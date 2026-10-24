-- New table owned by this app.
--
-- A row per registered scheduler.Job (kept in sync at startup by
-- Runner.Start -> Repository.Sync). The code decides what jobs exist; this
-- table decides when they run, whether they're enabled, and — via the
-- locked_at/locked_by lease columns — which instance is currently running
-- one, so two replicas never run the same tick twice.
CREATE TABLE scheduled_jobs (
    id CHAR(36) NOT NULL PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    cron_expression VARCHAR(100) NOT NULL,
    enabled TINYINT(1) NOT NULL DEFAULT 1,
    last_tick_at TIMESTAMP(6) NULL,
    locked_at TIMESTAMP(6) NULL,
    locked_by CHAR(36) NULL,
    last_run_at TIMESTAMP(6) NULL,
    last_status VARCHAR(20) NULL,
    last_message TEXT NULL,
    last_duration_ms BIGINT NULL,
    created_at TIMESTAMP(6) NULL,
    updated_at TIMESTAMP(6) NULL,
    UNIQUE INDEX scheduled_jobs_name_unique (name)
);
