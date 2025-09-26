# MCP Client Integration - Security Guide

This guide covers security considerations and best practices for the MCP Client Integration library.

## Security Features

### 1. Configuration Validation

The library includes comprehensive security validation:

- **Input Sanitization**: All configuration inputs are validated for injection attacks
- **Size Limits**: Configuration size is limited to prevent DoS attacks
- **Schema Validation**: JSON structure is strictly validated
- **Endpoint Validation**: URLs are validated for malicious patterns

### 2. Connection Security

#### SSL/TLS Configuration

```python
from mcp_client_integration import SimpleMCPClient

# Production mode - strict security (recommended)
client = SimpleMCPClient()  # Default: verify_ssl=True

# Development mode - relaxed validation (testing only)
config = MCPConfigurationManager(production_mode=False)
```

#### Production vs Development Mode

**Production Mode** (recommended for production):
- Only HTTPS connections allowed
- SSL certificate validation enforced
- Private IP addresses blocked
- Strict timeout limits

**Development Mode** (testing only):
- HTTP connections allowed for localhost
- SSL verification can be disabled
- More permissive validation

### 3. Environment Variable Security

#### Secure Configuration

```bash
# ✅ SECURE: Use environment variables
export MCP_SERVERS='{"atlassian": "https://mcp-server.company.com/sse"}'

# ❌ INSECURE: Don't hardcode in source code
client = SimpleMCPClient(config={"server": "https://secret-server.com"})
```

#### Configuration Validation

```python
from mcp_client_integration.common import MCPSecurityValidator

# Validate configuration before use
validator = MCPSecurityValidator(production_mode=True)
result = validator.validate_configuration_security(config_data)

if not result.valid:
    raise ValueError(f"Security validation failed: {result.error_message}")
```

## Security Best Practices

### 1. Network Security

```python
# ✅ SECURE: Always use HTTPS in production
{
  "atlassian": "https://mcp-atlassian.company.com/sse",
  "github": "https://mcp-github.company.com/sse"
}

# ❌ INSECURE: HTTP connections
{
  "atlassian": "http://mcp-atlassian.company.com/sse"  # Vulnerable to MITM
}
```

### 2. Credential Management

```python
# ✅ SECURE: Use environment variables or secret management
import os

mcp_config = {
    "atlassian": os.getenv("MCP_ATLASSIAN_URL"),
    "auth_token": os.getenv("MCP_AUTH_TOKEN")  # If auth is implemented
}

# ❌ INSECURE: Hardcoded credentials
mcp_config = {
    "atlassian": "https://user:password@mcp-server.com/sse"
}
```

### 3. Timeout Configuration

```python
# ✅ SECURE: Set reasonable timeouts
{
  "atlassian": {
    "endpoint": "https://mcp-atlassian.com/sse",
    "timeout": 30,  # 30 seconds maximum
    "connection_type": "external_route"
  }
}

# ❌ INSECURE: No timeout or excessive timeout
{
  "atlassian": {
    "endpoint": "https://mcp-atlassian.com/sse",
    "timeout": 3600  # 1 hour - too long, enables DoS
  }
}
```

### 4. Connection Pool Limits

```python
from mcp_client_integration.common import MCPConnectionPool

# ✅ SECURE: Limit connection pool size
pool = MCPConnectionPool(max_connections=10)  # Reasonable limit

# ❌ INSECURE: Unlimited connections
pool = MCPConnectionPool(max_connections=1000)  # Resource exhaustion risk
```

## Security Validation Examples

### Basic Security Check

```python
from mcp_client_integration.common import MCPSecurityValidator

validator = MCPSecurityValidator(production_mode=True)

# Valid configuration
config = {
    "atlassian": "https://mcp-atlassian.company.com/sse",
    "timeout": 30
}

result = validator.validate_configuration_security(config)
if result.valid:
    print("✅ Configuration is secure")
else:
    print(f"❌ Security issue: {result.error_message}")
```

### Production Security Validation

```python
# Production-grade validation
validator = MCPSecurityValidator(production_mode=True)

# This will fail in production mode
insecure_config = {
    "local": "http://localhost:8080/sse",  # Blocked in production
    "internal": "http://192.168.1.100/sse"  # Private IP blocked
}

result = validator.validate_configuration_security(insecure_config)
# Result: valid=False, error_message="Localhost/loopback addresses not allowed in production mode"
```

## Common Security Issues

### 1. Configuration Injection

```python
# ❌ DANGEROUS: User input directly used in configuration
user_input = request.json.get("mcp_server")
config = json.dumps({"server": user_input})  # Potential injection

# ✅ SAFE: Validate user input
from mcp_client_integration.common import MCPSecurityValidator

validator = MCPSecurityValidator(production_mode=True)
result = validator.validate_configuration_security({"server": user_input})

if result.valid:
    config = json.dumps({"server": user_input})
else:
    raise ValueError("Invalid server configuration")
```

### 2. SSL Certificate Issues

```python
# ❌ DANGEROUS: Disabling SSL verification
connection = ExternalRouteMCPConnection(
    "https://mcp-server.com/sse",
    verify_ssl=False  # Vulnerable to MITM attacks
)

# ✅ SAFE: Always verify SSL in production
connection = ExternalRouteMCPConnection(
    "https://mcp-server.com/sse",
    verify_ssl=True  # Default and recommended
)
```

### 3. Resource Exhaustion

```python
# ❌ DANGEROUS: No limits on configuration size
large_config = {"server_" + str(i): f"https://server{i}.com" for i in range(10000)}

# ✅ SAFE: Library automatically enforces limits
# MCPSecurityValidator.MAX_ENDPOINTS = 50
# MCPSecurityValidator.MAX_CONFIG_SIZE = 50KB
```

## Security Checklist

Before deploying to production:

- [ ] **Configuration Validation**: All configurations pass security validation
- [ ] **HTTPS Only**: All external connections use HTTPS
- [ ] **SSL Verification**: Certificate validation is enabled
- [ ] **Environment Variables**: Secrets stored in environment variables or secret management
- [ ] **Timeout Limits**: Reasonable timeout values configured
- [ ] **Connection Limits**: Connection pool size limits set
- [ ] **Production Mode**: `production_mode=True` for production deployments
- [ ] **Logging**: Security events are logged appropriately
- [ ] **Updates**: Dependencies are up-to-date with security patches

## Reporting Security Issues

If you discover a security vulnerability in the MCP Client Integration library:

1. **Do not** open a public GitHub issue
2. Email security concerns to: [security contact from project]
3. Include:
   - Description of the vulnerability
   - Steps to reproduce
   - Potential impact assessment
   - Suggested mitigation

## Security Dependencies

The library includes these security-focused dependencies:

- `certifi>=2024.8.30`: Up-to-date CA certificates
- `cryptography>=41.0.0`: Secure cryptographic operations
- `httpx[http2]>=0.28.1`: Secure HTTP client with HTTP/2 support

## Development Security Tools

For development and testing:

```bash
# Install security scanning tools
pip install bandit safety

# Run security linting
bandit -r src/

# Check for known vulnerabilities
safety check
```