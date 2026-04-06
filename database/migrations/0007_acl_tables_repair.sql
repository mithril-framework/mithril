-- +goose Up
-- Idempotent repair if permissions/roles/junction tables are missing (e.g. DB only had partial 0004 or column-only 0006).
CREATE TABLE IF NOT EXISTS public.permissions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    codename TEXT NOT NULL UNIQUE,
    description TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS public.roles (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL UNIQUE,
    description TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS public.role_permissions (
    role_id UUID NOT NULL REFERENCES public.roles(id) ON DELETE CASCADE,
    permission_id UUID NOT NULL REFERENCES public.permissions(id) ON DELETE CASCADE,
    PRIMARY KEY (role_id, permission_id)
);

CREATE INDEX IF NOT EXISTS idx_role_permissions_permission_id ON public.role_permissions(permission_id);

CREATE TABLE IF NOT EXISTS public.user_roles (
    user_id UUID NOT NULL REFERENCES public.users(id) ON DELETE CASCADE,
    role_id UUID NOT NULL REFERENCES public.roles(id) ON DELETE CASCADE,
    PRIMARY KEY (user_id, role_id)
);

CREATE INDEX IF NOT EXISTS idx_user_roles_role_id ON public.user_roles(role_id);

CREATE TABLE IF NOT EXISTS public.user_permissions (
    user_id UUID NOT NULL REFERENCES public.users(id) ON DELETE CASCADE,
    permission_id UUID NOT NULL REFERENCES public.permissions(id) ON DELETE CASCADE,
    PRIMARY KEY (user_id, permission_id)
);

CREATE INDEX IF NOT EXISTS idx_user_permissions_permission_id ON public.user_permissions(permission_id);

INSERT INTO public.permissions (codename, description) VALUES
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
SELECT 1;
