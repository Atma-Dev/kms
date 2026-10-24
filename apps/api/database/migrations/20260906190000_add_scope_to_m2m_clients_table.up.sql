-- Existing clients (registered before scoping existed) get '*' so this
-- migration does not silently cut off access they already had; an admin can
-- narrow them afterward via PATCH /management/m2m-clients/{id}.
ALTER TABLE m2m_clients ADD COLUMN scope JSON NULL;
UPDATE m2m_clients SET scope = JSON_ARRAY('*') WHERE scope IS NULL;
ALTER TABLE m2m_clients MODIFY scope JSON NOT NULL;
