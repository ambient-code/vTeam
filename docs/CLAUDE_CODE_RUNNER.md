# Claude Code Runner

This document explains how the Ambient Code Platform's Claude Code runner works and details all the prompts being added across the system.

## How the Claude Code Runner Works

### Core Architecture
The Claude Code runner (`components/runners/claude-code-runner/`) runs inside a Kubernetes Job created by the operator for each `AgenticSession`. It orchestrates AI-powered sessions by:

1. **Execution Environment**: Runs Claude Code CLI in a Kubernetes pod with workspace persistence
2. **Multi-Agent System**: Integrates with specialized AI agent personas (16 different roles)
3. **Spec-Kit Integration**: Supports spec-driven development with `/specify`, `/plan`, `/tasks` commands
4. **Git Integration**: Clones repositories, manages Git authentication (installation token or runner secret), creates branches
5. **Interactive vs Headless**: Supports both chat-based and one-shot execution modes

### Key Components

#### 1. **Runner Wrapper** (`wrapper.py`)
- **Session Management**: Manages session lifecycle, status updates, workspace sync
- **Claude Agent SDK Integration**: Invokes the Claude Agent SDK with configured tools and permissions
- **Mode Switching**: Handles both interactive chat and headless execution
- **Result Processing**: Captures and reports session results back to Kubernetes API

#### 2. **Agent System** (`agent_loader.py`)
- **Agent Personas**: Loads 16 specialized AI agents from YAML configurations
- **Dynamic Prompting**: Generates role-specific prompts for spec-kit workflows
- **Multi-Perspective Analysis**: Each agent provides domain-specific analysis

#### 3. **Spec-Kit Integration**
Handled via prompts and workflow tooling at a higher level; the runner focuses on session orchestration and SDK integration.

#### 4. **Git Integration** (in `wrapper.py`)
- **Authentication**: Uses short-lived GitHub tokens from the backend or project secrets
- **Repository Management**: Clones input repositories into the workspace (multi-repo supported)
- **Branch Operations**: Commits changes, pushes to output remotes, and optionally creates PRs

## All Prompts Being Added Across Components

### 1. **Core System Prompts** (main.py)

**Primary Claude Code System Prompt Enhancement:**
```python
append_system_prompt=self.prompt + "\n\nALWAYS consult sub agents to help with this task."
```

**Display Name Generation Prompt:**
```python
system_prompt = (
    "You are a helpful assistant that creates concise, descriptive names for tasks. "
    "Keep responses under 6 words and focus on the main action or objective."
)
user_prompt = (
    "Summarize this prompt into a short session display name.\n\n" + prompt
)
```

### 2. **Agent Persona System Prompts** (16 agent YAML files)

Each agent has a `systemMessage` that defines their personality and role:

**Engineering Manager (Emma):**
```yaml
systemMessage: |
  You are Emma, an Engineering Manager with expertise in team leadership and strategic planning.
  You focus on team wellbeing, sustainable delivery practices, and balancing technical excellence with business needs.
  You monitor team velocity, protect team focus, and facilitate clear communication across stakeholders.
```

**Staff Engineer (Stella):**
```yaml
systemMessage: |
  You are Stella, a Staff Engineer with expertise in technical leadership and implementation excellence.
  You bridge architectural vision to practical implementation, champion code quality, and mentor teams through complex technical challenges.
  You focus on hands-on technical leadership, performance optimization, and sustainable engineering practices.
```

**UX Researcher (Ryan):**
```yaml
systemMessage: |
  You are Ryan, a UX Researcher with expertise in user insights and evidence-based design.
  You challenge assumptions with data, plan research studies, and translate complex user insights into actionable design recommendations.
  You advocate for user voice and ensure design decisions are grounded in research and data.
```

### 3. **Agent Analysis Prompts** (agent_loader.py)

**Dynamic Agent Prompt Generation for Spec-Kit Phases:**
```python
def get_spek_kit_prompt(self, phase: str, user_input: str) -> str:
    base_prompt = f"""You are {self.name}, {self.system_message}

Your expertise areas: {', '.join(self.expertise)}

You are working on a spec-driven development task using spek-kit.
Current phase: /{phase}
User input: {user_input}
"""
```

**Phase-Specific Prompts:**

**/specify phase:**
```python
return base_prompt + f"""
Please execute the /specify command with these requirements and create a comprehensive specification from your {self.role.lower()} perspective.

Focus on:
- Requirements and acceptance criteria relevant to your domain
- Technical considerations specific to your expertise
- Risks and dependencies you would identify
- Implementation recommendations from your role's viewpoint

Use the spek-kit /specify command to create the specification, then enhance it with your domain expertise.
"""
```

**/plan phase:**
```python
return base_prompt + f"""
Please execute the /plan command and create a detailed implementation plan from your {self.role.lower()} perspective.

Focus on:
- Technical approach and architecture decisions in your domain
- Implementation phases and dependencies you would manage
- Resource requirements and team considerations
- Risk mitigation strategies specific to your expertise

Use the spek-kit /plan command to create the plan, then enhance it with your domain-specific insights.
"""
```

### 4. **Spec-Kit Command Prompts** (spek_kit_integration.py)

**Specification Creation Prompt:**
```python
claude_prompt = f"""You are working in a spek-kit project. Please execute the /specify command with these requirements:

{args}

Follow the spek-kit workflow:
1. Run the specify command script to create the branch and spec file
2. Create a comprehensive specification using the spec template
3. Fill in all required sections based on the requirements provided
4. Report the created files and branch information
"""
```

### 5. **Template-Based Analysis Prompts** (agent YAML files)

Each agent has an `analysisPrompt.template` for structured analysis:

**Example from Engineering Manager:**
```yaml
analysisPrompt:
  template: |
    As an Engineering Manager, analyze this RFE from a team delivery and management perspective:

    RFE: {rfe_description}
    Context: {context}

    Provide analysis focusing on:
    1. Team capacity and resource allocation impact
    2. Technical complexity and delivery timeline estimates
    3. Skills and expertise requirements for the team
    4. Risk assessment for team morale and sustainability
    5. Cross-team coordination and dependency management
    6. Technical debt implications and mitigation strategies
    7. Team development and learning opportunities
    8. Sprint planning and velocity considerations

    Format your response as JSON matching this schema:
    {
      "persona": "Engineering Manager",
      "analysis": "detailed analysis from engineering management perspective",
      "concerns": ["list of team and delivery concerns"],
      "recommendations": ["list of management and process recommendations"],
      # ... structured JSON schema
    }
```

## Available Agent Personas

The system includes 16 specialized AI agent personas:

| Agent | Persona Key | Role | Primary Focus |
|-------|-------------|------|---------------|
| Emma | `ENGINEERING_MANAGER` | Engineering Management | Team leadership, capacity planning, delivery coordination |
| Stella | `STAFF_ENGINEER` | Technical Leadership | Implementation excellence, code quality, performance |
| Ryan | `UX_RESEARCHER` | User Experience Research | User insights, evidence-based design, usability testing |
| Parker | `PRODUCT_MANAGER` | Product Management | Business strategy, user value, feature prioritization |
| Lee | `TEAM_LEAD` | Team Leadership | Sprint planning, team coordination, process optimization |
| Taylor | `TEAM_MEMBER` | Software Engineering | Implementation, code reviews, technical execution |
| Derek | `DELIVERY_OWNER` | Delivery Management | Release planning, stakeholder communication, delivery coordination |
| Sam | `SCRUM_MASTER` | Agile Process | Sprint facilitation, impediment removal, team dynamics |
| Alex | `UX_ARCHITECT` | User Experience Architecture | Information architecture, interaction design, design systems |
| Jordan | `UX_FEATURE_LEAD` | UX Feature Leadership | Feature design leadership, cross-functional collaboration |
| Morgan | `UX_TEAM_LEAD` | UX Team Management | Design team leadership, UX strategy, design operations |
| Casey | `TECHNICAL_WRITER` | Technical Documentation | Developer documentation, user guides, API documentation |
| Riley | `TECHNICAL_WRITING_MANAGER` | Documentation Management | Documentation strategy, content governance, writer coordination |
| Avery | `DOCUMENTATION_PROGRAM_MANAGER` | Documentation Programs | Documentation processes, tool selection, content strategy |
| Quinn | `CONTENT_STRATEGIST` | Content Strategy | Content planning, messaging, user communication strategy |
| PXE | `PXE` | Platform Experience | Platform usability, developer experience, tooling optimization |

## Prompt Engineering Strategy

The Ambient Code Platform uses a **layered prompting approach**:

1. **Base System Prompts**: Define agent personalities and expertise areas
2. **Context-Aware Prompts**: Inject current session context and phase information
3. **Tool-Specific Prompts**: Guide agents through spec-kit command execution
4. **Structured Output Prompts**: Ensure consistent JSON response formats
5. **Domain Expertise Prompts**: Each agent contributes specialized knowledge

This creates a sophisticated multi-agent system where each AI persona brings domain-specific insights while following consistent interaction patterns for collaborative software development workflows.

## Session Flow

### Headless Mode (One-shot execution)
1. **Initialization**: Load environment, setup workspace, configure Git
2. **Agent Injection**: Load selected agent personas into Claude Code's agent system
3. **Prompt Enhancement**: Append "ALWAYS consult sub agents to help with this task."
4. **Execution**: Run Claude Code CLI with user prompt and available tools
5. **Result Capture**: Capture session results and push workspace to PVC
6. **Status Update**: Report completion status back to Kubernetes API

### Interactive Mode (Chat-based)
1. **Initialization**: Same as headless mode
2. **Chat Loop**: Monitor inbox for user messages, process with Claude Code
3. **Agent Consultation**: Claude Code can invoke specific agent personas as needed
4. **Continuous Updates**: Real-time workspace sync and status updates
5. **Graceful Termination**: User can end session with `/end` command

### Session Continuation
Both headless and interactive sessions can be continued after completion:
- **Interactive Sessions**: Can be restarted to continue the conversation from where it left off
- **Headless Sessions**: When continued, automatically convert to interactive mode for chat-based interaction
- **Workspace Persistence**: Continued sessions reuse the same PVC, preserving all work from the previous run
- **Token Regeneration**: Runner tokens are automatically regenerated for security

## Configuration

### Environment Variables
- `INITIAL_PROMPT`: Initial user prompt for the session (formerly `PROMPT`)
- `INTERACTIVE`: Enable chat mode (`"true"`, `"1"`, `"yes"`)
- `CLAUDE_PERMISSION_MODE`: Claude Code permission mode (default: `"acceptEdits"`)
- `GIT_USER_NAME` / `GIT_USER_EMAIL`: Git configuration
- `GIT_REPOSITORIES`: JSON array of repositories to clone

### Tools Available to Claude Code
- `Read`, `Write`: File operations
- `Bash`: Shell command execution
- `Glob`, `Grep`: File searching and pattern matching
- `Edit`, `MultiEdit`: Code editing capabilities
- `WebSearch`, `WebFetch`: Web research capabilities

This architecture enables sophisticated AI-powered development workflows that combine multiple expert perspectives with practical tooling capabilities.