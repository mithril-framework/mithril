# mithril-rev Makefile

.PHONY: help init install run run-air build build-linux test clean docker-build docker-run crud dc dc-run dc-stop dc-start dc-down dc-logs backup restore backup-list routes swagger secret hash sha256 sha512 encode decode migrate-up migrate-down migrate-status migrate-reset seed install-tools kill

# Default target
help: ## Show this help message
	@echo "Available commands:"
	@grep -E '^[a-zA-Z0-9_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2}'

init: ## Symlink mithril to /usr/local/bin so you can run mithril from anywhere
	@ln -sf "$(CURDIR)/mithril" /usr/local/bin/mithril || (echo "Run: sudo ln -sf $(CURDIR)/mithril /usr/local/bin/mithril"; exit 1)

install: ## Install dependencies
	go mod tidy
	go mod download

i: ## Install dependencies
	go mod tidy
	go mod download

crud: ## Generate CRUD (repository, handlers, routes) from model. Usage: make crud MODEL=User
	@test -n "$$MODEL" || (echo "Usage: make crud MODEL=User"; exit 1); \
	go run ./cmd/crud "$$MODEL"

run: ## Run the application
	@echo "Starting application..."
	go run .

run-air: ## Run with live reload (Air)
	@echo "Starting with live reload..."
	go run github.com/air-verse/air@latest -c .air.toml

run dev: ## Run with live reload (Air)
	@echo "Starting with live reload..."
	go run github.com/air-verse/air@latest -c .air.toml

build: ## Build the application
	@echo "Building application..."
	go build -o bin/app .

build-linux: ## Build the application for Linux
	@echo "Building application..."
	GOOS=linux GOARCH=amd64 go build -o bin/app-linux .

docker-build: ## Build Docker image (tag: $(APP_NAME):latest)
	@echo "Building Docker image $(APP_NAME):latest..."
	docker build -t $(APP_NAME):latest .

docker-run: docker-build ## Build Docker image and run container (port 4000)
	@echo "Running $(APP_NAME):latest..."
	docker run --rm -p 4000:4000 $(APP_NAME):latest

DC_FILE := infrastructure/docker-compose.services.yml
# Per-service: make dc CMD=install SVC=postgres  or  make dc-install-postgres
CMD ?=
SVC ?=

dc: ## Per-service: make dc CMD=install SVC=postgres  (CMD: install|up|start|stop|restart|logs|shell)
	@cmd="$(CMD)"; svc="$(SVC)"; \
	if [ -z "$$svc" ]; then echo "Usage: make dc CMD=install|up|start|stop|restart|logs|shell SVC=postgres|pgadmin|adminer|redis|rabbitmq|kafka"; exit 1; fi; \
	case "$$cmd" in \
	  install) docker compose -f $(DC_FILE) pull $$svc ;; \
	  up) docker compose -f $(DC_FILE) up -d $$svc ;; \
	  start) docker compose -f $(DC_FILE) start $$svc ;; \
	  stop) docker compose -f $(DC_FILE) stop $$svc ;; \
	  restart) docker compose -f $(DC_FILE) restart $$svc ;; \
	  logs) docker compose -f $(DC_FILE) logs -f $$svc ;; \
	  shell) \
	    if [ "$$svc" = "postgres" ]; then docker compose -f $(DC_FILE) exec postgres psql -U postgres; \
	    else docker compose -f $(DC_FILE) exec $$svc sh; fi ;; \
	  *) echo "Usage: make dc CMD=install|up|start|stop|restart|logs|shell SVC=<service>"; exit 1 ;; \
	esac

# Per-service shortcuts: make dc-install-postgres, make dc-up-postgres, make dc-start-postgres, etc.
dc-install-%:
	@$(MAKE) dc CMD=install SVC=$*

dc-up-%:
	@$(MAKE) dc CMD=up SVC=$*

dc-start-%:
	@$(MAKE) dc CMD=start SVC=$*

dc-stop-%:
	@$(MAKE) dc CMD=stop SVC=$*

dc-restart-%:
	@$(MAKE) dc CMD=restart SVC=$*

dc-logs-%:
	@$(MAKE) dc CMD=logs SVC=$*

dc-shell-%:
	@$(MAKE) dc CMD=shell SVC=$*

dc-run: ## Start all dev containers in background
	docker compose -f $(DC_FILE) up -d

dc-stop: ## Stop all dev containers
	docker compose -f $(DC_FILE) stop

dc-start: ## Start all stopped dev containers
	docker compose -f $(DC_FILE) start

dc-down: ## Stop and remove dev containers (keeps volumes)
	docker compose -f $(DC_FILE) down

dc-logs: ## Follow all dev containers logs
	docker compose -f $(DC_FILE) logs -f

test: ## Run tests
	@echo "Running tests..."
	go test -v ./...

clean: ## Clean build artifacts
	@echo "Cleaning..."
	rm -rf bin/
	rm -rf tmp/
	rm -f coverage.out coverage.html

install-tools: ## Install goose CLI for migrations
	go install github.com/pressly/goose/v3/cmd/goose@latest

migrate-up: ## Run database migrations up (set DATABASE_URL or DB_* in .env)
	@[ -f .env ] && set -a && . ./.env && set +a; \
	if [ -z "$$DATABASE_URL" ] && [ -n "$$DB_HOST" ]; then \
	  DATABASE_URL="postgres://$${DB_USER:-postgres}:$${DB_PASSWORD}@$${DB_HOST}:$${DB_PORT:-5432}/$${DB_NAME:-mithril_rev}?sslmode=$${DB_SSLMODE:-disable}"; export DATABASE_URL; \
	fi; \
	test -n "$$DATABASE_URL" || (echo "Set DATABASE_URL or DB_* (e.g. DB_HOST, DB_USER, DB_PASSWORD, DB_NAME) in .env"; exit 1); \
	go run github.com/pressly/goose/v3/cmd/goose@latest -dir database/migrations postgres "$$DATABASE_URL" up

migrate-down: ## Run database migrations down
	@[ -f .env ] && set -a && . ./.env && set +a; \
	if [ -z "$$DATABASE_URL" ] && [ -n "$$DB_HOST" ]; then \
	  DATABASE_URL="postgres://$${DB_USER:-postgres}:$${DB_PASSWORD}@$${DB_HOST}:$${DB_PORT:-5432}/$${DB_NAME:-mithril_rev}?sslmode=$${DB_SSLMODE:-disable}"; export DATABASE_URL; \
	fi; \
	test -n "$$DATABASE_URL" || (echo "Set DATABASE_URL or DB_* in .env"; exit 1); \
	go run github.com/pressly/goose/v3/cmd/goose@latest -dir database/migrations postgres "$$DATABASE_URL" down

migrate-status: ## Show migration status
	@[ -f .env ] && set -a && . ./.env && set +a; \
	if [ -z "$$DATABASE_URL" ] && [ -n "$$DB_HOST" ]; then \
	  DATABASE_URL="postgres://$${DB_USER:-postgres}:$${DB_PASSWORD}@$${DB_HOST}:$${DB_PORT:-5432}/$${DB_NAME:-mithril_rev}?sslmode=$${DB_SSLMODE:-disable}"; export DATABASE_URL; \
	fi; \
	test -n "$$DATABASE_URL" || (echo "Set DATABASE_URL or DB_* in .env"; exit 1); \
	go run github.com/pressly/goose/v3/cmd/goose@latest -dir database/migrations postgres "$$DATABASE_URL" status

# Use after manually dropping tables (e.g. users). Clears Goose version table so migrate-up re-runs all migrations.
migrate-reset: ## Clear migration history and re-run all migrations (requires psql)
	@[ -f .env ] && set -a && . ./.env && set +a; \
	if [ -z "$$DATABASE_URL" ] && [ -n "$$DB_HOST" ]; then \
	  DATABASE_URL="postgres://$${DB_USER:-postgres}:$${DB_PASSWORD}@$${DB_HOST}:$${DB_PORT:-5432}/$${DB_NAME:-mithril_rev}?sslmode=$${DB_SSLMODE:-disable}"; export DATABASE_URL; \
	fi; \
	test -n "$$DATABASE_URL" || (echo "Set DATABASE_URL or DB_* in .env"; exit 1); \
	echo "Clearing goose_db_version and re-running migrations..."; \
	psql "$$DATABASE_URL" -c "DELETE FROM goose_db_version;" && \
	go run github.com/pressly/goose/v3/cmd/goose@latest -dir database/migrations postgres "$$DATABASE_URL" up

backup: ## Create database backup (compressed SQL in database/backups). Uses pg_dump, Docker, or Go.
	@[ -f .env ] && set -a && . ./.env && set +a; \
	if [ -z "$$DATABASE_URL" ] && [ -n "$$DB_HOST" ]; then \
	  DATABASE_URL="postgres://$${DB_USER:-postgres}:$${DB_PASSWORD}@$${DB_HOST}:$${DB_PORT:-5432}/$${DB_NAME:-mithril_rev}?sslmode=$${DB_SSLMODE:-disable}"; export DATABASE_URL; \
	fi; \
	test -n "$$DATABASE_URL" || (echo "Set DATABASE_URL or DB_* in .env"; exit 1); \
	go run ./cmd/backup backup

restore: ## Restore from backup. make restore (latest) or make restore f=path/to/backup.sql.gz
	@[ -f .env ] && set -a && . ./.env && set +a; \
	if [ -z "$$DATABASE_URL" ] && [ -n "$$DB_HOST" ]; then \
	  DATABASE_URL="postgres://$${DB_USER:-postgres}:$${DB_PASSWORD}@$${DB_HOST}:$${DB_PORT:-5432}/$${DB_NAME:-mithril_rev}?sslmode=$${DB_SSLMODE:-disable}"; export DATABASE_URL; \
	fi; \
	test -n "$$DATABASE_URL" || (echo "Set DATABASE_URL or DB_* in .env"; exit 1); \
	go run ./cmd/backup restore -file "$${f:-latest}" -force

backup-list: ## List backups in database/backups
	@go run ./cmd/backup list

routes: ## List all registered routes (no server start)
	@[ -f .env ] && set -a && . ./.env && set +a; LIST_ROUTES=1 go run .

swagger: ## Regenerate OpenAPI schema from route files via OpenAI (OPENAI_API_KEY in .env)
	@[ -f .env ] && set -a && . ./.env && set +a; \
	test -n "$$OPENAI_API_KEY" || { echo "Set OPENAI_API_KEY in .env"; exit 1; }; \
	go run ./cmd/swagger

secret: ## Generate a new JWT_SECRET and update .env
	@[ -f .env ] || { echo "Create .env first (e.g. copy from .env.sample)"; exit 1; }; \
	SECRET=$$(openssl rand -base64 32 | tr -d '\n'); \
	if grep -q '^JWT_SECRET=' .env 2>/dev/null; then \
	  sed "s|^JWT_SECRET=.*|JWT_SECRET=$$SECRET|" .env > .env.tmp && mv .env.tmp .env; \
	else \
	  echo "JWT_SECRET=$$SECRET" >> .env; \
	fi; \
	echo "Updated JWT_SECRET in .env"

hash: ## Hash password with bcrypt (e.g. make hash 123)
	@go run ./cmd/util hash '$(filter-out $@,$(MAKECMDGOALS))'

sha256: ## SHA-256 checksum (e.g. make sha256 hello)
	@go run ./cmd/util sha256 '$(filter-out $@,$(MAKECMDGOALS))'

sha512: ## SHA-512 checksum (e.g. make sha512 hello)
	@go run ./cmd/util sha512 '$(filter-out $@,$(MAKECMDGOALS))'

encode: ## Base64-encode (e.g. make encode hello)
	@go run ./cmd/util encode '$(filter-out $@,$(MAKECMDGOALS))'

decode: ## Base64-decode (e.g. make decode hello  or  make decode S='aGVsbG8=')
	@if [ -n "$$S" ]; then S="$$S" go run ./cmd/util decode; else go run ./cmd/util decode '$(filter-out $@,$(MAKECMDGOALS))'; fi

# Absorb extra goals so "make hash 123" does not try to build target "123"
%:
	@true

# Seed is manual-only. Do not call from run/run-air or any automatic step.
seed: ## Seed database (demo user). Run migrate-up first. Loads .env like migrate-*.
	@[ -f .env ] && set -a && . ./.env && set +a; \
	touch .seed-allowed && ALLOW_SEED=1 go run ./cmd/seed; rm -f .seed-allowed

kill: ## Kill app on port 4000 and Air (parent of that process)
	@pids=$$(lsof -t -i:4000); \
	if [ -n "$$pids" ]; then \
		for pid in $$pids; do \
			ppid=$$(ps -o ppid= -p $$pid 2>/dev/null | tr -d ' '); \
			[ -n "$$ppid" ] && [ "$$ppid" -gt 1 ] && kill -9 $$ppid 2>/dev/null; \
		done; \
		kill -9 $$pids 2>/dev/null; \
	fi; \
	air_pids=$$(pgrep -f '\\.air\\.toml' 2>/dev/null); \
	[ -n "$$air_pids" ] && kill -9 $$air_pids 2>/dev/null; \
	true
# Variables
APP_NAME ?= mithril-rev
