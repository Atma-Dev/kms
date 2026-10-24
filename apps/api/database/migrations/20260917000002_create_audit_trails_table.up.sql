-- One row per changed record within an audits row — see
-- 20260917000001_create_audits_table for what writes here.
CREATE TABLE audit_trails (
    id CHAR(36) NOT NULL PRIMARY KEY,
    audit_id CHAR(36) NOT NULL,
    object_table VARCHAR(255) NULL,
    object_id VARCHAR(255) NULL,
    -- `before`/`after` are backtick-quoted: BEFORE is a reserved word in
    -- MySQL (trigger timing, e.g. CREATE TRIGGER ... BEFORE INSERT), so the
    -- bare identifier is a syntax error.
    `before` JSON NULL,
    `after` JSON NULL,
    tag JSON NULL,
    additional JSON NULL,
    created_at TIMESTAMP NULL,
    updated_at TIMESTAMP NULL,
    INDEX audit_trails_audit_id_index (audit_id),
    -- "show every change ever made to this record", independent of which action triggered it
    INDEX audit_trails_object_index (object_table, object_id),
    CONSTRAINT audit_trails_audit_id_foreign FOREIGN KEY (audit_id) REFERENCES audits (id)
);
