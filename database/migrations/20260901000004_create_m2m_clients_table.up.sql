-- New table owned by this app.
--
-- Machine-to-machine callers authenticate with client + secret instead of the
-- SSO/JWT flow. secret stores a bcrypt hash, never the plaintext — the
-- plaintext is only ever returned once, at creation. revoked_at (not a delete)
-- keeps the row so a revoked client's past requests stay attributable.
CREATE TABLE m2m_clients (
    id CHAR(36) NOT NULL PRIMARY KEY,
    client VARCHAR(255) NOT NULL,
    secret VARCHAR(255) NOT NULL,
    revoked_at TIMESTAMP NULL,
    created_at TIMESTAMP NULL,
    updated_at TIMESTAMP NULL,
    UNIQUE KEY m2m_clients_client_unique (client)
);
