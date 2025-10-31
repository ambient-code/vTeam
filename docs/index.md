# Ambient Code Platform Documentation

The **Ambient Code Platform** is a Kubernetes-native AI automation platform that orchestrates intelligent agentic sessions through containerized microservices. Built on OpenShift/Kubernetes, the platform enables AI-powered automation for code analysis, development tasks, and engineering workflows.

## Architecture Overview

The platform follows a cloud-native microservices architecture:

- **Frontend**: Next.js web application with Shadcn UI for session management and monitoring
- **Backend API**: Go-based REST API managing Kubernetes Custom Resources with multi-tenant project isolation
- **Agentic Operator**: Kubernetes controller watching CRs and orchestrating Job execution
- **Claude Code Runner**: Python-based job pods executing Claude Code CLI with multi-agent collaboration

**Key Architectural Patterns:**
- Projects map to Kubernetes namespaces with RBAC-based isolation
- OpenShift OAuth integration for authentication with user bearer tokens
- Custom Resource Definitions (AgenticSession, ProjectSettings, RFEWorkflow)
- Operator-based reconciliation for declarative session management

## Quick Start

### Local Development

```bash
# Install OpenShift Local (CRC)
brew install crc
crc setup

# Clone and deploy
git clone https://github.com/ambient-code/vTeam.git
cd vTeam
make dev-start
```

See the [Getting Started Guide](user-guide/getting-started.md) for detailed setup instructions.

### Production Deployment

For production OpenShift clusters:
- [OpenShift Deployment Guide](OPENSHIFT_DEPLOY.md)
- [OAuth Configuration](OPENSHIFT_OAUTH.md)
- [GitHub App Setup](GITHUB_APP_SETUP.md)

## Key Features

**AgenticSession Management:**
- Create AI-powered automation sessions via web UI or API
- Interactive and headless execution modes
- Multi-repository support for cross-repo analysis
- Real-time status monitoring via WebSocket
- Kubernetes Job-based execution with automatic cleanup

**Multi-Tenancy & Security:**
- Project-scoped namespaces with RBAC isolation
- User token-based authentication (no shared credentials)
- Secure API key management via Kubernetes Secrets
- Fine-grained access control through ProjectSettings

**Developer Experience:**
- Modern Next.js frontend with React Query
- RESTful API with OpenAPI documentation
- Kubernetes-native tooling (kubectl, oc CLI)
- Comprehensive logging and troubleshooting

## Documentation Structure

### [📘 User Guide](user-guide/index.md)
Learn how to use the Ambient Code Platform for AI-powered automation:
- [Getting Started](user-guide/getting-started.md) - Installation and first session

### [🧪 Labs](labs/index.md)
Hands-on exercises to master the platform:
- [Lab 1: Your First Agentic Session](labs/basic/lab-1-first-rfe.md)

### [📖 Reference](reference/index.md)
Technical reference documentation:
- [Glossary](reference/glossary.md) - Key terms and concepts

### [🚀 Deployment Guides](OPENSHIFT_DEPLOY.md)
Production deployment resources:
- [OpenShift Deployment](OPENSHIFT_DEPLOY.md)
- [OAuth Setup](OPENSHIFT_OAUTH.md)
- [GitHub App Configuration](GITHUB_APP_SETUP.md)
- [Claude Code Runner](CLAUDE_CODE_RUNNER.md)

## Getting Help

- **Documentation Issues**: [GitHub Issues](https://github.com/ambient-code/vTeam/issues)
- **Questions**: [GitHub Discussions](https://github.com/ambient-code/vTeam/discussions)
- **Source Code**: [GitHub Repository](https://github.com/ambient-code/vTeam)

## Quick Links

- New to the platform? → [Getting Started](user-guide/getting-started.md)
- Want hands-on experience? → [Lab 1](labs/basic/lab-1-first-rfe.md)
- Need reference docs? → [Glossary](reference/glossary.md)
- Deploying to production? → [OpenShift Guide](OPENSHIFT_DEPLOY.md)