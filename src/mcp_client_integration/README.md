# MCP Client Integration

A Python library for Model Context Protocol (MCP) client integration with llama-index and other AI workflows.

## Features

- **Multi-server MCP client support** with JSON configuration
- **Connection pooling** and health monitoring 
- **OpenShift service discovery** patterns (external routes vs cluster services)
- **Standardized error handling** and validation
- **LlamaIndex integration** for AI workflows
- **Test-driven development** with comprehensive test coverage

## Installation

### As a dependency in your project

```bash
# Install from source (development)
pip install -e /path/to/vTeam/src/mcp_client_integration

# Or add to your requirements.txt
-e git+https://github.com/red-hat-data-services/vTeam.git#egg=mcp-client-integration&subdirectory=src/mcp_client_integration
```

### For development

```bash
cd src/mcp_client_integration
pip install -e ".[dev]"
```

## Quick Start

### Basic MCP Client Usage

```python
import asyncio
from mcp_client_integration import SimpleMCPClient

async def main():
    # Initialize client with JSON configuration from environment
    client = SimpleMCPClient()
    
    # Connect to all configured MCP servers
    await client.connect_all()
    
    # Send queries with automatic capability routing
    response = await client.query("What Jira tickets are assigned to me?")
    
    # Health check
    health = await client.health_check()
    print(f"Server health: {health}")
    
    # Cleanup
    await client.disconnect_all()

# Run the client
asyncio.run(main())
```

### Configuration

Set up MCP servers via environment variable:

```bash
export MCP_SERVERS='{
  "atlassian": "https://mcp-atlassian.apps.cluster.com/sse",
  "github": "https://mcp-github.apps.cluster.com/sse",
  "confluence": "mcp-confluence.default.svc.cluster.local:8080"
}'
```

### LlamaIndex Integration

```python
from mcp_client_integration import MCPLlamaIndexTool

# Create LlamaIndex tool
mcp_tool = MCPLlamaIndexTool()

# Add to your LlamaIndex agent
from llama_index.core.agent import ReActAgent

agent = ReActAgent.from_tools([mcp_tool])
response = agent.chat("Search for recent Jira tickets")
```

## Architecture

### Core Components

- **SimpleMCPClient**: Main client class with multi-server support
- **MCPConnectionPool**: Connection management and health monitoring
- **MCPConfigurationManager**: JSON configuration loading and validation
- **MCPEndpointConnector**: Endpoint validation and connectivity testing
- **MCPLlamaIndexTool**: LlamaIndex integration tool

### Common Utilities

- **Connection Management**: Standardized connection interfaces and pooling
- **Validation**: Endpoint and configuration validation utilities
- **Error Handling**: Structured error handling with context
- **Configuration**: Unified configuration management

## Testing

```bash
# Run tests
pytest

# With coverage
pytest --cov=. --cov-report=html

# Run specific test types
pytest tests/unit/
pytest tests/integration/
```

## Development

### Code Quality

```bash
# Format code
black .
isort .

# Lint
flake8 .

# Type check
mypy .
```

### Contributing

1. Follow TDD methodology - write tests first
2. Maintain >90% unit test coverage, >80% integration test coverage
3. Use common utilities to avoid code duplication
4. Follow the established error handling patterns

## Configuration Examples

### Simple Configuration

```json
{
  "atlassian": "https://mcp-atlassian.example.com/sse",
  "github": "https://mcp-github.example.com/sse"
}
```

### Advanced Configuration

```json
{
  "atlassian": {
    "endpoint": "https://mcp-atlassian.example.com/sse",
    "timeout": 60,
    "connection_type": "external_route",
    "enabled": true,
    "metadata": {"team": "platform"}
  },
  "confluence": {
    "endpoint": "mcp-confluence.default.svc.cluster.local:8080",
    "timeout": 30,
    "connection_type": "cluster_service",
    "enabled": true
  }
}
```

## License

MIT License - see LICENSE file for details.