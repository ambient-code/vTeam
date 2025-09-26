#!/bin/bash
set -euo pipefail

# vTeam Shared Configuration Enforcement Script
# Ensures latest vTeam configuration is active before Git operations

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SHARED_CONFIGS_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
CLAUDE_DIR="$HOME/.claude"

# Colors for output
RED='\033[0;31m'
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

error() {
    echo -e "${RED}[vTeam]${NC} $1"
}

step() {
    echo -e "${BLUE}[vTeam]${NC} $1"
}

# Check if we're in a vTeam repository
if [[ ! -d "$SHARED_CONFIGS_DIR" ]]; then
    error "vTeam shared-configs directory not found"
    exit 1
fi

# Create .claude directory if it doesn't exist
if [[ ! -d "$CLAUDE_DIR" ]]; then
    step "Creating ~/.claude directory"
    mkdir -p "$CLAUDE_DIR"
fi

# Function to check if symlink is up to date
check_symlink() {
    local target="$1"
    local link="$2"
    
    if [[ ! -L "$link" ]]; then
        return 1  # Not a symlink
    fi
    
    local current_target
    current_target="$(readlink "$link")"
    
    if [[ "$current_target" != "$target" ]]; then
        return 1  # Points to wrong target
    fi
    
    return 0  # Symlink is correct
}

# Check and update global configuration
GLOBAL_TARGET="$SHARED_CONFIGS_DIR/claude/global-CLAUDE.md"
GLOBAL_LINK="$CLAUDE_DIR/CLAUDE.md"

if ! check_symlink "$GLOBAL_TARGET" "$GLOBAL_LINK"; then
    step "Updating global CLAUDE.md configuration"
    
    # Backup existing file if it's not a symlink
    if [[ -f "$GLOBAL_LINK" ]] && [[ ! -L "$GLOBAL_LINK" ]]; then
        BACKUP_NAME="CLAUDE.md.backup-$(date +%Y%m%d-%H%M%S)"
        warn "Backing up existing $GLOBAL_LINK to $BACKUP_NAME"
        mv "$GLOBAL_LINK" "$CLAUDE_DIR/$BACKUP_NAME"
    fi
    
    # Remove existing link if present
    [[ -L "$GLOBAL_LINK" ]] && rm "$GLOBAL_LINK"
    
    # Create new symlink
    ln -sf "$GLOBAL_TARGET" "$GLOBAL_LINK"
    info "✓ Global configuration linked"
fi

# Check and update project templates
TEMPLATES_TARGET="$SHARED_CONFIGS_DIR/claude/project-templates"
TEMPLATES_LINK="$CLAUDE_DIR/project-templates"

if ! check_symlink "$TEMPLATES_TARGET" "$TEMPLATES_LINK"; then
    step "Updating project templates"
    
    # Backup existing directory if it's not a symlink
    if [[ -d "$TEMPLATES_LINK" ]] && [[ ! -L "$TEMPLATES_LINK" ]]; then
        BACKUP_NAME="project-templates.backup-$(date +%Y%m%d-%H%M%S)"
        warn "Backing up existing $TEMPLATES_LINK to $BACKUP_NAME"
        mv "$TEMPLATES_LINK" "$CLAUDE_DIR/$BACKUP_NAME"
    fi
    
    # Remove existing link if present
    [[ -L "$TEMPLATES_LINK" ]] && rm "$TEMPLATES_LINK"
    
    # Create new symlink
    ln -sf "$TEMPLATES_TARGET" "$TEMPLATES_LINK"
    info "✓ Project templates linked"
fi

# Verify installation
if [[ -L "$GLOBAL_LINK" ]] && [[ -L "$TEMPLATES_LINK" ]]; then
    info "✓ vTeam configuration is active and up to date"
else
    error "Configuration verification failed"
    exit 1
fi