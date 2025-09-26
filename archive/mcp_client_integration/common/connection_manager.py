#!/usr/bin/env python3
"""
MCP Connection Management Utilities

This module provides standardized connection interfaces and factories for MCP clients,
eliminating code duplication and providing consistent connection lifecycle management.
"""

import asyncio
import logging
import ssl
from abc import ABC, abstractmethod
from typing import Dict, Any, Optional, Union
from urllib.parse import urlparse

# Configure logging
logger = logging.getLogger(__name__)


class MCPConnectionInterface(ABC):
    """
    Standard interface for all MCP connections.
    
    This abstract base class defines the contract that all MCP connection
    implementations must follow, ensuring consistency across different
    connection types (external routes, cluster services, etc.).
    """
    
    @abstractmethod
    async def send_message(self, message: Dict[str, Any]) -> Dict[str, Any]:
        """
        Send a message through the connection.
        
        Args:
            message: The message to send
            
        Returns:
            Response from the server
            
        Raises:
            ConnectionError: If the connection is not available
            TimeoutError: If the message times out
        """
        pass
    
    @abstractmethod
    async def close(self) -> None:
        """
        Close the connection and cleanup resources.
        
        This method should be idempotent and safe to call multiple times.
        """
        pass
    
    @property
    @abstractmethod
    def connected(self) -> bool:
        """
        Check if the connection is currently active.
        
        Returns:
            True if connected, False otherwise
        """
        pass
    
    @property
    @abstractmethod
    def endpoint(self) -> str:
        """
        Get the endpoint this connection is connected to.
        
        Returns:
            The endpoint URL or address
        """
        pass


class MockMCPConnection(MCPConnectionInterface):
    """
    Mock MCP connection for testing and development.
    
    This implementation provides a standardized mock connection that can be
    used across all MCP components for testing purposes, eliminating the
    duplicate MockConnection classes found in the original implementation.
    """
    
    def __init__(self, endpoint: str, simulate_failure: bool = False):
        """
        Initialize mock connection.
        
        Args:
            endpoint: The endpoint to simulate connection to
            simulate_failure: Whether to simulate connection failures
        """
        self._endpoint = endpoint
        self._connected = True
        self._simulate_failure = simulate_failure
        self._message_count = 0
        
        logger.debug(f"Created mock connection to {endpoint}")
    
    async def send_message(self, message: Dict[str, Any]) -> Dict[str, Any]:
        """Send message through mock connection."""
        if self._simulate_failure:
            raise ConnectionError("Simulated connection failure")
        
        if not self._connected:
            raise ConnectionError("Connection is not active")
        
        self._message_count += 1
        
        # Simulate message processing
        response = {
            "status": "ok",
            "endpoint": self._endpoint,
            "message_id": self._message_count,
            "echo": message,
            "timestamp": asyncio.get_event_loop().time()
        }
        
        logger.debug(f"Mock connection sent message {self._message_count} to {self._endpoint}")
        return response
    
    async def close(self) -> None:
        """Close mock connection."""
        if self._connected:
            self._connected = False
            logger.debug(f"Mock connection to {self._endpoint} closed")
    
    @property
    def connected(self) -> bool:
        """Check if mock connection is active."""
        return self._connected
    
    @property
    def endpoint(self) -> str:
        """Get mock connection endpoint."""
        return self._endpoint
    
    def set_failure_mode(self, simulate_failure: bool) -> None:
        """Set whether to simulate failures."""
        self._simulate_failure = simulate_failure


class ExternalRouteMCPConnection(MCPConnectionInterface):
    """
    MCP connection for external routes (HTTPS/SSE).
    
    This implementation handles connections to external MCP servers via
    HTTPS routes, typically used in OpenShift environments.
    """
    
    def __init__(self, endpoint: str, timeout: int = 30, verify_ssl: bool = True):
        """
        Initialize external route connection.
        
        Args:
            endpoint: The HTTPS endpoint URL
            timeout: Connection timeout in seconds
            verify_ssl: Whether to verify SSL certificates (should be True in production)
        """
        self._endpoint = endpoint
        self._timeout = timeout
        self._verify_ssl = verify_ssl
        self._connected = False
        self._session = None
        
        # Validate endpoint format
        parsed = urlparse(endpoint)
        if parsed.scheme not in ['http', 'https']:
            raise ValueError(f"Invalid endpoint scheme: {parsed.scheme}")
        
        # Warn about insecure configurations
        if parsed.scheme == 'http':
            logger.warning(f"Insecure HTTP connection to {endpoint} - consider using HTTPS")
        
        if not verify_ssl:
            logger.warning(f"SSL verification disabled for {endpoint} - this is insecure for production")
        
        # Create SSL context for secure connections
        if parsed.scheme == 'https':
            self._ssl_context = ssl.create_default_context()
            if not verify_ssl:
                self._ssl_context.check_hostname = False
                self._ssl_context.verify_mode = ssl.CERT_NONE
        else:
            self._ssl_context = None
        
        logger.debug(f"Created external route connection to {endpoint} (SSL verify: {verify_ssl})")
    
    async def connect(self) -> None:
        """Establish connection to external route."""
        try:
            # In a real implementation, this would establish the SSE connection
            # For now, we simulate a successful connection
            self._connected = True
            logger.info(f"Connected to external route: {self._endpoint}")
        except Exception as e:
            logger.error(f"Failed to connect to {self._endpoint}: {e}")
            raise ConnectionError(f"Failed to connect to {self._endpoint}: {e}")
    
    async def send_message(self, message: Dict[str, Any]) -> Dict[str, Any]:
        """Send message through external route connection."""
        if not self._connected:
            raise ConnectionError("Connection not established")
        
        try:
            # In a real implementation, this would send via HTTPS/SSE
            # For now, we simulate a successful message exchange
            response = {
                "status": "ok",
                "endpoint": self._endpoint,
                "type": "external_route",
                "message": message,
                "timestamp": asyncio.get_event_loop().time()
            }
            
            logger.debug(f"Sent message via external route to {self._endpoint}")
            return response
            
        except Exception as e:
            logger.error(f"Failed to send message to {self._endpoint}: {e}")
            raise ConnectionError(f"Message send failed: {e}")
    
    async def close(self) -> None:
        """Close external route connection."""
        if self._connected:
            self._connected = False
            if self._session:
                await self._session.close()
                self._session = None
            logger.debug(f"External route connection to {self._endpoint} closed")
    
    @property
    def connected(self) -> bool:
        """Check if external route connection is active."""
        return self._connected
    
    @property
    def endpoint(self) -> str:
        """Get external route endpoint."""
        return self._endpoint


class ClusterServiceMCPConnection(MCPConnectionInterface):
    """
    MCP connection for cluster-internal services.
    
    This implementation handles connections to MCP servers running as
    Kubernetes services within the same cluster.
    """
    
    def __init__(self, endpoint: str, timeout: int = 30):
        """
        Initialize cluster service connection.
        
        Args:
            endpoint: The cluster service endpoint
            timeout: Connection timeout in seconds
        """
        self._endpoint = endpoint
        self._timeout = timeout
        self._connected = False
        self._websocket = None
        
        # Parse cluster service format
        if '.svc.cluster.local' not in endpoint:
            raise ValueError(f"Invalid cluster service format: {endpoint}")
        
        logger.debug(f"Created cluster service connection to {endpoint}")
    
    async def connect(self) -> None:
        """Establish connection to cluster service."""
        try:
            # In a real implementation, this would establish WebSocket connection
            # For now, we simulate a successful connection
            self._connected = True
            logger.info(f"Connected to cluster service: {self._endpoint}")
        except Exception as e:
            logger.error(f"Failed to connect to {self._endpoint}: {e}")
            raise ConnectionError(f"Failed to connect to {self._endpoint}: {e}")
    
    async def send_message(self, message: Dict[str, Any]) -> Dict[str, Any]:
        """Send message through cluster service connection."""
        if not self._connected:
            raise ConnectionError("Connection not established")
        
        try:
            # In a real implementation, this would send via WebSocket
            # For now, we simulate a successful message exchange
            response = {
                "status": "ok",
                "endpoint": self._endpoint,
                "type": "cluster_service",
                "message": message,
                "timestamp": asyncio.get_event_loop().time()
            }
            
            logger.debug(f"Sent message via cluster service to {self._endpoint}")
            return response
            
        except Exception as e:
            logger.error(f"Failed to send message to {self._endpoint}: {e}")
            raise ConnectionError(f"Message send failed: {e}")
    
    async def close(self) -> None:
        """Close cluster service connection."""
        if self._connected:
            self._connected = False
            if self._websocket:
                await self._websocket.close()
                self._websocket = None
            logger.debug(f"Cluster service connection to {self._endpoint} closed")
    
    @property
    def connected(self) -> bool:
        """Check if cluster service connection is active."""
        return self._connected
    
    @property
    def endpoint(self) -> str:
        """Get cluster service endpoint."""
        return self._endpoint


class MCPConnectionFactory:
    """
    Factory for creating appropriate MCP connection types.
    
    This factory eliminates code duplication by providing a centralized
    way to create connections based on endpoint format and requirements.
    """
    
    @staticmethod
    def create_connection(
        endpoint: str, 
        connection_type: Optional[str] = None,
        timeout: int = 30,
        mock: bool = False,
        verify_ssl: bool = True
    ) -> MCPConnectionInterface:
        """
        Create appropriate connection type based on endpoint.
        
        Args:
            endpoint: The endpoint to connect to
            connection_type: Optional explicit connection type
            timeout: Connection timeout in seconds
            mock: Whether to create a mock connection
            verify_ssl: Whether to verify SSL certificates for HTTPS connections
            
        Returns:
            Appropriate connection implementation
            
        Raises:
            ValueError: If endpoint format is invalid
        """
        if mock:
            return MockMCPConnection(endpoint)
        
        # Auto-detect connection type if not specified
        if connection_type is None:
            if endpoint.startswith(('http://', 'https://')):
                connection_type = 'external_route'
            elif '.svc.cluster.local' in endpoint:
                connection_type = 'cluster_service'
            else:
                raise ValueError(f"Cannot determine connection type for endpoint: {endpoint}")
        
        # Create appropriate connection type
        if connection_type == 'external_route':
            return ExternalRouteMCPConnection(endpoint, timeout, verify_ssl)
        elif connection_type == 'cluster_service':
            return ClusterServiceMCPConnection(endpoint, timeout)
        elif connection_type == 'mock':
            return MockMCPConnection(endpoint)
        else:
            raise ValueError(f"Unsupported connection type: {connection_type}")
    
    @staticmethod
    def create_mock_connection(endpoint: str, simulate_failure: bool = False) -> MockMCPConnection:
        """
        Create a mock connection for testing.
        
        Args:
            endpoint: The endpoint to mock
            simulate_failure: Whether to simulate failures
            
        Returns:
            Mock connection instance
        """
        return MockMCPConnection(endpoint, simulate_failure)


class MCPConnectionPool:
    """
    Connection pool for managing multiple MCP connections.
    
    This utility provides centralized connection lifecycle management,
    eliminating the need for each component to manage connections individually.
    """
    
    def __init__(self, timeout: int = 30, max_connections: int = 10):
        """
        Initialize connection pool.
        
        Args:
            timeout: Default connection timeout
            max_connections: Maximum number of connections allowed
        """
        self._connections: Dict[str, MCPConnectionInterface] = {}
        self._health: Dict[str, bool] = {}
        self._timeout = timeout
        self._max_connections = max_connections
        self._connection_count = 0
        
        logger.debug(f"Initialized MCP connection pool (max_connections={max_connections})")
    
    async def add_connection(
        self, 
        capability: str, 
        endpoint: str, 
        connection_type: Optional[str] = None,
        mock: bool = False
    ) -> bool:
        """
        Add a connection to the pool.
        
        Args:
            capability: The capability name for this connection
            endpoint: The endpoint to connect to
            connection_type: Optional explicit connection type
            mock: Whether to create a mock connection
            
        Returns:
            True if connection was successfully added
        """
        # Check connection pool limits
        if self._connection_count >= self._max_connections:
            logger.error(f"Connection pool limit reached ({self._max_connections}). Cannot add '{capability}'")
            return False
        
        try:
            connection = MCPConnectionFactory.create_connection(
                endpoint, connection_type, self._timeout, mock
            )
            
            # Establish connection for non-mock connections
            if not mock and hasattr(connection, 'connect'):
                await connection.connect()
            
            self._connections[capability] = connection
            self._health[capability] = True
            self._connection_count += 1
            
            logger.info(f"Added connection for capability '{capability}' to {endpoint} ({self._connection_count}/{self._max_connections})")
            return True
            
        except Exception as e:
            logger.error(f"Failed to add connection for '{capability}': {e}")
            self._health[capability] = False
            return False
    
    async def send_message(
        self, 
        capability: str, 
        message: Dict[str, Any]
    ) -> Dict[str, Any]:
        """
        Send message through a specific capability connection.
        
        Args:
            capability: The capability to send message through
            message: The message to send
            
        Returns:
            Response from the server
            
        Raises:
            KeyError: If capability not found
            ConnectionError: If connection is not healthy
        """
        if capability not in self._connections:
            raise KeyError(f"No connection found for capability: {capability}")
        
        if not self._health.get(capability, False):
            raise ConnectionError(f"Connection for '{capability}' is not healthy")
        
        try:
            connection = self._connections[capability]
            return await connection.send_message(message)
        except Exception as e:
            # Mark connection as unhealthy on failure
            self._health[capability] = False
            logger.warning(f"Connection for '{capability}' marked unhealthy: {e}")
            raise
    
    async def health_check(self) -> Dict[str, bool]:
        """
        Check health of all connections.
        
        Returns:
            Dictionary mapping capability to health status
        """
        health_results = {}
        
        for capability, connection in self._connections.items():
            try:
                # Simple ping to check health
                await connection.send_message({"type": "ping"})
                health_results[capability] = True
            except Exception as e:
                logger.warning(f"Health check failed for '{capability}': {e}")
                health_results[capability] = False
        
        # Update internal health tracking
        self._health.update(health_results)
        return health_results
    
    async def close_all(self) -> None:
        """Close all connections in the pool."""
        for capability, connection in self._connections.items():
            try:
                await connection.close()
                logger.debug(f"Closed connection for capability '{capability}'")
            except Exception as e:
                logger.warning(f"Error closing connection for '{capability}': {e}")
        
        self._connections.clear()
        self._health.clear()
        self._connection_count = 0
        logger.info("All connections closed")
    
    def get_health_status(self) -> Dict[str, bool]:
        """Get current health status of all connections."""
        return self._health.copy()
    
    def get_connection_info(self) -> Dict[str, Dict[str, Any]]:
        """Get detailed information about all connections."""
        info = {}
        
        for capability, connection in self._connections.items():
            info[capability] = {
                "endpoint": connection.endpoint,
                "connected": connection.connected,
                "healthy": self._health.get(capability, False),
                "type": type(connection).__name__
            }
        
        return info