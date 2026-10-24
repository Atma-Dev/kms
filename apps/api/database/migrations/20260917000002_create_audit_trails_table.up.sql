-- One row per changed record within an audits row — see
-- 20260917000001_create_audits_table for what writes here.
CREATE TABLE audit_trails (
    id UUID NOT NULL PRIMARY KEY,
    audit_id UUID NOT NULL,
    object_table VARCHAR(255) NULL,
    object_id VARCHAR(255) NULL,
    -- Quoted out of caution: BEFORE/AFTER are keywords (trigger timing), and
    -- although Postgres accepts them bare as column names, MySQL did not.
    "before" JSONB NULL,
    "after" JSONB NULL,
    tag JSONB NULL,
    additional JSONB NULL,
    created_at TIMESTAMPTZ NULL,
    updated_at TIMESTAMPTZ NULL,
    CONSTRAINT audit_trails_audit_id_foreign FOREIGN KEY (audit_id) REFERENCES audits (id)
);

CREATE INDEX audit_trails_audit_id_index ON audit_trails (audit_id);
-- "show every change ever made to this record", independent of which action triggered it
CREATE INDEX audit_trails_object_index ON audit_trails (object_table, object_id);
