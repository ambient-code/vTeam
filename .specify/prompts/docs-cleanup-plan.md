# vTeam Documentation Cleanup & Refresh

**Objective**: Streamline documentation to focus on critical user paths, eliminate incomplete placeholder content, fix technical inaccuracies, and standardize all repository URLs to the canonical upstream source.

**Strategy**: Strategic simplification over comprehensive coverage. Remove what doesn't work, fix what remains, ensure every published page provides value.

---

## Executive Summary

The current documentation suffers from **architectural misalignment** (v1 LlamaDeploy references in a v2 Kubernetes-native system) and **58% placeholder content**. This plan implements a focused cleanup that removes incomplete sections entirely while ensuring remaining documentation is technically accurate and actionable.

**Scope**: Remove developer guide, integrations section, and 8 incomplete labs. Fix critical inaccuracies in remaining user guide, reference docs, and Lab 1. Standardize all repository URLs.

---

## Phase 1: Structural Simplification

### Directory Removals

The following directories will be completely removed as they contain only placeholder stub files with no actionable content:

| Directory | File Count | Reason for Removal |
|-----------|------------|-------------------|
| `docs/developer-guide/` | 8 files | All stubs marked "Documentation Under Development" |
| `docs/labs/advanced/` | 3 files | Incomplete labs (4, 5, 6) with no content |
| `docs/labs/production/` | 3 files | Incomplete labs (7, 8, 9) with no content |
| `docs/labs/solutions/` | 3 files | Solution guides for non-existent labs |

### Individual File Deletions

These user guide and reference files are placeholders that will be removed:

**User Guide Stubs** (5 files):
```
docs/user-guide/creating-rfes.md
docs/user-guide/agent-framework.md
docs/user-guide/rfe-workflow.md
docs/user-guide/configuration.md
docs/user-guide/troubleshooting.md
```

**Reference Stubs** (3 files):
```
docs/reference/agent-personas.md
docs/reference/api-endpoints.md
docs/reference/configuration-schema.md
```

**Labs to Remove** (2 basic labs):
```
docs/labs/basic/lab-2-agent-interaction.md
docs/labs/basic/lab-3-workflow-basics.md
```

**Impact**: This reduces the documentation from 38 markdown files to 17 files, eliminating all placeholder content and focusing on what actually exists and works.

---

## Phase 2: Navigation Restructuring

### Current Navigation Structure
```yaml
nav:
  - Home: index.md
  - User Guide: [6 pages, 5 are stubs]
  - Developer Guide: [8 pages, 7 are stubs]
  - Labs: [9 labs, 8 incomplete]
  - Reference: [5 pages, 3 are stubs]
  - Integrations: [section exists]
```

### Target Navigation Structure
```yaml
nav:
  - Home: index.md
  - User Guide:
      - Overview: user-guide/index.md
      - Getting Started: user-guide/getting-started.md
  - Labs:
      - Overview: labs/index.md
      - Lab 1 - Your First Agentic Session: labs/basic/lab-1-first-rfe.md
  - Reference:
      - Overview: reference/index.md
      - Glossary: reference/glossary.md
  - Deployment Guides:
      - GitHub App Setup: GITHUB_APP_SETUP.md
      - OpenShift Deployment: OPENSHIFT_DEPLOY.md
      - OpenShift OAuth: OPENSHIFT_OAUTH.md
      - Claude Code Runner: CLAUDE_CODE_RUNNER.md
```

**Rationale**: Four focused sections instead of six sprawling ones. Every linked page has complete, tested content.

---

## Phase 3: Critical Technical Corrections

### Issue 1: Repository URL Standardization

**Current State**: Documentation references multiple repository URLs inconsistently (personal forks, old references, varied formats).

**Target State**: All references point to the canonical upstream repository.

**Pattern to Find**: `jeremyeder/vTeam`, `git@github.com:*/vTeam`, any non-canonical URLs

**Replacement**: `https://github.com/ambient-code/vTeam.git`

**Files Requiring URL Updates**: (scan all .md files)

### Issue 2: Architectural Terminology Corrections

The documentation currently describes a v1 architecture that doesn't exist in the v2 codebase. The following table shows what needs to change:

| Incorrect Reference | Actual Implementation | Files Affected |
|--------------------|-----------------------|----------------|
| "LlamaDeploy workflow orchestration" | Kubernetes operator with Custom Resources | user-guide/index.md, reference/index.md, glossary.md |
| "@llamaindex/server TypeScript framework" | Next.js with Shadcn UI and React Query | glossary.md, reference/index.md |
| "7-agent RFE council process" | Generic AgenticSession with Claude Code runner | labs/basic/lab-1-first-rfe.md, labs/index.md |
| API port 8000 | Actual backend port 8080 | reference/index.md |
| `/deployments/rhoai/tasks/create` | `/api/projects/:project/agentic-sessions` | reference/index.md |

### Issue 3: RFEWorkflow De-emphasis

**Current Problem**: Documentation prominently features RFEWorkflow and a 7-agent council process, but this isn't the primary user workflow.

**Solution**: Minimize or remove RFEWorkflow references. Lab 1 should demonstrate creating a basic AgenticSession, not an RFE workflow. The 7-agent council diagram in reference/index.md should be removed or clearly marked as a specialized feature.

**Affected Files**:
- `docs/labs/basic/lab-1-first-rfe.md` - Rewrite entirely to focus on AgenticSession
- `docs/labs/index.md` - Remove 7-agent council learning objectives
- `docs/reference/index.md` - Remove workflow state diagram, simplify to AgenticSession focus

### Issue 4: ProjectSettings Configuration Example

**Current (Incorrect)**:
```yaml
spec:
  apiKeys:
    anthropic: "sk-ant-api03-your-key-here"
  defaultModel: "claude-3-5-sonnet-20241022"
  timeout: 300
```

**Target (Correct)**:
```yaml
apiVersion: vteam.ambient-code/v1alpha1
kind: ProjectSettings
metadata:
  name: projectsettings
  namespace: vteam-dev
spec:
  groupAccess:
    - groupName: "developers"
      role: "edit"
  runnerSecretsName: "runner-secrets"
---
apiVersion: v1
kind: Secret
metadata:
  name: runner-secrets
  namespace: vteam-dev
type: Opaque
stringData:
  ANTHROPIC_API_KEY: "sk-ant-api03-your-key-here"
```

**File**: `docs/user-guide/getting-started.md` (lines 71-82)

---

## Phase 4: Content Rewrites

### Lab 1 Complete Rewrite

**Current Focus**: Creating an RFE workflow with 7-agent council process

**New Focus**: Creating a basic AgenticSession to analyze code or generate content

**New Title**: "Lab 1 - Your First Agentic Session"

**New Structure**:

```markdown
# Lab 1 - Your First Agentic Session

## What You'll Learn
Create and monitor an AgenticSession that uses Claude Code to perform
automated code analysis or content generation.

## Prerequisites
- vTeam installed and running (see Getting Started)
- ANTHROPIC_API_KEY configured in project settings
- A GitHub repository you want to analyze (or use a sample)

## Concepts Introduced
- AgenticSession Custom Resource
- Interactive vs headless execution modes
- Single-repo and multi-repo configurations
- Session status monitoring

## Procedure

### Step 1: Access the vTeam Web Interface
[detailed steps]

### Step 2: Create a New Project
[detailed steps]

### Step 3: Configure Your First Session
[detailed steps with actual AgenticSession spec fields]

### Step 4: Monitor Session Execution
[detailed steps showing status phases]

### Step 5: Review Results
[detailed steps showing output interpretation]

## Validation
- [ ] Session enters "Running" phase within 60 seconds
- [ ] Job pod is created in your project namespace
- [ ] Session completes with "Succeeded" status
- [ ] Results are visible in session details

## Troubleshooting
[common issues specific to AgenticSessions]

## Next Steps
Experiment with interactive mode and multi-repo configurations
(see Getting Started guide for advanced examples)
```

### Index Page Updates

**docs/index.md** - Remove developer guide references, add clear path to getting started:

```markdown
# vTeam Documentation

Welcome to vTeam, a Kubernetes-native AI automation platform that
orchestrates intelligent agentic sessions through containerized microservices.

## Quick Navigation

**New Users**: Start with the [Getting Started Guide](user-guide/getting-started.md)
to deploy vTeam locally and create your first agentic session.

**Hands-On Learning**: Work through [Lab 1](labs/basic/lab-1-first-rfe.md) to
understand core concepts through practical exercises.

**Deployment to Production**: Review our [OpenShift Deployment Guide](OPENSHIFT_DEPLOY.md)
for production deployment patterns.

## Architecture Overview

vTeam implements a Kubernetes operator pattern with Custom Resources...
[rest of architecture description, removing LlamaDeploy references]
```

**docs/user-guide/index.md** - Simplify to only reference existing content:

```markdown
# User Guide

This guide helps you deploy, configure, and use vTeam for AI-powered automation.

## Getting Started

The [Getting Started Guide](getting-started.md) walks you through:
- Local deployment using OpenShift Local (CRC)
- API key configuration
- Creating your first agentic session
- Understanding session states and results

## Concepts

### Projects and Namespaces
Each vTeam project maps to a Kubernetes namespace, providing multi-tenant
isolation and access control.

### AgenticSessions
An AgenticSession is a Custom Resource that represents an AI execution task...

### Interactive vs Headless Mode
Sessions can run in headless mode (single prompt execution) or interactive
mode (ongoing chat conversation)...

## Next Steps

Complete [Lab 1](../labs/basic/lab-1-first-rfe.md) for hands-on practice
with agentic sessions.
```

**docs/labs/index.md** - Focus on single lab:

```markdown
# Hands-On Labs

## Lab 1: Your First Agentic Session

This foundational lab introduces you to vTeam's core workflow by creating
and monitoring an AgenticSession. You'll learn how to configure sessions,
monitor execution, and interpret results.

**Time**: 30-45 minutes
**Level**: Beginner
**Prerequisites**: Completed Getting Started guide

[Link to Lab 1](basic/lab-1-first-rfe.md)

## Lab Format

Each lab follows this structure:
- **Learning Objectives**: What you'll accomplish
- **Prerequisites**: Required setup and knowledge
- **Procedure**: Step-by-step instructions with validation checkpoints
- **Troubleshooting**: Common issues and solutions
- **Validation**: Success criteria checklist

## Learning Path

After completing Lab 1, explore advanced AgenticSession features by
experimenting with multi-repo configurations and interactive mode
(see Getting Started guide for examples).
```

**docs/reference/index.md** - Simplify to remove RFE workflow, fix API endpoints:

```markdown
# Reference Documentation

## Custom Resources

### AgenticSession

The primary Custom Resource for AI-powered automation tasks.

**API Version**: `vteam.ambient-code/v1alpha1`
**Kind**: `AgenticSession`

Key spec fields:
- `prompt`: The task description for the AI agent
- `repos`: Array of repository configurations (input/output)
- `interactive`: Boolean for chat mode vs headless execution
- `timeout`: Maximum execution time in seconds
- `model`: Claude model to use (e.g., "claude-3-5-sonnet-20241022")

### ProjectSettings

Namespace-scoped configuration for vTeam projects.

**API Version**: `vteam.ambient-code/v1alpha1`
**Kind**: `ProjectSettings`

Key spec fields:
- `groupAccess`: Array of group permissions
- `runnerSecretsName`: Reference to Secret containing API keys

## REST API Endpoints

The vTeam backend API runs on port 8080 (development) or standard HTTPS in production.

### Projects
| Method | Endpoint | Purpose |
|--------|----------|---------|
| GET | `/api/projects` | List all accessible projects |
| POST | `/api/projects` | Create new project |
| GET | `/api/projects/:project` | Get project details |

### Agentic Sessions
| Method | Endpoint | Purpose |
|--------|----------|---------|
| GET | `/api/projects/:project/agentic-sessions` | List sessions in project |
| POST | `/api/projects/:project/agentic-sessions` | Create new session |
| GET | `/api/projects/:project/agentic-sessions/:name` | Get session details |
| DELETE | `/api/projects/:project/agentic-sessions/:name` | Delete session |

### Health & Status
| Method | Endpoint | Purpose |
|--------|----------|---------|
| GET | `/health` | Backend health check |

## WebSocket API

Real-time session updates available via WebSocket connection to backend.

[Additional reference content without workflow diagrams]
```

---

## Phase 5: Grammar and Style Standardization

### Terminology Reference Table

Apply these standards consistently across all documentation:

| Term | Capitalization | Context | Example Usage |
|------|----------------|---------|---------------|
| AgenticSession | PascalCase | Custom Resource type | "Create an AgenticSession to analyze code" |
| ProjectSettings | PascalCase | Custom Resource type | "Configure ProjectSettings for your namespace" |
| Project | Capitalized | vTeam concept | "Each Project maps to a Kubernetes namespace" |
| namespace | lowercase | Kubernetes resource | "Sessions run in the project's namespace" |
| web interface | lowercase | General reference | "Access the web interface at the URL shown" |
| Web Interface | Title Case | Headings only | "## Using the Web Interface" |
| interactive mode | lowercase | Session configuration | "Enable interactive mode for chat-based sessions" |
| headless mode | lowercase | Session configuration | "Headless mode executes a single prompt" |

### Style Guidelines

**Verb Tense**: Use present tense for procedures.
- ✅ "The operator creates a Job"
- ❌ "The operator will create a Job"

**Articles**: Include articles for clarity.
- ✅ "Try the hands-on exercises"
- ❌ "Try hands-on exercises"

**Consistency**: Use the same phrasing for repeated concepts.
- First reference: "Request for Enhancement (RFE)"
- Subsequent: "RFE"

---

## Execution Checklist

### Pre-Execution Validation
- [ ] Confirm current branch is `feature/docs-cleanup-v2`
- [ ] Verify working directory is clean
- [ ] Back up current docs state if needed

### Phase 1: Deletions
- [ ] Delete `docs/developer-guide/` directory
- [ ] Delete 5 user guide stub files
- [ ] Delete 3 reference stub files
- [ ] Delete 8 incomplete lab files
- [ ] Delete `docs/labs/solutions/` directory
- [ ] Remove empty directories (advanced/, production/)

### Phase 2: Navigation
- [ ] Update `mkdocs.yml` to remove Developer Guide section
- [ ] Update `mkdocs.yml` to remove Integrations section
- [ ] Update `mkdocs.yml` Labs section to show only Lab 1
- [ ] Simplify User Guide navigation to 2 pages

### Phase 3: URL Standardization
- [ ] Find/replace repository URLs in all .md files
- [ ] Verify replacement with grep: `grep -r "jeremyeder" docs/`
- [ ] Confirm all URLs point to `https://github.com/ambient-code/vTeam.git`

### Phase 4: Technical Corrections
- [ ] Remove all "LlamaDeploy" references (3+ files)
- [ ] Remove all "@llamaindex/server" references (2+ files)
- [ ] Fix API endpoints in `reference/index.md` (port 8000→8080, correct paths)
- [ ] Fix ProjectSettings YAML in `getting-started.md`
- [ ] Update technology stack descriptions

### Phase 5: Content Rewrites
- [ ] Rewrite `docs/labs/basic/lab-1-first-rfe.md` for AgenticSession
- [ ] Update `docs/index.md` (remove dev guide refs, fix architecture)
- [ ] Update `docs/user-guide/index.md` (remove dead links, simplify)
- [ ] Update `docs/labs/index.md` (focus on Lab 1 only)
- [ ] Update `docs/reference/index.md` (remove workflow, fix endpoints)
- [ ] Update `docs/reference/glossary.md` (fix terminology)

### Phase 6: Final Validation
- [ ] Run `mkdocs build --strict` to check for errors
- [ ] Fix any broken internal links
- [ ] Run `markdownlint docs/**/*.md`
- [ ] Fix any linting errors
- [ ] Test that GitHub Pages workflow still works
- [ ] Review changes with `git diff`

### Phase 7: Commit
- [ ] Stage all changes: `git add -A`
- [ ] Commit with message: "docs: streamline documentation, remove placeholders, fix technical inaccuracies"
- [ ] Push to origin: `git push origin feature/docs-cleanup-v2`

---

## Success Criteria

### Quantitative Metrics
- **Documentation files**: Reduced from 38 to ~17 (55% reduction)
- **Placeholder content**: 0% (down from 58%)
- **Build errors**: 0 (strict mode passes)
- **Broken links**: 0 (all internal links valid)

### Qualitative Criteria
- Every published page provides actionable, accurate information
- No references to unimplemented features (LlamaDeploy, @llamaindex)
- All code examples use correct YAML schemas and API endpoints
- Repository URLs consistently point to upstream canonical source
- Navigation structure reflects actual content availability

### User Impact
- Users can successfully follow Getting Started guide without encountering broken links
- Lab 1 demonstrates actual system capabilities (AgenticSession, not RFE workflow)
- Reference documentation matches actual API implementation
- No confusion about v1 vs v2 architecture

---

## Post-Execution Next Steps

After this cleanup is complete and merged:

**Short-term enhancements** (if desired in future):
- Add architecture diagrams to docs/index.md
- Add screenshots to getting-started.md
- Document multi-repo configuration patterns
- Add more AgenticSession examples

**Long-term expansions** (separate efforts):
- Consider re-adding developer guide with actual contributor content
- Create additional labs for advanced features (only when content is ready)
- Add API reference documentation (only when endpoints are stable)
- Document RFEWorkflow properly if/when that workflow is production-ready

---

## File Manifest

### Files to Delete (23 total)
```
docs/developer-guide/index.md
docs/developer-guide/setup.md
docs/developer-guide/architecture.md
docs/developer-guide/plugin-development.md
docs/developer-guide/api-reference.md
docs/developer-guide/contributing.md
docs/developer-guide/testing.md
docs/developer-guide/migration.md
docs/user-guide/creating-rfes.md
docs/user-guide/agent-framework.md
docs/user-guide/rfe-workflow.md
docs/user-guide/configuration.md
docs/user-guide/troubleshooting.md
docs/reference/agent-personas.md
docs/reference/api-endpoints.md
docs/reference/configuration-schema.md
docs/labs/basic/lab-2-agent-interaction.md
docs/labs/basic/lab-3-workflow-basics.md
docs/labs/advanced/lab-4-custom-agents.md
docs/labs/advanced/lab-5-workflow-modification.md
docs/labs/advanced/lab-6-integration-testing.md
docs/labs/production/lab-7-jira-integration.md
docs/labs/production/lab-8-openshift-deployment.md
docs/labs/production/lab-9-scaling-optimization.md
docs/labs/solutions/solutions-basic.md
docs/labs/solutions/solutions-advanced.md
docs/labs/solutions/solutions-production.md
```

### Files to Modify (15 total)
```
mkdocs.yml                              - Navigation restructure
docs/index.md                           - Remove dev guide, fix arch
docs/user-guide/index.md                - Remove dead links
docs/user-guide/getting-started.md      - Fix YAML, URLs
docs/labs/index.md                      - Focus on Lab 1
docs/labs/basic/lab-1-first-rfe.md      - Complete rewrite
docs/reference/index.md                 - Remove workflow, fix API
docs/reference/glossary.md              - Fix terminology
docs/CLAUDE_CODE_RUNNER.md             - Fix URLs
docs/GITHUB_APP_SETUP.md               - Fix URLs
docs/OPENSHIFT_DEPLOY.md               - Fix URLs
docs/OPENSHIFT_OAUTH.md                - Fix URLs
docs/README.md                          - Fix URLs
README.md                               - Fix URLs (root)
components/frontend/README.md           - Fix URLs
```

### Directories to Remove
```
docs/developer-guide/
docs/labs/advanced/
docs/labs/production/
docs/labs/solutions/
```

---

## Estimated Effort

**Total Time**: 4-6 hours

**Breakdown**:
- Deletions and navigation updates: 30 minutes
- URL find/replace across all files: 30 minutes
- Technical corrections (LlamaDeploy, API endpoints, YAML): 1 hour
- Lab 1 complete rewrite: 2 hours
- Index page updates: 1 hour
- Final validation and testing: 1 hour

**Complexity**: Medium (mostly deletions and find/replace, one significant rewrite)

---

*This plan serves as a comprehensive prompt for executing the documentation cleanup. Follow the execution checklist sequentially, validating each phase before proceeding to the next.*