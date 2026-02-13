# Database Backup and Restore

Full PostgreSQL backup and restore with no Postgres CLI required on the host: the tool prefers native `pg_dump`/`psql`/`pg_restore`, then falls back to **Docker** (postgres image), then **pure Go** (pgx).

## Commands

- **`make backup`** – Create a compressed SQL backup in `database/backups/` (uses `DATABASE_URL` or `DB_*` from `.env`).
- **`make restore FILE=path/to/backup.sql.gz`** or **`make restore FILE=latest`** – Restore from a file or the latest backup.
- **`make backup-list`** – List backups in `database/backups/`.

## CLI

```bash
go run ./cmd/backup backup [--schema-only] [--data-only] [--format=plain|custom] [--no-compress] [--output=path] [--use-docker|--use-go]
go run ./cmd/backup restore --file=path|latest [--force] [--use-docker|--use-go]
go run ./cmd/backup list
```

## Environment

- **`DATABASE_URL`** or **`DB_HOST`**, **`DB_PORT`**, **`DB_USER`**, **`DB_PASSWORD`**, **`DB_NAME`** – PostgreSQL connection.
- **`DB_BACKUP_PATH`** – Directory for backups (default: `database/backups`).

## Strategies

1. **Native** – Uses `pg_dump`, `psql`, `pg_restore` from `PATH` if present.
2. **Docker** – Runs `postgres:16-alpine` and executes the same tools inside the container. For a DB on the host, the container uses `host.docker.internal`. Requires Docker and a pull of the image on first use.
3. **Go** – Uses pgx only (no CLI, no Docker). Backup: schema (CREATE TABLE) + data (COPY format) + sequences. Restore: runs SQL and COPY blocks. **Custom `.dump` format is not supported** in the Go path; use native or Docker for `.dump` restore.

## Notes

- Restore of `.dump` files requires native `pg_restore` or Docker; plain `.sql`/`.sql.gz` can be restored with the Go fallback.
- Backup files are written under `database/backups/` and are ignored by git (see `.gitignore`).
