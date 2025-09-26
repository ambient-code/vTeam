# Global Claude Configuration

This file contains global Claude Code configuration that applies to all projects.
This should be symlinked to ~/.claude/CLAUDE.md

## Global Operating Principles

### Git and Version Control
- **MANDATORY BRANCH VERIFICATION**: ALWAYS check current git branch with `git branch --show-current` as the FIRST action before ANY task that could modify files
- When using git, ALWAYS work in feature branches unless told explicitly otherwise
- **Always squash commits** for clean history
- **Make sure to commit frequently** with succinct commit messages that are immediately useful to the reader

### Development Standards
- **Always use python virtual environments** to avoid affecting system python packages
- **ALWAYS use uv instead of pip** where possible
- **ALWAYS run markdownlint locally** on any markdown files that you work with
- **ALWAYS automatically resolve any issues reported by linters**
- **ALWAYS try to minimize rework**

### GitHub Best Practices
- **When setting up GitHub projects, ALWAYS use repository-level projects**. NEVER use user-level projects
- **NEVER change visibility of a github repository** without explicitly being told to do so
- When working with GitHub repositories, always follow GitHub Flow
- **ALWAYS setup dependabot automation** when creating a new github repository
- **Warn if in a GitHub git repo and GitHub Actions integration is not installed** for Claude

### Code Quality
- **NEVER push if linters report errors or warnings**
- **NEVER push if tests fail** 
- **ALWAYS fix issues immediately after running linters**
- **ALWAYS make sure all dates that you use match reality**
- When creating new python applications, you only need to support versions N and N-1

### File Management
- **ALWAYS keep your utility/working scripts in git** and well-isolated from the primary codebase
- **NOTHING may ever depend on these scripts**
- **NEVER make changes to files unless you are on the correct feature branch** for those changes
- **NEVER create files unless they're absolutely necessary** for achieving your goal
- **ALWAYS prefer editing an existing file to creating a new one**
- **NEVER proactively create documentation files** (*.md) or README files unless explicitly requested

### Linting Workflow
**MANDATORY: ALWAYS run the complete linting workflow locally before ANY git push or commit**

Check the project's CLAUDE.md file for language-specific linting commands and workflows.

### Testing Best Practices
- **ALWAYS run tests immediately after making implementation changes**
- When fixing API bugs, update both implementation AND corresponding tests in the same commit
- Never assume tests will pass after changing HTTP methods, endpoints, or response formats
- Implementation changes without test updates = guaranteed CI failures