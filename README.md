# Mithril

A batteries-included, AI-friendly Go web framework built on [Fiber](https://gofiber.io), inspired by Django, Laravel, FastAPI, and NestJS.

**Documentation:** [mithril-docs-nine.vercel.app](https://mithril-docs-nine.vercel.app/docs/getting-started/introduction)

## Features

| Available now | Planned (see [ROADMAP.md](ROADMAP.md)) |
|---------------|----------------------------------------|
| JWT auth + refresh tokens | CSRF, rate limiting, 2FA |
| Django-style RBAC | WebSockets, gRPC, queues |
| PostgreSQL + goose migrations | OpenTelemetry, Sentry, Prometheus |
| CRUD generator | FastAPI-style validation layer |
| Admin panel + embedded DBMS | Object storage (S3/MinIO) |
| Docker + Kubernetes + CI/CD | i18n, CMS module |

## Quick Start

### Install CLI

```bash
curl -fsSL https://raw.githubusercontent.com/mithril-framework/mithril/main/install.sh | sh
```

Or from source:

```bash
go install github.com/mithril-framework/mithril/cmd/mithril@main
```

### Create a project

```bash
mithril new hello-mithril
cd hello-mithril
cp env.example .env
make dc-up-postgres
make migrate-up
make seed              # demo user: user@example.com / password
# or: make createsuperuser
make run
```

Visit `http://localhost:4000` — API docs at `/docs`, health at `/health`.

### Clone and run this repo

```bash
git clone https://github.com/mithril-framework/mithril.git
cd mithril
make install
cp env.example .env
make dc-up-postgres
make migrate-up
make run
```

## CLI Commands

Inside a project, `mithril` delegates to `make`:

```bash
mithril migrate-up      # Run migrations
mithril crud MODEL=Blog # Generate CRUD from model
mithril seed              # Seed demo data
mithril admin-enable      # Enable /admin panel
mithril dbms              # Start embedded DBMS on :5050
mithril --help            # Full command list
```

## Project Structure

```
├── main.go              # Application entry point
├── routes/              # Route registration
├── internal/            # Auth, ACL, admin, CRUD handlers
├── database/            # Models, repositories, migrations
├── cmd/                 # CLI tools (mithril, crud, acl, seed, …)
├── pkg/utils/           # Shared utilities
├── public/admin/        # Admin SPA
└── infrastructure/      # Docker, K8s, compose services
```

## Configuration

See [env.example](env.example). Required for production:

- `JWT_SECRET` — must be set when `APP_ENV=production`
- `DATABASE_URL` or `DB_*` — PostgreSQL connection

## Security

> **Warning:** Change `JWT_SECRET` before deploying. Never commit `.env` files.

Registration is disabled by default. Set `ENABLE_REGISTER=true` to allow `POST /auth/register`.

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md).

## Troubleshooting

**Wrong `mithril` on PATH**

If `mithril --version` prints `dev` or a scaffold message instead of `mithril 1.x.x (github.com/mithril-framework/mithril)`, another binary is shadowing the framework CLI:

```bash
which -a mithril
go install github.com/mithril-framework/mithril/cmd/mithril@main
export PATH="$(go env GOPATH)/bin:$PATH"
# or symlink to /usr/local/bin:
sudo "$(go env GOPATH)/bin/mithril" init
mithril --version
```

**Quick login without interactive `createsuperuser`**

```bash
mithril seed
# user@example.com / password
```

Or non-interactive superuser:

```bash
mithril createsuperuser --email admin@example.com --password 'your-secure-password'
```

## License

[MIT](LICENSE)
