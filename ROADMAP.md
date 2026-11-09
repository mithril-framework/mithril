# Mithril Roadmap

This document tracks planned features and milestones. Items marked **Done** ship in the current release.

## Phase 1 — Open Source Foundation (current)

- [x] README, LICENSE, CONTRIBUTING
- [x] Module path `github.com/mithril-framework/mithril`
- [x] Global CLI (`mithril new`, `install.sh`)
- [x] Docs aligned with real codebase
- [x] Security hygiene (JWT, CORS, auth stubs)

## Phase 2 — Developer Experience (v1.2, beta)

- [x] Importable library (`app.New`, `app.Main`) and thin `mithril new` template
- [x] Declarative resources: CRUD + validation + OpenAPI + permissions + admin from one struct
- [x] Typed handlers with binding and validation (FastAPI-style)
- [x] OpenAPI 3.1 generated from Go types (replaces AI-generated Swagger)
- [x] Embedded migrations, views and assets: single-binary deploys
- [x] `mithril make:resource` generator with dry run
- [x] HTTP handler test suite expansion (auth, users, resources, api, app, scaffold)
- [ ] `migrate-gen`: generate migrations from model diffs
- [ ] Resource hooks (before/after save), filters, search and relationship preloading
- [ ] `mithriltest` helpers (transactional DB, `ActingAs(user)`)

## Phase 3 — Security & Middleware

- [ ] Server-side sessions with refresh-token rotation and revocation
- [ ] CSRF protection
- [ ] Rate limiting (per-IP, per-user, per-route)
- [ ] Session management (DB-backed)
- [ ] Built-in captcha helpers
- [ ] DDoS / attack protection
- [ ] Honeypot endpoints

## Phase 4 — Platform Features

- [ ] WebSocket support (socket.io-style API on top of native WS)
- [ ] Background jobs / queue system
- [ ] Object storage (S3/MinIO)
- [ ] i18n with `Accept-Language` header
- [ ] CMS-style content module (Strapi-inspired)

## Phase 5 — Observability

- [ ] OpenTelemetry tracing
- [ ] Sentry integration
- [ ] Prometheus metrics endpoint
- [ ] Structured JSON logging levels

## Phase 6 — Architecture

- [ ] Microservices helpers
- [ ] CQRS patterns
- [ ] gRPC support
- [ ] Blue-green / zero-downtime deployment guides

## Phase 7 — AI & Tooling

- [x] MCP server (`go run . mcp`): routes, resources, OpenAPI, migrations, scaffold_resource
- [x] `AGENTS.md` and `.mcp.json` in generated projects
- [ ] More MCP tools (run tests, read logs, explain permissions)

## Already Available

- Fiber v3 HTTP server with middleware stack
- PostgreSQL via pgx + goose migrations
- JWT authentication (access + refresh tokens)
- Django-style RBAC (roles, permissions, ACL CLI)
- CRUD API for User and Blog models
- Admin panel (`/admin`) and embedded DBMS (`:5050`)
- Makefile / `mithril` CLI for migrations, seed, backup, swagger, docker services
- Docker multi-stage build and Kubernetes manifests
- GitHub Actions CI/CD pipeline
