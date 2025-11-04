# Deployment Checklist for LangGraph MVP

## ✅ Already Done
- ✅ Logged into OpenShift cluster (as `clusteradmin`)
- ✅ Built `langgraph-wrapper:base` image
- ✅ Built example workflow image

## 🔨 What You Need to Do

### 1. Build All Component Images

The LangGraph MVP requires these images to be built and deployed:
- **Backend** (includes Postgres DB code)
- **Frontend** 
- **Operator** (handles LangGraph workflow pods)
- **Runner** (legacy Claude runner)

```bash
# Build all images (using podman, targeting linux/amd64)
make build-all CONTAINER_ENGINE=podman PLATFORM=linux/amd64
```

### 2. Push Images to Registry

**Option A: Use Existing Images from quay.io/ambient_code**
- Skip building/pushing if you trust the existing images
- The deploy script uses `quay.io/ambient_code/*:latest` by default

**Option B: Build and Push Your Own**
```bash
# Set your registry
export REGISTRY="quay.io/gkrumbach07"

# Tag and push
make push-all CONTAINER_ENGINE=podman REGISTRY=$REGISTRY
```

### 3. Deploy to Cluster

```bash
# From project root
cd components/manifests

# Copy env file if first time
cp env.example .env

# Edit .env and set at least:
# - ANTHROPIC_API_KEY=your-key
# - (Optional) CONTAINER_REGISTRY if using custom images

# Deploy (from project root)
make deploy
```

The deploy script will:
- ✅ Check you're logged in (already done)
- ✅ Create namespace `ambient-code`
- ✅ Deploy Postgres (new!)
- ✅ Deploy Backend, Frontend, Operator
- ✅ Set up RBAC, CRDs, Routes

### 4. Verify Deployment

```bash
# Check pods
oc get pods -n ambient-code

# Wait for all pods to be Running
oc get pods -n ambient-code -w

# Check Postgres is running (NEW!)
oc get pods -n ambient-code | grep postgres

# Check services
oc get services -n ambient-code

# Get frontend route
oc get route frontend-route -n ambient-code
```

## 📝 Important Notes

1. **LangGraph Wrapper**: Already built and pushed to `quay.io/gkrumbach07/langgraph-wrapper:base` ✅
   - This is the base image users extend for their workflows
   - Doesn't need to be in Makefile - it's a separate concern

2. **Postgres**: Will be deployed automatically via `make deploy`
   - New StatefulSet, Service, Secret, ConfigMap
   - Backend connects to it automatically

3. **Cluster Login**: Already done ✅
   - `oc whoami` shows `clusteradmin`
   - `make deploy` will work

4. **Image Registry**: 
   - Default: Uses `quay.io/ambient_code/*:latest`
   - Custom: Set `CONTAINER_REGISTRY` in `.env` or override in deploy script

## 🚀 Quick Start

```bash
# 1. Build images (if needed)
make build-all CONTAINER_ENGINE=podman PLATFORM=linux/amd64

# 2. Deploy everything
make deploy

# 3. Watch pods start
oc get pods -n ambient-code -w

# 4. Get frontend URL
oc get route frontend-route -n ambient-code
```

