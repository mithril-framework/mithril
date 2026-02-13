# mithril-rev Makefile

.PHONY: help install run run-air build build-linux test clean docker-build backup restore backup-list migrate-up migrate-down migrate-status migrate-reset seed install-tools kill

# Default target
help: ## Show this help message
	@echo "Available commands:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2}'

install: ## Install dependencies
	go mod tidy
	go mod download

run: ## Run the application
	@echo "Starting application..."
	go run .

run-air: ## Run with live reload (Air)
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
