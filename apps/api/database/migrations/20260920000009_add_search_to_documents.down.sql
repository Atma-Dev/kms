DROP INDEX IF EXISTS documents_search_vector_index;
ALTER TABLE documents DROP COLUMN search_vector;
ALTER TABLE documents DROP COLUMN search_body;
