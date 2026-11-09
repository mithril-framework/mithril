# Mithril Code Review — Logic & Security Audit

Branch: `review/logic-security-audit` · Date: 2026-11-08 · Base: `main` @ `957df0b`

Scope: everything in this repo (`main.go`, `routes/`, `internal/`, `database/`, `cmd/`, `pkg/utils`, `public/admin`, Docker/K8s/install files), checked against the vision in `mithril-docs` (a batteries-included, secure-by-default framework in the style of Django/Laravel, built on Go and Fiber).

> **Status:** ✅ = fixed on this branch (S1–S6, L1; L2 fixed for users only). See "Fix log" at the end.

Severity: **Critical** = remote takeover or privilege escalation · **High** = auth bypass, data integrity, or exposure · **Medium** = hardening gap with a real impact · **Low** = cleanup.

---

## 1. Security issues

### Critical

| # | Issue | Where | Impact / Fix |
|---|---|---|---|
| S1 ✅ | **Default JWT secret `"secret"`** is used whenever `APP_ENV` isn't exactly `production`. If someone deploys without setting `APP_ENV` (common), anyone can forge a token with `is_superuser: true`. The placeholder secrets in `env.example` and `k8s/secret.yaml` (`change-me-in-production`) are also accepted in production. | `main.go:resolveJWTSecret` | Full takeover. **Fix:** refuse to start without a secret unless `APP_ENV=development` is set explicitly. Require at least 32 bytes and reject known placeholder values. Have `mithril new` generate a random secret. |
| S2 ✅ | **Mass assignment in `POST /api/users`.** The handler binds the request body straight into `models.User`, so the caller controls `is_superuser`, `is_active`, `id`, and `password_hash` (stored as sent, with no hashing). | `internal/crud/user/handlers.go:Create` | Anyone with `users.add` can create a superuser. **Fix:** use a dedicated request DTO (like the admin handler already does), hash `password`, and never bind `is_superuser` or `password_hash`. |
| S3 ✅ | **Privilege escalation in `PUT /api/users/:id`.** Anyone with `users.change` can set `is_superuser=true` on any account, including their own (`canElevate` checks `users.change` instead of superuser). They can also overwrite another user's `password_hash`, which lets them take over a superuser account. | `internal/crud/user/handlers.go:Update` | **Fix:** only a superuser may change `is_superuser`. Use a DTO and never accept `password_hash`. Password changes go through a separate endpoint that checks the current password. |
| S4 ✅ | **Self-service update lets a user change sensitive fields.** Any logged-in user can `PUT /api/users/{self}` and change their own `email` (no verification), `password_hash` (no current-password check), and `is_active`. | same | With a stolen token, an attacker can lock in the takeover. **Fix:** self-service accepts only profile fields. |
| S5 ✅ | **`admin.access` is effectively superuser.** Admin API holders who aren't superusers can create superusers, set `is_superuser` on anyone, grant themselves any permission or role, and delete or demote real superusers. | `internal/admin/handlers.go` (CreateUser/UpdateUser/Assign*) | **Fix:** require superuser for superuser flags and ACL edits, or add separate `admin.manage_acl` / `users.manage_superuser` permissions. Block deleting or demoting the last superuser. |
| S6 ✅ | **Token `type` is never checked.** Refresh tokens (7 days) work as access tokens on `/api/*` and `/admin/api/*`, and access tokens work at `/auth/refresh`. | `acl/context.go`, `auth/handlers.go:Refresh` | Long-lived bearer access, and a stolen access token can be refreshed forever. **Fix:** require `type=="access"` in `JWTClaimsMiddleware` and `type=="refresh"` in `Refresh`. |

### High

| # | Issue | Where | Impact / Fix |
|---|---|---|---|
| S7 | **No revocation or session model.** Logout only clears cookies, so tokens stay valid. Disabled, deleted, or demoted users keep access for up to 24 hours, because `is_superuser` from the JWT short-circuits every ACL check. `Refresh` doesn't check `IsActive`, so a disabled user can refresh forever. | `acl/service.go:HasPermission`, `auth/handlers.go` | **Fix:** store sessions or refresh-token families in the DB (rotate on use, detect reuse), check `IsActive` on refresh, and shorten access tokens to about 15 minutes, as the docs already describe. Trust the DB rather than the JWT for superuser status. |
| S8 (partly) | **Embedded DBMS (`:5050`) is exposed to the network.** It binds `:5050` (all interfaces) while the log line says `127.0.0.1`. There are no server timeouts (slowloris). The host field on the login form is free text, which makes it an SSRF / internal port-scan tool: connection errors are shown verbatim. | `internal/dbms/server.go:ListenAndServe, handleLogin` | **Fix:** default to `127.0.0.1:5050`, use `http.Server` with timeouts, optionally restrict to an allow-list of hosts, and print a warning when binding to a non-loopback address. |
| S9 | **DBMS session store grows without limit and never expires.** Every request without a cookie creates a session (memory DoS). Sessions keep the DB password in plaintext for the life of the process. The session ID isn't rotated on login. The cookie has no `Secure` flag. | `internal/dbms/server.go:ensureSession` | **Fix:** create sessions only on login, add a TTL plus a sweeper, cap the session count, and rotate the ID on login. |
| S10 ✅ | **`/monitor` is public.** It exposes CPU, memory, PIDs, and connection counts to anyone. | `routes/core.go` | **Fix:** put it behind `RequireSuperuser` or a separate flag, off in production. |
| S11 (partly) | **Internal errors are sent to clients.** The global `ErrorHandler`, every CRUD 500, ACL errors, `/health` (raw DB error), and `token_error` all return `err.Error()`, which leaks SQL, constraint, and schema details. | `main.go`, CRUD/ACL/admin handlers | **Fix:** log the details server-side with the request ID and return a generic message plus the `request_id`. |
| S12 | **No brute-force protection on `/auth/login` or `/auth/register`.** The docs list rate limiting as a core feature. | `routes/auth.go` | **Fix:** ship `limiter` on `/auth/*` by default, keyed by IP and email. |
| S13 | **User enumeration on login.** A disabled account gets `403 account_disabled` before the password is checked, and an unknown email returns quickly (no bcrypt run), which leaks through timing. | `auth/handlers.go:Login` | **Fix:** check the password first and compare against a dummy hash when the user is missing. Return the same 401 in all cases. |

### Medium

| # | Issue | Where |
|---|---|---|
| S14 | The JWT middleware doesn't pin the algorithm (`JWTAlg` is empty), and `Refresh` parses without `jwt.WithValidMethods`. It isn't exploitable today because the key is an HMAC `[]byte`, but pin `HS256` (plus issuer and audience) anyway. | `routes/register.go`, `auth/handlers.go` |
| S15 | The auth cookies don't set `Secure`. They're also never read by `jwtware`, which only checks the Authorization header. If cookie auth gets wired up later, CSRF protection is needed; today the `remember` feature is half-built. | `auth/handlers.go:Login` |
| S16 | Registration has no email validation or normalization (emails are case-sensitive, so `A@x.com` and `a@x.com` can both register). There's no password policy, and bcrypt silently truncates passwords at 72 bytes. The response says "check your email" but no verification email is sent. The user's email is logged in plaintext. | `auth/handlers.go:Register` |
| S17 | `session_id = "session_<unix seconds>"` is predictable and collides across logins in the same second, so it's useless for revocation. Use a UUID or random value. | `auth/handlers.go:Login` |
| S18 | Backup and restore pass a DSN containing the password as argv (`pg_restore -d <dsn>`, `psql <dsn>`), and Docker gets `-e PGPASSWORD=...`. Both are visible in `ps`. Backup files are created with mode 0644 in a 0755 directory. | `internal/db/backup/native.go`, `docker.go` |
| S19 ✅ | `/admin` (static) and `/docs` (Swagger UI) are served without auth. That's fine for static assets, but Swagger should be off in production by default. | `main.go`, `routes/admin.go` |
| S20 ✅ | Helmet, compression, and CORS hardening are opt-in. A framework that markets itself as "secure by default" should turn on Helmet and request size limits by default. | `main.go` |
| S21 | `install.sh` and the README install `@main`, which is unpinned and makes every install a supply-chain risk. Install the latest tag and verify checksums. The `GOPROXY=direct` in the Dockerfile skips the module proxy cache. | `install.sh`, `Dockerfile` |
| S22 ✅ | `mithril swagger` sends route source code to OpenAI without telling the user. Document this and add a confirmation prompt or a `--yes` flag. | `cmd/swagger/main.go` |

### Low
- `GenerateRandomBytes` ignores the error from `rand.Read`. `GenerateRandomString` ignores errors and has modulo-free but slow per-char `big.Int` usage (fine).
- DBMS `validCSRF` uses `==`. Use `subtle.ConstantTimeCompare`.
- DBMS `Content-Disposition` filename isn't quoted (schema and table names can contain `;`/`"`).
- PgAdmin in `docker-compose.services.yml` uses `admin/admin`. That's OK for dev, but bind it to localhost only.

---

## 2. Logic issues

| # | Issue | Where | Effect |
|---|---|---|---|
| L1 ✅ | **`/auth/refresh` sits behind the JWT middleware**, so it needs a *valid access token*. Once the access token expires (the only time you'd refresh), refreshing is impossible. | `routes/auth.go` | The refresh flow is broken. Move `/refresh` to the public section (and see S6). |
| L2 ✅ | **PUT does a full replace with zero values.** `PUT /api/users/:id {"first_name":"x"}` sets `is_active=false` and `last_name=""`. On blogs, `PUT {"title":"x"}` wipes `content` and unpublishes the post. A user can deactivate their own account by accident. | `crud/user`, `crud/blog`, `admin.UpdateBlog` | Use PATCH semantics with pointer fields (as `admin.UpdateUser` already does). |
| L3 ✅ | **Blog Create/Update accept a client-supplied `id`, `created_at`, and `updated_at`.** The Update response echoes the request struct, so it returns zero timestamps instead of the stored row. | `crud/blog/handlers.go` | Re-read the row after update and ignore ID and timestamps from the body. |
| L4 ✅ | The DSN is built with `fmt.Sprintf` and no URL escaping, so a password containing `@ / : # ?` breaks the connection. | `internal/db/dsn.go` | Use `url.URL{User: url.UserPassword(...)}` (the DBMS already does this). |
| L5 ✅ | The Dockerfile copies only the binary. `views/`, `public/` (admin SPA, static files), `docs/swagger.json`, and the migrations are missing, so `/html`, `/admin`, and `/docs` all break in the container. | `Dockerfile` | Copy the assets or use `embed.FS`, which is better. |
| L6 ✅ | `engine.Reload(true)` is always on, so templates are re-read from disk on every render in production. | `main.go:setupApp` | Turn it on only in development. |
| L7 ✅ | The admin `ListUsers`/`ListBlogs` handlers don't cap `limit` or validate `offset`, so `?limit=100000000` dumps the whole table. | `internal/admin/handlers.go` | Clamp the values as the CRUD handlers do. |
| L8 ✅ | If the DB is down, the app starts with no `/api` routes at all (outside production). Clients get 404 instead of 503, and nothing reconnects. | `main.go`, `routes/register.go` | Always register the routes, return 503 while the DB is unavailable, and let pgxpool reconnect lazily. |
| L9 | Roles come from the JWT, but permissions come from the DB, so the two can disagree for 24 hours after a change. `/auth/me` returns the stale roles stored in the token. | `acl`, `auth.Me` | Read `/auth/me` from the DB. |
| L10 | A role named `"user"` is injected into the token when the user has no roles, but no such role exists in the DB. `RequireRole("user")` would fail while the token claims the role. | `auth/handlers.go` | Pick one source of truth. |
| L11 | Redundant ACL queries: Blog Update/Delete call `UserIsSuperuser` and `HasPermission` 2–4 times each (the superuser query isn't cached per request). | `crud/blog` | Cache the per-request principal in `Locals` once. |
| L12 ✅ | `loadEnvFile` is copied in several `cmd/*` mains. It doesn't handle `export KEY=`, inline `# comments`, or single quotes, and an empty value in the real environment (`KEY=`) gets overridden by `.env`. | `main.go`, `cmd/dbms` | Add one shared `internal/config` package. |
| L13 ✅ | The `TestDefaultModulePathFallback` test fails on any machine with `gh` logged in or `github.user` set (it expects `myapp` and gets `github.com/<user>/myapp`). CI only passes because the runner has no GitHub user configured. | `cmd/mithril/main_test.go` | Inject the user lookup so the test doesn't depend on the machine. |
| L14 ✅ | The docs and the code disagree. The docs describe `CORS_ALLOWED_ORIGINS`, `APP_PORT`, `JWT_ACCESS_TOKEN_EXPIRY=15m`, and `artisan`; the code uses `CORS_ORIGINS`, `PORT`, a hard-coded 24h/7d, and `mithril`. | docs ↔ code | Generate the config reference from a single typed config struct. |
| L15 ✅ | `routes/crud_blog.go` and `crud_user.go` re-create repositories that `RegisterAll` already built. There are also `getEnv` helpers duplicated in three packages. | routes | Wire dependencies once (a container or an `App` struct). |

Test status today: `go build` and `go vet` pass. `go test ./...` fails only on L13. There are no tests for the CRUD handlers, admin, dbms, backup, or utils packages, which is exactly where S2–S5 live.

---

## 3. Suggestions to make Mithril popular

**Fix the trust story first.** A framework that sells "JWT + RBAC built in" can't ship S1–S6. Fix those, add a `SECURITY.md` with a disclosure policy, and run `gosec`, `govulncheck`, and CodeQL in CI. Then say so in the README.

1. **Become an importable library, not just a template.** Today `mithril new` clones the whole repo, so users can't `go get -u` fixes. Split it into `github.com/mithril-framework/mithril` (library: `app`, `auth`, `acl`, `admin`, `config`, `db`) plus a thin starter template. This is the biggest adoption blocker: security fixes like the ones above never reach existing projects.
2. **One typed config struct** (`env` tags, validation, fail-fast). Generate the docs' config page from it so the docs never drift again (L14).
3. **Declarative resources, Django-admin-style:** `admin.Register(&Blog{}, admin.Options{List: ..., Search: ...})` should auto-generate CRUD, the admin UI, OpenAPI, and permissions (`blogs.view/add/change/delete`). This is Mithril's unique selling point against Gin, Echo, and Fiber. Make it the headline feature.
4. **Typed handlers with validation and OpenAPI generated from types** (Huma/FastAPI style: `func(ctx, *CreateBlogIn) (*BlogOut, error)`). Deterministic spec generation from types beats AI-generated Swagger: it's reproducible, works offline, and doesn't send code to a third party. Keep the AI as an optional "write descriptions" helper.
5. **Code-first migrations:** `migrate-gen` that diffs models against the schema (Atlas or a custom tool). Consider `sqlc` for typed queries instead of hand-written scan lists.
6. **Auth that's complete out of the box:** sessions or refresh rotation, password reset, email verification, TOTP 2FA, OAuth (Google, GitHub), and API keys, with a pluggable mailer (SMTP, Resend, SES). Today those endpoints return 501; finishing them is what makes people switch from rolling their own.
7. **Secure-by-default middleware preset:** Helmet, rate limiting on `/auth`, body limits, CSRF for cookie auth, strict CORS, and request-ID-correlated JSON logs (`slog`). Pass a single `mithril.Production()` option to turn it all on.
8. **`embed.FS` for views, admin, and migrations** → a single-binary deployment (and it fixes L5). "One binary, `scp`, run" is a big selling point in the Go world.
9. **Observability built in:** `/metrics` (Prometheus), OpenTelemetry tracing for HTTP and pgx, and `slog` with trace IDs. A Grafana dashboard JSON in the repo.
10. **Background jobs and a scheduler** (River on Postgres is a natural fit since you're Postgres-first), plus WebSockets. Laravel's queues are a big part of its appeal.
11. **Testing helpers:** `mithriltest.NewApp(t)` with a transactional DB (testcontainers), plus `ActingAs(user)` and JSON assertions. Easy testing drives framework adoption.
12. **AI-native positioning (from the roadmap):** ship an MCP server exposing routes, models, and migrations; `AGENTS.md` and Cursor rules in generated projects; and `mithril make:*` generators that LLMs can drive reliably. Few Go frameworks do this, so it's a real differentiator.
13. **DX polish:** `mithril make:model|handler|migration|seeder`, an interactive `mithril new` (pick DB, auth, admin), `mithril routes`, `mithril doctor`, and shell completion. Use semver tags with a CHANGELOG and stop installing `@main`.
14. **Docs and community:** a 5-minute "build a blog API with auth + admin" tutorial, a comparison page with honest benchmarks (TechEmpower-style), an example app gallery, a Discord server, `good first issue` labels, and a public roadmap board. Publish a demo admin panel on a live URL.
15. **Mark planned features clearly.** The docs currently show code for `pkg/core`, CSRF, sessions, and a DI container that don't exist. Keep the `PlannedFeatureBanner`, but also add a status table (Shipped / Beta / Planned). Overpromising in docs is the fastest way to lose early adopters.

### Suggested fix order
1. S1, S2, S3, S5, S6, L1 (an afternoon's work, and it closes the takeover paths)
2. S7 (sessions and revocation), S11, S12, S13
3. L2/L3 (PATCH semantics with DTOs), L5 (embed), S8/S9 (DBMS hardening)
4. Handler tests for the CRUD and admin auth matrix, so the fixes don't regress

---

## Fix log (this branch)

- **S1:** `resolveJWTSecret` now fails closed. Outside `APP_ENV=development|dev|local|test` (including when `APP_ENV` is unset) it requires a `JWT_SECRET` of at least 32 bytes that isn't a known placeholder. In development an empty secret gets a random per-process value instead of `"secret"`. `env.example` now ships `APP_ENV=development` with an empty `JWT_SECRET`, and `mithril new` writes a random secret into `.env` (mode 0600).
- **S2:** `POST /api/users` binds a `createUserRequest` DTO, validates email and password (8–72 bytes), and bcrypt-hashes the password. Only a DB superuser may set `is_superuser`.
- **S3/S4:** `PUT /api/users/:id` uses a pointer-field DTO (PATCH semantics, which also fixes L2 for users). Policy in `internal/crud/user/policy.go`:
  - only a DB superuser can change `is_superuser` or edit a superuser account;
  - nobody can change their own superuser or active flags;
  - changing your own email or password requires `current_password`;
  - `password_hash` is never accepted from clients.
  Delete: no self-delete, and only superusers can delete superusers.
- **S5:** Admin ACL mutations (permissions, roles, assign/revoke) now require `acl.RequireSuperuserDB`. Admin user create, update and delete enforce the same superuser rules, checked against the DB.
- **S6:** `JWTClaimsMiddleware` rejects any token whose `type` isn't `access`, and it now also runs on the protected `/auth/*` routes. `Refresh` accepts only `type=refresh`, pins HS256, requires `exp`, rejects inactive users, and takes the email from the DB. The JWT middleware pins `JWTAlg: HS256` and no longer echoes parse errors.
- **L1:** `/auth/refresh` is now a public route; it authenticates with the refresh token itself.

Tests added: `main_test.go` (secret policy), `routes/auth_test.go` (access vs refresh token, alg pinning, public refresh), `internal/crud/user/policy_test.go` (escalation, self-service, partial update, delete rules), `internal/acl/middleware_test.go` (`RequireSuperuserDB`).

Still open: everything not marked ✅, notably S7 (revocation and sessions), S8/S9 (DBMS), S11–S13, L2 for blogs and admin blogs, and L13 (the pre-existing machine-dependent test).

---

## Fix log, round 2: adoption suggestions (this branch)

Implemented suggestions 1–5 and 15 from section 3 (library, declarative resources, OpenAPI from types, single binary, AI tooling, honest docs). Side effects that close review items:

- **L2/L3 ✅:** resources use PATCH semantics and ignore read-only fields (id, owner, timestamps), so mass assignment isn't possible.
- **L4 ✅:** `config.DSNFromEnv` URL-escapes credentials.
- **L5 ✅:** migrations, views, static assets and the admin UI are embedded; the binary runs from an empty directory.
- **L6 ✅:** template reloading only in development.
- **L7 ✅:** list limits are clamped in admin and resources.
- **L8 ✅:** routes are always registered; while the DB is down they answer 503, and the app reconnects in the background (development only; production fails fast).
- **L12 ✅:** one `.env` loader (`config.LoadDotEnv`) with quotes, `export` and comment support. Present-but-empty variables win.
- **L13 ✅:** the GitHub-login lookup is injectable; `go test ./...` is green on any machine.
- **L14 ✅:** `config.Config` is the single source of configuration, and the docs list exactly its variables.
- **L15 ✅:** the `routes/` package is gone; wiring lives in `app`.
- **S8 (partly):** `dbms` binds `127.0.0.1:5050` by default. Session expiry and the SSRF host restriction are still open.
- **S10 ✅:** `/monitor` is off by default and superuser-only.
- **S11 (partly):** the global error handler, typed handlers, resources and users return generic 500s with a request ID. Some admin ACL handlers still echo DB errors.
- **S19 ✅:** Swagger UI is on only in development by default (`ENABLE_DOCS`).
- **S20 ✅:** Helmet is on by default.
- **S22 ✅:** AI Swagger generation removed (no code is sent to third parties).

Bugs found and fixed while building: the typed-handler middleware order (middleware ran after the handler), and global middleware added after `app.New` skipping routes (added `app.WithMiddleware`).

Verification: unit tests in every new package; 46 end-to-end HTTP checks against PostgreSQL 16 (auth, token types, ownership rules, mass assignment, validation, admin restrictions, embedded assets); `mithril new` → `make:resource` → `go test` → `migrate up` on a generated project; MCP stdio handshake and all 7 tools; upgrade of a database built by the old 0001–0007 migrations.

Still open: S7, S9, S12–S18, S21, plus the docs suggestions not yet built (DI container, sessions, testing helpers, observability).

---

## Fix log, round 3: performance defaults (v1.3, branch `perf/fast-defaults`)

Applied the `mithril-performance` playbook: measure first, fix the top item, re-measure with the same load.

| Route (oha, keep-alive, c=64, macOS loopback) | v1.2 | v1.3 |
|---|---|---|
| `GET /` | 57,043 req/s · p99 4.93 ms | 81,220 req/s · p99 0.98 ms |
| `GET /api/blogs` (list, non-superuser) | 4,237 · p50 14.7 ms | 14,022 · p50 4.2 ms (3.3×) |
| `GET /api/blogs/:id` | 5,881 · p50 10.6 ms | 17,142 · p50 3.5 ms (2.9×) |

What changed:
- **Per-request queries:** list 4 → 2, get 3 → 2, patch 5 → 3. The superuser flag and permission set now load in one query cached per request, and lists use `count(*) OVER()`.
- **Pool:** sized 4 per CPU (10–50) instead of a fixed 10.
- **Defaults:** errors-only request logs, best-speed compression, trusted-proxy client IPs, timeouts, a cached `/openapi.json`, opt-in pprof (loopback only), and opt-in Prefork and high-throughput mode.

Found while benchmarking:
- **New bug, fixed:** SIGTERM was ignored. `app.Main` caught the signal but `Serve` never stopped, so every container deploy ended in a SIGKILL that dropped in-flight requests. It now shuts down gracefully (10 s).
- **S7 (partly):** DB-checked permission paths now deny deactivated or deleted users even when their token is still valid.

Verified with unit and race tests, the security e2e suite (52/52), a generated project (77k req/s on `/hello`, p99 1.0 ms, profile mostly syscalls), and `nginx -t` on the generated nginx config.
