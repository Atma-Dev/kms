-- Started as a copy of Laravel's default users table.
--
-- email_verified_at / remember_token / deleted_at are unused by this API; they
-- are kept because the SSO-side user record still carries them.

CREATE TABLE users (
    id CHAR(36) NOT NULL PRIMARY KEY,
    sso_id CHAR(36) NULL,
    number BIGINT UNSIGNED NOT NULL,
    first_name VARCHAR(255) NOT NULL,
    last_name VARCHAR(255) NULL,
    email VARCHAR(255) NOT NULL,
    phone_number VARCHAR(255) NULL,
    email_verified_at TIMESTAMP NULL,
    password VARCHAR(255) NULL,
    remember_token VARCHAR(100) NULL,
    created_at TIMESTAMP NULL,
    updated_at TIMESTAMP NULL,
    deleted_at TIMESTAMP NULL,
    marketing_id VARCHAR(255) NULL,
    is_active TINYINT(1) NOT NULL DEFAULT 0,
    last_activity TIMESTAMP NULL,
    UNIQUE KEY users_number_unique (number),
    UNIQUE KEY users_email_unique (email),
    UNIQUE KEY users_phone_number_unique (phone_number),
    INDEX users_sso_id_index (sso_id)
);