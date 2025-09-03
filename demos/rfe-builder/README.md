# RFE Builder - AI-Powered Workflow Platform

[![CI/CD Pipeline](https://github.com/jeremyeder/vTeam/actions/workflows/ci.yml/badge.svg)](https://github.com/jeremyeder/vTeam/actions/workflows/ci.yml)
[![Coverage](https://codecov.io/gh/jeremyeder/vTeam/branch/main/graph/badge.svg)](https://codecov.io/gh/jeremyeder/vTeam)
[![Python 3.10+](https://img.shields.io/badge/python-3.10+-blue.svg)](https://www.python.org/downloads/)

RFE Builder is an interactive Streamlit web application that guides Request for Enhancement (RFE) submissions through a structured 7-step council review process with AI-powered assistance and enterprise integration.

## 🚀 Features

### Phase 1 (Foundation)
- **📊 Visual Workflow Management**: Interactive Mermaid diagram showing the complete RFE Council process
- **👥 Multi-Agent System**: 7 specialized agent roles with dedicated dashboards
- **📈 Progress Tracking**: Real-time status updates and step-by-step progression
- **🔄 State Management**: Persistent RFE data with comprehensive history tracking
- **📋 Role-Based Interfaces**: Customized dashboards for each workflow participant
- **✅ Comprehensive Testing**: Full test coverage with CI/CD pipeline

### Phase 2 (Conversational AI) ✨ **NEW**
- **💬 AI-Powered RFE Creation**: Natural language RFE creation using Anthropic Claude
- **🤖 Intelligent Agent Assistants**: Role-specific AI guidance for all 7 workflow agents
- **🎯 Smart Recommendations**: Context-aware suggestions based on RFE content
- **💰 Cost Optimization**: Token usage tracking and intelligent prompt management
- **📊 Dynamic Prompting**: Workflow-aware AI assistance that adapts to current step

## 🏗️ Architecture

### Agent Roles

The RFE Builder implements a 7-agent workflow system:

- **📊 Parker (Product Manager)** - RFE prioritization and stakeholder communication
- **🏛️ Archie (Architect)** - Technical review and acceptance criteria validation
- **⭐ Stella (Staff Engineer)** - Completeness assessment and final approval
- **📋 Olivia (Product Owner)** - Information gathering and requirement clarification
- **👥 Lee (Team Lead)** - Impact assessment and resource planning
- **💻 Taylor (Team Member)** - Technical impact evaluation
- **🚀 Derek (Delivery Owner)** - Feature ticket creation and assignment

### Workflow Steps

1. **Prioritize RFE** (Parker) - Business impact and priority assessment
2. **Review RFE** (Archie) - Technical feasibility and architecture review
3. **Completeness Check** (Stella) - Requirements validation and gap analysis
4. **Acceptance Criteria** (Archie) - Final technical approval
5. **Accept/Reject Decision** (Stella) - Final disposition
6. **Communicate Assessment** (Parker) - Stakeholder notification
7. **Create Feature Ticket** (Derek) - Implementation planning and assignment

## 🔧 Installation

### Prerequisites

- Python 3.10 or higher
- pip package manager

### Setup

1. **Clone the repository**
   ```bash
   git clone https://github.com/jeremyeder/vTeam.git
   cd vTeam/demos/rfe-builder
   ```

2. **Create virtual environment**
   ```bash
   python -m venv venv
   source venv/bin/activate  # On Windows: venv\Scripts\activate
   ```

3. **Install dependencies**
   ```bash
   # Using uv (recommended for speed)
   uv pip install -r requirements.txt
   # OR: pip install -r requirements.txt
   ```

4. **Configure AI features (Phase 2 - Optional)**

   **Option A: Direct Anthropic API (Standard)**
   ```bash
   # Copy secrets template
   cp .streamlit/secrets.toml.template .streamlit/secrets.toml

   # Edit .streamlit/secrets.toml and add your Anthropic API key
   # Get your key from: https://console.anthropic.com/
   ```

   **Option B: Google Cloud Vertex AI (Enterprise)**
   ```bash
   # IMPORTANT: See VERTEX_AI_SETUP.md for complete setup instructions
   
   # Quick setup (requires Google Cloud project with Vertex AI enabled):
   export CLAUDE_CODE_USE_VERTEX=1
   export CLOUD_ML_REGION=us-east5  # Your preferred region
   export ANTHROPIC_VERTEX_PROJECT_ID=your-gcp-project-id
   export ANTHROPIC_MODEL='claude-sonnet-4@20250514'
   export ANTHROPIC_SMALL_FAST_MODEL='claude-3-5-haiku@20241022'

   # Authenticate with Google Cloud
   gcloud auth application-default login

   # Verify your setup
   python check_vertex_setup.py
   ```
   
   Note: When running in Vertex AI mode, an Anthropic API key is not required. Leave `ANTHROPIC_API_KEY` unset (recommended). If you keep a `.streamlit/secrets.toml`, you may remove the key or set `ANTHROPIC_API_KEY = "using-vertex-ai"` to explicitly indicate Vertex usage.
   
   📋 **For detailed Vertex AI setup:** See [`VERTEX_AI_SETUP.md`](./VERTEX_AI_SETUP.md)

5. **Run the application**
   ```bash
   streamlit run app.py
   ```

6. **Open in browser**
   - The app will automatically open at `http://localhost:8501`

## 📖 Usage

### Creating an RFE

#### Option 1: AI-Powered Conversational Creation (Phase 2) ⭐ **Recommended**
1. Navigate to **"💬 AI Chat RFE"** in the sidebar
2. Describe your enhancement idea in natural language
3. The AI assistant will guide you through gathering all necessary information
4. Review the generated RFE draft and create when ready

#### Option 2: Traditional Form-Based Creation
1. Navigate to **"📝 Create RFE"** in the sidebar
2. Fill in the required fields:
   - **Title**: Brief descriptive title
   - **Description**: Detailed enhancement description
   - **Business Justification** (optional): Business value explanation
   - **Technical Requirements** (optional): Technical constraints
   - **Success Criteria** (optional): Implementation success metrics
3. Click **"Create RFE"** to submit

### Managing Workflow

1. **View Progress**: Use **"📊 Workflow Overview"** to see visual workflow status
2. **Agent Actions**: Access **"👥 Agent Dashboard"** to perform role-specific tasks
3. **Track RFEs**: Monitor all submissions in **"📈 RFE List"** with filtering options

### Agent-Specific Actions

Each agent role has specific capabilities with AI-powered assistance:

- **Parker (PM)**:
  - Prioritize new RFEs with AI business impact analysis
  - Draft stakeholder communications with AI assistance
- **Archie (Architect)**:
  - Review technical feasibility with AI architectural guidance
  - Validate acceptance criteria with AI recommendations
- **Stella (Staff Engineer)**:
  - Assess RFE completeness with AI gap analysis
  - Make final accept/reject decisions with AI decision support
- **Derek (Delivery Owner)**:
  - Generate JIRA tickets and development tasks with AI assistance
- **Other Agents**: Specialized assessment functions with role-specific AI guidance

## 🔧 Troubleshooting

### AI Configuration Issues

#### Vertex AI Connection Problems
If you see "I'm having trouble connecting to the AI service" with Vertex AI:

1. **Verify environment variables are set:**
   ```bash
   echo $CLAUDE_CODE_USE_VERTEX
   echo $ANTHROPIC_VERTEX_PROJECT_ID
   echo $CLOUD_ML_REGION
   ```

2. **Test your configuration:**
   ```bash
   # Run the setup verification script
   python check_vertex_setup.py
   ```

3. **Common issues:**
   - Missing Google Cloud authentication: Run `gcloud auth application-default login`
   - Wrong project ID: Verify your GCP project has Vertex AI API enabled
   - Missing dependencies: Ensure `anthropic[vertex]` and `google-cloud-aiplatform` are installed
   - Incorrect region: Use a region that supports Claude models (e.g., `us-east5`, `us-central1`)
   - Unsupported model: Verify your model is supported (see supported models below)
   - Connection timeouts: Increase `ANTHROPIC_TIMEOUT` for slow networks
   - Intermittent failures: The system will retry up to `ANTHROPIC_MAX_RETRIES` times automatically

#### Supported Vertex AI Models
- `claude-3-5-sonnet@20241022` (recommended)
- `claude-3-5-haiku@20241022` (fast, cost-effective)
- `claude-sonnet-4@20250514` (latest, most capable)
- `claude-3-sonnet@20240229`
- `claude-3-haiku@20240307`

#### Direct API Issues
If using direct Anthropic API:
- Verify your API key in `.streamlit/secrets.toml`
- Check your account has sufficient credits at https://console.anthropic.com/

If you are using Vertex AI mode (`CLAUDE_CODE_USE_VERTEX=1`), `ANTHROPIC_API_KEY` is optional and can be omitted.

## 🧪 Testing

### Run Tests

```bash
# Run all tests
pytest

# Run with coverage
pytest --cov=data --cov=components --cov-report=html

# Run specific test categories
pytest tests/test_rfe_models.py -v
pytest tests/test_workflow.py -v
```

### Test Coverage

- **RFE Models**: Data structure validation, workflow state management
- **Workflow Components**: Step progression, agent assignment, status tracking
- **Integration**: Complete workflow simulation, boundary conditions

## 🔄 CI/CD Pipeline

The project includes comprehensive GitHub Actions workflows:

- **Continuous Integration**: Automated testing across Python 3.10 and 3.11
- **Code Quality**: Linting with flake8, type checking with mypy
- **Security Scanning**: Safety and bandit security analysis
- **Streamlit Validation**: App syntax and import testing
- **Build Artifacts**: Deployment package creation

## 📁 Project Structure

```
rfe-builder/
├── app.py                      # Main Streamlit application
├── data/
│   ├── __init__.py
│   └── rfe_models.py          # Data models and state management
├── components/
│   ├── __init__.py
│   ├── workflow.py            # Workflow visualization components
│   ├── chat_interface.py      # AI-powered conversational interface
│   └── ai_assistants.py       # Agent-specific AI assistants
├── ai_models/
│   ├── __init__.py
│   ├── cost_tracker.py        # AI usage cost tracking
│   └── prompt_manager.py      # AI prompt management
├── prompts/
│   ├── conversational_rfe_creation.yaml
│   └── agents/                # Agent-specific prompt templates
├── pages/
│   └── parker_pm.py           # Agent-specific page (example)
├── tests/
│   ├── __init__.py
│   ├── test_rfe_models.py     # Model tests
│   └── test_workflow.py       # Workflow tests
├── .streamlit/
│   ├── secrets.toml.template  # Configuration template
│   └── secrets.toml           # API credentials (not in git)
├── .github/
│   └── workflows/
│       └── ci.yml             # CI/CD pipeline
├── requirements.txt           # Dependencies
└── README.md                  # This file
```

## 🔮 Roadmap

### Phase 2: Conversational Interface (Planned)
- AI-powered chat interface for RFE creation
- Natural language processing for requirement extraction
- Context-aware form generation
- Smart suggestions and auto-completion

### Phase 3: Enterprise Integration (Planned)
- Jira/GitHub Issues integration
- Bi-directional data synchronization
- Webhook support for real-time updates
- API gateway for third-party integrations

### Phase 4: Advanced Intelligence (Planned)
- Predictive analytics for RFE success rates
- Automated impact assessment
- Historical data analysis and insights
- Custom reporting and dashboards

## 🤝 Contributing

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Make your changes
4. Add tests for new functionality
5. Ensure all tests pass (`pytest`)
6. Commit your changes (`git commit -m 'Add amazing feature'`)
7. Push to the branch (`git push origin feature/amazing-feature`)
8. Open a Pull Request

## 📄 License

This project is part of the vTeam repository and follows the same licensing terms.

## 🙋 Support

For questions or issues:
- Open an issue in the [vTeam repository](https://github.com/jeremyeder/vTeam/issues)
- Tag issues with `rfe-builder` for faster response

## 🎯 Phase 1 Success Criteria

- [x] ✅ Basic workflow functional with all agent roles
- [x] ✅ Visual workflow matches mermaid diagram
- [x] ✅ RFE state persistence across sessions
- [x] ✅ Comprehensive test coverage (>80%)
- [x] ✅ CI/CD pipeline with automated testing
- [x] ✅ Agent-specific interfaces functional
- [x] ✅ Step progression and status tracking working

---

**Built with ❤️ using Streamlit and Python**
