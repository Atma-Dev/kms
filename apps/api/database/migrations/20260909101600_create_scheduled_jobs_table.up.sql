-- New table owned by this app.
--
-- A row per registered scheduler.Job (kept in sync at startup by
-- Runner.Start -> Repository.Sync). The code decides what jobs exist; this
-- table decides when they run, whether they're enabled, and — via the
-- locked_at/locked_by lease columns — which instance is currently running
-- one, so two replicas never run the same tick twice.
CREATE TABLE scheduled_jobs (
    id UUID NOT NULL PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    cron_expression VARCHAR(100) NOT NULL,
    enabled BOOLEAN NOT NULL DEFAULT TRUE,
    last_tick_at TIMESTAMPTZ NULL,
    locked_at TIMESTAMPTZ NULL,
    locked_by UUID NULL,
    last_run_at TIMESTAMPTZ NULL,
    last_status VARCHAR(20) NULL,
    last_message TEXT NULL,
    last_duration_ms BIGINT NULL,
    created_at TIMESTAMPTZ NULL,
    updated_at TIMESTAMPTZ NULL,
    CONSTRAINT scheduled_jobs_name_unique UNIQUE (name)
);
