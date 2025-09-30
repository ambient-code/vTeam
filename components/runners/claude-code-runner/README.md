# Claude Code Runner

Python service that orchestrates AI-powered agentic sessions using Claude Code CLI with multi-agent collaboration.

## Key Features

- **Multi-Agent System**: 16 specialized AI personas (Engineering Manager, Staff Engineer, UX Researcher, etc.)
- **Spec-Kit Integration**: Supports `/specify`, `/plan`, `/tasks` commands for spec-driven development
- **Git Integration**: Clones repositories, manages authentication, creates branches
- **Execution Modes**: Both interactive chat and headless one-shot execution
- **Workspace Persistence**: Kubernetes-native with PVC storage

## Documentation

For complete architecture details, prompt engineering strategy, and usage examples, see:

**[📖 docs/CLAUDE_CODE_RUNNER.md](../../../docs/CLAUDE_CODE_RUNNER.md)**

## Quick Start

The runner is deployed as a Kubernetes Job by the vTeam operator when agentic sessions are created through the web interface.
See:

**[📖 docs/OPENSHIFT_DEPLOY.md](../../../docs/OPENSHIFT_DEPLOY.md)**


Environment variables:
- `PROMPT`: Initial user prompt for the session
- `INTERACTIVE`: Enable chat mode (`"true"` for interactive, `"false"` for headless)
- `CLAUDE_PERMISSION_MODE`: Claude Code permission mode (default: `"acceptEdits"`)
