.PHONY: help phase1-deploy phase1-clean phase1-logs phase1-test phase1-status

KUSTOMIZE := kubectl kustomize
KUBECTL := kubectl
NAMESPACE := openwebui

help:
	@echo "Open WebUI + LiteLLM Deployment"
	@echo "================================"
	@echo ""
	@echo "Phase 1 (Kind) Commands:"
	@echo "  make phase1-deploy   - Deploy Open WebUI + LiteLLM to Kind cluster"
	@echo "  make phase1-status   - Check deployment status"
	@echo "  make phase1-logs     - View Open WebUI logs"
	@echo "  make phase1-test     - Run health checks"
	@echo "  make phase1-clean    - Remove deployment"
	@echo ""
	@echo "Prerequisites:"
	@echo "  1. Kind cluster running (run: make -C ../../e2e setup-kind)"
	@echo "  2. API key configured (edit: overlays/phase1-kind/secrets.yaml)"
	@echo ""
	@echo "Access:"
	@echo "  Docker: http://vteam.local/chat"
	@echo "  Podman: http://vteam.local:8080/chat"

phase1-deploy:
	@echo "Deploying Phase 1 (Kind)..."
	@if ! $(KUBECTL) get namespace $(NAMESPACE) >/dev/null 2>&1; then \
		echo "Creating namespace $(NAMESPACE)..."; \
	fi
	$(KUSTOMIZE) overlays/phase1-kind | $(KUBECTL) apply -f -
	@echo ""
	@echo "Waiting for deployments to be ready..."
	$(KUBECTL) wait --for=condition=available --timeout=300s \
		deployment/litellm deployment/openwebui -n $(NAMESPACE) || true
	@echo ""
	@make phase1-status
	@echo ""
	@echo "✅ Phase 1 deployed!"
	@echo ""
	@echo "Access Open WebUI:"
	@echo "  Docker: http://vteam.local/chat"
	@echo "  Podman: http://vteam.local:8080/chat"

phase1-clean:
	@echo "Removing Phase 1 deployment..."
	$(KUSTOMIZE) overlays/phase1-kind | $(KUBECTL) delete -f - --ignore-not-found=true
	@echo "✅ Phase 1 cleaned up"

phase1-logs:
	@echo "Open WebUI logs (Ctrl+C to exit):"
	$(KUBECTL) logs -f -l app=openwebui -n $(NAMESPACE) --tail=50

phase1-logs-litellm:
	@echo "LiteLLM logs (Ctrl+C to exit):"
	$(KUBECTL) logs -f -l app=litellm -n $(NAMESPACE) --tail=50

phase1-test:
	@echo "Testing LiteLLM health..."
	@$(KUBECTL) exec -n $(NAMESPACE) deployment/openwebui -- \
		curl -s http://litellm-service:4000/health || echo "❌ LiteLLM health check failed"
	@echo ""
	@echo "Testing Open WebUI connectivity to LiteLLM..."
	@$(KUBECTL) exec -n $(NAMESPACE) deployment/openwebui -- \
		curl -s http://litellm-service:4000/v1/models -H "Authorization: Bearer sk-litellm-dev-master-key" || echo "❌ Model list failed"
	@echo ""
	@echo "✅ Health checks complete"

phase1-status:
	@echo "Pod Status:"
	@$(KUBECTL) get pods -n $(NAMESPACE)
	@echo ""
	@echo "Services:"
	@$(KUBECTL) get svc -n $(NAMESPACE)
	@echo ""
	@echo "Ingress:"
	@$(KUBECTL) get ingress -n $(NAMESPACE) || true
	@echo ""
	@echo "PVC:"
	@$(KUBECTL) get pvc -n $(NAMESPACE)

phase1-shell-webui:
	@echo "Opening shell in Open WebUI pod..."
	$(KUBECTL) exec -it -n $(NAMESPACE) deployment/openwebui -- /bin/sh

phase1-shell-litellm:
	@echo "Opening shell in LiteLLM pod..."
	$(KUBECTL) exec -it -n $(NAMESPACE) deployment/litellm -- /bin/sh

phase1-port-forward:
	@echo "Port forwarding Open WebUI to localhost:8080..."
	@echo "Access at: http://localhost:8080"
	$(KUBECTL) port-forward -n $(NAMESPACE) svc/openwebui-service 8080:8080
