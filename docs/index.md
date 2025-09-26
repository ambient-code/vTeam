# vTeam

vTeam is an OpenShift-native platform for AI-assisted engineering and agentic automation. It consists of a Next.js frontend, a Go backend API, and a Go operator that reconciles Kubernetes resources.

## Current Architecture

- Projects map to OpenShift namespaces labeled `ambient-code.io/managed=true`.
- Frontend authenticates via oauth-proxy and forwards identity headers to the backend.
- Backend exposes project-scoped APIs for sessions, RFE workflows, permissions, keys, and runner secrets. All Kubernetes calls are done strictly with the caller’s token.
- Operator watches `AgenticSession` CRs and creates per-session Jobs with a PVC and a content sidecar, and updates session status.

## Quick Start

See `docs/OPENSHIFT_DEPLOY.md` to deploy on OpenShift, and `docs/OPENSHIFT_OAUTH.md` to enable oauth-proxy login.

## Key Features

- Agentic sessions (CRD) with start/stop, real-time messages via WebSocket, and runner token provisioning
- RFE workflows (CRD) with umbrella/supporting repos, Jira link management, and optional seeding
- GitHub App installation linking and repo browsing proxies
- RBAC-backed project permissions and access keys

## Where to next

- Developer overview: `docs/developer-guide/index.md`
- API reference: `docs/reference/api-endpoints.md`
- User guide: `docs/user-guide/index.md`