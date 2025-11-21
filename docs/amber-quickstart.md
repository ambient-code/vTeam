# Amber Issue-to-PR Automation - Quickstart

Get Amber handling GitHub issues in 5 minutes.

## Prerequisites

- GitHub repository with Actions enabled
- Anthropic API key

## Setup (One-Time)

### 1. Add Anthropic API Key to GitHub Secrets

```bash
# Via GitHub CLI
gh secret set ANTHROPIC_API_KEY

# Or via GitHub UI:
# Settings → Secrets and variables → Actions → New repository secret
# Name: ANTHROPIC_API_KEY
# Value: sk-ant-...
```

### 2. Enable GitHub Actions Permissions

**Settings → Actions → General → Workflow permissions**:
- ✅ Read and write permissions
- ✅ Allow GitHub Actions to create and approve pull requests

### 3. Verify Workflow Exists

```bash
ls .github/workflows/amber-issue-handler.yml
# Should exist - if not, check installation
```

## Usage

### Method 1: Create Issue with Template (Recommended)

1. **Go to Issues → New Issue**
2. **Select Template**:
   - 🤖 Amber Auto-Fix Request
   - 🔧 Amber Refactoring Request
   - 🧪 Amber Test Coverage Request
3. **Fill Out Form**
4. **Submit Issue**

Amber will automatically trigger and create a PR.

---

### Method 2: Add Label to Existing Issue

1. **Create or open an issue**
2. **Add one of these labels**:
   - `amber:auto-fix`
   - `amber:refactor`
   - `amber:test-coverage`
3. **Amber triggers immediately**

---

### Method 3: Comment Trigger

On any existing issue, comment:

```
/amber execute
```

Amber will execute the proposal described in the issue body.

---

## Example: Fix Linting Errors

### Step 1: Create Issue

**Title**: `[Amber] Fix Go formatting in backend`

**Label**: `amber:auto-fix`

**Body**:
```markdown
## Problem
Backend handlers fail gofmt checks.

## Files
File: `components/backend/handlers/*.go`

## Fix Type
Code Formatting
```

### Step 2: Wait for Amber

- GitHub Actions workflow starts (~30 seconds)
- Amber analyzes code and applies fixes (~1-2 minutes)
- PR is created automatically

### Step 3: Review PR

```bash
# View the PR
gh pr list --label amber-generated

# Review changes
gh pr view 123

# Merge if good
gh pr merge 123 --squash
```

**Done!** Linting errors fixed automatically.

---

## Example: Refactor Large File

### Step 1: Create Issue

**Title**: `[Amber Refactor] Break sessions.go into modules`

**Label**: `amber:refactor`

**Body**:
```markdown
## Current State
File: `components/backend/handlers/sessions.go`
Issue: 3,495 lines, violates Constitution Principle V

## Desired State
Break into modules:
- sessions/lifecycle.go (create, delete)
- sessions/status.go (status updates)
- sessions/jobs.go (job management)
- sessions/validation.go (input validation)

## Constraints
- Maintain backward compatibility
- All existing tests must pass
- Follow CLAUDE.md standards

## Priority
P0 - Critical
```

### Step 2: Amber Creates Detailed PR

- Analyzes current structure
- Creates modular file structure
- Updates imports across codebase
- Ensures all tests pass
- Creates PR with before/after comparison

### Step 3: Review & Merge

```bash
# Review the refactoring
gh pr view 124 --web

# Check CI status
gh pr checks 124

# Merge when ready
gh pr merge 124 --squash
```

---

## Example: Add Missing Tests

### Step 1: Create Issue

**Title**: `[Amber Tests] Add contract tests for sessions API`

**Label**: `amber:test-coverage`

**Body**:
```markdown
## Untested Code
File: `components/backend/handlers/sessions.go`
Functions:
- CreateSession (line 123)
- UpdateSessionStatus (line 456)

## Test Type
Contract Tests (API endpoints)

## Test Scenarios
- Happy path: create session with valid spec
- Error: missing API key
- Error: invalid namespace
- Edge case: very long prompt (>10K chars)

## Target Coverage
60%
```

### Step 2: Amber Generates Tests

- Creates `handlers/sessions_test.go`
- Writes table-driven tests (Go convention)
- Covers happy paths and error cases
- Ensures all tests pass

### Step 3: Verify Coverage

```bash
# Check test coverage
cd components/backend
go test -coverprofile=coverage.out ./handlers
go tool cover -func=coverage.out | grep sessions.go

# Should show ~60% coverage
```

---

## Monitoring Amber

### View All Amber PRs

```bash
gh pr list --label amber-generated
```

### View Workflow Runs

```bash
gh run list --workflow=amber-issue-handler.yml
```

### Check Specific Run

```bash
# Get run ID from issue comment or Actions tab
gh run view 123456789 --log
```

---

## Troubleshooting

### Amber Didn't Trigger

**Check**:
1. Label is exact: `amber:auto-fix` (not `amber auto-fix`)
2. Secret `ANTHROPIC_API_KEY` is set
3. Workflow permissions enabled (Settings → Actions)

**Debug**:
```bash
gh run list --workflow=amber-issue-handler.yml --limit 5
```

### PR Tests Failed

**Action**: Review PR, add feedback comment, Amber can update PR

### Amber Commented "Error"

**Action**: Read error message, update issue with more context, re-trigger

---

## Configuration

### Adjust Amber's Behavior

Edit `.claude/amber-config.yml`:

```yaml
# Change risk thresholds
automation_policies:
  auto_fix:
    max_files_per_pr: 10  # Change to 20 for larger auto-fixes

# Add new patterns
  categories:
    - name: "Your Custom Pattern"
      patterns: ["custom pattern"]
```

### Disable Amber Temporarily

```bash
# Disable workflow
gh workflow disable amber-issue-handler.yml

# Re-enable
gh workflow enable amber-issue-handler.yml
```

---

## Best Practices

### ✅ Do

- Use specific file paths when possible
- Provide clear success criteria
- Start with small, low-risk tasks (auto-fix)
- Review all PRs before merging

### ❌ Don't

- Request breaking changes (Amber will reject)
- Provide vague instructions ("make it better")
- Skip PR review (always review before merge)
- Hardcode secrets (Amber will refuse)

---

## Next Steps

1. **Try auto-fix first**: Create issue with `amber:auto-fix` label
2. **Review generated PR**: Understand Amber's approach
3. **Graduate to refactoring**: Try `amber:refactor` for tech debt
4. **Add tests**: Use `amber:test-coverage` to improve coverage
5. **Monitor metrics**: Track PR merge rate, time-to-merge

---

## Full Documentation

- [Complete Guide](amber-automation.md) - Detailed documentation
- [Amber Config](.claude/amber-config.yml) - Automation policies
- [Project Standards](../CLAUDE.md) - Conventions Amber follows

---

## Support

**Questions?** Create issue with label `amber:help`

**Feature Requests?** Title issue `[Amber Feature Request] ...`

**Bugs?** Title issue `[Amber Bug] ...` with workflow run link

---

**You're ready to use Amber!** Create your first issue and watch the automation magic happen. 🤖
