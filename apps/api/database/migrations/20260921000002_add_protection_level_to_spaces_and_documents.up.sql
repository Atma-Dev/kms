-- The protection level (Modul 4 FR-6): how hard a document defends itself.
--
--   normal  watermark overlay on the web viewer, watermarked text PDF
--   strict  the viewer is server-rendered page images with the watermark
--           already burned in, and the PDF is built from those same images
--
-- The PRD models this as a DocumentProtectionSetting table with a nullable
-- document_id standing for "the space default". It is two columns here
-- instead, for two reasons:
--
--   1. MySQL treats NULLs as distinct in a unique index, so
--      UNIQUE (space_id, document_id) would not stop a second, conflicting
--      space default from being inserted. The column cannot disagree with
--      itself.
--   2. Resolving the level is part of loading the document, and a column is
--      read by the SELECT that was already happening. A table would add a
--      query to the hot path of every document open.
--
-- documents.protection_level is NULL for "inherit the space", which is what
-- the PRD's absent row meant. created_by is not carried: who changed a
-- setting is what the audit trail (go-audit) already records, and a column
-- beside it is a second answer to the same question.
ALTER TABLE spaces
    ADD COLUMN protection_level VARCHAR(8) NOT NULL DEFAULT 'normal' AFTER description;

ALTER TABLE documents
    ADD COLUMN protection_level VARCHAR(8) NULL AFTER status;
