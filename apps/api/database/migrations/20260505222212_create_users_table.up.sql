-- Started as a copy of Laravel's default users table.
--
-- email_verified_at / remember_token / deleted_at are unused by this API; they
-- are kept because the SSO-side user record still carries them.
--
-- The email unique index is on LOWER(email): Postgres compares text
-- case-sensitively, and "Budi@x.id" and "budi@x.id" are one mailbox. Login
-- looks the address up the same way (LOWER(email) = LOWER(?)), so the index
-- serves it.

CREATE TABLE users (
    id UUID NOT NULL PRIMARY KEY,
    sso_id UUID NULL,
    number BIGINT NOT NULL CHECK (number >= 0),
    first_name VARCHAR(255) NOT NULL,
    last_name VARCHAR(255) NULL,
    email VARCHAR(255) NOT NULL,
    phone_number VARCHAR(255) NULL,
    email_verified_at TIMESTAMPTZ NULL,
    password VARCHAR(255) NULL,
    remember_token VARCHAR(100) NULL,
    created_at TIMESTAMPTZ NULL,
    updated_at TIMESTAMPTZ NULL,
    deleted_at TIMESTAMPTZ NULL,
    marketing_id VARCHAR(255) NULL,
    is_active BOOLEAN NOT NULL DEFAULT FALSE,
    last_activity TIMESTAMPTZ NULL,
    CONSTRAINT users_number_unique UNIQUE (number),
    CONSTRAINT users_phone_number_unique UNIQUE (phone_number)
);

CREATE UNIQUE INDEX users_email_unique ON users (LOWER(email));
CREATE INDEX users_sso_id_index ON users (sso_id);
