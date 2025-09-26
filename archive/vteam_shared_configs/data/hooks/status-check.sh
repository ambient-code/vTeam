#!/bin/bash
set -euo pipefail

# vTeam Configuration Status Check Script
# Displays current vTeam configuration status on Claude session start

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SHARED_CONFIGS_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
CLAUDE_DIR="$HOME/.claude"

# Colors for output
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

info() {
    echo -e "${GREEN}[vTeam]${NC} $1"
}

warn() {
    echo -e "${YELLOW}[vTeam]${NC} $1"
}

status() {
    echo -e "${BLUE}[vTeam]${NC} $1"
}

# Only show status if we're in a vTeam repository
if [[ ! -d "$SHARED_CONFIGS_DIR" ]]; then
    exit 0  # Silent exit if not in vTeam repo
fi

# Check configuration status
GLOBAL_LINK="$CLAUDE_DIR/CLAUDE.md"
TEMPLATES_LINK="$CLAUDE_DIR/project-templates"

echo
status "=== vTeam Configuration Status ==="

# Check global config
if [[ -L "$GLOBAL_LINK" ]]; then
    TARGET=$(readlink "$GLOBAL_LINK")
    if [[ "$TARGET" == "$SHARED_CONFIGS_DIR/claude/global-CLAUDE.md" ]]; then
        info "✓ Global configuration: Active"
    else
        warn "⚠ Global configuration: Linked to different source"
    fi
else
    warn "⚠ Global configuration: Not linked (will be auto-configured on first Git operation)"
fi

# Check project templates
if [[ -L "$TEMPLATES_LINK" ]]; then
    TARGET=$(readlink "$TEMPLATES_LINK")
    if [[ "$TARGET" == "$SHARED_CONFIGS_DIR/claude/project-templates" ]]; then
        info "✓ Project templates: Active"
    else
        warn "⚠ Project templates: Linked to different source"
    fi
else
    warn "⚠ Project templates: Not linked (will be auto-configured on first Git operation)"
fi

# Check for local overrides
LOCAL_SETTINGS=".claude/settings.local.json"
if [[ -f "$LOCAL_SETTINGS" ]]; then
    info "✓ Local overrides: Present in $LOCAL_SETTINGS"
else
    status "ℹ Local overrides: None (create $LOCAL_SETTINGS for personal customizations)"
fi

echo
status "Team standards will be automatically enforced on Git operations"
status "Use '.claude/settings.local.json' for personal overrides"
echo