# OpenShift Deployment Guide

This guide explains how to deploy the RHOAI AI Feature Sizing Platform to your OpenShift environment.

## Overview

The application consists of:
- **LlamaDeploy Backend**: Multi-agent workflow orchestration 
- **Chat UI**: TypeScript-based modern chat interface
- **Multiple Services**: RFE builder and Jira RFE to architecture workflows
- **Persistent Storage**: For outputs and vector indices

## Prerequisites

### Required Tools
```bash
# OpenShift CLI
curl -LO https://mirror.openshift.com/pub/openshift-v4/clients/ocp/latest/openshift-client-linux.tar.gz

# Docker (for building images)
# Install according to your OS

# Verify installations
oc version
docker --version
```

### Authentication
```bash
# Login to your OpenShift cluster
oc login https://your-openshift-cluster.com

# Verify access
oc whoami
oc get projects
```

### Container Registry
Ensure you have access to a container registry (e.g., OpenShift internal registry, Docker Hub, Quay.io):

```bash
# Example: Using OpenShift internal registry
oc get routes -n openshift-image-registry

# Example: Login to external registry
docker login your-registry.com
```

## Quick Deployment

> **📋 For build and deployment instructions, see [openshift/README.md](openshift/README.md)**

The OpenShift README covers:
- Building images with podman/docker auto-detection
- Deploy script usage with environment variables
- Kustomize deployment
- Individual manifest deployment

## Configuration

### 1. API Keys (Required)
Update the secret with your API keys:

```bash
# Update OpenAI API key
oc patch secret rhoai-secrets -n rhoai-ai-feature-sizing -p '{"stringData":{"OPENAI_API_KEY":"sk-your-openai-key-here"}}'

# Optional: Add Anthropic API key
oc patch secret rhoai-secrets -n rhoai-ai-feature-sizing -p '{"stringData":{"ANTHROPIC_API_KEY":"your-anthropic-key-here"}}'
```

### 2. Environment Variables
Modify `openshift/configmap.yaml` to adjust:
- LLM provider settings
- Chunk size and overlap
- Upload limits
- Agent timeout settings

### 3. Storage Classes
Update `openshift/pvc.yaml` to use your cluster's storage class:
```yaml
storageClassName: gp2  # Change to your storage class
```

Find available storage classes:
```bash
oc get storageclass
```

### 4. Resource Limits
Adjust resource requests/limits in `openshift/deployment.yaml`:
```yaml
resources:
  requests:
    memory: "512Mi"
    cpu: "250m"
  limits:
    memory: "2Gi"
    cpu: "1000m"
```

## Architecture in OpenShift

```
┌─────────────────────────────────────────────────┐
│                OpenShift Cluster                │
│                                                 │
│  ┌─────────────────────────────────────────┐   │
│  │     Namespace: rhoai-ai-feature-sizing  │   │
│  │                                         │   │
│  │  ┌─────────────┐    ┌────────────────┐ │   │
│  │  │    Pod      │    │  ConfigMap &   │ │   │
│  │  │             │    │    Secrets     │ │   │
│  │  │ ┌─────────┐ │    │                │ │   │
│  │  │ │LlamaDep-│ │    │ - OPENAI_KEY   │ │   │
│  │  │ │loy App  │ │    │ - Config vars  │ │   │
│  │  │ │Port 4501│ │    └────────────────┘ │   │
│  │  │ │Port 8000│ │                       │   │
│  │  │ └─────────┘ │    ┌────────────────┐ │   │
│  │  └─────────────┘    │      PVCs      │ │   │
│  │         │           │                │ │   │
│  │  ┌─────────────┐    │ - output-pvc   │ │   │
│  │  │  Services   │    │                │ │   │
│  │  │             │    └────────────────┘ │   │
│  │  │ - rhoai-api │                       │   │
│  │  │ - rhoai-ui  │                       │   │
│  │  └─────────────┘                       │   │
│  │         │                               │   │
│  │  ┌─────────────┐                       │   │
│  │  │   Routes    │                       │   │
│  │  │             │                       │   │
│  │  │ - API Route │                       │   │
│  │  │ - UI Route  │                       │   │
│  │  └─────────────┘                       │   │
│  └─────────────────────────────────────────┘   │
└─────────────────────────────────────────────────┘
```

## Accessing the Application

After deployment, get the route URLs:
```bash
# Get API route
oc get route rhoai-api -n rhoai-ai-feature-sizing -o jsonpath='{.spec.host}'

# Get UI route  
oc get route rhoai-ui -n rhoai-ai-feature-sizing -o jsonpath='{.spec.host}'
```

Access points:
- **UI**: `https://[ui-route]/deployments/rhoai-ai-feature-sizing/ui`
- **API**: `https://[api-route]`  
- **Docs**: `https://[api-route]/docs`

## Monitoring and Troubleshooting

### Check Deployment Status
```bash
# View all resources
oc get all -n rhoai-ai-feature-sizing

# Check pod status
oc get pods -n rhoai-ai-feature-sizing

# View deployment status
oc rollout status deployment/rhoai-ai-feature-sizing -n rhoai-ai-feature-sizing
```

### View Logs
```bash
# Stream application logs
oc logs -f deployment/rhoai-ai-feature-sizing -n rhoai-ai-feature-sizing

# View specific pod logs
oc logs [pod-name] -n rhoai-ai-feature-sizing
```

### Debug Common Issues

#### Pod Not Starting
```bash
# Describe pod for events
oc describe pod [pod-name] -n rhoai-ai-feature-sizing

# Check resource limits
oc top pods -n rhoai-ai-feature-sizing
```

#### Storage Issues
```bash
# Check PVC status
oc get pvc -n rhoai-ai-feature-sizing

# Describe PVC for events
oc describe pvc rhoai-output-pvc -n rhoai-ai-feature-sizing
```

#### Networking Issues
```bash
# Test service connectivity
oc port-forward service/rhoai-ai-feature-sizing 8000:8000 -n rhoai-ai-feature-sizing

# Check route configuration
oc describe route rhoai-api -n rhoai-ai-feature-sizing
```

### Health Checks
The application includes health endpoints:
- `http://localhost:8000/health` (inside pod)
- Readiness probe: Checks API availability
- Liveness probe: Monitors application health

## Security Considerations

### OpenShift Security
- Runs as non-root user (UID 1000)
- Security Context Constraints (SCC) compliance
- Read-only root filesystem where possible
- Capability dropping (ALL capabilities dropped)

### Network Security
- TLS termination at OpenShift router
- Internal service-to-service communication
- Network policies can be applied for micro-segmentation

### Secrets Management
- API keys stored in OpenShift secrets
- Environment variable injection
- Secret rotation supported

## Scaling

### Horizontal Scaling
```bash
# Scale deployment
oc scale deployment rhoai-ai-feature-sizing --replicas=3 -n rhoai-ai-feature-sizing
```

### Vertical Scaling
Update resource limits in `openshift/deployment.yaml` and apply:
```bash
oc apply -f openshift/deployment.yaml
```

### Autoscaling
```yaml
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: rhoai-hpa
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: rhoai-ai-feature-sizing
  minReplicas: 1
  maxReplicas: 5
  metrics:
  - type: Resource
    resource:
      name: cpu
      target:
        type: Utilization
        averageUtilization: 70
```

## Updates and Maintenance

### Rolling Updates
```bash
# Update image tag
oc patch deployment rhoai-ai-feature-sizing -n rhoai-ai-feature-sizing -p '{"spec":{"template":{"spec":{"containers":[{"name":"rhoai-app","image":"your-registry.com/rhoai-ai-feature-sizing:new-tag"}]}}}}'

# Monitor rollout
oc rollout status deployment/rhoai-ai-feature-sizing -n rhoai-ai-feature-sizing
```

### Backup
```bash
# Backup persistent volumes (example)
oc create job backup-output --from=cronjob/volume-backup -n rhoai-ai-feature-sizing

# Export configuration
oc export all --selector=app=rhoai-ai-feature-sizing -n rhoai-ai-feature-sizing
```

## Performance Optimization

### Resource Tuning
Based on usage patterns, adjust:
- CPU/Memory requests and limits
- Storage size and performance class
- JVM heap sizes if applicable

### Caching
- Vector index caching in persistent storage
- Session context persistence
- Static asset caching at router level

## Support

For issues and questions:
1. Check application logs: `oc logs -f deployment/rhoai-ai-feature-sizing -n rhoai-ai-feature-sizing`
2. Review OpenShift events: `oc get events -n rhoai-ai-feature-sizing`
3. Verify configuration: `oc get configmap,secret -n rhoai-ai-feature-sizing`
4. Test connectivity: Port-forward and direct API calls

## Cleanup

To remove the deployment:
```bash
# Delete all resources
oc delete project rhoai-ai-feature-sizing

# Or delete individual components
oc delete -f openshift/
```
