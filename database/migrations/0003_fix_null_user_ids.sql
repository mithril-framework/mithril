-- +goose Up
-- Fix rows that have NULL id (can happen when DEFAULT was not applied during 0002 INSERT).
UPDATE users SET id = gen_random_uuid() WHERE id IS NULL;

-- Ensure id column is NOT NULL and has primary key (idempotent if already set).
ALTER TABLE users ALTER COLUMN id SET DEFAULT gen_random_uuid();
ALTER TABLE users ALTER COLUMN id SET NOT NULL;

-- +goose Down
-- Cannot reliably undo; leave schema as is.
