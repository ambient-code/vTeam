# Makefile for ambient-code-backend

.PHONY: help build test test-unit test-contract test-integration clean run container-build container-run

# Default target
help: ## Show this help message
	@echo "Available targets:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  %-20s %s\n", $$1, $$2}'

# Build targets
build: ## Build the backend binary
	go build -o backend .

clean: ## Clean build artifacts
	rm -f backend main
	go clean

# Test targets
test: test-unit test-contract ## Run all tests (excluding integration tests)

test-unit: ## Run unit tests
	go test ./tests/unit/... -v

test-contract: ## Run contract tests
	go test ./tests/contract/... -v

test-integration: ## Run integration tests (requires Kubernetes cluster)
	@echo "Running integration tests (requires Kubernetes cluster access)..."
	go test ./tests/integration/... -v -timeout=5m

test-integration-short: ## Run integration tests with short timeout
	go test ./tests/integration/... -v -short

test-all: test test-integration ## Run all tests including integration tests

# Test with specific configuration
test-integration-local: ## Run integration tests with local configuration
	@echo "Running integration tests with local configuration..."
	TEST_NAMESPACE=ambient-code-test \
	CLEANUP_RESOURCES=true \
	go test ./tests/integration/... -v -timeout=5m

test-integration-ci: ## Run integration tests for CI (no cleanup for debugging)
	@echo "Running integration tests for CI..."
	TEST_NAMESPACE=ambient-code-ci \
	CLEANUP_RESOURCES=false \
	go test ./tests/integration/... -v -timeout=10m -json

test-permissions: ## Run permission and RBAC integration tests specifically
	@echo "Running permission boundary and RBAC tests..."
	TEST_NAMESPACE=ambient-code-test \
	CLEANUP_RESOURCES=true \
	go test ./tests/integration/ -v -run TestPermission -timeout=5m

test-permissions-verbose: ## Run permission tests with detailed output
	@echo "Running permission tests with verbose output..."
	TEST_NAMESPACE=ambient-code-test \
	CLEANUP_RESOURCES=true \
	go test ./tests/integration/ -v -run TestPermission -timeout=5m -count=1

# Coverage targets
test-coverage: ## Run tests with coverage
	go test ./tests/unit/... ./tests/contract/... -coverprofile=coverage.out
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# Development targets
run: ## Run the backend server locally
	go run .

dev: ## Run with live reload (requires air: go install github.com/cosmtrek/air@latest)
	air

# Container targets
CONTAINER_ENGINE ?= podman

container-build: ## Build container image
	$(CONTAINER_ENGINE) build -t ambient-code-backend .

container-run: ## Run container
	$(CONTAINER_ENGINE) run -p 8080:8080 ambient-code-backend

# Linting and formatting
fmt: ## Format Go code
	go fmt ./...

vet: ## Run go vet
	go vet ./...

lint: ## Run golangci-lint (requires golangci-lint to be installed)
	golangci-lint run

# Dependency management
deps: ## Download dependencies
	go mod download

deps-update: ## Update dependencies
	go get -u ./...
	go mod tidy

deps-verify: ## Verify dependencies
	go mod verify

# Installation targets for development tools
install-tools: ## Install development tools
	@echo "Installing development tools..."
	go install github.com/cosmtrek/air@latest
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# Kubernetes-specific targets for integration testing
k8s-setup: ## Setup local Kubernetes for testing (requires kubectl and kind)
	@echo "Setting up local Kubernetes cluster for testing..."
	kind create cluster --name ambient-test || true
	kubectl config use-context kind-ambient-test
	@echo "Installing test CRDs..."
	kubectl apply -f ../manifests/crds/ || echo "Warning: Could not install CRDs"

k8s-teardown: ## Teardown local Kubernetes test cluster
	@echo "Tearing down test cluster..."
	kind delete cluster --name ambient-test || true

# Pre-commit hooks
pre-commit: fmt vet test ## Run pre-commit checks

# Build information
version: ## Show version information
	@echo "Go version: $(shell go version)"
	@echo "Git commit: $(shell git rev-parse --short HEAD 2>/dev/null || echo 'unknown')"
	@echo "Build time: $(shell date)"

# Environment validation
check-env: ## Check environment setup for development
	@echo "Checking environment..."
	@go version >/dev/null 2>&1 || (echo "❌ Go not installed"; exit 1)
	@echo "✅ Go installed: $(shell go version)"
	@kubectl version --client >/dev/null 2>&1 || echo "⚠️  kubectl not found (needed for integration tests)"
	@podman version >/dev/null 2>&1 || echo "⚠️  Podman not found (needed for container builds)"
	@echo "Environment check complete"