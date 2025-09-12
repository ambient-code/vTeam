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
	cd components/runners/claude-code-runner && $(CONTAINER_ENGINE) build $(PLATFORM_FLAG) $(BUILD_FLAGS) -t $(RUNNER_IMAGE) .

# Kubernetes deployment
deploy: ## Deploy all components to Kubernetes
	@echo "Deploying to Kubernetes..."
	cd components/manifests && ./deploy.sh

# Cleanup
clean: ## Clean up all Kubernetes resources
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
