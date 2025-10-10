# vTeam Documentation

Welcome to vTeam, an OpenShift-native AI automation platform for intelligent agentic sessions and feature refinement workflows.

## What is vTeam?

vTeam is a platform that orchestrates Claude Code SDK sessions with specialized AI agent personas in a Kubernetes-native environment. It enables teams to:

- **Execute Agentic Sessions**: Run AI-powered code analysis, documentation, and development workflows
- **Create RFE Workflows**: Multi-phase feature refinement with specialized agent perspectives
- **Manage Projects**: Isolated multi-tenant project namespaces with RBAC
- **Work with Repositories**: Clone, modify, and push to Git repositories directly from AI sessions
- **Monitor in Real-time**: WebSocket-based live updates and interactive sessions

## Architecture Overview

vTeam consists of four main components:

- **Frontend** (Next.js 15): React-based web interface for managing sessions and workflows
- **Backend API** (Go): REST API with project-scoped multi-tenancy and WebSocket support
- **Operator** (Go): Kubernetes operator that reconciles AgenticSession and RFEWorkflow CRs
- **Runner** (Python): Executes Claude Code SDK sessions with workspace and agent persona management

Projects map to OpenShift namespaces labeled `ambient-code.io/managed=true`. The operator watches Custom Resources and creates Jobs with PVCs for persistent workspace storage.

## Quick Start

```bash
# Deploy to OpenShift
cd components/manifests
cp env.example .env
# Edit .env and set ANTHROPIC_API_KEY
make deploy

# Verify deployment
oc get pods -n ambient-code

# Get frontend URL
oc get route frontend-route -n ambient-code
```

See [Getting Started](user-guide/getting-started.md) for detailed setup instructions.

## Key Features

### Agentic Sessions
- Claude Code SDK integration with streaming messages
- Multi-repo workspace support (input/output repos)
- Interactive and one-shot execution modes
- Persistent workspace storage with PVCs
- Cost tracking and usage metrics

### RFE Workflows
- Multi-phase feature refinement (ideation, specification, planning, implementation)
- Specialized agent personas (PM, UX, engineering, docs)
- Umbrella and supporting repository management
- Jira integration for artifact publishing
- Workspace seeding and artifact tracking

### Multi-Tenancy
- Project-scoped namespaces with RBAC
- Per-project secret management
- User and group permissions
- Access keys for API authentication

### Integrations
- **GitHub App**: Private repo access, branch creation, PR automation
- **OpenShift OAuth**: SSO authentication with cluster credentials
- **Jira**: Publish artifacts to Jira issues
- **WebSocket**: Real-time message streaming

## Where to Go Next

### User Documentation
- **[Getting Started](user-guide/getting-started.md)**: Deploy vTeam and create your first session
- **[User Guide](user-guide/index.md)**: Learn how to use vTeam effectively

### Developer Documentation
- **[Developer Guide](developer-guide/index.md)**: Contribute to vTeam development
- **[Architecture Details](OPENSHIFT_DEPLOY.md)**: Understand the deployment model

### Reference
- **[OpenShift Deployment](OPENSHIFT_DEPLOY.md)**: Deployment guide and configuration
- **[OAuth Setup](OPENSHIFT_OAUTH.md)**: Enable OpenShift SSO authentication
- **[GitHub Integration](GITHUB_APP_SETUP.md)**: Configure GitHub App integration
- **[Runner Architecture](CLAUDE_CODE_RUNNER.md)**: Understand the Claude Code runner
- **[Glossary](reference/glossary.md)**: Terms and definitions

## Getting Help

- **GitHub Issues**: [Report bugs and request features](https://github.com/red-hat-data-services/vTeam/issues)
- **Documentation**: Browse the guides in the navigation menu
- **Community**: Join discussions and share your experience