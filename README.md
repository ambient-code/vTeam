# vTeam: Refinement Agent Team System

> AI-powered automation system to reduce engineering refinement time and improve ticket quality

## Overview

The **vTeam** repository contains a dual-purpose AI automation platform:

1. **Refinement Agent Team System** - A Streamlit-based AI automation solution that dramatically reduces the time engineering teams spend in refinement meetings through a 7-step agent council workflow
2. **vTeam Shared Configs** - A Python package for managing shared Claude Code configuration across development teams

The Refinement Agent Team system transforms Request for Enhancement (RFE) submissions into well-refined, implementation-ready tickets through intelligent AI agent collaboration, enabling engineering teams to start work immediately with comprehensive context and clear acceptance criteria.

## Problem Statement

Engineering teams currently spend excessive time in refinement meetings due to:
- Poorly prepared tickets lacking necessary context
- Missing detailed feature breakdowns  
- Unclear acceptance criteria
- Disconnected information across RFEs, code repositories, and architectural documents

## Solution

The Refinement Agent Team system addresses these challenges through intelligent AI automation:
- **7-Agent Council Process** - Specialized AI agents (PM, Architect, Staff Engineer, PO, Team Lead, Team Member, Delivery Owner) handle different refinement aspects
- **Conversational RFE Creation** - Natural language interface powered by Anthropic Claude for intuitive ticket creation
- **Comprehensive Context Assembly** - Automatically enriches tickets with business justification, technical requirements, and success criteria
- **Workflow Orchestration** - Guided progression through standardized refinement steps
- **Integration Ready** - Built for seamless integration with existing Jira workflows

## Success Metrics

- 🎯 **90% ticket readiness** for immediate engineering execution
- ⏱️ **50% reduction** in refinement meeting duration
- 🚀 **25% improvement** in engineering velocity
- 📊 **Measurable time savings** in refinement hours per ticket

## System Architecture

### Technology Stack
- **Web Framework**: Streamlit for interactive UI
- **AI Integration**: Anthropic Claude API with Google Vertex AI support
- **Data Models**: Pydantic for type-safe RFE workflow management
- **Language**: Python 3.13+
- **Development Tools**: pre-commit hooks with black, isort, flake8, mypy

### Agent Council Workflow
The system implements a 7-step refinement process with specialized AI agents:

1. **Parker (PM)** - RFE Prioritization
2. **Archie (Architect)** - Technical Review
3. **Stella (Staff Engineer)** - Completeness Check
4. **Archie (Architect)** - Acceptance Criteria Validation
5. **Stella (Staff Engineer)** - Accept/Reject Decision
6. **Parker (PM)** - Assessment Communication
7. **Derek (Delivery Owner)** - Feature Ticket Creation

### Integration Points
- **Jira API** - Epic creation and synchronization (implemented)
- **Anthropic Claude** - Conversational AI and agent assistance
- **Google Vertex AI** - Alternative AI provider support
- **Git Repositories** - Future integration for code context

## Key Features

- **Conversational RFE Creation**: Natural language interface with real-time structured data extraction
- **Multi-Agent Workflow**: Specialized AI agents model realistic software team dynamics
- **Visual Workflow Tracking**: Progress visualization with step-by-step status updates
- **Cost Management**: Built-in API usage tracking and response caching
- **Jira Integration**: Automated Epic creation from refined RFEs
- **Agent Dashboard**: Role-specific views for different team members

## Getting Started

### Prerequisites
- Python 3.13+ (or Python 3.12+)
- Git
- Anthropic Claude API key (for AI features)

### Installation

#### 1. Clone the Repository
```bash
git clone https://github.com/jeremyeder/vTeam.git
cd vTeam
```

#### 2. Set Up Python Environment
```bash
# Create virtual environment (recommended)
python -m venv venv
source venv/bin/activate  # On Windows: venv\Scripts\activate

# Install dependencies using uv (preferred) or pip
cd demos/rfe-builder
uv pip install -r requirements.txt
# OR: pip install -r requirements.txt
```

#### 3. Configure AI API Access
Create a `.streamlit/secrets.toml` file in the `demos/rfe-builder/` directory:
```toml
[anthropic]
api_key = "your-anthropic-api-key-here"

[vertex]
project_id = "your-gcp-project-id"  # Optional, for Vertex AI
location = "us-central1"            # Optional, for Vertex AI

[jira]  # Optional, for Jira integration
base_url = "https://your-domain.atlassian.net"
username = "your-email@company.com"
api_token = "your-jira-api-token"
project_key = "PROJECT"
```

#### 4. Run the Application
```bash
cd demos/rfe-builder
streamlit run app.py
```

The application will open in your browser at `http://localhost:8501`.

### Development Setup

#### Install Development Tools
```bash
# Install pre-commit hooks
cd demos/rfe-builder
pre-commit install

# Run all pre-commit checks
pre-commit run --all-files
```

#### Linting and Testing
```bash
# Format code
black .
isort --profile black .

# Lint code
flake8 --max-line-length=88 --extend-ignore=E203,W503 .

# Type checking
mypy --ignore-missing-imports .

# Run tests
python -m pytest
```

## File Structure

```
vTeam/
├── demos/rfe-builder/              # Refinement Agent Team demo application
│   ├── app.py                      # Main Streamlit application
│   ├── components/                 # UI components and integrations
│   │   ├── chat_interface.py       # Conversational AI interface
│   │   ├── jira_integration.py     # Jira Epic creation
│   │   └── workflow.py             # Workflow management UI
│   ├── data/                       # Data models and state management
│   │   └── rfe_models.py           # Pydantic models for RFE workflow
│   ├── ai_models/                  # AI integration modules
│   │   ├── anthropic_client.py     # Claude API client
│   │   ├── cost_tracker.py         # Usage monitoring
│   │   └── prompt_manager.py       # Prompt templates
│   ├── prompts/                    # AI prompt templates
│   ├── tests/                      # Test suite
│   └── requirements.txt            # Python dependencies
├── src/vteam_shared_configs/       # Shared configuration package
│   ├── cli.py                      # Command-line interface
│   └── installer.py                # Configuration management
├── pyproject.toml                  # Package configuration
├── rhoai-ux-agents-vTeam.md        # Complete agent framework docs
└── CLAUDE.md                       # Claude Code guidance
```

## Usage

### Creating RFEs

1. **Conversational Creation** (💬 AI Chat RFE)
   - Describe your enhancement idea in natural language
   - AI assistant guides you through the process
   - Structured data extracted automatically

2. **Form-Based Creation** (📝 Create RFE)
   - Traditional form interface
   - Manual field completion
   - Immediate workflow assignment

### Agent Council Workflow

Once an RFE is created, it flows through the 7-step agent council:

1. **Parker (PM)** prioritizes the RFE for business value
2. **Archie (Architect)** reviews technical feasibility
3. **Stella (Staff Engineer)** validates completeness
4. **Archie (Architect)** ensures acceptance criteria are clear
5. **Stella (Staff Engineer)** makes final accept/reject decision
6. **Parker (PM)** communicates results to stakeholders
7. **Derek (Delivery Owner)** creates implementation tickets

### Agent Framework

The system uses a sophisticated multi-agent framework with different seniority levels and specializations. Each agent has:
- **Distinct personality** and communication style
- **Technical competency levels** matching real software teams
- **Domain expertise** in their area of responsibility
- **Realistic interaction patterns** with other agents

See `rhoai-ux-agents-vTeam.md` for complete agent specifications and interaction protocols.

## Shared Configuration

This repository includes shared Claude Code configuration for team development standards and workflows.

### vTeam Shared-Configs

Automated team configuration management via Python package:

- **🔄 Automatic enforcement** - Hooks ensure team standards on every Git operation
- **⚙️ Developer flexibility** - Personal overrides via `.claude/settings.local.json`
- **📊 Visual documentation** - Mermaid workflow diagrams show configuration hierarchy
- **🛠️ Project templates** - Python, JavaScript, Shell development templates

**Quick Setup:**
```bash
uv pip install -e .  # Install from source
vteam-config install  # Set up configuration
```

**Available Commands:**
```bash
vteam-config status      # Show current configuration
vteam-config update      # Update to latest version
vteam-config uninstall   # Remove configuration
```