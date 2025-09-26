"""
MCP Client Integration for Llama Index

This module provides MCP (Model Context Protocol) client integration for llama index
deployments, enabling access to Jira and Confluence data through standardized interfaces.

Based on SPIKE-001 and SPIKE-002 validated patterns.
Refactored with common utilities for reduced code duplication and standardized interfaces.
"""

from .simple_mcp_client import SimpleMCPClient
from .endpoint_connector import MCPEndpointConnector
from .llama_integration import MCPEnhancedLlamaIndex
from .llama_index_tool import MCPLlamaIndexTool, create_mcp_tool

# Export common utilities for advanced users
from .common import (
    MCPConnectionPool,
    MCPConfigurationManager,
    MCPEndpointValidator,
    MCPErrorHandler,
    MCPError,
    MCPConnectionError,
    MCPConfigurationError,
    handle_mcp_errors
)

__version__ = "1.0.0"
__all__ = [
    # Main client classes
    "SimpleMCPClient",
    "MCPEndpointConnector", 
    "MCPEnhancedLlamaIndex",
    "MCPLlamaIndexTool",
    "create_mcp_tool",
    
    # Common utilities
    "MCPConnectionPool",
    "MCPConfigurationManager", 
    "MCPEndpointValidator",
    "MCPErrorHandler",
    "MCPError",
    "MCPConnectionError",
    "MCPConfigurationError",
    "handle_mcp_errors"
]