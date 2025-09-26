# MCP Client Integration - Installation Guide

This guide covers installation and usage of the MCP Client Integration library for various deployment scenarios.

## Quick Start

### Prerequisites

- Python 3.8 or higher
- Virtual environment (recommended)

### Basic Installation

```bash
# From the vTeam project root
cd src/mcp_client_integration
pip install -e .
```

### Development Installation

```bash
# Install with development dependencies
cd src/mcp_client_integration
pip install -e ".[dev]"
```

## Installation Scenarios

### 1. Local Development

For local development and testing:

```bash
# Clone the repository
git clone https://github.com/red-hat-data-services/vTeam.git
cd vTeam/src/mcp_client_integration

# Create virtual environment
python -m venv venv
source venv/bin/activate  # Linux/Mac
# or
venv\Scripts\activate     # Windows

# Install in development mode
pip install -e ".[dev]"
```

### 2. Integration in Other Projects

#### Option A: Direct Installation from Source

Add to your `requirements.txt`:
```
-e git+https://github.com/red-hat-data-services/vTeam.git#egg=mcp-client-integration&subdirectory=src/mcp_client_integration
```

#### Option B: Local Path Installation

```bash
# Install from local path
pip install -e /path/to/vTeam/src/mcp_client_integration
```

#### Option C: Copy Package

Copy the entire `src/mcp_client_integration` directory to your project and install:

```bash
pip install -e ./mcp_client_integration
```

### 3. Production Deployment

For production deployments, create a wheel package:

```bash
cd src/mcp_client_integration
pip install build
python -m build
pip install dist/mcp_client_integration-1.0.0-py3-none-any.whl
```

## Configuration

### Environment Variables

Set up MCP servers via environment variable:

```bash
export MCP_SERVERS='{
  "atlassian": "https://mcp-atlassian.apps.cluster.com/sse",
  "github": "https://mcp-github.apps.cluster.com/sse",
  "confluence": "mcp-confluence.default.svc.cluster.local:8080"
}'
```

### Security Configuration

For production deployments, use production mode for enhanced security:

```python
from mcp_client_integration.common import MCPConfigurationManager

# Production mode - strict security validation
config_manager = MCPConfigurationManager(production_mode=True)

# Development mode - more permissive for testing
config_manager = MCPConfigurationManager(production_mode=False)
```

⚠️ **Security Note**: Always use `production_mode=True` in production environments for enhanced security validation. See [SECURITY.md](SECURITY.md) for detailed security guidelines.

### Advanced Configuration

```bash
export MCP_SERVERS='{
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
}'
```

## Usage Examples

### Basic Usage

```python
import asyncio
from mcp_client_integration import SimpleMCPClient

async def main():
    # Initialize client
    client = SimpleMCPClient()
    
    # Connect to all servers
    await client.connect_all()
    
    # Send queries
    response = await client.query("What Jira tickets are assigned to me?")
    print(response)
    
    # Health check
    health = await client.health_check()
    print(f"Server health: {health}")
    
    # Cleanup
    await client.disconnect_all()

# Run the client
asyncio.run(main())
```

### LlamaIndex Integration

```python
from mcp_client_integration import MCPLlamaIndexTool

# Create tool
mcp_tool = MCPLlamaIndexTool()

# Use with LlamaIndex (requires llama-index to be installed separately)
try:
    from llama_index.core.agent import ReActAgent
    
    # Create agent with MCP tool
    agent = ReActAgent.from_tools([mcp_tool.to_llama_index_tool()])
    response = agent.chat("Search for recent Jira tickets")
    print(response)
except ImportError:
    # Use directly without LlamaIndex
    result = mcp_tool("Search for recent Jira tickets")
    print(result)
```

### Endpoint Validation

```python
from mcp_client_integration import MCPEndpointConnector

async def validate_endpoints():
    connector = MCPEndpointConnector()
    
    # Validate endpoint formats
    valid = connector.validate_endpoint_config("https://mcp-server.com/sse")
    print(f"Endpoint valid: {valid}")
    
    # Test connectivity
    result = await connector.test_connectivity("https://mcp-server.com/sse")
    print(f"Connectivity: {result}")

asyncio.run(validate_endpoints())
```

## Integration with demos/rfe-builder

The MCP client integration is designed to work seamlessly with the RFE builder demo:

```python
# In demos/rfe-builder application
from mcp_client_integration import SimpleMCPClient, MCPLlamaIndexTool

# Use in your RFE workflow
async def enhance_rfe_with_data():
    client = SimpleMCPClient()
    await client.connect_all()
    
    # Query for related tickets
    jira_data = await client.query("Find related tickets for this RFE", "atlassian")
    
    # Process with LlamaIndex
    tool = MCPLlamaIndexTool()
    enhanced_result = await tool.call("Analyze this data for RFE insights")
    
    return enhanced_result
```

## Docker Deployment

### Dockerfile Example

```dockerfile
FROM python:3.11-slim

WORKDIR /app

# Copy MCP client integration
COPY src/mcp_client_integration ./mcp_client_integration

# Install dependencies
RUN pip install -e ./mcp_client_integration

# Copy your application code
COPY your_app ./your_app

# Set environment variables
ENV MCP_SERVERS='{"atlassian": "https://mcp-atlassian.svc.cluster.local/sse"}'

CMD ["python", "-m", "your_app"]
```

### Docker Compose Example

```yaml
version: '3.8'
services:
  mcp-client-app:
    build: .
    environment:
      - MCP_SERVERS={"atlassian": "https://mcp-atlassian.apps.cluster.local/sse"}
    depends_on:
      - mcp-atlassian-server
  
  mcp-atlassian-server:
    image: mcp-atlassian:latest
    ports:
      - "8080:8080"
```

## Kubernetes Deployment

### ConfigMap for Configuration

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: mcp-client-config
data:
  MCP_SERVERS: |
    {
      "atlassian": "https://mcp-atlassian.apps.cluster.local/sse",
      "github": "mcp-github.default.svc.cluster.local:8080"
    }
```

### Deployment with ConfigMap

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: mcp-client-app
spec:
  replicas: 1
  selector:
    matchLabels:
      app: mcp-client-app
  template:
    metadata:
      labels:
        app: mcp-client-app
    spec:
      containers:
      - name: app
        image: your-app:latest
        envFrom:
        - configMapRef:
            name: mcp-client-config
```

## Testing

### Run Tests

```bash
# Run all tests
pytest

# Run with coverage
pytest --cov=. --cov-report=html

# Run specific test types
pytest tests/unit/
pytest tests/integration/
```

### Mock Mode for Testing

```python
# Use mock mode for testing
client = SimpleMCPClient(mock=True)
await client.connect_all()  # Uses mock connections

# Mock responses for specific tests
from unittest.mock import patch
with patch.object(client.connection_pool, 'send_message') as mock_send:
    mock_send.return_value = {"result": "test data"}
    response = await client.query("test query")
```

## Troubleshooting

### Common Issues

1. **Import Errors**
   ```bash
   # Ensure package is installed
   pip install -e .
   
   # Check Python path
   python -c "import mcp_client_integration; print(mcp_client_integration.__file__)"
   ```

2. **Configuration Errors**
   ```bash
   # Validate JSON configuration
   python -c "import json; json.loads('$MCP_SERVERS')"
   
   # Test basic connectivity
   python -c "from mcp_client_integration import MCPEndpointConnector; print('Import successful')"
   ```

3. **Connection Issues**
   ```python
   # Debug connectivity
   import asyncio
   from mcp_client_integration import MCPEndpointConnector
   
   async def debug():
       connector = MCPEndpointConnector()
       result = await connector.test_connectivity("your-endpoint")
       print(result)
   
   asyncio.run(debug())
   ```

### Logging

Enable debug logging:

```python
import logging
logging.basicConfig(level=logging.DEBUG)

# MCP-specific logging
mcp_logger = logging.getLogger('mcp_client_integration')
mcp_logger.setLevel(logging.DEBUG)
```

## Dependencies

### Core Dependencies
- `httpx[http2]>=0.28.1` - HTTP client with HTTP/2 support
- `websockets>=13.1` - WebSocket support for SSE  
- `certifi>=2024.8.30` - Up-to-date CA certificates for secure connections
- `cryptography>=41.0.0` - Cryptographic operations for secure connections

### Development Dependencies
- `pytest>=8.3.5` - Testing framework
- `pytest-asyncio>=0.24.0` - Async test support
- `pytest-cov>=5.0.0` - Test coverage reporting
- `bandit` - Security linting
- `safety` - Dependency vulnerability scanning
- Development tools (black, isort, flake8, mypy)

### Optional Dependencies
- LlamaIndex components (install separately as needed)

## Support

For issues and questions:
- GitHub Issues: https://github.com/red-hat-data-services/vTeam/issues
- Documentation: See README.md in the package directory
- Examples: Check the `demos/rfe-builder` integration