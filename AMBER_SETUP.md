# Amber Issue-to-PR Automation - Setup Complete ✅

This document summarizes the Amber automation system installed in this repository.

## What Was Installed

### 1. GitHub Actions Workflow
**File**: `.github/workflows/amber-issue-handler.yml`

**What It Does**:
- Monitors GitHub issues for specific labels (`amber:auto-fix`, `amber:refactor`, `amber:test-coverage`)
- Extracts instructions and file paths from issue body
- Executes Amber agent via Claude Code SDK
- Creates pull request with automated fixes
- Links PR back to original issue

**Security Features**:
- Environment variable injection prevention (no command injection vulnerabilities)
- Minimal permissions (contents:write, issues:write, pull-requests:write)
- Token redaction in logs
- Follows GitHub security best practices

---

### 2. Issue Templates
**Location**: `.github/ISSUE_TEMPLATE/`

**Templates**:
1. **amber-auto-fix.yml** - Low-risk fixes (formatting, linting)
2. **amber-refactor.yml** - Medium-risk changes (breaking files, extracting patterns)
3. **amber-test-coverage.yml** - Test additions (unit/contract/integration)
4. **config.yml** - Template configuration with documentation links

**Features**:
- Structured forms with validation
- Pre-filled labels for automatic triggering
- Clear instructions and examples
- Risk acknowledgment checkboxes

---

### 3. Automation Configuration
**File**: `.claude/amber-config.yml`

**Defines**:
- Risk-based automation policies (low/medium/high)
- Auto-fix categories (formatting, linting, docs)
- Proposal workflow for medium-risk changes
- Report-only mode for high-risk changes
- Constitution compliance checks
- Monitoring schedules (daily/weekly/monthly)

**Key Settings**:
```yaml
max_auto_prs_per_day: 5           # Safety guardrail
require_tests_pass: true           # Quality check
never_push_to_main: true           # Branch protection
always_create_branch: true         # Safe workflow
```

---

### 4. Documentation
**Files**:
1. **docs/amber-automation.md** - Complete automation guide (4,000+ words)
   - How it works
   - Available workflows
   - Configuration
   - Security
   - Troubleshooting
   - Best practices
   - FAQ

2. **docs/amber-quickstart.md** - Get started in 5 minutes
   - Prerequisites
   - Setup steps
   - Three usage examples
   - Monitoring commands

3. **.github/workflows/README.md** - Workflows documentation
   - All active workflows
   - Permissions
   - Security considerations
   - Troubleshooting

4. **AGENTS.md** - Updated with Amber section
   - Quick links to documentation
   - Common workflows summary

---

## Setup Required (One-Time)

### 1. Add Anthropic API Key

```bash
# Via GitHub CLI
gh secret set ANTHROPIC_API_KEY
# Paste your key: sk-ant-...

# Or via GitHub UI:
# Settings → Secrets and variables → Actions → New repository secret
```

### 2. Enable GitHub Actions Permissions

**Settings → Actions → General → Workflow permissions**:
- ✅ Read and write permissions
- ✅ Allow GitHub Actions to create and approve pull requests

### 3. Verify Installation

```bash
# Check workflow exists
gh workflow view amber-issue-handler.yml

# Should show: "amber-issue-handler.yml (active)"
```

---

## Quick Test

### Create Test Issue

1. Go to **Issues → New Issue**
2. Select **🤖 Amber Auto-Fix Request**
3. Fill in:
   ```
   Title: [Amber] Test automation
   Description: Test Amber workflow with a trivial fix
   Files: README.md
   Fix Type: Code Formatting
   ```
4. Submit

### Expected Outcome

1. **~30 seconds**: GitHub Actions workflow starts
2. **~2 minutes**: Amber analyzes and creates branch
3. **~2-3 minutes**: PR is created and linked to issue

### Verify

```bash
# View the PR
gh pr list --label amber-generated

# View workflow run
gh run list --workflow=amber-issue-handler.yml --limit 1
```

---

## Usage Examples

### Example 1: Fix Linting

**Issue**:
```yaml
Title: [Amber] Fix Go formatting in backend
Label: amber:auto-fix
Files: components/backend/**/*.go
```

**Outcome**: PR with `gofmt -w .` applied

---

### Example 2: Refactor Large File

**Issue**:
```yaml
Title: [Amber Refactor] Break sessions.go into modules
Label: amber:refactor
Current State: handlers/sessions.go (3,495 lines)
Desired State: Break into lifecycle.go, status.go, jobs.go
```

**Outcome**: PR with modular file structure, all tests passing

---

### Example 3: Add Tests

**Issue**:
```yaml
Title: [Amber Tests] Add contract tests for Projects API
Label: amber:test-coverage
Untested Code: handlers/projects.go (CreateProject, DeleteProject)
Target Coverage: 60%
```

**Outcome**: PR with table-driven tests (Go convention)

---

## Monitoring

### View All Amber Activity

```bash
# All Amber PRs
gh pr list --label amber-generated

# Recent workflow runs
gh run list --workflow=amber-issue-handler.yml

# Success rate
gh pr list --label amber-generated --state merged | wc -l
gh pr list --label amber-generated --state closed | wc -l
```

### Metrics to Track

- **PR Merge Rate**: Target 90%+ (high-quality automation)
- **Time to Merge**: Faster than manual fixes
- **Issue Resolution Time**: Should decrease over time
- **Developer Satisfaction**: Gather feedback on usefulness

---

## Configuration Tuning

### Increase Automation Aggressiveness

Edit `.claude/amber-config.yml`:

```yaml
automation_policies:
  auto_fix:
    max_files_per_pr: 20  # Default: 10
    auto_merge: true       # Default: false (requires approval)
```

### Add New Fix Categories

```yaml
categories:
  - name: "Import Optimization"
    patterns:
      - "unused imports in TypeScript"
      - "import sorting violations"
    commands:
      typescript: ["npm run lint:fix"]
```

### Change Monitoring Schedule

```yaml
monitoring:
  schedule:
    daily:
      - "check_recent_commits"
      - "run_security_scan"  # Add custom check
```

---

## Troubleshooting

### Workflow Not Triggering

**Symptoms**: Issue labeled but no workflow run

**Debug**:
```bash
# Check workflow is enabled
gh workflow view amber-issue-handler.yml

# Check permissions
# Settings → Actions → General → Workflow permissions
# Should be "Read and write permissions"

# Check secret exists
gh secret list | grep ANTHROPIC_API_KEY
```

---

### Amber Commented "Error"

**Symptoms**: Workflow ran but commented error on issue

**Actions**:
1. Click workflow run link in error comment
2. Review logs: `gh run view <run-id> --log`
3. Common issues:
   - Missing file paths in issue
   - Vague instructions
   - Changes too complex for automation

**Fix**: Update issue with more context, re-label to re-trigger

---

### PR Tests Failing

**Symptoms**: Amber created PR but CI fails

**Actions**:
1. Review PR diff manually
2. Amber should have run linters before committing
3. CI may have additional checks Amber doesn't know about
4. Add failing check to `.claude/amber-config.yml`

---

## Next Steps

### 1. Test with Low-Risk Issue (5 min)

Create simple auto-fix issue to verify workflow

### 2. Review Generated PR (10 min)

Understand Amber's approach and code quality

### 3. Add Custom Patterns (30 min)

Extend `.claude/amber-config.yml` with project-specific patterns

### 4. Train Team (1 hour)

Share quickstart guide, demonstrate creating issues

### 5. Monitor & Iterate (ongoing)

Track metrics, gather feedback, tune configuration

---

## Architecture Notes

### Why GitHub Actions?

- **Self-Service**: Any team member can trigger Amber via issues
- **Audit Trail**: All changes tracked via issues → PRs → commits
- **Access Control**: GitHub's built-in permissions system
- **No Infrastructure**: Runs on GitHub's hosted runners
- **Integration**: Native GitHub API access

### Why Issue-Driven?

- **Async Workflow**: Create issue, come back to PR later
- **Discussion**: Team can discuss approach in issue before Amber executes
- **Approval**: `/amber execute` comment requires explicit trigger
- **Linking**: PRs automatically close issues, creating traceability

### Why Labels?

- **Declarative**: Label = intent, no complex commands
- **Filterable**: Easy to find all `amber:*` issues
- **Automatable**: GitHub Actions triggers on label events
- **User-Friendly**: Point-and-click vs. CLI commands

---

## Advanced Usage

### Custom Amber Agents

Extend workflow to support custom agents:

```yaml
# In amber-issue-handler.yml
if: github.event.label.name == 'amber:custom-agent'
```

Then specify agent in issue body:

```markdown
Agent: Steve (UX Designer)
Task: Create mockup for new feature page
```

### Integration with Other Tools

Amber can integrate with:
- **Jira**: Comment on linked Jira ticket when PR created
- **Slack**: Notify channel of Amber activity
- **PagerDuty**: Create incident if Amber fails repeatedly
- **Datadog**: Send metrics on Amber performance

Add integrations in workflow's final steps.

---

## Security Considerations

### Secrets Management

- ✅ `ANTHROPIC_API_KEY` stored as GitHub secret (encrypted at rest)
- ✅ Never logged or exposed in workflow runs
- ✅ Only accessible to workflow, not forks
- ✅ Rotation: Update secret, no code changes required

### Command Injection Prevention

- ✅ All user input passed via environment variables
- ✅ No direct interpolation of `${{ github.event.* }}`
- ✅ Reviewed by security hook in Claude Code
- ✅ Follows GitHub's official security guidelines

### Branch Protection

- ✅ Amber never pushes to `main` (creates feature branches)
- ✅ All changes go through PR review
- ✅ CI must pass before merge
- ✅ Human approval required (no auto-merge by default)

---

## Cost Estimate

### Anthropic API Usage

**Per Workflow Run**:
- Auto-fix: ~10K tokens = $0.03
- Refactoring: ~50K tokens = $0.15
- Test coverage: ~30K tokens = $0.09

**Monthly (example: 50 runs)**:
- 30 auto-fixes × $0.03 = $0.90
- 15 refactorings × $0.15 = $2.25
- 5 test additions × $0.09 = $0.45
- **Total: ~$3.60/month**

**ROI**: If each Amber run saves 30 minutes of developer time at $50/hour = $25 saved per run. 50 runs = $1,250 value for $3.60 cost.

---

## Support

### Documentation

- [Complete Guide](docs/amber-automation.md)
- [Quickstart](docs/amber-quickstart.md)
- [Workflows README](.github/workflows/README.md)

### Getting Help

**Issues with Amber**:
1. Create issue with label `amber:help`
2. Include workflow run link
3. Describe expected vs. actual behavior

**Feature Requests**:
- Title: `[Amber Feature Request] ...`
- Describe desired capability
- Include use case examples

**Bugs**:
- Title: `[Amber Bug] ...`
- Include workflow run link
- Steps to reproduce

---

## Success Metrics

Track these to measure Amber's effectiveness:

### Quantitative

- **PR Merge Rate**: % of Amber PRs merged (target: 90%+)
- **Time to Merge**: Time from issue creation to PR merge
- **Developer Time Saved**: Estimated hours saved per week
- **Issue Resolution Rate**: % of issues successfully automated

### Qualitative

- **Developer Satisfaction**: Survey team on usefulness
- **Code Quality**: Review generated code quality
- **Adoption Rate**: % of eligible issues using Amber
- **Team Feedback**: Gather suggestions for improvement

---

## Roadmap

### Phase 1: Foundation (Current)

- ✅ Basic auto-fix workflow
- ✅ Issue templates
- ✅ Documentation

### Phase 2: Enhancement (Next)

- [ ] Auto-merge for trusted patterns
- [ ] Slack notifications
- [ ] Metrics dashboard
- [ ] Custom agent support

### Phase 3: Intelligence (Future)

- [ ] Amber learns from rejected PRs
- [ ] Proactive issue creation (Amber finds issues)
- [ ] Multi-agent collaboration
- [ ] Predictive maintenance

---

**Amber is ready to use!** 🤖

Create your first issue and experience automated development workflows.

**Quick Links**:
- [Quickstart](docs/amber-quickstart.md)
- [Full Documentation](docs/amber-automation.md)
- [Create Auto-Fix Issue](../../issues/new?template=amber-auto-fix.yml)
- [Create Refactoring Issue](../../issues/new?template=amber-refactor.yml)
- [Create Test Coverage Issue](../../issues/new?template=amber-test-coverage.yml)
