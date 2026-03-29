.PHONY: help install-tools lint-go format-go test-go

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

test-go: ## Run Go tests
	@echo "Running Go tests..."
	@go test -v -race -coverprofile=coverage.out ./...
	@go tool cover -func=coverage.out

test-go-coverage: test-go ## Run Go tests with coverage report
	@go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"
