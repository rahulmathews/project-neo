.PHONY: help install-tools lint-go format-go format-check-go docker-up docker-down docker-build docker-logs supabase-start supabase-stop dev-up dev-down

help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Available targets:'
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}'

install-tools: ## Install Go development tools (run once after cloning)
	@echo "Installing Go tools..."
	@go install mvdan.cc/gofumpt@latest
	@go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	@go install github.com/air-verse/air@latest
	@go install golang.org/x/tools/cmd/goimports@latest
	@echo "✓ Go tools installed"
	@echo "Note: dart format comes with the Flutter SDK (no install needed)"

lint-go: ## Lint Go code (full suite)
	@echo "Linting Go code..."
	@golangci-lint run --config .golangci.yml ./...

lint-go-fast: ## Lint Go code (fast subset, same as pre-commit)
	@echo "Linting Go code (fast)..."
	@golangci-lint run --fast --timeout 60s ./...

lint-go-fix: ## Lint Go code and auto-fix what's possible
	@echo "Linting and fixing Go code..."
	@golangci-lint run --fix --config .golangci.yml ./...

format-go: ## Format Go code (imports + gofumpt)
	@echo "Formatting Go code..."
	@goimports -w .
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
