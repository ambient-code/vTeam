#!/usr/bin/env python3
"""
SimpleMCPClient - Multi-MCP Server Client Implementation (Refactored)

This module provides the core MCP client implementation for llama index integration,
supporting multiple MCP servers with simplified JSON configuration.

Based on SPIKE-001 and SPIKE-002 validated patterns.
Refactored to use common utilities for reduced code duplication.
"""

import asyncio
import logging
from typing import Dict, Any, Optional

from .common import (
    MCPConnectionPool,
    MCPConfigurationManager,
    MCPErrorHandler,
    MCPConfiguration,
    handle_mcp_errors,
    MCPConnectionError,
    MCPConfigurationError
)

# Configure logging
logger = logging.getLogger(__name__)


class SimpleMCPClient:
    """
    Simple MCP client with multi-server support using opinionated JSON configuration.
    
    This class implements the enhanced US-001 pattern with SPIKE-001 and SPIKE-002
    validated connectivity patterns. Refactored to use common utilities for
    improved maintainability and reduced code duplication.
    """
    
    def __init__(self, env_var: str = "MCP_SERVERS", mock: bool = False):
        """
        Initialize SimpleMCPClient with JSON configuration from environment.
        
        Args:
            env_var: Environment variable containing MCP server configuration
            mock: Whether to use mock connections for testing
        """
        # Initialize utilities
        self.config_manager = MCPConfigurationManager()
        self.connection_pool = MCPConnectionPool()
        self.error_handler = MCPErrorHandler(__name__)
        
        # Load and validate configuration
        try:
            self.config = self.config_manager.load_configuration(env_var)
        except Exception as e:
            raise MCPConfigurationError(f"Failed to load configuration: {e}", e)
        
        # Store configuration details for compatibility
        self.servers = self.config.get_server_endpoints()
        self.mock = mock
        
        logger.info(f"Initialized SimpleMCPClient with {len(self.servers)} servers: {list(self.servers.keys())}")
    
    @property
    def connections(self) -> Dict[str, Any]:
        """Get connection pool info for compatibility."""
        return self.connection_pool.get_connection_info()
    
    @property 
    def health(self) -> Dict[str, bool]:
        """Get health status for compatibility."""
        return self.connection_pool.get_health_status()
    
    @handle_mcp_errors("connect_all")
    async def connect_all(self):
        """Connect to all configured MCP servers using connection pool."""
        logger.info("Connecting to all configured MCP servers...")
        
        enabled_servers = self.config.get_enabled_servers()
        connection_tasks = []
        
        for capability, server_config in enabled_servers.items():
            task = asyncio.create_task(
                self.connection_pool.add_connection(
                    capability,
                    server_config.endpoint,
                    server_config.connection_type,
                    self.mock
                )
            )
            connection_tasks.append((capability, task))
        
        # Wait for all connections to complete
        results = await asyncio.gather(
            *[task for _, task in connection_tasks], 
            return_exceptions=True
        )
        
        # Process results
        successful_connections = 0
        for i, result in enumerate(results):
            capability = connection_tasks[i][0]
            if isinstance(result, Exception):
                logger.warning(f"Failed to connect to {capability}: {result}")
            elif result:
                logger.info(f"Successfully connected to {capability}")
                successful_connections += 1
            else:
                logger.warning(f"Connection to {capability} returned False")
        
        logger.info(f"Connected to {successful_connections}/{len(enabled_servers)} MCP servers")
    
    @handle_mcp_errors("query")
    async def query(self, request: str, capability: str = None) -> Any:
        """
        Send query to appropriate MCP server based on capability routing.
        
        Args:
            request: The query string to send
            capability: Optional explicit capability to target
            
        Returns:
            Query response from the MCP server
        """
        # Auto-detect capability from request if not specified
        if not capability:
            capability = self._detect_capability(request)
        
        logger.debug(f"Routing query to capability '{capability}': {request[:100]}...")
        
        try:
            # Use connection pool to send message
            return await self.connection_pool.send_message(capability, {"query": request})
        except KeyError:
            # Try fallback to any healthy server
            health_status = await self.health_check()
            healthy_capabilities = [cap for cap, healthy in health_status.items() if healthy]
            
            if not healthy_capabilities:
                raise MCPConnectionError("No healthy MCP servers available", "unknown")
            
            fallback_capability = healthy_capabilities[0]
            logger.info(f"Falling back to {fallback_capability} for query")
            return await self.connection_pool.send_message(fallback_capability, {"query": request})
    
    def _detect_capability(self, request: str) -> str:
        """
        Simple keyword-based capability detection.
        
        This method analyzes the request string for capability keywords
        and returns the appropriate server capability name.
        """
        request_lower = request.lower()
        
        # Check for capability keywords in request
        for capability in self.servers.keys():
            if capability.lower() in request_lower:
                return capability
        
        # Check for common Atlassian keywords
        if any(keyword in request_lower for keyword in ['jira', 'ticket', 'issue', 'project']):
            # Look for atlassian capability
            if 'atlassian' in self.servers:
                return 'atlassian'
        
        # Check for GitHub keywords
        if any(keyword in request_lower for keyword in ['github', 'repository', 'repo', 'commit']):
            # Look for github capability
            if 'github' in self.servers:
                return 'github'
        
        # Check for Confluence keywords
        if any(keyword in request_lower for keyword in ['confluence', 'wiki', 'document', 'page']):
            # Look for confluence capability
            if 'confluence' in self.servers:
                return 'confluence'
        
        # Default to first configured server
        return next(iter(self.servers.keys()))
    
    @handle_mcp_errors("health_check")
    async def health_check(self) -> Dict[str, bool]:
        """
        Perform health check on all configured servers using connection pool.
        
        Returns:
            Dictionary mapping capability names to health status
        """
        logger.debug("Performing health check on all servers")
        return await self.connection_pool.health_check()
    
    @handle_mcp_errors("disconnect_all")
    async def disconnect_all(self):
        """Disconnect from all MCP servers using connection pool."""
        logger.info("Disconnecting from all MCP servers")
        await self.connection_pool.close_all()
    
    def get_server_status(self) -> Dict[str, Dict[str, Any]]:
        """
        Get comprehensive status of all configured servers.
        
        Returns:
            Dictionary with server status information
        """
        connection_info = self.connection_pool.get_connection_info()
        health_status = self.connection_pool.get_health_status()
        
        status = {}
        
        for capability, endpoint in self.servers.items():
            connection_data = connection_info.get(capability, {})
            
            status[capability] = {
                "endpoint": endpoint,
                "connected": connection_data.get("connected", False),
                "healthy": health_status.get(capability, False),
                "connection_type": connection_data.get("type", "unknown")
            }
        
        return status