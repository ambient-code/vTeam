"""
Common utilities for MCP client integration.

This package provides shared utilities for connection management, validation,
error handling, and configuration management across MCP components.

This refactored architecture eliminates code duplication and provides
standardized interfaces for MCP operations.
"""

# Connection management
from .connection_manager import (
    MCPConnectionInterface,
    MCPConnectionFactory,
    MCPConnectionPool,
    MockMCPConnection,
    ExternalRouteMCPConnection,
    ClusterServiceMCPConnection
)

# Validation utilities
from .validation import (
    MCPEndpointValidator,
    MCPConfigurationValidator,
    MCPSecurityValidator,
    ValidationResult
)

# Error handling
from .error_handler import (
    MCPError,
    MCPConnectionError,
    MCPConfigurationError,
    MCPValidationError,
    MCPProtocolError,
    MCPTimeoutError,
    MCPErrorHandler,
    MCPErrorCategory,
    MCPErrorContext,
    handle_mcp_errors,
    default_error_handler
)

# Configuration management
from .configuration import (
    MCPConfigurationManager,
    MCPConfiguration,
    MCPServerConfig,
    load_mcp_configuration,
    create_simple_configuration,
    validate_mcp_configuration_dict
)

__version__ = "1.0.0"

__all__ = [
    # Connection management
    "MCPConnectionInterface",
    "MCPConnectionFactory",
    "MCPConnectionPool",
    "MockMCPConnection",
    "ExternalRouteMCPConnection",
    "ClusterServiceMCPConnection",
    
    # Validation
    "MCPEndpointValidator",
    "MCPConfigurationValidator",
    "MCPSecurityValidator",
    "ValidationResult",
    
    # Error handling
    "MCPError",
    "MCPConnectionError",
    "MCPConfigurationError",
    "MCPValidationError",
    "MCPProtocolError",
    "MCPTimeoutError",
    "MCPErrorHandler",
    "MCPErrorCategory",
    "MCPErrorContext",
    "handle_mcp_errors",
    "default_error_handler",
    
    # Configuration
    "MCPConfigurationManager",
    "MCPConfiguration",
    "MCPServerConfig",
    "load_mcp_configuration",
    "create_simple_configuration",
    "validate_mcp_configuration_dict"
]