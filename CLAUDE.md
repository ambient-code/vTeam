# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

**vTeam** is a dual-purpose repository containing:

1. **RAT (Refinement Agent Team) System**: An AI-powered automation system to reduce engineering refinement time and improve ticket quality through intelligent Jira ticket preparation
2. **vTeam Shared Configs**: A Python package providing shared Claude Code configuration for team development standards

## Architecture

### RAT System (`demos/rfe-builder/`)
- **Technology Stack**: Streamlit web application with Python backend
- **AI Integration**: Anthropic Claude API with Vertex AI support for conversational RFE creation
- **Data Models**: Pydantic models for RFE workflow management (7-step council review process)
- **Agent System**: Multi-agent workflow with specialized roles (PM, Architect, Staff Engineer, PO, Team Lead, Team Member, Delivery Owner)

### vTeam Shared Configs (`src/vteam_shared_configs/`)
- **Purpose**: Manages Claude Code configuration across team projects
- **Installation**: CLI tool (`vteam-config`) for setup and maintenance
- **Configuration**: Global settings, project templates, and team hooks via symlinks

## Development Commands

### Python Environment Setup
```bash
# Use virtual environments for all Python development
python -m venv venv
source venv/bin/activate  # or `venv\Scripts\activate` on Windows

# Install dependencies (prefer uv over pip)
uv pip install -r requirements.txt
```

### RFE Builder Demo
```bash
cd demos/rfe-builder
streamlit run app.py
```

### Package Development (vTeam Shared Configs)
```bash
# Install development dependencies
uv pip install -e ".[dev]"

# Run linting workflow
black .
isort --profile black .
flake8 --max-line-length=88 --extend-ignore=E203,W503 .

# Run tests
python -m pytest

# Run mypy type checking
mypy --ignore-missing-imports src/
```

### Pre-commit Hooks
```bash
# Install pre-commit hooks (configured in demos/rfe-builder/.pre-commit-config.yaml)
pre-commit install
pre-commit run --all-files
```

## Key Architecture Patterns

### RAT System Workflow
1. **7-Step Council Process**: RFEs flow through specialized agent roles
2. **State Management**: WorkflowState class manages RFE lifecycle with Pydantic models
3. **Agent Roles**: Enum-based agent system with predefined responsibilities
4. **Conversational AI**: ChatInterface provides natural language RFE creation
5. **Cost Tracking**: Built-in usage monitoring for AI API calls

### Agent Role Mapping
- **Parker (PM)**: Prioritizes RFEs and communicates assessments
- **Archie (Architect)**: Reviews RFEs and checks acceptance criteria  
- **Stella (Staff Engineer)**: Performs completeness checks and accept/reject decisions
- **Olivia (PO)**: Manages acceptance criteria
- **Lee (Team Lead)**: Coordinates team execution
- **Taylor (Team Member)**: Handles implementation details
- **Derek (Delivery Owner)**: Creates feature tickets and manages delivery

### Configuration Management
- **Symlink-based**: Global configs linked to user's `.claude/` directory
- **Template System**: Project templates for different development scenarios
- **Team Hooks**: Shared Git hooks and development standards
- **Local Overrides**: `.claude/settings.local.json` for personal customization

## Testing Strategy
- **Unit Tests**: pytest framework with coverage reporting
- **Type Checking**: mypy with additional dependencies for Pydantic
- **Linting**: black, isort, flake8 with pre-commit automation
- **AI Integration Tests**: Mock Anthropic API calls for testing

## File Structure Significance

```
vTeam/
├── src/vteam_shared_configs/     # Installable package for team configuration
│   ├── cli.py                    # Command-line interface (vteam-config)
│   └── installer.py              # Configuration management logic
├── demos/rfe-builder/            # RAT system demonstration
│   ├── app.py                    # Main Streamlit application
│   ├── components/               # UI components and integrations
│   │   ├── chat_interface.py     # Conversational AI interface
│   │   └── jira_integration.py   # (Future) Jira API integration
│   └── data/                     # Data models and state management
│       └── rfe_models.py         # Pydantic models for RFE workflow
├── pyproject.toml               # Package configuration
└── rhoai-ux-agents-vTeam.md     # Complete agent framework documentation
```

## AI Integration Notes
- **Primary API**: Anthropic Claude with Vertex AI fallback
- **Model Selection**: Configurable via `get_model_name()` function
- **Cost Management**: Built-in token counting and caching system
- **Error Handling**: Graceful degradation when AI services unavailable

## Agent-Driven Development
This project implements an AI agent system modeled after real software teams with different seniority levels and responsibilities. The agent framework (detailed in `rhoai-ux-agents-vTeam.md`) provides realistic team dynamics for AI-assisted development workflows.

## Configuration Standards
- **Python Formatting**: black (88 char line length, double quotes)
- **Import Sorting**: isort with black profile
- **Linting**: flake8 with line length 88, ignoring E203,W503
- **Type Checking**: mypy with missing import tolerance
- **Git Workflow**: Feature branches, squashed commits, conventional commit messages