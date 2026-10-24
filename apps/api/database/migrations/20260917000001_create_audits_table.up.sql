-- New table owned by this app. Written by app/audit.GormStorer, one row per
-- audited action (a mutation an app/* action explicitly opts into recording,
-- via github.com/iqbalatma/go-audit). Distinct from activity_logs, which is
-- HTTP-request shaped (method/path/status_code) and logs every mutating
-- request automatically — this is field-change shaped and only exists where
-- a call site chose to record one.
CREATE TABLE audits (
    id CHAR(36) NOT NULL PRIMARY KEY,
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
    tag JSON NULL,
    additional JSON NULL,
    created_at TIMESTAMP NULL,
    updated_at TIMESTAMP NULL,
    -- "show full history of this record" — the most common query against this table
    INDEX audits_entry_object_index (entry_object_table, entry_object_id),
    -- "what did this user do"
    INDEX audits_actor_index (actor_table, actor_id),
    -- time-range filtering / pagination / retention cleanup
    INDEX audits_created_at_index (created_at)
);
