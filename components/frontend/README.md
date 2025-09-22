## Ambient Agentic Runner — Frontend (Next.js)

Next.js UI for managing Agentic Sessions and Projects. In local development it proxies API calls to the backend and forwards incoming auth/context headers; it does not spoof identities.

### Prerequisites
- Node.js 20+ and npm
- Go 1.24+ (to run the backend locally)
- oc/kubectl configured to your OpenShift/Kubernetes cluster

### Backend (local) quick start
Run the backend locally while targeting your cluster.

1) Install CRDs to your cluster
```bash
oc apply -f ../manifests/crd.yaml
oc apply -f ../manifests/projectsettings-crd.yaml
```

2) Create/label a project namespace (example: my-project)
```bash
oc new-project my-project || oc project my-project
oc label namespace my-project ambient-code.io/managed=true --overwrite
oc annotate namespace my-project \
  ambient-code.io/display-name="My Project" --overwrite
```

3) Start the backend (defaults to port 8080)
```bash
cd ../backend
export KUBECONFIG="$HOME/.kube/config"   # or your kubeconfig path
go run .
# Health: curl http://localhost:8080/health
```

### Frontend (local) quick start

**Recommended: Use integrated CRC development environment:**
```bash
# From repository root - single command setup
make dev-start
# Access: https://vteam-frontend-vteam-dev.apps-crc.testing
```

**Alternative: Standalone frontend development:**
```bash
# From this directory, install and run:
npm ci
export BACKEND_URL=http://localhost:8080/api  # Adjust for your backend
npm run dev
# Open http://localhost:3000
```

### Header forwarding model (dev and prod)
Next.js API routes forward incoming headers to the backend. They do not auto-inject user identity. In development, you can optionally provide values via environment or `oc`:

- Forwarded when present on the request:
  - `X-Forwarded-User`, `X-Forwarded-Email`, `X-Forwarded-Preferred-Username`
  - `X-Forwarded-Groups`
  - `X-OpenShift-Project`
  - `Authorization: Bearer <token>` (forwarded as `X-Forwarded-Access-Token`)
- Optional dev helpers:
  - `OC_USER`, `OC_EMAIL`, `OC_TOKEN`
  - `ENABLE_OC_WHOAMI=1` to let the server call `oc whoami` / `oc whoami -t`

In production, put an OAuth/ingress proxy in front of the app to set these headers.

### Environment variables
- `BACKEND_URL` (default: `http://localhost:8080/api`)
  - Used by server-side API routes to reach the backend.
- Optional dev helpers: `OC_USER`, `OC_EMAIL`, `OC_TOKEN`, `ENABLE_OC_WHOAMI=1`

You can also put these in a `.env.local` file in this folder:
```
BACKEND_URL=http://localhost:8080/api
# Optional dev helpers
# OC_USER=your.name
# OC_EMAIL=your.name@example.com
# OC_TOKEN=...
# ENABLE_OC_WHOAMI=1
```

### Verifying requests
Backend directly (requires headers):
```bash
curl -i http://localhost:8080/api/projects/my-project/agentic-sessions \
  -H "X-OpenShift-Project: my-project" \
  -H "X-Forwarded-User: dev" \
  -H "X-Forwarded-Groups: ambient-project:my-project:admin"
```

Through the frontend route (forwards headers to backend):
```bash
curl -i http://localhost:3000/api/projects/my-project/agentic-sessions \
  -H "X-OpenShift-Project: my-project"
```

### Common issues
- 400 “Project is required …”
  - Use path `/api/projects/{project}/…` or include `X-OpenShift-Project`.
- 403 “Project is not managed by Ambient”
  - Ensure namespace is labeled `ambient-code.io/managed=true`.
- Missing auth header
  - In dev, provide `Authorization: Bearer <token>` (or use `OC_TOKEN` / `ENABLE_OC_WHOAMI`).

### Production notes
- Do not spoof identities. Forward real headers from your OAuth/ingress proxy.
- Provide a project selection mechanism and forward it as `X-OpenShift-Project` (or use project path in API URLs).
