# mithril-rev Makefile

.PHONY: help install run run-air build test clean

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

test: ## Run tests
	@echo "Running tests..."
	go test -v ./...

clean: ## Clean build artifacts
	@echo "Cleaning..."
	rm -rf bin/
	rm -rf tmp/
	rm -f coverage.out coverage.html

kill:
	@pids=$$(lsof -t -i:4000); \
	if [ -n "$$pids" ]; then \
		kill -9 $$pids; \
	fi
# Variables
APP_NAME ?= mithril-rev
