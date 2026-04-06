-- +goose Up
-- Idempotent repair: Goose does not read Go models — schema changes live only in these SQL files.
-- If is_superuser is missing (drift, partial apply, wrong DB), this fixes public.users only.
ALTER TABLE IF EXISTS public.users ADD COLUMN IF NOT EXISTS is_superuser BOOLEAN NOT NULL DEFAULT false;

-- +goose Down
SELECT 1;
