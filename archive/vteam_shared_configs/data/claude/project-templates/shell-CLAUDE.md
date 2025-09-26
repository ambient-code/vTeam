# CLAUDE.md - Shell Script Project

This file provides guidance to Claude Code (claude.ai/code) when working with this shell script project.

## Development Commands

### Code Quality
```bash
# Lint shell scripts
shellcheck *.sh

# Lint with specific exclusions (if needed)
shellcheck -e SC1090 -e SC1091 *.sh

# Format shell scripts (if using shfmt)
shfmt -w *.sh
```

### Testing
```bash
# Run tests (if using bats)
bats tests/

# Run specific test file
bats tests/test-example.bats

# Manual testing
bash script-name.sh --help
bash script-name.sh --test-mode
```

### Execution
```bash
# Make scripts executable
chmod +x *.sh

# Run script
./script-name.sh

# Source script (for functions/aliases)
source script-name.sh
```

## Project Architecture

<!-- Describe your shell scripts, their purposes, and how they interact -->

## Configuration

### Shell Compatibility
- Target: Bash 4.0+ (specify if using other shells)
- Shebang: `#!/usr/bin/env bash` or `#!/bin/bash`

### Code Style
- Indentation: 2 or 4 spaces (consistent)
- Variable naming: snake_case for local, UPPER_CASE for globals/exports
- Function naming: snake_case
- Error handling: Use `set -euo pipefail` for strict mode

### Testing Framework
- Test runner: bats-core (or specify alternative)
- Test location: tests/ directory
- Test files: *.bats format

## Shell Script Best Practices

### Error Handling
```bash
#!/usr/bin/env bash
set -euo pipefail  # Exit on error, undefined vars, pipe failures
```

### Function Structure
```bash
function_name() {
    local arg1="$1"
    local arg2="${2:-default_value}"
    
    # Function body
    echo "Processing: $arg1"
}
```

### Variable Quoting
```bash
# Always quote variables
echo "$variable"
cp "$source_file" "$destination_file"
```

## Pre-commit Requirements

Before any commit, ALWAYS run:
1. `shellcheck *.sh` (or with project-specific exclusions)
2. Manual testing of modified scripts
3. Verify executable permissions are set correctly

All ShellCheck warnings must be resolved or explicitly excluded with comments.