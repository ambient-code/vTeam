#!/bin/bash
set -euo pipefail

echo "Waiting for all deployments to be ready..."
echo ""

# Wait for backend
echo "⏳ Waiting for backend-api..."
kubectl wait --for=condition=available --timeout=300s \
  deployment/backend-api \
  -n ambient-code

# Wait for operator
echo "⏳ Waiting for agentic-operator..."
kubectl wait --for=condition=available --timeout=300s \
  deployment/agentic-operator \
  -n ambient-code

# Wait for frontend
echo "⏳ Waiting for frontend..."
kubectl wait --for=condition=available --timeout=300s \
  deployment/frontend \
  -n ambient-code

echo ""
echo "✅ All pods are ready!"
echo ""

# Show pod status
kubectl get pods -n ambient-code

