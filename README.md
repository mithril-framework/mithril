# Mithril

A batteries-included, AI-friendly Go web framework built on [Fiber](https://gofiber.io), inspired by Django, Laravel and FastAPI. Mithril is a **library**: you import it, and framework fixes arrive with `go get -u`.

**Documentation:** [mithril-docs-nine.vercel.app](https://mithril-docs-nine.vercel.app/docs/getting-started/introduction) · [Feature status](https://mithril-docs-nine.vercel.app/docs/getting-started/feature-status)

```go
type Post struct {
	ID        uuid.UUID `json:"id"`
	Title     string    `json:"title" validate:"required,max=200"`
	Body      string    `json:"body" mithril:"text"`
	AuthorID  uuid.UUID `json:"author_id" mithril:"owner"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

//go:embed migrations
var migrations embed.FS

func main() {
	a := app.Must(app.New(app.WithMigrations(migrations, "migrations")))
	a.Register(resource.Define[Post](resource.Options{Name: "posts"}))
	app.Main(a) // serve | migrate up | routes | openapi | createsuperuser | acl | mcp
}
```

That one registration gives you a validated, paginated CRUD API at `/api/posts`, ownership-aware permissions (`posts.view/add/change/change_any/delete/delete_any`), OpenAPI 3.1 operations, Swagger UI at `/docs` and an admin page at `/admin`, in a single self-contained binary.

## Features

| Shipped | Beta (v1.2) | Planned ([ROADMAP](ROADMAP.md)) |
|---|---|---|
| JWT access/refresh tokens (HS256, type-checked) | Importable library + `mithril new` template | CSRF, rate limiting, sessions |
| Django-style RBAC + superusers | Declarative resources (`resource.Define[T]`) | Token revocation, 2FA, password reset |
| Admin panel | Typed handlers with validation (`api.Get/Post/...`) | WebSockets, gRPC, queues |
| PostgreSQL (pgx v5) | OpenAPI 3.1 generated from Go types | OpenTelemetry, Prometheus, Sentry |
| Secure defaults (fail-closed secrets, Helmet, no leaked errors) | Embedded migrations, single-binary deploys | `migrate-gen`, DI container |
| Docker, Kubernetes, CI | MCP server, `AGENTS.md`, `make:resource` generator | Testing helpers, i18n, storage |

## Quick start

```bash
go install github.com/mithril-framework/mithril/cmd/mithril@latest

mithril new -module github.com/you/shop shop
cd shop
docker compose up -d postgres
mithril migrate up
mithril createsuperuser --email admin@shop.test --password 'change-me-now'
mithril make:resource Product name:string:required price:float --owner
mithril migrate up
mithril serve            # http://localhost:4000 · /docs · /admin
```

## Packages

| Package | Purpose |
|---|---|
| `app` | Application builder, middleware stack, management commands |
| `resource` | Declarative CRUD resources |
| `api`, `openapi` | Typed handlers, validation, OpenAPI generation |
| `auth`, `acl`, `users`, `admin` | JWT auth, RBAC, user management, admin panel |
| `config`, `db`, `migrate` | Typed configuration, pgx pool, embedded goose migrations |
| `mcpserver`, `scaffold` | MCP server for AI agents, code generators |
| `pkg/utils` | Hashing, encryption, strings, time, pagination helpers |

## This repository

The repository root is the **reference app**: it's built on the library and registers one example resource (`examples/blog`).

```bash
git clone https://github.com/mithril-framework/mithril.git
cd mithril
cp env.example .env
make dc-up-postgres
make migrate-up
make createsuperuser ARGS='--email admin@example.com --password change-me-now'
make run
```

Try framework changes in a separate project with `mithril new -local ../mithril demo`.

## Security

- Outside `APP_ENV=development`, including when `APP_ENV` is unset, the server refuses to start without a strong `JWT_SECRET` (32+ bytes, not a placeholder).
- Registration is off by default (`ENABLE_REGISTER=true` enables it).
- Report vulnerabilities privately through GitHub security advisories.

## Upgrading from 1.1

Projects created with 1.1 were full clones of this repository. They keep working (`mithril <target>` still runs their `make` targets). To move to the library layout, create a new project and copy your models and migrations; existing databases upgrade with `migrate up` (see the [migration guide](https://mithril-docs-nine.vercel.app/docs/database/migrations#upgrading-a-pre-12-project)).

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md). Run `go test ./...` before opening a PR.

## License

[MIT](LICENSE)
