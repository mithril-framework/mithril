-- +goose Up
-- Schema is defined only here (and sibling .sql files). Goose does not sync database/models/*.go — add ALTER/CREATE in SQL, then migrate-up.
ALTER TABLE users ADD COLUMN IF NOT EXISTS is_superuser BOOLEAN NOT NULL DEFAULT false;

CREATE TABLE permissions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    codename TEXT NOT NULL UNIQUE,
    description TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE roles (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL UNIQUE,
    description TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE role_permissions (
    role_id UUID NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    permission_id UUID NOT NULL REFERENCES permissions(id) ON DELETE CASCADE,
    PRIMARY KEY (role_id, permission_id)
);

CREATE INDEX idx_role_permissions_permission_id ON role_permissions(permission_id);

CREATE TABLE user_roles (
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role_id UUID NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    PRIMARY KEY (user_id, role_id)
);

CREATE INDEX idx_user_roles_role_id ON user_roles(role_id);

CREATE TABLE user_permissions (
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    permission_id UUID NOT NULL REFERENCES permissions(id) ON DELETE CASCADE,
    PRIMARY KEY (user_id, permission_id)
);

CREATE INDEX idx_user_permissions_permission_id ON user_permissions(permission_id);

INSERT INTO permissions (codename, description) VALUES
    ('admin.access', 'Use admin panel API'),
    ('users.view', 'List and view users'),
    ('users.add', 'Create users'),
    ('users.change', 'Update users'),
    ('users.delete', 'Delete users'),
    ('blogs.view', 'List and view blogs'),
    ('blogs.add', 'Create blogs'),
    ('blogs.change', 'Update own blogs'),
    ('blogs.change_any', 'Update any blog'),
    ('blogs.delete', 'Delete own blogs'),
    ('blogs.delete_any', 'Delete any blog')
ON CONFLICT (codename) DO NOTHING;

-- +goose Down
DROP TABLE IF EXISTS user_permissions;
DROP TABLE IF EXISTS user_roles;
DROP TABLE IF EXISTS role_permissions;
DROP TABLE IF EXISTS roles;
DROP TABLE IF EXISTS permissions;
ALTER TABLE users DROP COLUMN IF EXISTS is_superuser;
