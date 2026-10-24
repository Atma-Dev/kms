-- RBAC, rebuilt for what this app actually uses rather than copied from
-- Spatie's schema:
--
--   * no model_type. Spatie's pivots are polymorphic because a role can be
--     attached to any model; here a role only ever belongs to a user, so the
--     column held one constant value on every row. model_has_roles becomes a
--     plain user_roles pivot, which can carry a real foreign key to users --
--     something the polymorphic version could not.
--   * no model_has_permissions. Permissions reach a user only through their
--     role (user_roles -> role_has_permissions -> permissions); nothing grants
--     one directly, so the table was never read.
--   * no guard_name. This app has no concept of guards; the column existed to
--     satisfy Laravel's NOT NULL and its (name, guard_name) unique key. The
--     unique key is now on name alone.

CREATE TABLE permissions (
    id UUID NOT NULL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    description VARCHAR(255) NULL,
    feature_group VARCHAR(255) NULL,
    created_at TIMESTAMPTZ NULL,
    updated_at TIMESTAMPTZ NULL,
    CONSTRAINT permissions_name_unique UNIQUE (name)
);

CREATE TABLE roles (
    id UUID NOT NULL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    is_mutable BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NULL,
    updated_at TIMESTAMPTZ NULL,
    CONSTRAINT roles_name_unique UNIQUE (name)
);

CREATE TABLE role_has_permissions (
    permission_id UUID NOT NULL,
    role_id UUID NOT NULL,
    PRIMARY KEY (permission_id, role_id),
    CONSTRAINT role_has_permissions_permission_id_foreign
        FOREIGN KEY (permission_id) REFERENCES permissions (id) ON DELETE CASCADE,
    CONSTRAINT role_has_permissions_role_id_foreign
        FOREIGN KEY (role_id) REFERENCES roles (id) ON DELETE CASCADE
);

CREATE INDEX role_has_permissions_role_id_index ON role_has_permissions (role_id);

-- user_id first in the primary key: every read is "the roles of this user".
-- The separate role_id index is for the one query that goes the other way,
-- RoleRepository.GetUserIDs.
CREATE TABLE user_roles (
    user_id UUID NOT NULL,
    role_id UUID NOT NULL,
    PRIMARY KEY (user_id, role_id),
    CONSTRAINT user_roles_user_id_foreign
        FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE,
    CONSTRAINT user_roles_role_id_foreign
        FOREIGN KEY (role_id) REFERENCES roles (id) ON DELETE CASCADE
);

CREATE INDEX user_roles_role_id_index ON user_roles (role_id);
