ALTER TABLE documents DROP INDEX documents_search_body_fulltext;
ALTER TABLE documents DROP INDEX documents_title_fulltext;
ALTER TABLE documents DROP COLUMN search_body;
