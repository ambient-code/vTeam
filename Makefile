.PHONY: help setup-env build-all build-frontend build-backend build-operator build-runner deploy clean dev-frontend dev-backend lint test registry-login push-all dev-start dev-stop dev-test dev-logs-operator dev-restart-operator dev-operator-status dev-test-operator e2e-test e2e-setup e2e-clean setup-hooks remove-hooks

# Default target
help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Configuration Variables:'
	@echo '  CONTAINER_ENGINE   Container engine to use (default: docker, can be set to podman)'
	@echo '  PLATFORM           Target platform (e.g., linux/amd64, linux/arm64)'
	@echo '  BUILD_FLAGS        Additional flags to pass to build command'
	@echo '  REGISTRY           Container registry for push operations'
	@echo ''
	@echo 'Examples:'
	@echo '  make build-all CONTAINER_ENGINE=podman'
	@echo '  make build-all PLATFORM=linux/amd64'
	@echo '  make build-all BUILD_FLAGS="--no-cache --pull"'
	@echo '  make build-all CONTAINER_ENGINE=podman PLATFORM=linux/arm64'
	@echo ''
	@echo 'Targets:'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-15s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

# Container engine configuration
CONTAINER_ENGINE ?= docker
PLATFORM ?= linux/amd64
BUILD_FLAGS ?= 


# Construct platform flag if PLATFORM is set
ifneq ($(PLATFORM),)
PLATFORM_FLAG := --platform=$(PLATFORM)
else
PLATFORM_FLAG := 
endif

# Docker image tags
FRONTEND_IMAGE ?= vteam_frontend:latest
BACKEND_IMAGE ?= vteam_backend:latest
OPERATOR_IMAGE ?= vteam_operator:latest
RUNNER_IMAGE ?= vteam_claude_runner:latest

# Docker registry operations (customize REGISTRY as needed)
REGISTRY ?= your-registry.com

# Build all images
build-all: build-frontend build-backend build-operator build-runner ## Build all container images

# Build individual components
build-frontend: ## Build the frontend container image
	@echo "Building frontend image with $(CONTAINER_ENGINE)..."
	cd components/frontend && $(CONTAINER_ENGINE) build $(PLATFORM_FLAG) $(BUILD_FLAGS) -t $(FRONTEND_IMAGE) .

build-backend: ## Build the backend API container image
	@echo "Building backend image with $(CONTAINER_ENGINE)..."
	cd components/backend && $(CONTAINER_ENGINE) build $(PLATFORM_FLAG) $(BUILD_FLAGS) -t $(BACKEND_IMAGE) .

build-operator: ## Build the operator container image
	@echo "Building operator image with $(CONTAINER_ENGINE)..."
	cd components/operator && $(CONTAINER_ENGINE) build $(PLATFORM_FLAG) $(BUILD_FLAGS) -t $(OPERATOR_IMAGE) .

build-runner: ## Build the Claude Code runner container image
	@echo "Building Claude Code runner image with $(CONTAINER_ENGINE)..."
	cd components/runners && $(CONTAINER_ENGINE) build $(PLATFORM_FLAG) $(BUILD_FLAGS) -t $(RUNNER_IMAGE) -f claude-code-runner/Dockerfile .

# Kubernetes deployment
deploy: ## Deploy all components to OpenShift (production overlay)
	@echo "Deploying to OpenShift..."
	cd components/manifests && ./deploy.sh

# Cleanup
clean: ## Clean up all Kubernetes resources (production overlay)
	@echo "Cleaning up Kubernetes resources..."
	cd components/manifests && ./deploy.sh clean



push-all: ## Push all images to registry
	$(CONTAINER_ENGINE) tag $(FRONTEND_IMAGE) $(REGISTRY)/$(FRONTEND_IMAGE)
	$(CONTAINER_ENGINE) tag $(BACKEND_IMAGE) $(REGISTRY)/$(BACKEND_IMAGE)
	$(CONTAINER_ENGINE) tag $(OPERATOR_IMAGE) $(REGISTRY)/$(OPERATOR_IMAGE)
	$(CONTAINER_ENGINE) tag $(RUNNER_IMAGE) $(REGISTRY)/$(RUNNER_IMAGE)
	$(CONTAINER_ENGINE) push $(REGISTRY)/$(FRONTEND_IMAGE)
	$(CONTAINER_ENGINE) push $(REGISTRY)/$(BACKEND_IMAGE)
	$(CONTAINER_ENGINE) push $(REGISTRY)/$(OPERATOR_IMAGE)
	$(CONTAINER_ENGINE) push $(REGISTRY)/$(RUNNER_IMAGE)

# Git hooks for branch protection
setup-hooks: ## Install git hooks for branch protection
	@./scripts/install-git-hooks.sh

remove-hooks: ## Remove git hooks
	@echo "Removing git hooks..."
	@rm -f .git/hooks/pre-commit
	@rm -f .git/hooks/pre-push
	@echo "✅ Git hooks removed"

# Local dev helpers (OpenShift Local/CRC-based)
dev-start: setup-hooks ## Start local dev (CRC + OpenShift + backend + frontend)
	@bash components/scripts/local-dev/crc-start.sh

dev-stop: ## Stop local dev processes
	@bash components/scripts/local-dev/crc-stop.sh

dev-test: ## Run local dev smoke tests
	@bash components/scripts/local-dev/crc-test.sh

# Additional CRC options
dev-stop-cluster: ## Stop local dev and shutdown CRC cluster
	@bash components/scripts/local-dev/crc-stop.sh --stop-cluster

dev-clean: ## Stop local dev and delete OpenShift project  
	@bash components/scripts/local-dev/crc-stop.sh --delete-project

# Development mode with hot-reloading
dev-start-hot: ## Start local dev with hot-reloading enabled
	@DEV_MODE=true bash components/scripts/local-dev/crc-start.sh

dev-sync: ## Start file sync for hot-reloading (run in separate terminal)
	@bash components/scripts/local-dev/crc-dev-sync.sh both

dev-sync-backend: ## Sync only backend files
	@bash components/scripts/local-dev/crc-dev-sync.sh backend

dev-sync-frontend: ## Sync only frontend files
	@bash components/scripts/local-dev/crc-dev-sync.sh frontend

dev-logs: ## Show logs for both backend and frontend
	@echo "Backend logs:"
	@oc logs -f deployment/vteam-backend -n vteam-dev --tail=20 &
	@echo -e "\n\nFrontend logs:"
	@oc logs -f deployment/vteam-frontend -n vteam-dev --tail=20

dev-logs-backend: ## Show backend logs with Air output
	@oc logs -f deployment/vteam-backend -n vteam-dev

dev-logs-frontend: ## Show frontend logs with Next.js output
	@oc logs -f deployment/vteam-frontend -n vteam-dev

dev-logs-operator: ## Show operator logs
	@oc logs -f deployment/vteam-operator -n vteam-dev

dev-restart-operator: ## Restart operator deployment
	@echo "Restarting operator..."
	@oc rollout restart deployment/vteam-operator -n vteam-dev
	@oc rollout status deployment/vteam-operator -n vteam-dev --timeout=60s

dev-operator-status: ## Show operator status and recent events
	@echo "Operator Deployment Status:"
	@oc get deployment vteam-operator -n vteam-dev
	@echo ""
	@echo "Operator Pod Status:"
	@oc get pods -n vteam-dev -l app=vteam-operator
	@echo ""
	@echo "Recent Operator Events:"
	@oc get events -n vteam-dev --field-selector involvedObject.kind=Deployment,involvedObject.name=vteam-operator --sort-by='.lastTimestamp' | tail -10

dev-test-operator: ## Run only operator tests
	@echo "Running operator-specific tests..."
	@bash components/scripts/local-dev/crc-test.sh 2>&1 | grep -A 1 "Operator"

# E2E Testing with kind
e2e-test: ## Run complete e2e test suite (setup, deploy, test, cleanup)
	@echo "Running e2e tests..."
	@# Clean up any existing cluster first
	@cd e2e && CONTAINER_ENGINE=$(CONTAINER_ENGINE) ./scripts/cleanup.sh 2>/dev/null || true
	@# Setup and deploy (allows password prompt for /etc/hosts)
	cd e2e && CONTAINER_ENGINE=$(CONTAINER_ENGINE) ./scripts/setup-kind.sh
	cd e2e && CONTAINER_ENGINE=$(CONTAINER_ENGINE) ./scripts/deploy.sh
	@# Run tests with cleanup trap (no more password prompts needed)
	@cd e2e && trap 'CONTAINER_ENGINE=$(CONTAINER_ENGINE) ./scripts/cleanup.sh' EXIT; ./scripts/run-tests.sh

e2e-setup: ## Install e2e test dependencies
	@echo "Installing e2e test dependencies..."
	cd e2e && npm install

e2e-clean: ## Clean up e2e test environment
	@echo "Cleaning up e2e environment..."
	cd e2e && CONTAINER_ENGINE=$(CONTAINER_ENGINE) ./scripts/cleanup.sh
