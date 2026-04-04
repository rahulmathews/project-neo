.PHONY: help install-tools lint-go format-go format-check-go docker-up docker-down docker-build docker-logs supabase-start supabase-stop dev-up dev-down

help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Available targets:'
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}'

install-tools: ## Install Go development tools
	@echo "Installing Go tools..."
	@go install mvdan.cc/gofumpt@latest
	@go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	@go install github.com/air-verse/air@latest
	@echo "✓ Go tools installed"

lint-go: ## Lint Go code
	@echo "Linting Go code..."
	@golangci-lint run --config .golangci.yml ./...

format-go: ## Format Go code
	@echo "Formatting Go code..."
	@gofumpt -w .

format-check-go: ## Check Go code formatting
	@echo "Checking Go code formatting..."
	@test -z "$$(gofumpt -l .)" || (echo "Go files need formatting:" && gofumpt -l . && exit 1)

docker-up: ## Start all app services
	@docker compose up -d

docker-down: ## Stop all app services
	@docker compose down

docker-build: ## Build all Docker images
	@docker compose build

docker-logs: ## Tail logs from all services
	@docker compose logs -f

supabase-start: ## Start Supabase local instance
	@supabase start

supabase-stop: ## Stop Supabase local instance
	@supabase stop

dev-up: ## Start everything (Supabase + app services)
	@supabase start
	@docker compose up -d

dev-down: ## Stop everything (app services + Supabase)
	@docker compose down
	@supabase stop
