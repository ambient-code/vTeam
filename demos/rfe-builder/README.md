# RHOAI AI Feature Sizing Platform

A production-ready multi-agent system for analyzing Request for Enhancement (RFE) descriptions using specialized AI personas with RAG-powered knowledge bases, built on **LlamaDeploy** and **@llamaindex/server**.

## Overview

RHOAI uses 7 specialized AI agents working together to provide comprehensive feature analysis:

- **UX Designer (UXD)** - User experience, interface design, accessibility
- **Product Manager (PM)** - Business requirements, prioritization, stakeholder alignment  
- **Backend Engineer** - System architecture, APIs, database design
- **Frontend Engineer** - React implementation, TypeScript, state management
- **Architect** - Overall system design, integration patterns, scalability
- **Product Owner** - Business value, acceptance criteria, stakeholder management
- **SME/Researcher** - Domain expertise, industry best practices, research

## Features

- **🐍 Production Python Backend**: LlamaDeploy workflow orchestration with native Python LlamaIndex
- **🟨 Modern TypeScript Frontend**: Professional chat UI powered by @llamaindex/server
- **🤖 Multi-Agent Analysis**: Each agent analyzes RFEs from their specialized perspective
- **📚 RAG Knowledge Bases**: Agents access domain-specific knowledge from configured data sources
- **🔗 MCP Integration**: Connect to JIRA, GitHub, and Confluence via Model Context Protocol
- **🚀 Production Ready**: Enterprise-grade deployment, monitoring, and scaling
- **🔗 API Access**: Full REST API for programmatic integration
- **📊 Real-time Progress**: Streaming responses and workflow observability

## Architecture

### Production System Design
```
┌─────────────────────────────┐    ┌─────────────────────────────┐
│     Python Backend          │    │   TypeScript Frontend       │
│     (LlamaDeploy)           │    │   (@llamaindex/server)      │
│                             │    │                             │
│ • Multi-Agent Workflows     │────│ • Modern Chat Interface     │
│ • RAG Vector Retrieval      │    │ • Real-time Updates         │
│ • Python LlamaIndex v0.12+  │    │ • API Integration           │
│ • Production Orchestration  │    │ • Professional UI           │
└─────────────────────────────┘    └─────────────────────────────┘
```

### Agent Workflow
1. **RFE Input** - User submits feature description via chat UI
2. **Multi-Agent Analysis** - LlamaDeploy orchestrates all 7 agents simultaneously 
3. **Knowledge Retrieval** - RAG system provides domain-specific context for each agent
4. **Synthesis** - Comprehensive analysis combining all agent perspectives
5. **Deliverables** - Component teams, architecture diagrams, implementation timeline

## Prerequisites

If you haven't installed uv, you can follow the instructions [here](https://docs.astral.sh/uv/getting-started/installation/) to install it.

You can configure [LLM model](https://docs.llamaindex.ai/en/stable/module_guides/models/llms) and [embedding model](https://docs.llamaindex.ai/en/stable/module_guides/models/embeddings) in [src/settings.py](src/settings.py).

You must also install `pnpm` globally
```bash
npm i -g pnpm
```

Please setup their API keys in the `.env` file (copy from `.env.template`).

## MCP Integration

RFE Builder now supports Model Context Protocol (MCP) integration to enhance RFE creation with real-time data from external systems:

### Supported MCP Servers
- **Atlassian (JIRA)**: Search for related tickets and project context
- **GitHub**: Repository information and development context  
- **Confluence**: Documentation and knowledge base search

### MCP Configuration

1. Copy the environment template:
   ```bash
   cp .env.template .env
   ```

2. Configure MCP servers in `.env`:
   ```bash
   MCP_SERVERS='{
     "atlassian": "https://your-mcp-atlassian-server/sse",
     "github": "https://your-mcp-github-server/sse",
     "confluence": "https://your-mcp-confluence-server/sse"
   }'
   ```

3. Test MCP integration:
   ```bash
   uv run python -m src.mcp_cli validate
   uv run python -m src.mcp_cli health
   uv run python -m src.mcp_cli test
   ```

### MCP CLI Commands

The RFE Builder includes comprehensive CLI tools for MCP management:

```bash
# Validate MCP configuration
uv run python -m src.mcp_cli validate

# Check system health
uv run python -m src.mcp_cli health

# Test connectivity
uv run python -m src.mcp_cli test

# Search JIRA tickets
uv run python -m src.mcp_cli search-jira --query "authentication" --project "PROJ"

# Get GitHub repository info
uv run python -m src.mcp_cli github-info --owner "myorg" --repo "myrepo"

# Search Confluence docs
uv run python -m src.mcp_cli search-confluence --query "api documentation"
```

## Installation

Both the SDK and the CLI are part of the LlamaDeploy Python package. To install, just run:

```bash
uv sync
```

## Generate Index

Generate the embeddings of the documents in the `./data` directory:

```shell
uv run generate
```

## Running the Deployment

At this point we have all we need to run this deployment. Ideally, we would have the API server already running
somewhere in the cloud, but to get started let's start an instance locally. Run the following python script
from a shell:

```
$ uv run -m llama_deploy.apiserver
INFO:     Started server process [10842]
INFO:     Waiting for application startup.
INFO:     Application startup complete.
INFO:     Uvicorn running on http://0.0.0.0:4501 (Press CTRL+C to quit)
```

From another shell, use the CLI, `llamactl`, to create the deployment:

```
$ uv run llamactl deploy deployment.yml
Deployment successful: rhoai-ai-feature-sizing
```

## UI Interface

LlamaDeploy will serve the UI through the apiserver. Point the browser to [http://localhost:4501/deployments/rhoai-ai-feature-sizing/ui](http://localhost:4501/deployments/rhoai-ai-feature-sizing/ui) to interact with your deployment through a user-friendly interface.

## API endpoints

You can find all the endpoints in the [API documentation](http://localhost:4501/docs). To get started, you can try the following endpoints:

Create a new task:

```bash
curl -X POST 'http://localhost:4501/deployments/rhoai-ai-feature-sizing/tasks/create' \
  -H 'Content-Type: application/json' \
  -d '{
    "input": "{\"user_msg\":\"Hello\",\"chat_history\":[]}",
    "service_id": "rfe-builder-workflow"
  }'
```

Stream events:

```bash
curl 'http://localhost:4501/deployments/rhoai-ai-feature-sizing/tasks/0b411be6-005d-43f0-9b6b-6a0017f08002/events?session_id=dd36442c-45ca-4eaa-8d75-b4e6dad1a83e&raw_event=true' \
  -H 'Content-Type: application/json'
```

Note that the task_id and session_id are returned when creating a new task.

## Use Case

We have prepared a comprehensive RFE Builder workflow system that helps you interactively build RFEs with multi-agent collaboration, generate multiple artifacts, and edit them through chat.
The main workflow is in [`src/rfe_builder_workflow.py`](src/rfe_builder_workflow.py).

## Customize the UI

The UI is served by LlamaDeploy, you can configure the UI by modifying the `uiConfig` in the [ui/index.ts](ui/index.ts) file.

The following are the available options:

- `starterQuestions`: Predefined questions for chat interface
- `componentsDir`: Directory for custom event components
- `layoutDir`: Directory for custom layout components
- `llamaDeploy`: The LlamaDeploy configuration (deployment name and workflow name that defined in the [deployment.yml](deployment.yml) file)

## Agent Configuration

Agents are configured in YAML files in `src/agents/`. Each agent specifies:

- **Persona & Role** - Name and domain expertise
- **Data Sources** - Knowledge base directories or GitHub repositories  
- **Analysis Prompts** - Structured prompts for consistent output
- **Sample Knowledge** - Fallback knowledge when no custom data available

Example agent configuration:
```yaml
name: "Frontend Engineer"
persona: "FRONTEND_ENG"
expertise: ["react", "typescript", "ui-components"]

dataSources:
  - "frontend-patterns"
  - name: "react-docs"
    type: "github"
    source: "facebook/react"
    options:
      path: "docs/"
```

## Data Sources

### Local Directories
Place documentation in `data/` subdirectories matching agent data source names.

### GitHub Repositories  
Configure in agent YAML files. Python pipeline handles cloning and indexing.

### Hybrid Loading
1. **Python indexes** - Loaded first if available
2. **Local directories** - TypeScript fallback for simple cases
3. **Sample knowledge** - Built-in fallback for testing

## Technical Stack

- **Python** - Core workflow engine, agent coordination, LlamaDeploy orchestration
- **TypeScript** - Modern UI configuration and customization
- **LlamaIndex** - RAG system, vector stores, document processing, workflows
- **LlamaDeploy** - Production deployment and service orchestration
- **OpenAI** - Language model and embeddings
- **YAML** - Agent configuration with structured definitions

## Development

```bash
# Start with hot reload for development
uv run -m llama_deploy.apiserver

# In another terminal, deploy your changes
uv run llamactl deploy deployment.yml
```

## File Structure

```
/
├── src/agents/          # Agent YAML configurations  
├── src/                # Core Python workflow and settings
├── ui/                 # TypeScript UI configuration
├── data/               # Local knowledge bases
└── deployment.yml      # LlamaDeploy configuration
```

## Learn More

- [LlamaIndex Documentation](https://docs.llamaindex.ai) - learn about LlamaIndex.
- [Workflows Introduction](https://docs.llamaindex.ai/en/stable/understanding/workflows/) - learn about LlamaIndex workflows.
- [LlamaDeploy GitHub Repository](https://github.com/run-llama/llama_deploy)
- [Chat-UI Documentation](https://ts.llamaindex.ai/docs/chat-ui)

You can check out [the LlamaIndex GitHub repository](https://github.com/run-llama/llama_index) - your feedback and contributions are welcome!

This system provides a foundation for multi-agent RFE analysis with extensible agent configurations and flexible data source integration.
