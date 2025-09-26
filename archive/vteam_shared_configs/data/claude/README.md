# Claude Configuration Management

This directory contains Claude Code configuration files for managing global and project-specific settings.

## Structure

```
claude/
├── README.md                 # This file
├── global-CLAUDE.md         # Global configuration (symlink to ~/.claude/CLAUDE.md)
└── project-templates/       # Templates for common project types
    ├── python-CLAUDE.md
    ├── javascript-CLAUDE.md
    └── shell-CLAUDE.md
```

## Setup Instructions

### Global Configuration
```bash
# Create symlink for global Claude configuration
ln -sf ~/repos/dotfiles/claude/global-CLAUDE.md ~/.claude/CLAUDE.md
```

### Project-Specific Configuration
For new projects, copy the appropriate template:
```bash
# For Python projects
cp ~/repos/dotfiles/claude/project-templates/python-CLAUDE.md /path/to/project/CLAUDE.md

# For JavaScript projects  
cp ~/repos/dotfiles/claude/project-templates/javascript-CLAUDE.md /path/to/project/CLAUDE.md

# For shell projects
cp ~/repos/dotfiles/claude/project-templates/shell-CLAUDE.md /path/to/project/CLAUDE.md
```

## Best Practices

1. **Global Configuration**: Use `~/.claude/CLAUDE.md` for settings that apply to ALL projects
2. **Project Configuration**: Use `PROJECT_ROOT/CLAUDE.md` for project-specific commands and context
3. **Version Control**: Keep both global and project configurations in git
4. **Symlinks**: Use symlinks to maintain a single source of truth for global config
5. **Templates**: Use project templates to ensure consistency across similar projects

## Configuration Hierarchy

Claude Code follows this configuration hierarchy (highest to lowest priority):
1. Project-specific `CLAUDE.md` (in project root)
2. Global `~/.claude/CLAUDE.md` 
3. Built-in Claude Code defaults

This allows you to:
- Set organization-wide standards in global config
- Override with project-specific requirements
- Maintain consistency across all your projects