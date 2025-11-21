# GitHub Actions Workflows

This directory contains automated workflows for the Ambient Code Platform.

## Active Workflows

### 🤖 Amber Issue-to-PR Handler (`amber-issue-handler.yml`)

**Purpose**: Automatically processes GitHub issues and creates pull requests using the Amber background agent.

**Triggers**:
- Issue labeled with `amber:auto-fix`, `amber:refactor`, or `amber:test-coverage`
- Issue comment containing `/amber execute`

**What It Does**:
1. Parses issue for file paths, instructions, and context
2. Executes Amber agent with appropriate prompt
3. Creates feature branch with fixes/refactoring/tests
4. Opens pull request with changes
5. Links PR back to original issue

**Requirements**:
- `ANTHROPIC_API_KEY` secret configured
- Workflow permissions: read/write for contents, issues, PRs

**Documentation**: [Amber Automation Guide](../../docs/amber-automation.md)

---

### 🏗️ Components Build & Deploy (`components-build-deploy.yml`)

**Purpose**: Builds and deploys platform components on changes.

**Triggers**:
- Push to `main` branch
- Pull requests affecting component directories

**What It Does**:
1. Detects which components changed
2. Builds multi-platform Docker images (amd64, arm64)
3. Pushes to `quay.io/ambient_code` registry (main branch only)
4. Runs component-specific tests

**Change Detection**:
- Frontend: `components/frontend/**`
- Backend: `components/backend/**`
- Operator: `components/operator/**`
- Claude Runner: `components/runners/claude-code-runner/**`

---

### 🧪 E2E Tests (`e2e.yml`)

**Purpose**: Runs end-to-end tests in Kind (Kubernetes in Docker).

**Triggers**:
- Pull requests
- Manual workflow dispatch

**What It Does**:
1. Sets up Kind cluster
2. Deploys full vTeam stack
3. Runs Cypress tests against UI
4. Reports results

**Documentation**: [E2E Testing Guide](../../docs/testing/e2e-guide.md)

---

### 🔧 Test Local Dev (`test-local-dev.yml`)

**Purpose**: Validates local development setup works correctly.

**Triggers**:
- Changes to dev scripts (`Makefile`, `dev-start.sh`, etc.)
- Manual workflow dispatch

**What It Does**:
1. Simulates local development environment
2. Tests `make dev-start`, `make dev-stop`
3. Verifies CRC integration

---

### 🔄 Dependabot Auto-Merge (`dependabot-auto-merge.yml`)

**Purpose**: Automatically merges Dependabot dependency updates.

**Triggers**:
- Dependabot PR creation

**What It Does**:
1. Checks if PR is from Dependabot
2. Waits for CI to pass
3. Auto-merges patch/minor version updates
4. Requires manual review for major updates

---

### 🔄 Amber Dependency Sync (`amber-dependency-sync.yml`)

**Purpose**: Keeps Amber agent's dependency knowledge current.

**Triggers**:
- Daily at 7 AM UTC
- Manual workflow dispatch

**What It Does**:
1. Extracts dependency versions from go.mod, pyproject.toml, package.json
2. Updates `agents/amber.md` with current versions
3. Validates sync accuracy
4. Validates constitution compliance
5. Auto-commits changes

**Documentation**: [Amber Automation Guide](../../docs/amber-automation.md)

---

### 🤝 Claude Code Integration (`claude.yml`)

**Purpose**: Integrates Claude Code with GitHub workflows.

**Triggers**:
- Issue/PR comments with @claude mentions
- Issue/PR opened or assigned

**What It Does**:
- Enables Claude Code AI assistance in issues/PRs
- Provides AI-powered code review and suggestions
- Supports fork-compatible checkouts

---

### 🔍 Claude Code Review (`claude-code-review.yml`)

**Purpose**: Automated code reviews using Claude.

**Triggers**:
- Pull requests opened or synchronized

**What It Does**:
1. Checks out PR head (supports forks)
2. Minimizes old review comments
3. Runs comprehensive code review
4. Posts structured review (Blocker/Critical/Major/Minor issues)

**Requirements**:
- `CLAUDE_CODE_OAUTH_TOKEN` secret configured

---

### 🛠️ Go Lint (`go-lint.yml`)

**Purpose**: Go code quality enforcement.

**Triggers**:
- Push to main
- Pull requests affecting Go code

**What It Does**:
1. Detects changes to backend/operator Go code
2. Checks gofmt formatting
3. Runs go vet
4. Runs golangci-lint

---

### 🎨 Frontend Lint (`frontend-lint.yml`)

**Purpose**: Frontend code quality enforcement.

**Triggers**:
- Push to main
- Pull requests affecting TypeScript/JavaScript code

**What It Does**:
1. Detects changes to frontend code
2. Runs ESLint
3. TypeScript type checking
4. Build validation (`npm run build`)

---

### 🚀 Production Release Deploy (`prod-release-deploy.yaml`)

**Purpose**: Production releases with semver versioning.

**Triggers**:
- Manual workflow dispatch only

**What It Does**:
1. Calculates next version (major/minor/patch bump)
2. Generates changelog from git commits
3. Creates git tag and GitHub release
4. Builds all component images with release tag
5. Deploys to production OpenShift cluster

**Requirements**:
- `PROD_OPENSHIFT_SERVER` and `PROD_OPENSHIFT_TOKEN` secrets

---

### 📚 Documentation Deploy (`docs.yml`)

**Purpose**: Deploy MkDocs documentation to GitHub Pages.

**Triggers**:
- Push to main
- Manual workflow dispatch

**What It Does**:
1. Builds docs with MkDocs in UBI9 container
2. Deploys to GitHub Pages

---

## Workflow Permissions

All workflows follow **principle of least privilege**:

```yaml
permissions:
  contents: read      # Default for reading code
  issues: write       # Only for issue-handling workflows
  pull-requests: write # Only for PR-creating workflows
  packages: write     # Only for image publishing
```

## Security Considerations

### Secrets Required

| Secret | Used By | Purpose |
|--------|---------|---------|
| `ANTHROPIC_API_KEY` | amber-issue-handler.yml | Claude API access |
| `CLAUDE_CODE_OAUTH_TOKEN` | claude-code-review.yml | Claude Code action authentication |
| `QUAY_USERNAME`, `QUAY_PASSWORD` | components-build-deploy.yml, prod-release-deploy.yaml | Quay.io registry access |
| `REDHAT_USERNAME`, `REDHAT_PASSWORD` | components-build-deploy.yml, prod-release-deploy.yaml | Red Hat registry access |
| `OPENSHIFT_SERVER`, `OPENSHIFT_TOKEN` | components-build-deploy.yml | OpenShift cluster access (dev) |
| `PROD_OPENSHIFT_SERVER`, `PROD_OPENSHIFT_TOKEN` | prod-release-deploy.yaml | OpenShift cluster access (prod) |
| `GITHUB_TOKEN` | All workflows | GitHub API access (auto-provided) |

### Command Injection Prevention

All workflows use **environment variables** to pass user input (issue titles, bodies, comments) to prevent command injection attacks.

**Example (Safe)**:
```yaml
env:
  ISSUE_TITLE: ${{ github.event.issue.title }}
run: echo "$ISSUE_TITLE"
```

**Anti-Pattern (Unsafe)**:
```yaml
run: echo "${{ github.event.issue.title }}"  # ❌ Vulnerable to injection
```

**Reference**: [GitHub Actions Security Guide](https://github.blog/security/vulnerability-research/how-to-catch-github-actions-workflow-injections-before-attackers-do/)

---

## Monitoring

### View All Workflow Runs

```bash
gh run list
```

### View Specific Workflow

```bash
gh run list --workflow=amber-issue-handler.yml
```

### Watch Running Workflow

```bash
gh run watch
```

### View Logs

```bash
gh run view <run-id> --log
```

---

## Troubleshooting

### Workflow Not Triggering

**Check**:
1. Workflow file syntax: `gh workflow list`
2. Trigger conditions match event
3. Workflow permissions enabled (Settings → Actions)

**Debug**:
```bash
# View workflow status
gh workflow view amber-issue-handler.yml

# Check recent runs
gh run list --workflow=amber-issue-handler.yml --limit 5
```

### Workflow Failing

**Common Issues**:
1. Missing secret (check Settings → Secrets)
2. Insufficient permissions (check workflow `permissions:`)
3. Syntax error in YAML (use `yamllint`)

**Debug**:
```bash
# View failure logs
gh run view <run-id> --log-failed

# Re-run failed jobs
gh run rerun <run-id> --failed
```

---

## Adding New Workflows

### Checklist

- [ ] Define clear trigger conditions (`on:`)
- [ ] Set minimal permissions (`permissions:`)
- [ ] Use env vars for user input (prevent injection)
- [ ] Add documentation to this README
- [ ] Test in fork before merging
- [ ] Add workflow badge to main README (optional)

### Template

```yaml
name: Your Workflow Name

on:
  # Define triggers
  push:
    branches: [main]

permissions:
  contents: read  # Minimal permissions

jobs:
  your-job:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - name: Your step
        env:
          # Use env vars for user input
          INPUT: ${{ github.event.inputs.value }}
        run: |
          echo "$INPUT"
```

---

## Best Practices

### ✅ Do

- Use latest action versions (`actions/checkout@v4`)
- Set explicit permissions per workflow
- Pass user input via environment variables
- Cache dependencies (npm, pip, Go modules)
- Fail fast for critical errors

### ❌ Don't

- Use `permissions: write-all` (too broad)
- Interpolate user input directly in `run:` commands
- Hardcode secrets (use GitHub Secrets)
- Run workflows on every push (use path filters)
- Ignore security warnings from GitHub

---

## Related Documentation

- [Amber Automation Guide](../../docs/amber-automation.md)
- [E2E Testing Guide](../../docs/testing/e2e-guide.md)
- [GitHub Actions Docs](https://docs.github.com/en/actions)
- [Security Best Practices](https://docs.github.com/en/actions/security-for-github-actions)

---

**Questions?** Create an issue with label `workflow:help`
