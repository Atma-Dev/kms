-- Per-document access grants (FR-9).
--
-- A document with NO rows here follows the ordinary permissions. A document
-- WITH rows is restricted to the subjects listed — which is what the PRD's own
-- example asks for: "dokumen sensitif hanya untuk beberapa user tertentu
-- meski mereka Viewer di space itu". Adding the first row narrows access; it
-- never widens it.
--
-- subject_type + subject_id is a deliberate polymorphic pair (user or role)
-- and therefore carries no foreign key: one column cannot reference two
-- tables. Rows are cleaned up by the application when a role is deleted.
CREATE TABLE document_access (
    id UUID NOT NULL PRIMARY KEY,
    document_id UUID NOT NULL,
    subject_type VARCHAR(8) NOT NULL,
    subject_id UUID NOT NULL,
    permission VARCHAR(8) NOT NULL,
    created_at TIMESTAMPTZ NULL,
    updated_at TIMESTAMPTZ NULL,
    CONSTRAINT document_access_subject_unique UNIQUE (document_id, subject_type, subject_id),
    CONSTRAINT document_access_document_id_foreign FOREIGN KEY (document_id) REFERENCES documents (id) ON DELETE CASCADE
);
