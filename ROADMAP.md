# Mithril Roadmap

This document tracks planned features and milestones. Items marked **Done** ship in the current release.

## Phase 1 — Open Source Foundation (current)

- [x] README, LICENSE, CONTRIBUTING
- [x] Module path `github.com/mithril-framework/mithril`
- [x] Global CLI (`mithril new`, `install.sh`)
- [x] Docs aligned with real codebase
- [x] Security hygiene (JWT, CORS, auth stubs)

## Phase 2 — Developer Experience

- [ ] `migrate-gen` — generate migrations from models
- [ ] CRUD generator: AST-based route registration, dry-run improvements
- [ ] Project template polish and more scaffolds
- [ ] HTTP handler test suite expansion

## Phase 3 — Security & Middleware

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

- [ ] AI assistant integration
- [ ] MCP server support
- [ ] Cursor skills / rules templates

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
