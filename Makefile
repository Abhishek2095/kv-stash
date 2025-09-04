# Makefile for kv-stash project
.PHONY: help build test lint lint-docker clean docker-build docker-run fmt vet deps check pre-commit install-tools

# Default target
help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Targets:'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-15s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod
GOFMT=gofmt
GOVET=$(GOCMD) vet

# Binary name and paths
BINARY_NAME=kvstash
BINARY_PATH=./bin/$(BINARY_NAME)
CMD_PATH=./cmd/kvstash

# Docker settings
DOCKER_IMAGE=kvstash
DOCKER_TAG=latest
GOLANGCI_LINT_VERSION=v1.57.2

# Build flags
LDFLAGS=-ldflags="-s -w"
BUILD_FLAGS=-v $(LDFLAGS)

# Test flags
TEST_FLAGS=-v -race -coverprofile=coverage.out
BENCH_FLAGS=-v -bench=. -benchmem

all: clean deps lint test build ## Run all: clean, deps, lint, test, build

build: ## Build the binary
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p bin
	$(GOBUILD) $(BUILD_FLAGS) -o $(BINARY_PATH) $(CMD_PATH)
	@echo "Binary built: $(BINARY_PATH)"

build-linux: ## Build Linux binary
	@echo "Building $(BINARY_NAME) for Linux..."
	@mkdir -p bin
	GOOS=linux GOARCH=amd64 $(GOBUILD) $(BUILD_FLAGS) -o $(BINARY_PATH)-linux-amd64 $(CMD_PATH)

build-darwin: ## Build macOS binary
	@echo "Building $(BINARY_NAME) for macOS..."
	@mkdir -p bin
	GOOS=darwin GOARCH=amd64 $(GOBUILD) $(BUILD_FLAGS) -o $(BINARY_PATH)-darwin-amd64 $(CMD_PATH)
	GOOS=darwin GOARCH=arm64 $(GOBUILD) $(BUILD_FLAGS) -o $(BINARY_PATH)-darwin-arm64 $(CMD_PATH)

build-windows: ## Build Windows binary
	@echo "Building $(BINARY_NAME) for Windows..."
	@mkdir -p bin
	GOOS=windows GOARCH=amd64 $(GOBUILD) $(BUILD_FLAGS) -o $(BINARY_PATH)-windows-amd64.exe $(CMD_PATH)

build-all: build-linux build-darwin build-windows ## Build for all platforms

test: ## Run tests
	@echo "Running tests..."
	$(GOTEST) $(TEST_FLAGS) ./...

test-verbose: ## Run tests with verbose output
	@echo "Running tests with verbose output..."
	$(GOTEST) -v -race ./...

test-coverage: ## Run tests and show coverage
	@echo "Running tests with coverage..."
	$(GOTEST) $(TEST_FLAGS) ./...
	$(GOCMD) tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

bench: ## Run benchmarks
	@echo "Running benchmarks..."
	$(GOTEST) $(BENCH_FLAGS) ./...

# Linting targets

lint: ## Run golangci-lint locally (requires golangci-lint installed)
	@echo "Running golangci-lint..."
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run --config .golangci.yml --timeout 5m; \
	else \
		echo "golangci-lint not found. Please install it or use 'make lint-docker'"; \
		echo "Install with: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest"; \
		exit 1; \
	fi

lint-docker: ## Run golangci-lint in Docker container (recommended)
	@echo "Running golangci-lint in Docker..."
	@docker run --rm \
		-v $(PWD):/app \
		-v ~/.cache/golangci-lint/$(GOLANGCI_LINT_VERSION):/root/.cache \
		-w /app \
		golangci/golangci-lint:$(GOLANGCI_LINT_VERSION) \
		golangci-lint run --config .golangci.yml --timeout 5m

lint-fix: ## Run golangci-lint with auto-fix
	@echo "Running golangci-lint with auto-fix..."
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run --config .golangci.yml --timeout 5m --fix; \
	else \
		echo "golangci-lint not found. Using Docker..."; \
		docker run --rm \
			-v $(PWD):/app \
			-v ~/.cache/golangci-lint/$(GOLANGCI_LINT_VERSION):/root/.cache \
			-w /app \
			golangci/golangci-lint:$(GOLANGCI_LINT_VERSION) \
			golangci-lint run --config .golangci.yml --timeout 5m --fix; \
	fi

lint-docker-fix: ## Run golangci-lint in Docker with auto-fix
	@echo "Running golangci-lint in Docker with auto-fix..."
	@docker run --rm \
		-v $(PWD):/app \
		-v ~/.cache/golangci-lint/$(GOLANGCI_LINT_VERSION):/root/.cache \
		-w /app \
		golangci/golangci-lint:$(GOLANGCI_LINT_VERSION) \
		golangci-lint run --config .golangci.yml --timeout 5m --fix

# Formatting and checking

fmt: ## Format Go code
	@echo "Formatting Go code..."
	@$(GOFMT) -s -w .
	@if command -v gofumpt >/dev/null 2>&1; then \
		gofumpt -w .; \
	fi
	@if command -v goimports >/dev/null 2>&1; then \
		goimports -w -local github.com/Abhishek2095/kv-stash .; \
	fi

vet: ## Run go vet
	@echo "Running go vet..."
	$(GOVET) ./...

check: fmt vet lint test ## Run all checks: format, vet, lint, test

# Dependencies

deps: ## Download and verify dependencies
	@echo "Downloading dependencies..."
	$(GOMOD) download
	$(GOMOD) verify

deps-update: ## Update dependencies
	@echo "Updating dependencies..."
	$(GOMOD) tidy
	$(GOGET) -u ./...

deps-vendor: ## Vendor dependencies
	@echo "Vendoring dependencies..."
	$(GOMOD) vendor

# Tools installation

install-tools: ## Install development tools
	@echo "Installing development tools..."
	@go install github.com/golangci/golangci-lint/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION)
	@go install mvdan.cc/gofumpt@latest
	@go install golang.org/x/tools/cmd/goimports@latest
	@go install github.com/securecodewarrior/gosec/v2/cmd/gosec@latest
	@echo "Tools installed successfully"

# Docker targets

docker-build: ## Build Docker image
	@echo "Building Docker image..."
	docker build -t $(DOCKER_IMAGE):$(DOCKER_TAG) .

docker-run: docker-build ## Run Docker container
	@echo "Running Docker container..."
	docker run --rm -p 6380:6380 $(DOCKER_IMAGE):$(DOCKER_TAG)

docker-shell: ## Run Docker container with shell
	@echo "Running Docker container with shell..."
	docker run --rm -it $(DOCKER_IMAGE):$(DOCKER_TAG) /bin/sh

# Development workflow

dev: ## Start development server (build and run)
	@make build
	@echo "Starting development server..."
	@$(BINARY_PATH)

pre-commit: clean deps fmt lint test ## Pre-commit checks
	@echo "All pre-commit checks passed!"

# Cleanup

clean: ## Clean build artifacts
	@echo "Cleaning up..."
	$(GOCLEAN)
	@rm -rf bin/
	@rm -f coverage.out coverage.html
	@rm -rf vendor/

clean-cache: ## Clean Go build cache
	@echo "Cleaning Go build cache..."
	$(GOCMD) clean -cache
	$(GOCMD) clean -testcache
	$(GOCMD) clean -modcache

# Release

release-snapshot: ## Build release snapshot with goreleaser
	@echo "Building release snapshot..."
	@if command -v goreleaser >/dev/null 2>&1; then \
		goreleaser release --snapshot --clean; \
	else \
		echo "goreleaser not found. Please install it first."; \
		echo "Install with: go install github.com/goreleaser/goreleaser/v2@latest"; \
		exit 1; \
	fi

# CI/CD helpers

ci-setup: deps install-tools ## Setup CI environment
	@echo "CI environment setup complete"

ci-test: fmt vet lint-docker test ## Run CI tests
	@echo "CI tests completed"

ci-build: build-all ## Build all platforms for CI
	@echo "CI build completed"

# Documentation

docs: ## Generate documentation
	@echo "Generating documentation..."
	@if command -v godoc >/dev/null 2>&1; then \
		echo "Starting godoc server on http://localhost:6060"; \
		godoc -http=:6060; \
	else \
		echo "godoc not found. Install with: go install golang.org/x/tools/cmd/godoc@latest"; \
	fi

# Security

security: ## Run security checks
	@echo "Running security checks..."
	@if command -v gosec >/dev/null 2>&1; then \
		gosec ./...; \
	else \
		echo "gosec not found. Install with: go install github.com/securecodewarrior/gosec/v2/cmd/gosec@latest"; \
	fi

# Monitoring the running process

run: build ## Build and run the server
	@echo "Running $(BINARY_NAME)..."
	@$(BINARY_PATH)

run-debug: build ## Build and run with debug logging
	@echo "Running $(BINARY_NAME) with debug logging..."
	@LOG_LEVEL=debug $(BINARY_PATH)

# Help with colors
.PHONY: help
help: ## Show this help message
	@echo '\033[1;32mAvailable targets:\033[0m'
	@echo ''
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[1;36m%-15s\033[0m %s\n", $$1, $$2}'
	@echo ''
	@echo '\033[1;33mCommon workflows:\033[0m'
	@echo '  make pre-commit  # Run all checks before committing'
	@echo '  make dev         # Build and run development server'
	@echo '  make lint-docker # Lint code using Docker (recommended)'
	@echo '  make ci-test     # Run all CI tests'
