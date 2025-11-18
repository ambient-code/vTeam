.PHONY: help build-all build-frontend build-backend build-operator build-runner deploy clean registry-login push-all local-up local-down local-clean local-status local-rebuild local-reload-backend local-reload-frontend local-reload-operator test-all local-test-dev local-test-quick local-logs local-logs-backend local-logs-frontend local-logs-operator local-shell local-shell-frontend local-test local-url local-port-forward local-troubleshoot check-minikube check-kubectl dev-test-operator e2e-test e2e-setup e2e-clean validate-makefile lint-makefile check-shell makefile-health

# Default target
.DEFAULT_GOAL := help

# Configuration
CONTAINER_ENGINE ?= podman
PLATFORM ?= linux/amd64
BUILD_FLAGS ?= 
NAMESPACE ?= ambient-code
REGISTRY ?= quay.io/your-org

# Image tags
FRONTEND_IMAGE ?= vteam-frontend:latest
BACKEND_IMAGE ?= vteam-backend:latest
OPERATOR_IMAGE ?= vteam-operator:latest
RUNNER_IMAGE ?= vteam-runner:latest

# Colors for output
COLOR_RESET := \033[0m
COLOR_BOLD := \033[1m
COLOR_GREEN := \033[32m
COLOR_YELLOW := \033[33m
COLOR_BLUE := \033[34m
COLOR_RED := \033[31m

# Platform flag
ifneq ($(PLATFORM),)
PLATFORM_FLAG := --platform=$(PLATFORM)
else
PLATFORM_FLAG :=
endif

##@ General

help: ## Display this help message
	@echo '$(COLOR_BOLD)Ambient Code Platform - Development Makefile$(COLOR_RESET)'
	@echo ''
	@echo '$(COLOR_BOLD)Quick Start:$(COLOR_RESET)'
	@echo '  $(COLOR_GREEN)make local-up$(COLOR_RESET)            Start local development environment'
	@echo '  $(COLOR_GREEN)make local-status$(COLOR_RESET)        Check status of local environment'
	@echo '  $(COLOR_GREEN)make local-logs$(COLOR_RESET)          View logs from all components'
	@echo '  $(COLOR_GREEN)make local-down$(COLOR_RESET)          Stop local environment'
	@echo ''
	@echo '$(COLOR_BOLD)Quality Assurance:$(COLOR_RESET)'
	@echo '  $(COLOR_GREEN)make validate-makefile$(COLOR_RESET)   Validate Makefile quality (runs in CI)'
	@echo '  $(COLOR_GREEN)make makefile-health$(COLOR_RESET)     Run comprehensive health check'
	@echo ''
	@awk 'BEGIN {FS = ":.*##"; printf "$(COLOR_BOLD)Available Targets:$(COLOR_RESET)\n"} /^[a-zA-Z_-]+:.*?##/ { printf "  $(COLOR_BLUE)%-20s$(COLOR_RESET) %s\n", $$1, $$2 } /^##@/ { printf "\n$(COLOR_BOLD)%s$(COLOR_RESET)\n", substr($$0, 5) } ' $(MAKEFILE_LIST)
	@echo ''
	@echo '$(COLOR_BOLD)Configuration Variables:$(COLOR_RESET)'
	@echo '  CONTAINER_ENGINE=$(CONTAINER_ENGINE)  (docker or podman)'
	@echo '  NAMESPACE=$(NAMESPACE)'
	@echo '  PLATFORM=$(PLATFORM)'
	@echo ''
	@echo '$(COLOR_BOLD)Examples:$(COLOR_RESET)'
	@echo '  make local-up CONTAINER_ENGINE=docker'
	@echo '  make local-reload-backend'
	@echo '  make build-all PLATFORM=linux/arm64'

##@ Building

build-all: build-frontend build-backend build-operator build-runner ## Build all container images

build-frontend: ## Build frontend image (production)
	@echo "$(COLOR_BLUE)▶$(COLOR_RESET) Building frontend with $(CONTAINER_ENGINE)..."
	@cd components/frontend && $(CONTAINER_ENGINE) build $(PLATFORM_FLAG) $(BUILD_FLAGS) -t $(FRONTEND_IMAGE) .
	@echo "$(COLOR_GREEN)✓$(COLOR_RESET) Frontend built: $(FRONTEND_IMAGE)"

build-backend: ## Build backend image
	@echo "$(COLOR_BLUE)▶$(COLOR_RESET) Building backend with $(CONTAINER_ENGINE)..."
	@cd components/backend && $(CONTAINER_ENGINE) build $(PLATFORM_FLAG) $(BUILD_FLAGS) -t $(BACKEND_IMAGE) .
	@echo "$(COLOR_GREEN)✓$(COLOR_RESET) Backend built: $(BACKEND_IMAGE)"

build-operator: ## Build operator image
	@echo "$(COLOR_BLUE)▶$(COLOR_RESET) Building operator with $(CONTAINER_ENGINE)..."
	@cd components/operator && $(CONTAINER_ENGINE) build $(PLATFORM_FLAG) $(BUILD_FLAGS) -t $(OPERATOR_IMAGE) .
	@echo "$(COLOR_GREEN)✓$(COLOR_RESET) Operator built: $(OPERATOR_IMAGE)"

build-runner: ## Build Claude Code runner image
	@echo "$(COLOR_BLUE)▶$(COLOR_RESET) Building runner with $(CONTAINER_ENGINE)..."
	@cd components/runners && $(CONTAINER_ENGINE) build $(PLATFORM_FLAG) $(BUILD_FLAGS) -t $(RUNNER_IMAGE) -f claude-code-runner/Dockerfile .
	@echo "$(COLOR_GREEN)✓$(COLOR_RESET) Runner built: $(RUNNER_IMAGE)"

##@ Registry Operations

registry-login: ## Login to container registry
	@echo "$(COLOR_BLUE)▶$(COLOR_RESET) Logging in to $(REGISTRY)..."
	@$(CONTAINER_ENGINE) login $(REGISTRY)

push-all: registry-login ## Push all images to registry
	@echo "$(COLOR_BLUE)▶$(COLOR_RESET) Pushing images to $(REGISTRY)..."
	@for image in $(FRONTEND_IMAGE) $(BACKEND_IMAGE) $(OPERATOR_IMAGE) $(RUNNER_IMAGE); do \
		echo "  Tagging and pushing $$image..."; \
		$(CONTAINER_ENGINE) tag $$image $(REGISTRY)/$$image && \
		$(CONTAINER_ENGINE) push $(REGISTRY)/$$image; \
	done
	@echo "$(COLOR_GREEN)✓$(COLOR_RESET) All images pushed"

##@ Local Development (Minikube)

local-up: check-minikube check-kubectl ## Start local development environment (minikube)
	@echo "$(COLOR_BOLD)🚀 Starting Ambient Code Platform Local Environment$(COLOR_RESET)"
	@echo ""
	@echo "$(COLOR_BLUE)▶$(COLOR_RESET) Step 1/8: Starting minikube..."
	@minikube start --memory=4096 --cpus=2 2>/dev/null || \
		(minikube status >/dev/null 2>&1 && echo "$(COLOR_GREEN)✓$(COLOR_RESET) Minikube already running") || \
		(echo "$(COLOR_RED)✗$(COLOR_RESET) Failed to start minikube" && exit 1)
	@echo "$(COLOR_BLUE)▶$(COLOR_RESET) Step 2/8: Enabling addons..."
	@minikube addons enable ingress >/dev/null 2>&1 || true
	@minikube addons enable storage-provisioner >/dev/null 2>&1 || true
	@echo "$(COLOR_BLUE)▶$(COLOR_RESET) Step 3/8: Building images..."
	@$(MAKE) --no-print-directory _build-and-load
	@echo "$(COLOR_BLUE)▶$(COLOR_RESET) Step 4/8: Creating namespace..."
	@kubectl create namespace $(NAMESPACE) --dry-run=client -o yaml | kubectl apply -f - >/dev/null 2>&1
	@echo "$(COLOR_BLUE)▶$(COLOR_RESET) Step 5/8: Applying CRDs and RBAC..."
	@kubectl apply -f components/manifests/base/crds/ >/dev/null 2>&1 || true
	@kubectl apply -f components/manifests/base/rbac/ >/dev/null 2>&1 || true
	@kubectl apply -f components/manifests/minikube/local-dev-rbac.yaml >/dev/null 2>&1 || true
	@echo "$(COLOR_BLUE)▶$(COLOR_RESET) Step 6/8: Creating storage..."
	@kubectl apply -f components/manifests/base/workspace-pvc.yaml -n $(NAMESPACE) >/dev/null 2>&1 || true
	@echo "$(COLOR_BLUE)▶$(COLOR_RESET) Step 7/8: Deploying services..."
	@kubectl apply -f components/manifests/minikube/backend-deployment.yaml >/dev/null 2>&1
	@kubectl apply -f components/manifests/minikube/backend-service.yaml >/dev/null 2>&1
	@kubectl apply -f components/manifests/minikube/frontend-deployment-dev.yaml >/dev/null 2>&1
	@kubectl apply -f components/manifests/minikube/frontend-service.yaml >/dev/null 2>&1
	@kubectl apply -f components/manifests/minikube/operator-deployment.yaml >/dev/null 2>&1
	@echo "$(COLOR_BLUE)▶$(COLOR_RESET) Step 8/8: Setting up ingress..."
	@kubectl wait --namespace ingress-nginx --for=condition=ready pod \
		--selector=app.kubernetes.io/component=controller --timeout=90s >/dev/null 2>&1 || true
	@kubectl apply -f components/manifests/minikube/ingress.yaml >/dev/null 2>&1 || true
	@echo ""
	@echo "$(COLOR_GREEN)✓ Ambient Code Platform is starting up!$(COLOR_RESET)"
	@echo ""
	@$(MAKE) --no-print-directory _show-access-info
	@echo ""
	@echo "$(COLOR_YELLOW)⚠  Next steps:$(COLOR_RESET)"
	@echo "  • Wait ~30s for pods to be ready"
	@echo "  • Run: $(COLOR_BOLD)make local-status$(COLOR_RESET) to check deployment"
	@echo "  • Run: $(COLOR_BOLD)make local-logs$(COLOR_RESET) to view logs"

local-down: check-kubectl ## Stop Ambient Code Platform (keep minikube running)
	@echo "$(COLOR_BLUE)▶$(COLOR_RESET) Stopping Ambient Code Platform..."
	@kubectl delete namespace $(NAMESPACE) --ignore-not-found=true --timeout=60s
	@echo "$(COLOR_GREEN)✓$(COLOR_RESET) Ambient Code Platform stopped (minikube still running)"
	@echo "  To stop minikube: $(COLOR_BOLD)make local-clean$(COLOR_RESET)"

local-clean: check-minikube ## Delete minikube cluster completely
	@echo "$(COLOR_BLUE)▶$(COLOR_RESET) Deleting minikube cluster..."
	@minikube delete
	@echo "$(COLOR_GREEN)✓$(COLOR_RESET) Minikube cluster deleted"

local-status: check-kubectl ## Show status of local deployment
	@echo "$(COLOR_BOLD)📊 Ambient Code Platform Status$(COLOR_RESET)"
	@echo ""
	@echo "$(COLOR_BOLD)Minikube:$(COLOR_RESET)"
	@minikube status 2>/dev/null || echo "$(COLOR_RED)✗$(COLOR_RESET) Minikube not running"
	@echo ""
	@echo "$(COLOR_BOLD)Pods:$(COLOR_RESET)"
	@kubectl get pods -n $(NAMESPACE) -o wide 2>/dev/null || echo "$(COLOR_RED)✗$(COLOR_RESET) Namespace not found"
	@echo ""
	@echo "$(COLOR_BOLD)Services:$(COLOR_RESET)"
	@kubectl get svc -n $(NAMESPACE) 2>/dev/null | grep -E "NAME|NodePort" || echo "No services found"
	@echo ""
	@$(MAKE) --no-print-directory _show-access-info

local-rebuild: ## Rebuild and reload all components
	@echo "$(COLOR_BOLD)🔄 Rebuilding all components...$(COLOR_RESET)"
	@$(MAKE) --no-print-directory _build-and-load
	@$(MAKE) --no-print-directory _restart-all
	@echo "$(COLOR_GREEN)✓$(COLOR_RESET) All components rebuilt and reloaded"

local-reload-backend: ## Rebuild and reload backend only
	@echo "$(COLOR_BLUE)▶$(COLOR_RESET) Rebuilding backend..."
	@cd components/backend && $(CONTAINER_ENGINE) build -t $(BACKEND_IMAGE) . >/dev/null 2>&1
	@minikube image load $(BACKEND_IMAGE) >/dev/null 2>&1
	@echo "$(COLOR_BLUE)▶$(COLOR_RESET) Restarting backend..."
	@kubectl rollout restart deployment/backend-api -n $(NAMESPACE) >/dev/null 2>&1
	@kubectl rollout status deployment/backend-api -n $(NAMESPACE) --timeout=60s
	@echo "$(COLOR_GREEN)✓$(COLOR_RESET) Backend reloaded"

local-reload-frontend: ## Rebuild and reload frontend only
	@echo "$(COLOR_BLUE)▶$(COLOR_RESET) Rebuilding frontend..."
	@cd components/frontend && $(CONTAINER_ENGINE) build -t vteam-frontend-dev:latest -f Dockerfile.dev . >/dev/null 2>&1
	@minikube image load vteam-frontend-dev:latest >/dev/null 2>&1
	@echo "$(COLOR_BLUE)▶$(COLOR_RESET) Restarting frontend..."
	@kubectl rollout restart deployment/frontend -n $(NAMESPACE) >/dev/null 2>&1
	@kubectl rollout status deployment/frontend -n $(NAMESPACE) --timeout=60s
	@echo "$(COLOR_GREEN)✓$(COLOR_RESET) Frontend reloaded"


local-reload-operator: ## Rebuild and reload operator only
	@echo "$(COLOR_BLUE)▶$(COLOR_RESET) Rebuilding operator..."
	@cd components/operator && $(CONTAINER_ENGINE) build -t $(OPERATOR_IMAGE) . >/dev/null 2>&1
	@minikube image load $(OPERATOR_IMAGE) >/dev/null 2>&1
	@echo "$(COLOR_BLUE)▶$(COLOR_RESET) Restarting operator..."
	@kubectl rollout restart deployment/agentic-operator -n $(NAMESPACE) >/dev/null 2>&1
	@kubectl rollout status deployment/agentic-operator -n $(NAMESPACE) --timeout=60s
	@echo "$(COLOR_GREEN)✓$(COLOR_RESET) Operator reloaded"

##@ Testing

test-all: local-test-quick local-test-dev ## Run all tests (quick + comprehensive)

##@ Quality Assurance

validate-makefile: lint-makefile check-shell ## Validate Makefile quality and syntax
	@echo "$(COLOR_GREEN)✓ Makefile validation passed$(COLOR_RESET)"

lint-makefile: ## Lint Makefile for syntax and best practices
	@echo "$(COLOR_BLUE)▶$(COLOR_RESET) Linting Makefile..."
	@# Check that all targets have help text or are internal/phony
	@missing_help=$$(awk '/^[a-zA-Z_-]+:/ && !/##/ && !/^_/ && !/^\.PHONY/ && !/^\.DEFAULT_GOAL/' $(MAKEFILE_LIST)); \
	if [ -n "$$missing_help" ]; then \
		echo "$(COLOR_YELLOW)⚠$(COLOR_RESET)  Targets missing help text:"; \
		echo "$$missing_help" | head -5; \
	fi
	@# Check for common mistakes
	@if grep -n "^\t " $(MAKEFILE_LIST) | grep -v "^#" >/dev/null 2>&1; then \
		echo "$(COLOR_RED)✗$(COLOR_RESET) Found tabs followed by spaces (use tabs only for indentation)"; \
		grep -n "^\t " $(MAKEFILE_LIST) | head -3; \
		exit 1; \
	fi
	@# Check for undefined variable references (basic check)
	@if grep -E '\$$[^($$@%<^+?*]' $(MAKEFILE_LIST) | grep -v "^#" | grep -v '\$$\$$' >/dev/null 2>&1; then \
		echo "$(COLOR_YELLOW)⚠$(COLOR_RESET)  Possible unprotected variable references found"; \
	fi
	@# Verify .PHONY declarations exist
	@if ! grep -q "^\.PHONY:" $(MAKEFILE_LIST); then \
		echo "$(COLOR_RED)✗$(COLOR_RESET) No .PHONY declarations found"; \
		exit 1; \
	fi
	@echo "$(COLOR_GREEN)✓$(COLOR_RESET) Makefile syntax validated"

check-shell: ## Validate shell scripts with shellcheck (if available)
	@echo "$(COLOR_BLUE)▶$(COLOR_RESET) Checking shell scripts..."
	@if command -v shellcheck >/dev/null 2>&1; then \
		echo "  Running shellcheck on test scripts..."; \
		shellcheck tests/local-dev-test.sh 2>/dev/null || echo "$(COLOR_YELLOW)⚠$(COLOR_RESET)  shellcheck warnings in tests/local-dev-test.sh"; \
		if [ -d e2e/scripts ]; then \
			shellcheck e2e/scripts/*.sh 2>/dev/null || echo "$(COLOR_YELLOW)⚠$(COLOR_RESET)  shellcheck warnings in e2e scripts"; \
		fi; \
		echo "$(COLOR_GREEN)✓$(COLOR_RESET) Shell scripts checked"; \
	else \
		echo "$(COLOR_YELLOW)⚠$(COLOR_RESET)  shellcheck not installed (optional)"; \
		echo "  Install with: brew install shellcheck (macOS) or apt-get install shellcheck (Linux)"; \
	fi

makefile-health: check-minikube check-kubectl ## Run comprehensive Makefile health check
	@echo "$(COLOR_BOLD)🏥 Makefile Health Check$(COLOR_RESET)"
	@echo ""
	@echo "$(COLOR_BOLD)Prerequisites:$(COLOR_RESET)"
	@minikube version >/dev/null 2>&1 && echo "$(COLOR_GREEN)✓$(COLOR_RESET) minikube available" || echo "$(COLOR_RED)✗$(COLOR_RESET) minikube missing"
	@kubectl version --client >/dev/null 2>&1 && echo "$(COLOR_GREEN)✓$(COLOR_RESET) kubectl available" || echo "$(COLOR_RED)✗$(COLOR_RESET) kubectl missing"
	@command -v $(CONTAINER_ENGINE) >/dev/null 2>&1 && echo "$(COLOR_GREEN)✓$(COLOR_RESET) $(CONTAINER_ENGINE) available" || echo "$(COLOR_RED)✗$(COLOR_RESET) $(CONTAINER_ENGINE) missing"
	@echo ""
	@echo "$(COLOR_BOLD)Configuration:$(COLOR_RESET)"
	@echo "  CONTAINER_ENGINE = $(CONTAINER_ENGINE)"
	@echo "  NAMESPACE = $(NAMESPACE)"
	@echo "  PLATFORM = $(PLATFORM)"
	@echo ""
	@$(MAKE) --no-print-directory validate-makefile
	@echo ""
	@echo "$(COLOR_GREEN)✓ Makefile health check complete$(COLOR_RESET)"

local-test-dev: ## Run local developer experience tests
	@echo "$(COLOR_BLUE)▶$(COLOR_RESET) Running local developer experience tests..."
	@./tests/local-dev-test.sh

local-test-quick: check-kubectl check-minikube ## Quick smoke test of local environment
	@echo "$(COLOR_BOLD)🧪 Quick Smoke Test$(COLOR_RESET)"
	@echo ""
	@echo "$(COLOR_BLUE)▶$(COLOR_RESET) Testing minikube..."
	@minikube status >/dev/null 2>&1 && echo "$(COLOR_GREEN)✓$(COLOR_RESET) Minikube running" || (echo "$(COLOR_RED)✗$(COLOR_RESET) Minikube not running" && exit 1)
	@echo "$(COLOR_BLUE)▶$(COLOR_RESET) Testing namespace..."
	@kubectl get namespace $(NAMESPACE) >/dev/null 2>&1 && echo "$(COLOR_GREEN)✓$(COLOR_RESET) Namespace exists" || (echo "$(COLOR_RED)✗$(COLOR_RESET) Namespace missing" && exit 1)
	@echo "$(COLOR_BLUE)▶$(COLOR_RESET) Testing pods..."
	@kubectl get pods -n $(NAMESPACE) 2>/dev/null | grep -q "Running" && echo "$(COLOR_GREEN)✓$(COLOR_RESET) Pods running" || (echo "$(COLOR_RED)✗$(COLOR_RESET) No pods running" && exit 1)
	@echo "$(COLOR_BLUE)▶$(COLOR_RESET) Testing backend health..."
	@curl -sf http://$$(minikube ip):30080/health >/dev/null 2>&1 && echo "$(COLOR_GREEN)✓$(COLOR_RESET) Backend healthy" || (echo "$(COLOR_RED)✗$(COLOR_RESET) Backend not responding" && exit 1)
	@echo "$(COLOR_BLUE)▶$(COLOR_RESET) Testing frontend..."
	@curl -sf http://$$(minikube ip):30030 >/dev/null 2>&1 && echo "$(COLOR_GREEN)✓$(COLOR_RESET) Frontend accessible" || (echo "$(COLOR_RED)✗$(COLOR_RESET) Frontend not responding" && exit 1)
	@echo ""
	@echo "$(COLOR_GREEN)✓ Quick smoke test passed!$(COLOR_RESET)"

dev-test-operator: ## Run only operator tests
	@echo "Running operator-specific tests..."
	@bash components/scripts/local-dev/crc-test.sh 2>&1 | grep -A 1 "Operator"

##@ Development Tools

local-logs: check-kubectl ## Show logs from all components (follow mode)
	@echo "$(COLOR_BOLD)📋 Streaming logs from all components (Ctrl+C to stop)$(COLOR_RESET)"
	@kubectl logs -n $(NAMESPACE) -l 'app in (backend-api,frontend,agentic-operator)' --tail=20 --prefix=true -f 2>/dev/null || \
		echo "$(COLOR_RED)✗$(COLOR_RESET) No pods found. Run 'make local-status' to check deployment."

local-logs-backend: check-kubectl ## Show backend logs only
	@kubectl logs -n $(NAMESPACE) -l app=backend-api --tail=100 -f

local-logs-frontend: check-kubectl ## Show frontend logs only
	@kubectl logs -n $(NAMESPACE) -l app=frontend --tail=100 -f

local-logs-operator: check-kubectl ## Show operator logs only
	@kubectl logs -n $(NAMESPACE) -l app=agentic-operator --tail=100 -f

local-shell: check-kubectl ## Open shell in backend pod
	@echo "$(COLOR_BLUE)▶$(COLOR_RESET) Opening shell in backend pod..."
	@kubectl exec -it -n $(NAMESPACE) $$(kubectl get pod -n $(NAMESPACE) -l app=backend-api -o jsonpath='{.items[0].metadata.name}' 2>/dev/null) -- /bin/sh 2>/dev/null || \
		echo "$(COLOR_RED)✗$(COLOR_RESET) Backend pod not found or not ready"

local-shell-frontend: check-kubectl ## Open shell in frontend pod
	@echo "$(COLOR_BLUE)▶$(COLOR_RESET) Opening shell in frontend pod..."
	@kubectl exec -it -n $(NAMESPACE) $$(kubectl get pod -n $(NAMESPACE) -l app=frontend -o jsonpath='{.items[0].metadata.name}' 2>/dev/null) -- /bin/sh 2>/dev/null || \
		echo "$(COLOR_RED)✗$(COLOR_RESET) Frontend pod not found or not ready"

local-test: local-test-quick ## Alias for local-test-quick (backward compatibility)

local-url: check-minikube ## Display access URLs
	@$(MAKE) --no-print-directory _show-access-info

local-port-forward: check-kubectl ## Port-forward for direct access (8080→backend, 3000→frontend)
	@echo "$(COLOR_BOLD)🔌 Setting up port forwarding$(COLOR_RESET)"
	@echo ""
	@echo "  Backend:  http://localhost:8080"
	@echo "  Frontend: http://localhost:3000"
	@echo ""
	@echo "$(COLOR_YELLOW)Press Ctrl+C to stop$(COLOR_RESET)"
	@echo ""
	@trap 'echo ""; echo "$(COLOR_GREEN)✓$(COLOR_RESET) Port forwarding stopped"; exit 0' INT; \
	(kubectl port-forward -n $(NAMESPACE) svc/backend-service 8080:8080 >/dev/null 2>&1 &); \
	(kubectl port-forward -n $(NAMESPACE) svc/frontend-service 3000:3000 >/dev/null 2>&1 &); \
	wait

local-troubleshoot: check-kubectl ## Show troubleshooting information
	@echo "$(COLOR_BOLD)🔍 Troubleshooting Information$(COLOR_RESET)"
	@echo ""
	@echo "$(COLOR_BOLD)Pod Status:$(COLOR_RESET)"
	@kubectl get pods -n $(NAMESPACE) -o wide 2>/dev/null || echo "$(COLOR_RED)✗$(COLOR_RESET) No pods found"
	@echo ""
	@echo "$(COLOR_BOLD)Recent Events:$(COLOR_RESET)"
	@kubectl get events -n $(NAMESPACE) --sort-by='.lastTimestamp' | tail -10 2>/dev/null || echo "No events"
	@echo ""
	@echo "$(COLOR_BOLD)Failed Pods (if any):$(COLOR_RESET)"
	@kubectl get pods -n $(NAMESPACE) --field-selector=status.phase!=Running,status.phase!=Succeeded 2>/dev/null || echo "All pods are running"
	@echo ""
	@echo "$(COLOR_BOLD)Pod Descriptions:$(COLOR_RESET)"
	@for pod in $$(kubectl get pods -n $(NAMESPACE) -o name 2>/dev/null | head -3); do \
		echo ""; \
		echo "$(COLOR_BLUE)$$pod:$(COLOR_RESET)"; \
		kubectl describe -n $(NAMESPACE) $$pod | grep -A 5 "Conditions:\|Events:" | head -10; \
	done

##@ Production Deployment

deploy: ## Deploy to production Kubernetes cluster
	@echo "$(COLOR_BLUE)▶$(COLOR_RESET) Deploying to Kubernetes..."
	@cd components/manifests && ./deploy.sh
	@echo "$(COLOR_GREEN)✓$(COLOR_RESET) Deployment complete"

clean: ## Clean up Kubernetes resources
	@echo "$(COLOR_BLUE)▶$(COLOR_RESET) Cleaning up..."
	@cd components/manifests && ./deploy.sh clean
	@echo "$(COLOR_GREEN)✓$(COLOR_RESET) Cleanup complete"

##@ Internal Helpers (do not call directly)

check-minikube: ## Check if minikube is installed
	@command -v minikube >/dev/null 2>&1 || \
		(echo "$(COLOR_RED)✗$(COLOR_RESET) minikube not found. Install: https://minikube.sigs.k8s.io/docs/start/" && exit 1)

check-kubectl: ## Check if kubectl is installed
	@command -v kubectl >/dev/null 2>&1 || \
		(echo "$(COLOR_RED)✗$(COLOR_RESET) kubectl not found. Install: https://kubernetes.io/docs/tasks/tools/" && exit 1)

_build-and-load: ## Internal: Build and load images
	@$(CONTAINER_ENGINE) build -t $(BACKEND_IMAGE) components/backend >/dev/null 2>&1
	@$(CONTAINER_ENGINE) build -t vteam-frontend-dev:latest -f components/frontend/Dockerfile.dev components/frontend >/dev/null 2>&1
	@$(CONTAINER_ENGINE) build -t $(OPERATOR_IMAGE) components/operator >/dev/null 2>&1
	@minikube image load $(BACKEND_IMAGE) >/dev/null 2>&1
	@minikube image load vteam-frontend-dev:latest >/dev/null 2>&1
	@minikube image load $(OPERATOR_IMAGE) >/dev/null 2>&1
	@echo "$(COLOR_GREEN)✓$(COLOR_RESET) Images built and loaded"

_restart-all: ## Internal: Restart all deployments
	@kubectl rollout restart deployment -n $(NAMESPACE) >/dev/null 2>&1
	@echo "$(COLOR_BLUE)▶$(COLOR_RESET) Waiting for deployments to be ready..."
	@kubectl rollout status deployment -n $(NAMESPACE) --timeout=90s >/dev/null 2>&1 || true

_show-access-info: ## Internal: Show access information
	@echo "$(COLOR_BOLD)🌐 Access URLs:$(COLOR_RESET)"
	@MINIKUBE_IP=$$(minikube ip 2>/dev/null) && \
		echo "  Frontend: $(COLOR_BLUE)http://$$MINIKUBE_IP:30030$(COLOR_RESET)" && \
		echo "  Backend:  $(COLOR_BLUE)http://$$MINIKUBE_IP:30080$(COLOR_RESET)" || \
		echo "  $(COLOR_RED)✗$(COLOR_RESET) Cannot get minikube IP"
	@echo ""
	@echo "$(COLOR_BOLD)Alternative:$(COLOR_RESET) Port forward for localhost access"
	@echo "  Run: $(COLOR_BOLD)make local-port-forward$(COLOR_RESET)"
	@echo "  Then access:"
	@echo "    Frontend: $(COLOR_BLUE)http://localhost:3000$(COLOR_RESET)"
	@echo "    Backend:  $(COLOR_BLUE)http://localhost:8080$(COLOR_RESET)"
	@echo ""
	@echo "$(COLOR_YELLOW)⚠  SECURITY NOTE:$(COLOR_RESET) Authentication is DISABLED for local development."

##@ E2E Testing
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
