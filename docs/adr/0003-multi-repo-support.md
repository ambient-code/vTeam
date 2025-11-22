# ADR-0003: Multi-Repository Support in AgenticSessions

**Status:** Accepted
**Date:** 2024-11-21
**Deciders:** Product Team, Engineering Team
**Technical Story:** User request for cross-repo analysis and modification

## Context and Problem Statement

Users needed to execute AI sessions that operate across multiple Git repositories simultaneously. For example:

- Analyze dependencies between frontend and backend repos
- Make coordinated changes across microservices
- Generate documentation that references multiple codebases

Original design: AgenticSession operated on a single repository.

How should we extend AgenticSessions to support multiple repositories while maintaining simplicity and clear semantics?

## Decision Drivers

- **User need:** Cross-repo analysis and modification workflows
- **Clarity:** Need clear semantics for which repo is "primary"
- **Workspace model:** Claude Code expects a single working directory
- **Git operations:** Push/PR creation needs per-repo configuration
- **Status tracking:** Need to track per-repo outcomes (pushed vs. abandoned)
- **Backward compatibility:** Don't break single-repo workflows

## Considered Options

1. **Multiple repos with mainRepoIndex (chosen)**
2. **Separate sessions per repo with orchestration layer**
3. **Multi-root workspace (multiple working directories)**
4. **Merge all repos into monorepo temporarily**

## Decision Outcome

Chosen option: "Multiple repos with mainRepoIndex", because:

1. **Claude Code compatibility:** Single working directory aligns with claude-code CLI
2. **Clear semantics:** mainRepoIndex explicitly specifies "primary" repo
3. **Flexibility:** Can reference other repos via relative paths
4. **Status tracking:** Per-repo pushed/abandoned status in CR
5. **Backward compatible:** Single-repo sessions just have one entry in repos array

### Consequences

**Positive:**

- Enables cross-repo workflows (analysis, coordinated changes)
- Per-repo push status provides clear outcome tracking
- mainRepoIndex makes "primary repository" explicit
- Backward compatible with single-repo sessions
- Supports different git configs per repo (fork vs. direct push)

**Negative:**

- Increased complexity in session CR structure
- Clone order matters (mainRepo must be cloned first to establish working directory)
- File paths between repos can be confusing for users
- Workspace cleanup more complex with multiple repos

**Risks:**

- Users might not understand which repo is "main"
- Large number of repos could cause workspace size issues
- Git credentials management across repos more complex

## Implementation Notes

**AgenticSession Spec Structure:**

```yaml
apiVersion: vteam.ambient-code/v1alpha1
kind: AgenticSession
metadata:
  name: multi-repo-session
spec:
  prompt: "Analyze API compatibility between frontend and backend"

  # repos is an array of repository configurations
  repos:
    - input:
        url: "https://github.com/org/frontend"
        branch: "main"
      output:
        type: "fork"
        targetBranch: "feature-update"
        createPullRequest: true

    - input:
        url: "https://github.com/org/backend"
        branch: "main"
      output:
        type: "direct"
        pushBranch: "feature-update"

  # mainRepoIndex specifies which repo is the working directory (0-indexed)
  mainRepoIndex: 0  # frontend is the main repo

  interactive: false
  timeout: 3600
```

**Status Structure:**

```yaml
status:
  phase: "Completed"
  startTime: "2024-11-21T10:00:00Z"
  completionTime: "2024-11-21T10:30:00Z"

  # Per-repo status tracking
  repoStatuses:
    - repoURL: "https://github.com/org/frontend"
      status: "pushed"
      message: "PR #123 created"

    - repoURL: "https://github.com/org/backend"
      status: "abandoned"
      message: "No changes made"
```

**Clone Implementation Pattern:**

```python
# components/runners/claude-code-runner/wrapper.py

def clone_repositories(repos, main_repo_index, workspace):
    """Clone repos in correct order: mainRepo first, others after."""

    # Clone main repo first to establish working directory
    main_repo = repos[main_repo_index]
    main_path = clone_repo(main_repo["input"]["url"], workspace)
    os.chdir(main_path)  # Set as working directory

    # Clone other repos relative to workspace
    for i, repo in enumerate(repos):
        if i == main_repo_index:
            continue
        clone_repo(repo["input"]["url"], workspace)

    return main_path
```

**Key Files:**
- `components/backend/types/session.go:RepoConfig` - Repo configuration types
- `components/backend/handlers/sessions.go:227` - Multi-repo validation
- `components/runners/claude-code-runner/wrapper.py:clone_repositories` - Clone logic
- `components/operator/internal/handlers/sessions.go:150` - Status tracking

**Patterns Established:**

- mainRepoIndex defaults to 0 if not specified
- repos array must have at least one entry
- Per-repo output configuration (fork vs. direct push)
- Per-repo status tracking (pushed, abandoned, error)

## Validation

**Testing Scenarios:**

- ✅ Single-repo session (backward compatibility)
- ✅ Two-repo session with mainRepoIndex=0
- ✅ Two-repo session with mainRepoIndex=1
- ✅ Cross-repo file analysis
- ✅ Per-repo push status correctly reported
- ✅ Clone failure in secondary repo doesn't block main repo

**User Feedback:**

- Positive: Enables new workflow patterns (monorepo analysis)
- Confusion: Initially unclear which repo is "main"
- Resolution: Added documentation and examples

## Links

- Related: ADR-0001 (Kubernetes-Native Architecture)
- Implementation PR: #XXX
- User documentation: `docs/user-guide/multi-repo-sessions.md`
