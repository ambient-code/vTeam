# User Guide

Welcome to the Ambient Code Platform User Guide! This section provides everything you need to effectively use the platform for AI-powered automation and agentic development workflows.

## What You'll Learn

This guide covers the essential aspects of using the Ambient Code Platform:

### 🚀 [Getting Started](getting-started.md)
- Complete setup and installation (local and production)
- Configure your Anthropic API key
- Create your first AgenticSession
- Verify your environment is working
- Troubleshoot common issues

## Core Concepts

Before diving in, understand these key concepts:

### **AgenticSession**
An AgenticSession is a Kubernetes Custom Resource representing an AI-powered automation task. Each session:
- Executes a prompt using Claude Code
- Can operate on one or multiple GitHub repositories
- Runs as a Kubernetes Job with isolated workspace
- Supports interactive (long-running) and headless (batch) modes
- Tracks status, results, and per-repo push/abandon decisions

### **Projects & Namespaces**
The platform uses Kubernetes namespaces for multi-tenant isolation:
- Each project maps to a namespace
- Users authenticate with OpenShift OAuth
- RBAC controls who can create/view sessions
- ProjectSettings CR manages API keys and defaults

### **Session Modes**

**Headless Mode** (`interactive: false`):
- Single-prompt execution with timeout
- Ideal for batch tasks, CI/CD automation
- Session completes and exits automatically

**Interactive Mode** (`interactive: true`):
- Long-running chat sessions
- Uses inbox/outbox files for communication
- Ideal for iterative development, debugging
- Runs until explicitly stopped

## User Workflows

### For Developers

**Automate repetitive tasks:**
- Code analysis and documentation generation
- Refactoring and modernization
- Test generation and coverage improvements
- Security vulnerability scanning

**Example session:**
```yaml
apiVersion: vteam.ambient-code/v1alpha1
kind: AgenticSession
metadata:
  name: analyze-repo
  namespace: my-project
spec:
  prompt: "Analyze this codebase and generate comprehensive API documentation"
  repos:
    - input:
        url: https://github.com/myorg/myrepo
        branch: main
  interactive: false
  timeout: 3600
```

### For Engineering Teams

**Improve development velocity:**
- Automated code reviews
- Cross-repository analysis
- Migration and upgrade automation
- Consistency checking across microservices

**Multi-repo session example:**
```yaml
spec:
  prompt: "Compare authentication patterns in these services and create a unified approach"
  repos:
    - input:
        url: https://github.com/myorg/service-a
    - input:
        url: https://github.com/myorg/service-b
  mainRepoIndex: 0  # service-a is the working directory
```

### For Team Leads

**Manage automation at scale:**
- Configure ProjectSettings for your team
- Set default models and timeouts
- Manage API keys via Kubernetes Secrets
- Monitor session execution and costs
- Review session results and approve PRs

## Prerequisites

Before using the platform, ensure you have:

- [ ] OpenShift or Kubernetes cluster access
- [ ] Ambient Code Platform deployed and running ([Deployment Guides](../OPENSHIFT_DEPLOY.md))
- [ ] Anthropic Claude API key
- [ ] Project created with your user granted access
- [ ] Basic familiarity with GitHub workflows

## Quick Navigation

- **New to the platform?** → Start with [Getting Started](getting-started.md)
- **Want hands-on practice?** → Try [Lab 1: Your First Agentic Session](../labs/basic/lab-1-first-rfe.md)
- **Need technical details?** → Check the [Reference Documentation](../reference/index.md)
- **Deploying the platform?** → See [OpenShift Deployment Guide](../OPENSHIFT_DEPLOY.md)

## Getting Help

If you encounter issues:

- **Common problems**: See the [Troubleshooting section](getting-started.md#common-issues) in Getting Started
- **Documentation bugs**: [Submit an issue](https://github.com/ambient-code/platform/issues)
- **Questions**: [GitHub Discussions](https://github.com/ambient-code/platform/discussions)
- **CLAUDE.md**: Check the project root for detailed development documentation

---

Ready to get started? Jump to the [Getting Started Guide](getting-started.md) to install the Ambient Code Platform and create your first AgenticSession!
