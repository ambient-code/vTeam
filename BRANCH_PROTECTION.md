# Branch Protection Configuration

This document explains the branch protection settings for the vTeam repository.

## Current Configuration

The `main` branch has minimal protection rules optimized for solo development:

- ✅ **Admin enforcement enabled** - Ensures consistency in protection rules
- ❌ **Required PR reviews disabled** - Allows self-merging of PRs
- ❌ **Status checks disabled** - No CI/CD requirements (can be added later)
- ❌ **Restrictions disabled** - No user/team restrictions on merging

## Rationale

This configuration is designed for **solo development** scenarios where:

1. **Jeremy is the primary/only developer** - Self-review doesn't add value
2. **Maintains Git history** - PRs are still encouraged for tracking changes
3. **Removes friction** - No waiting for external approvals
4. **Preserves flexibility** - Can easily revert when team grows

## Usage Patterns

### Recommended Workflow
1. Create feature branches for significant changes
2. Create PRs for change documentation and review history
3. Self-merge PRs when ready (no approval needed)
4. Use direct pushes only for hotfixes or minor updates

### When to Use PRs vs Direct Push
- **PRs**: New features, architecture changes, documentation updates
- **Direct Push**: Typo fixes, quick configuration changes, emergency hotfixes

## Future Considerations

When the team grows beyond solo development, consider re-enabling:

```bash
# Re-enable required reviews (example)
gh api --method PUT repos/red-hat-data-services/vTeam/branches/main/protection \
  --field required_status_checks=null \
  --field enforce_admins=true \
  --field required_pull_request_reviews='{"required_approving_review_count":1,"dismiss_stale_reviews":true,"require_code_owner_reviews":false}' \
  --field restrictions=null
```

## Commands Used

To disable branch protection (current state):
```bash
gh api --method PUT repos/red-hat-data-services/vTeam/branches/main/protection \
  --field required_status_checks=null \
  --field enforce_admins=true \
  --field required_pull_request_reviews=null \
  --field restrictions=null
```

To check current protection status:
```bash
gh api repos/red-hat-data-services/vTeam/branches/main/protection
```