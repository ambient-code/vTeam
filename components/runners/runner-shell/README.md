# Runner Shell

Standardized shell framework for AI agent runners in the vTeam platform.

## Architecture

The Runner Shell provides a common framework for different AI agents (Claude, OpenAI, etc.) with standardized:

- **Protocol**: Common message format and types
- **Transport**: WebSocket communication with backend
- **Sink**: S3 persistence for message durability
- **Context**: Session information and utilities

## Components

### Core
- `shell.py` - Main orchestrator
- `protocol.py` - Message definitions
- `transport_ws.py` - WebSocket transport
- `sink_s3.py` - S3 message persistence
- `context.py` - Runner context

### Adapters
- `adapters/claude/` - Claude AI adapter


## Usage

```bash
runner-shell \
  --session-id sess-123 \
  --workspace-path /workspace \
  --websocket-url ws://backend:8080/session/sess-123/ws \
  --s3-bucket ambient-code-sessions \
  --adapter claude
```

## Development

```bash
# Install in development mode
pip install -e ".[dev]"

# Format code
black runner_shell/
```

## Environment Variables

- `ANTHROPIC_API_KEY` - Claude API key
- `AWS_ACCESS_KEY_ID` - AWS credentials for S3
- `AWS_SECRET_ACCESS_KEY` - AWS credentials for S3