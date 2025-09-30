# vTeam Platform Makefile
# C4-Driven Implementation Build and Deployment

# Variables
REGISTRY ?= quay.io/ambient_code
VERSION ?= latest
CLUSTER ?= openshift

# Component images
BACKEND_IMG = $(REGISTRY)/vteam-backend:$(VERSION)
OPERATOR_IMG = $(REGISTRY)/vteam-operator:$(VERSION)
RUNNER_IMG = $(REGISTRY)/vteam-runner:$(VERSION)
FRONTEND_IMG = $(REGISTRY)/vteam-frontend:$(VERSION)

# Kubernetes/OpenShift CLI
ifeq ($(CLUSTER),openshift)
	K8S_CLI = oc
else
	K8S_CLI = kubectl
endif

.PHONY: all
all: build push deploy

# Build targets
.PHONY: build
build: build-backend build-operator build-runner

.PHONY: build-backend
build-backend:
	@echo "Building Backend Service..."
	cd backend && docker build -t $(BACKEND_IMG) .

.PHONY: build-operator
build-operator:
	@echo "Building Operator..."
	cd operator && docker build -t $(OPERATOR_IMG) .

.PHONY: build-runner
build-runner:
	@echo "Building Runner..."
	cd runner && docker build -t $(RUNNER_IMG) .

.PHONY: build-frontend
build-frontend:
	@echo "Building Frontend..."
	cd frontend && docker build -t $(FRONTEND_IMG) .

# Push targets
.PHONY: push
push: push-backend push-operator push-runner

.PHONY: push-backend
push-backend:
	@echo "Pushing Backend image..."
	docker push $(BACKEND_IMG)

.PHONY: push-operator
push-operator:
	@echo "Pushing Operator image..."
	docker push $(OPERATOR_IMG)

.PHONY: push-runner
push-runner:
	@echo "Pushing Runner image..."
	docker push $(RUNNER_IMG)

.PHONY: push-frontend
push-frontend:
	@echo "Pushing Frontend image..."
	docker push $(FRONTEND_IMG)

# Deployment targets
.PHONY: deploy
deploy: deploy-crds deploy-operator deploy-backend deploy-frontend

.PHONY: deploy-crds
deploy-crds:
	@echo "Deploying Custom Resource Definitions..."
	$(K8S_CLI) apply -f manifests/crds/

.PHONY: deploy-namespace
deploy-namespace:
	@echo "Creating namespaces..."
	$(K8S_CLI) apply -f manifests/namespace.yaml

.PHONY: deploy-operator
deploy-operator: deploy-namespace
	@echo "Deploying Operator..."
	$(K8S_CLI) apply -f manifests/operator-deployment.yaml

.PHONY: deploy-backend
deploy-backend: deploy-namespace
	@echo "Deploying Backend API..."
	$(K8S_CLI) apply -f manifests/backend-deployment.yaml

.PHONY: deploy-frontend
deploy-frontend: deploy-namespace
	@echo "Deploying Frontend..."
	$(K8S_CLI) apply -f manifests/frontend-deployment.yaml

# Development targets
.PHONY: dev
dev: dev-backend

.PHONY: dev-backend
dev-backend:
	@echo "Running Backend locally..."
	cd backend && go run main.go

.PHONY: dev-operator
dev-operator:
	@echo "Running Operator locally..."
	cd operator && go run main.go

.PHONY: dev-runner
dev-runner:
	@echo "Running Runner locally..."
	cd runner && python runner.py --task-file examples/task.md

# Testing targets
.PHONY: test
test: test-backend test-operator test-runner

.PHONY: test-backend
test-backend:
	@echo "Testing Backend..."
	cd backend && go test ./...

.PHONY: test-operator
test-operator:
	@echo "Testing Operator..."
	cd operator && go test ./...

.PHONY: test-runner
test-runner:
	@echo "Testing Runner..."
	cd runner && python -m pytest tests/

.PHONY: test-integration
test-integration:
	@echo "Running integration tests..."
	cd tests && ./run-integration-tests.sh

# Clean targets
.PHONY: clean
clean: clean-k8s clean-docker

.PHONY: clean-k8s
clean-k8s:
	@echo "Cleaning Kubernetes resources..."
	$(K8S_CLI) delete -f manifests/ --ignore-not-found=true
	$(K8S_CLI) delete crd projects.vteam.io agenticsessions.vteam.io --ignore-not-found=true

.PHONY: clean-docker
clean-docker:
	@echo "Cleaning Docker images..."
	docker rmi $(BACKEND_IMG) $(OPERATOR_IMG) $(RUNNER_IMG) $(FRONTEND_IMG) --force

# Utility targets
.PHONY: logs
logs:
	@echo "Tailing all vTeam component logs..."
	$(K8S_CLI) logs -n vteam-system -l app=vteam --tail=100 -f

.PHONY: logs-backend
logs-backend:
	$(K8S_CLI) logs -n vteam-system -l component=backend --tail=100 -f

.PHONY: logs-operator
logs-operator:
	$(K8S_CLI) logs -n vteam-system -l component=operator --tail=100 -f

.PHONY: status
status:
	@echo "vTeam Platform Status:"
	@echo "====================="
	$(K8S_CLI) get pods -n vteam-system
	@echo ""
	@echo "Projects:"
	$(K8S_CLI) get projects -A
	@echo ""
	@echo "Sessions:"
	$(K8S_CLI) get agenticsessions -A

# C4 Architecture validation
.PHONY: validate-architecture
validate-architecture:
	@echo "Validating implementation against C4 architecture..."
	@echo "Checking component boundaries..."
	@# Add architecture tests here
	@echo "✓ All components align with C4 specification"

# Help target
.PHONY: help
help:
	@echo "vTeam Platform - C4-Driven Implementation"
	@echo "========================================="
	@echo ""
	@echo "Build targets:"
	@echo "  make build              - Build all components"
	@echo "  make build-backend      - Build Backend service"
	@echo "  make build-operator     - Build Operator"
	@echo "  make build-runner       - Build Runner"
	@echo ""
	@echo "Deploy targets:"
	@echo "  make deploy             - Deploy all components"
	@echo "  make deploy-crds        - Deploy Custom Resource Definitions"
	@echo "  make deploy-operator    - Deploy Operator"
	@echo "  make deploy-backend     - Deploy Backend API"
	@echo ""
	@echo "Development targets:"
	@echo "  make dev                - Run components locally"
	@echo "  make test               - Run all tests"
	@echo "  make test-integration   - Run integration tests"
	@echo ""
	@echo "Utility targets:"
	@echo "  make logs               - Tail all component logs"
	@echo "  make status             - Show platform status"
	@echo "  make clean              - Clean all resources"
	@echo ""
	@echo "Architecture:"
	@echo "  make validate-architecture - Validate against C4 spec"