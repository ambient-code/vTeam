#!/usr/bin/env python3
"""
MCPLlamaIndexTool - LlamaIndex Tool Integration

This module provides a LlamaIndex tool wrapper for MCP client integration,
enabling direct use within LlamaIndex agents and workflows.
"""

import asyncio
import logging
from typing import Dict, Any, Optional

from .simple_mcp_client import SimpleMCPClient
from .common import handle_mcp_errors, MCPErrorHandler

# Configure logging
logger = logging.getLogger(__name__)


class MCPLlamaIndexTool:
    """
    LlamaIndex tool wrapper for MCP client integration.
    
    This class provides a standardized tool interface for LlamaIndex agents
    to interact with MCP servers for data retrieval and analysis.
    """
    
    def __init__(self, env_var: str = "MCP_SERVERS", mock: bool = False):
        """
        Initialize MCPLlamaIndexTool.
        
        Args:
            env_var: Environment variable containing MCP server configuration
            mock: Whether to use mock connections for testing
        """
        self.mcp_client = SimpleMCPClient(env_var, mock)
        self.error_handler = MCPErrorHandler(__name__)
        self._initialized = False
        
        # Tool metadata for LlamaIndex integration
        self.name = "mcp_query"
        self.description = (
            "Query MCP servers for data from Atlassian (Jira, Confluence), "
            "GitHub, and other configured services. Automatically routes "
            "queries to appropriate servers based on content."
        )
        
        logger.info("MCPLlamaIndexTool initialized")
    
    @handle_mcp_errors("initialize_tool")
    async def initialize(self):
        """Initialize MCP connections if not already initialized."""
        if not self._initialized:
            await self.mcp_client.connect_all()
            self._initialized = True
            logger.info("MCP tool connections initialized")
    
    @handle_mcp_errors("query_tool")
    async def call(self, query: str, capability: Optional[str] = None) -> Dict[str, Any]:
        """
        Call the MCP tool with a query.
        
        This method is the primary interface for LlamaIndex agents to
        interact with MCP servers.
        
        Args:
            query: The query string to send to MCP servers
            capability: Optional specific capability to target
            
        Returns:
            Dictionary containing query results and metadata
        """
        # Ensure tool is initialized
        if not self._initialized:
            await self.initialize()
        
        logger.debug(f"MCPLlamaIndexTool processing query: {query[:100]}...")
        
        try:
            # Query MCP servers
            result = await self.mcp_client.query(query, capability)
            
            # Format response for LlamaIndex
            return {
                "tool": "mcp_query",
                "query": query,
                "capability": capability,
                "result": result,
                "success": True,
                "metadata": {
                    "server_status": self.mcp_client.get_server_status(),
                    "query_length": len(query)
                }
            }
            
        except Exception as e:
            logger.error(f"MCP tool query failed: {e}")
            return {
                "tool": "mcp_query",
                "query": query,
                "error": str(e),
                "success": False
            }
    
    def __call__(self, query: str, capability: Optional[str] = None) -> Dict[str, Any]:
        """
        Synchronous call interface for LlamaIndex compatibility.
        
        This method wraps the async call method for synchronous usage
        in LlamaIndex agents that may not support async tools.
        
        Args:
            query: The query string to send to MCP servers
            capability: Optional specific capability to target
            
        Returns:
            Dictionary containing query results and metadata
        """
        # Run async call in event loop
        try:
            loop = asyncio.get_event_loop()
            return loop.run_until_complete(self.call(query, capability))
        except RuntimeError:
            # Create new event loop if none exists
            loop = asyncio.new_event_loop()
            asyncio.set_event_loop(loop)
            try:
                return loop.run_until_complete(self.call(query, capability))
            finally:
                loop.close()
    
    @handle_mcp_errors("get_tool_status")
    async def get_status(self) -> Dict[str, Any]:
        """
        Get tool status and MCP server health information.
        
        Returns:
            Dictionary with tool and server status information
        """
        if not self._initialized:
            await self.initialize()
        
        return {
            "tool_name": self.name,
            "initialized": self._initialized,
            "mcp_servers": self.mcp_client.get_server_status(),
            "health": await self.mcp_client.health_check()
        }
    
    async def close(self):
        """Close all MCP connections."""
        if self._initialized:
            await self.mcp_client.disconnect_all()
            self._initialized = False
            logger.info("MCP tool connections closed")
    
    def to_llama_index_tool(self):
        """
        Convert to LlamaIndex tool format.
        
        This method creates a properly formatted tool for LlamaIndex agents.
        Note: Requires llama-index to be installed by the consuming application.
        
        Returns:
            LlamaIndex tool instance
        """
        try:
            from llama_index.core.tools import FunctionTool
            
            return FunctionTool.from_defaults(
                fn=self.__call__,
                name=self.name,
                description=self.description,
                return_direct=False
            )
        except ImportError as e:
            logger.error(
                "llama-index not available. Install llama-index dependencies "
                "in your consuming application to use this method."
            )
            raise ImportError(
                "llama-index not installed. Please install llama-index "
                "dependencies to use LlamaIndex integration."
            ) from e
    
    def get_tool_metadata(self) -> Dict[str, Any]:
        """
        Get tool metadata for registration with LlamaIndex.
        
        Returns:
            Dictionary with tool metadata
        """
        return {
            "name": self.name,
            "description": self.description,
            "version": "1.0.0",
            "type": "mcp_integration",
            "async_support": True,
            "sync_support": True,
            "capabilities": list(self.mcp_client.servers.keys()) if hasattr(self.mcp_client, 'servers') else []
        }


# Convenience function for quick tool creation
def create_mcp_tool(env_var: str = "MCP_SERVERS", mock: bool = False) -> MCPLlamaIndexTool:
    """
    Convenience function to create an MCPLlamaIndexTool.
    
    Args:
        env_var: Environment variable containing MCP server configuration
        mock: Whether to use mock connections for testing
        
    Returns:
        Configured MCPLlamaIndexTool instance
    """
    return MCPLlamaIndexTool(env_var, mock)