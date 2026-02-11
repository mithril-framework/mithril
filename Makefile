# myproject7lication Makefile

.PHONY: help install run build test clean docker-up docker-down migrate seed

# Default target
help: ## Show this help message
	@echo "Available commands:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2}'

install: ## Install dependencies
	go mod tidy
	go mod download

run: ## Run the application in development mode
	@echo "Starting Mithril application..."
	go run main.go

run-air: ## Run with live reload (uses air via go run)
	@echo "Starting with live reload..."
	go run github.com/air-verse/air@latest -c .air.toml

build: ## Build the application
	@echo "Building application..."
	go build -o bin/app main.go

test: ## Run tests
	@echo "Running tests..."
	go test -v ./...

test-coverage: ## Run tests with coverage
	@echo "Running tests with coverage..."
	go test -v -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

clean: ## Clean build artifacts
	@echo "Cleaning..."
	rm -rf bin/
	rm -f coverage.out coverage.html

# Database commands
migrate: ## Run database migrations
	@echo "Running migrations..."
	@if [ -f artisan ]; then ./artisan migrate; else echo "Artisan not found. Run 'make install' first."; fi

migrate-rollback: ## Rollback last migration
	@echo "Rolling back migration..."
	@if [ -f artisan ]; then ./artisan migrate:rollback; else echo "Artisan not found. Run 'make install' first."; fi

migrate-fresh: ## Fresh migration (drop and recreate)
	@echo "Fresh migration..."
	@if [ -f artisan ]; then ./artisan migrate:fresh; else echo "Artisan not found. Run 'make install' first."; fi

seed: ## Seed the database
	@echo "Seeding database..."
	@if [ -f artisan ]; then ./artisan db:seed; else echo "Artisan not found. Run 'make install' first."; fi

# Docker commands
docker-build: ## Build Docker image
	@echo "Building Docker image..."
	docker build -t $(APP_NAME) .

docker-up: ## Start Docker containers
	@echo "Starting Docker containers..."
	docker-compose -f docker-compose.dev.yml up -d

docker-up-prod: ## Start production Docker containers
	@echo "Starting production Docker containers..."
	docker-compose -f docker-compose.prod.yml up -d

docker-down: ## Stop Docker containers
	@echo "Stopping Docker containers..."
	docker-compose -f docker-compose.dev.yml down
	docker-compose -f docker-compose.prod.yml down

docker-logs: ## Show Docker logs
	@echo "Showing Docker logs..."
	docker-compose -f docker-compose.dev.yml logs -f

# Development commands
dev-setup: install migrate seed ## Setup development environment
	@echo "Development environment setup complete!"

dev-reset: clean docker-down docker-up migrate seed ## Reset development environment
	@echo "Development environment reset complete!"

# Production commands
prod-build: ## Build for production
	@echo "Building for production..."
	CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o bin/app main.go

prod-deploy: prod-build docker-up-prod ## Deploy to production
	@echo "Deployment complete!"

# Utility commands
lint: ## Run linter
	@echo "Running linter..."
	golangci-lint run

format: ## Format code
	@echo "Formatting code..."
	go fmt ./...

# Variables
APP_NAME ?= mithril-app
