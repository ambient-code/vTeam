# Git Hooks for Branch Protection

This directory contains git hooks that enforce the feature branch workflow to prevent accidental commits and pushes to protected branches.

## What's Included

### `pre-commit`

Runs before every `git commit` and blocks commits to protected branches:

- `main`
- `master`
- `production`

**Error message example:**

```
⚠️  Attempting to commit directly to 'main'
❌ Direct commits to 'main' are not allowed.

💡 Create a feature branch instead:
   git checkout -b feature/your-feature-name
   git checkout -b fix/bug-description
   git checkout -b issue-123-description

🔧 Or force commit (not recommended):
   git commit --no-verify -m 'your message'
```

### `pre-push`

Runs before every `git push` and blocks pushes to protected branches:

- Blocks if you're currently on a protected branch
- Blocks if you're pushing to a protected remote branch

**Error message example:**

```
⚠️  Attempting to push directly to 'main'
❌ Direct pushes to 'main' are not allowed.

💡 Use the pull request workflow instead:
   1. Create a feature branch: git checkout -b feature/your-feature
   2. Push your feature branch: git push origin feature/your-feature
   3. Open a pull request on GitHub

🔧 Or force push (not recommended):
   git push --no-verify
```

## Installation

### Automatic Installation (Recommended)

Hooks are automatically installed when you run:

```bash
make dev-start
```

### Manual Installation

If you need to install hooks manually:

```bash
./scripts/install-git-hooks.sh
```

Or via Makefile:

```bash
make setup-hooks
```

### Verify Installation

Check that symlinks exist in `.git/hooks/`:

```bash
ls -la .git/hooks/ | grep -E 'pre-commit|pre-push'
```

You should see symlinks pointing to `../../scripts/git-hooks/pre-commit` and `pre-push`.

## Usage

### Normal Workflow (Hooks Active)

```bash
# Create a feature branch
git checkout -b feature/my-feature

# Make changes and commit (hook allows this)
git add .
git commit -m "feat: add new feature"

# Push to feature branch (hook allows this)
git push origin feature/my-feature

# Try to commit to main (hook blocks this)
git checkout main
git commit -m "oops"  # ❌ Blocked by pre-commit hook
```

### Override When Necessary

In rare cases where you need to commit/push to a protected branch (e.g., hotfix, CI automation):

```bash
# Override pre-commit hook
git commit --no-verify -m "hotfix: critical security patch"

# Override pre-push hook
git push --no-verify origin main
```

**⚠️ Use `--no-verify` sparingly!** It bypasses all safety checks.

## Uninstallation

To remove the git hooks:

```bash
make remove-hooks
```

Or manually:

```bash
rm .git/hooks/pre-commit
rm .git/hooks/pre-push
```

Backed-up hooks (if any existed before installation) are saved as `.git/hooks/pre-commit.backup`.

## How It Works

### Symlinks

The installation script creates symlinks from `.git/hooks/` to `scripts/git-hooks/`:

```
.git/hooks/pre-commit  →  ../../scripts/git-hooks/pre-commit
.git/hooks/pre-push    →  ../../scripts/git-hooks/pre-push
```

**Benefits:**

- Changes to hook scripts automatically apply (no reinstall needed)
- Hooks are tracked in git (in `scripts/git-hooks/`)
- Easy to update and maintain

### Hook Execution Flow

**Pre-commit:**

1. Git runs `.git/hooks/pre-commit` before creating commit
2. Hook checks current branch with `git rev-parse --abbrev-ref HEAD`
3. If branch is protected → exit code 1 (block commit)
4. If branch is allowed → exit code 0 (proceed)

**Pre-push:**

1. Git runs `.git/hooks/pre-push` before pushing
2. Hook receives push targets via stdin (local ref, remote ref)
3. Checks both current branch and target branch
4. If either is protected → exit code 1 (block push)
5. If both are allowed → exit code 0 (proceed)

## Customization

### Add More Protected Branches

Edit the `PROTECTED_BRANCHES` list in both hook files:

```python
PROTECTED_BRANCHES: List[str] = ["main", "master", "production", "release"]
```

### Change Suggested Branch Prefixes

Edit the `SUGGESTED_PREFIXES` list in `pre-commit`:

```python
SUGGESTED_PREFIXES: List[str] = [
    "feature/",
    "fix/",
    "docs/",
    "refactor/",
    "test/",
    "issue-",
    "jira-",  # Add your custom prefixes
]
```

### Disable Hooks Temporarily

Instead of uninstalling, you can temporarily disable:

```bash
# Disable pre-commit
mv .git/hooks/pre-commit .git/hooks/pre-commit.disabled

# Re-enable
mv .git/hooks/pre-commit.disabled .git/hooks/pre-commit
```

Or use `--no-verify` for individual operations.

## Troubleshooting

### Hook Not Running

**Check if hook is executable:**

```bash
ls -l .git/hooks/pre-commit
# Should show: lrwxr-xr-x (symlink with execute permissions)
```

**Verify symlink target:**

```bash
readlink .git/hooks/pre-commit
# Should show: ../../scripts/git-hooks/pre-commit
```

**Reinstall hooks:**

```bash
make remove-hooks
make setup-hooks
```

### Python Not Found

Hooks require Python 3.11+ (project standard). Verify:

```bash
python3 --version
```

If Python is not in your PATH, update the shebang in hook files:

```python
#!/usr/bin/env python3.11  # Specific version
```

### Hook Fails with Error

Run the hook manually to see detailed errors:

```bash
python3 scripts/git-hooks/pre-commit
python3 scripts/git-hooks/pre-push < /dev/null
```

### Backed-Up Hook Lost

If the installer backed up your previous hook, restore it:

```bash
mv .git/hooks/pre-commit.backup .git/hooks/pre-commit
```

## Contributing

When modifying hooks:

1. Test thoroughly on both protected and non-protected branches
2. Ensure error messages are clear and actionable
3. Follow project Python standards (black, isort, type hints)
4. Update this README if behavior changes

## References

- [Git Hooks Documentation](https://git-scm.com/docs/githooks)
- [CONTRIBUTING.md](../../CONTRIBUTING.md) - Project contribution guide
- [Makefile](../../Makefile) - Development commands
