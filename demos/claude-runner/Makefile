.PHONY: help setup-env build-all build-frontend build-backend build-operator build-runner deploy clean dev-frontend dev-backend lint test registry-login push-all

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
	@echo '  make setup-env                              # Create .env from template'
	@echo '  make build-all CONTAINER_ENGINE=podman'
	@echo '  make build-all PLATFORM=linux/amd64'
	@echo '  make build-all BUILD_FLAGS="--no-cache --pull"'
	@echo '  make build-all CONTAINER_ENGINE=podman PLATFORM=linux/arm64'
	@echo ''
	@echo 'Targets:'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-15s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

# Container engine configuration
CONTAINER_ENGINE ?= docker
PLATFORM ?= 
BUILD_FLAGS ?= 

# Construct platform flag if PLATFORM is set
ifneq ($(PLATFORM),)
PLATFORM_FLAG := --platform=$(PLATFORM)
else
PLATFORM_FLAG := 
endif

# Docker image tags
FRONTEND_IMAGE ?= claude-runner-frontend:latest
BACKEND_IMAGE ?= claude-runner-backend:latest
OPERATOR_IMAGE ?= research-operator:latest
RUNNER_IMAGE ?= claude-runner:latest

# Build all images
build-all: build-frontend build-backend build-operator build-runner ## Build all container images

# Build individual components
build-frontend: ## Build the frontend container image
	@echo "Building frontend image with $(CONTAINER_ENGINE)..."
	cd frontend && $(CONTAINER_ENGINE) build $(PLATFORM_FLAG) $(BUILD_FLAGS) -t $(FRONTEND_IMAGE) .

build-backend: ## Build the backend API container image
	@echo "Building backend image with $(CONTAINER_ENGINE)..."
	cd backend && $(CONTAINER_ENGINE) build $(PLATFORM_FLAG) $(BUILD_FLAGS) -t $(BACKEND_IMAGE) .

build-operator: ## Build the operator container image
	@echo "Building operator image with $(CONTAINER_ENGINE)..."
	cd operator && $(CONTAINER_ENGINE) build $(PLATFORM_FLAG) $(BUILD_FLAGS) -t $(OPERATOR_IMAGE) .

build-runner: ## Build the Claude runner container image
	@echo "Building Claude runner image with $(CONTAINER_ENGINE)..."
	cd claude-runner && $(CONTAINER_ENGINE) build $(PLATFORM_FLAG) $(BUILD_FLAGS) -t $(RUNNER_IMAGE) .

# Development targets
dev-frontend: ## Start frontend in development mode
	cd frontend && npm install && npm run dev

dev-backend: ## Start backend in development mode
	cd backend && go run main.go

dev-operator: ## Start operator in development mode
	cd operator && go run main.go

# Environment setup
setup-env: ## Create .env file from template
	@if [ ! -f .env ]; then \
		echo "Creating .env file from template..."; \
		cp manifests/env.example .env; \
		echo "✓ Created .env file. Please edit it with your actual values:"; \
		echo "  - Set ANTHROPIC_API_KEY to your actual API key"; \
		echo "  - Adjust other settings as needed"; \
	else \
		echo ".env file already exists"; \
	fi

# Kubernetes deployment
deploy: ## Deploy all components to Kubernetes
	@echo "Deploying to Kubernetes..."
	cd manifests && ./deploy.sh

deploy-crd: ## Deploy only the Custom Resource Definition
	kubectl apply -f manifests/crd.yaml

deploy-rbac: ## Deploy only RBAC configuration
	kubectl apply -f manifests/rbac.yaml

deploy-secrets: ## Deploy secrets and config
	kubectl apply -f manifests/secrets.yaml

deploy-backend: ## Deploy only the backend service
	kubectl apply -f manifests/backend-deployment.yaml

deploy-operator: ## Deploy only the operator
	kubectl apply -f manifests/operator-deployment.yaml

deploy-frontend: ## Deploy only the frontend
	kubectl apply -f manifests/frontend-deployment.yaml

# Cleanup
clean: ## Clean up all Kubernetes resources
	@echo "Cleaning up Kubernetes resources..."
	cd manifests && ./deploy.sh clean

# Status and monitoring
status: ## Show deployment status
	@echo "Deployment Status:"
	@echo "=================="
	kubectl get pods -l 'app in (backend-api,research-operator,frontend)' -n claude-research
	@echo ""
	kubectl get services -l 'app in (backend-api,frontend)' -n claude-research
	@echo ""
	kubectl get researchsessions -n claude-research

logs-backend: ## View backend logs
	kubectl logs -l app=backend-api -f -n claude-research

logs-operator: ## View operator logs
	kubectl logs -l app=research-operator -f -n claude-research

logs-frontend: ## View frontend logs
	kubectl logs -l app=frontend -f -n claude-research

# Port forwarding for local access
port-forward-frontend: ## Port forward frontend service to localhost:3000
	kubectl port-forward svc/frontend-service 3000:3000 -n claude-research

port-forward-backend: ## Port forward backend service to localhost:8080
	kubectl port-forward svc/backend-service 8080:8080 -n claude-research

# Development setup with Kind
kind-create: ## Create a local Kubernetes cluster with Kind
	kind create cluster --name claude-research

kind-load: build-all ## Load all images into Kind cluster
	kind load $(CONTAINER_ENGINE)-image $(FRONTEND_IMAGE) --name claude-research
	kind load $(CONTAINER_ENGINE)-image $(BACKEND_IMAGE) --name claude-research
	kind load $(CONTAINER_ENGINE)-image $(OPERATOR_IMAGE) --name claude-research
	kind load $(CONTAINER_ENGINE)-image $(RUNNER_IMAGE) --name claude-research

kind-deploy: kind-load deploy ## Deploy to Kind cluster

kind-clean: ## Delete the Kind cluster
	kind delete cluster --name claude-research

# Linting and testing
lint-frontend: ## Lint frontend code
	cd frontend && npm run lint

lint-backend: ## Lint backend code
	cd backend && go fmt ./... && go vet ./...

lint-operator: ## Lint operator code
	cd operator && go fmt ./... && go vet ./...

lint: lint-frontend lint-backend lint-operator ## Lint all code

# Testing
test-frontend: ## Run frontend tests
	cd frontend && npm test

test-backend: ## Run backend tests
	cd backend && go test ./...

test-operator: ## Run operator tests
	cd operator && go test ./...

test: test-frontend test-backend test-operator ## Run all tests

# Docker registry operations (customize REGISTRY as needed)
REGISTRY ?= your-registry.com

push-all: build-all ## Push all images to registry
	$(CONTAINER_ENGINE) tag $(FRONTEND_IMAGE) $(REGISTRY)/$(FRONTEND_IMAGE)
	$(CONTAINER_ENGINE) tag $(BACKEND_IMAGE) $(REGISTRY)/$(BACKEND_IMAGE)
	$(CONTAINER_ENGINE) tag $(OPERATOR_IMAGE) $(REGISTRY)/$(OPERATOR_IMAGE)
	$(CONTAINER_ENGINE) tag $(RUNNER_IMAGE) $(REGISTRY)/$(RUNNER_IMAGE)
	$(CONTAINER_ENGINE) push $(REGISTRY)/$(FRONTEND_IMAGE)
	$(CONTAINER_ENGINE) push $(REGISTRY)/$(BACKEND_IMAGE)
	$(CONTAINER_ENGINE) push $(REGISTRY)/$(OPERATOR_IMAGE)
	$(CONTAINER_ENGINE) push $(REGISTRY)/$(RUNNER_IMAGE)

# Utility targets
install-deps: ## Install development dependencies
	@echo "Installing frontend dependencies..."
	cd frontend && npm install
	@echo "Installing Go dependencies..."
	cd backend && go mod tidy
	cd operator && go mod tidy

create-namespace: ## Create the claude-research namespace if it doesn't exist
	kubectl apply -f manifests/namespace.yaml

# Example research session
create-example: ## Create an example research session
	kubectl apply -f - <<EOF
	apiVersion: research.example.com/v1
	kind: ResearchSession
	metadata:
	  name: example-research
	  namespace: claude-research
	spec:
	  prompt: "Analyze this website and provide insights about its design and user experience"
	  websiteURL: "https://example.com"
	  llmSettings:
	    model: "claude-3-5-sonnet-20241022"
	    temperature: 0.7
	    maxTokens: 4000
	  timeout: 300
	EOF

# Full development setup
dev-setup: install-deps build-all kind-create kind-deploy ## Complete development setup
