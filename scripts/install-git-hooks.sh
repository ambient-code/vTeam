#!/usr/bin/env bash
#
# Install git hooks for branch protection
#
# This script creates symlinks from .git/hooks/ to scripts/git-hooks/
# for automated branch protection enforcement.

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Determine repository root
REPO_ROOT="$(git rev-parse --show-toplevel 2>/dev/null)"
if [ -z "$REPO_ROOT" ]; then
    echo -e "${RED}❌ Not in a git repository${NC}"
    exit 1
fi

HOOKS_SOURCE_DIR="$REPO_ROOT/scripts/git-hooks"
HOOKS_TARGET_DIR="$REPO_ROOT/.git/hooks"

# Ensure source directory exists
if [ ! -d "$HOOKS_SOURCE_DIR" ]; then
    echo -e "${RED}❌ Hooks source directory not found: $HOOKS_SOURCE_DIR${NC}"
    exit 1
fi

# Ensure target directory exists
mkdir -p "$HOOKS_TARGET_DIR"

# List of hooks to install
HOOKS=("pre-commit" "pre-push")

echo "🔧 Installing git hooks for branch protection..."
echo ""

installed_count=0
skipped_count=0

for hook in "${HOOKS[@]}"; do
    source_path="$HOOKS_SOURCE_DIR/$hook"
    target_path="$HOOKS_TARGET_DIR/$hook"

    # Check if source exists
    if [ ! -f "$source_path" ]; then
        echo -e "${YELLOW}⚠️  Source hook not found: $source_path (skipping)${NC}"
        continue
    fi

    # Check if target already exists
    if [ -e "$target_path" ]; then
        # Check if it's already a symlink to our hook
        if [ -L "$target_path" ] && [ "$(readlink "$target_path")" = "$source_path" ]; then
            echo -e "${GREEN}✓${NC} $hook already installed (symlink exists)"
            skipped_count=$((skipped_count + 1))
            continue
        else
            echo -e "${YELLOW}⚠️  $hook already exists (backing up to ${hook}.backup)${NC}"
            mv "$target_path" "${target_path}.backup"
        fi
    fi

    # Create symlink
    ln -s "$source_path" "$target_path"
    echo -e "${GREEN}✓${NC} Installed $hook"
    installed_count=$((installed_count + 1))
done

echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
if [ $installed_count -gt 0 ]; then
    echo -e "${GREEN}✅ Successfully installed $installed_count hook(s)${NC}"
fi
if [ $skipped_count -gt 0 ]; then
    echo -e "${GREEN}✓${NC}  $skipped_count hook(s) already installed"
fi
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""
echo "📋 Installed hooks:"
echo "   • pre-commit  - Prevents commits to main/master/production"
echo "   • pre-push    - Prevents pushes to main/master/production"
echo ""
echo "💡 To override a hook when needed:"
echo "   git commit --no-verify"
echo "   git push --no-verify"
echo ""
echo "🗑️  To uninstall hooks:"
echo "   make remove-hooks"
echo ""
