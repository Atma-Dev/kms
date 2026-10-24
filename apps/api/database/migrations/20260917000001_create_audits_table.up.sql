-- New table owned by this app. Written by app/audit.GormStorer, one row per
-- audited action (a mutation an app/* action explicitly opts into recording,
-- via github.com/iqbalatma/go-audit). Distinct from activity_logs, which is
-- HTTP-request shaped (method/path/status_code) and logs every mutating
-- request automatically — this is field-change shaped and only exists where
-- a call site chose to record one.
CREATE TABLE audits (
    id UUID NOT NULL PRIMARY KEY,
    action VARCHAR(255) NULL,
    message VARCHAR(500) NULL,
    app_name VARCHAR(255) NULL,
    actor_table VARCHAR(255) NULL,
    actor_id VARCHAR(255) NULL,
    actor_name VARCHAR(255) NULL,
    actor_email VARCHAR(255) NULL,
    actor_phone VARCHAR(255) NULL,
    ip_address VARCHAR(45) NULL,
    method VARCHAR(10) NULL,
    endpoint VARCHAR(255) NULL,
    user_agent VARCHAR(500) NULL,
    entry_object_table VARCHAR(255) NULL,
    entry_object_id VARCHAR(255) NULL,
    tag JSONB NULL,
    additional JSONB NULL,
    created_at TIMESTAMPTZ NULL,
    updated_at TIMESTAMPTZ NULL
);

-- "show full history of this record" — the most common query against this table
CREATE INDEX audits_entry_object_index ON audits (entry_object_table, entry_object_id);
-- "what did this user do"
CREATE INDEX audits_actor_index ON audits (actor_table, actor_id);
-- time-range filtering / pagination / retention cleanup
CREATE INDEX audits_created_at_index ON audits (created_at);
