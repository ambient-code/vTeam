#!/usr/bin/env python3
"""
MCPEndpointConnector - Endpoint Validation and Connectivity

This module provides endpoint validation and connectivity testing for MCP clients,
based on SPIKE-002 validated OpenShift service discovery patterns.

Refactored to use common utilities for reduced code duplication.
"""

import asyncio
import logging
import socket
import ssl
import time
from typing import Optional, Dict, Any
from urllib.parse import urlparse

from .common import (
    MCPEndpointValidator,
    ValidationResult,
    MCPErrorHandler,
    handle_mcp_errors,
    MCPConnectionError,
    MCPValidationError
)

# Configure logging
logger = logging.getLogger(__name__)


class MCPEndpointConnector:
    """
    MCP endpoint connector with validation based on SPIKE-002 findings.
    
    This class provides endpoint validation for both external routes and
    cluster-internal services, implementing patterns validated in SPIKE-002.
    
    Refactored to use common validation utilities for reduced code duplication.
    """
    
    def __init__(self, timeout_seconds: int = 30):
        """Initialize MCPEndpointConnector with common utilities.
        
        Args:
            timeout_seconds: Connection timeout in seconds
        """
        self.timeout_seconds = timeout_seconds
        self.validator = MCPEndpointValidator()
        self.error_handler = MCPErrorHandler(__name__)
        
        logger.debug(f"MCPEndpointConnector initialized with {timeout_seconds}s timeout")
    
    def validate_endpoint_config(self, endpoint: str) -> bool:
        """
        Validate endpoint configuration format using common validator.
        
        This method implements SPIKE-002 validated endpoint validation patterns
        for both external routes and cluster-internal services.
        
        Args:
            endpoint: The endpoint URL/address to validate
            
        Returns:
            True if endpoint format is valid, False otherwise
        """
        result = self.validator.validate_endpoint(endpoint)
        return result.valid
    
    def get_validation_result(self, endpoint: str) -> ValidationResult:
        """
        Get detailed validation result using common validator.
        
        Args:
            endpoint: The endpoint URL/address to validate
            
        Returns:
            ValidationResult with detailed validation information
        """
        return self.validator.validate_endpoint(endpoint)
    
    def _is_external_route(self, endpoint: str) -> bool:
        """Check if endpoint is an external route format."""
        return self.validator._is_external_route(endpoint)
    
    def _is_cluster_service(self, endpoint: str) -> bool:
        """Check if endpoint is a cluster service format."""
        return self.validator._is_cluster_service(endpoint)
    
    @handle_mcp_errors("test_connectivity")
    async def test_connectivity(self, endpoint: str) -> Dict[str, Any]:
        """
        Test connectivity to an endpoint with error handling.
        
        This method performs actual connectivity testing based on SPIKE-002
        validated patterns using common error handling utilities.
        
        Args:
            endpoint: The endpoint to test
            
        Returns:
            Dictionary with connectivity test results
        """
        result = {
            "endpoint": endpoint,
            "reachable": False,
            "response_time_ms": None,
            "error": None,
            "endpoint_type": None
        }
        
        # Use common validator
        validation_result = self.get_validation_result(endpoint)
        if not validation_result.valid:
            result["error"] = validation_result.error_message
            return result
        
        try:
            async with self.error_handler.handle_connection_errors("test_connectivity", endpoint):
                if self._is_external_route(endpoint):
                    result["endpoint_type"] = "external_route"
                    return await self._test_external_route_connectivity(endpoint, result)
                else:
                    result["endpoint_type"] = "cluster_service"
                    return await self._test_cluster_service_connectivity(endpoint, result)
                    
        except Exception as e:
            result["error"] = str(e)
            return result
    
    async def _test_external_route_connectivity(self, endpoint: str, result: Dict[str, Any]) -> Dict[str, Any]:
        """Test connectivity to external route."""
        try:
            parsed = urlparse(endpoint)
            host = parsed.hostname
            port = parsed.port or (443 if parsed.scheme == 'https' else 80)
            
            start_time = time.time()
            
            # Test TCP connectivity
            sock = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
            sock.settimeout(self.timeout_seconds)
            
            try:
                if parsed.scheme == 'https':
                    # Test SSL connectivity
                    context = ssl.create_default_context()
                    sock = context.wrap_socket(sock, server_hostname=host)
                
                sock.connect((host, port))
                
                end_time = time.time()
                result["reachable"] = True
                result["response_time_ms"] = int((end_time - start_time) * 1000)
                
            finally:
                sock.close()
                
        except socket.timeout:
            result["error"] = f"Connection timeout after {self.timeout_seconds}s"
        except socket.gaierror as e:
            result["error"] = f"DNS resolution failed: {e}"
        except Exception as e:
            result["error"] = f"Connection failed: {e}"
        
        return result
    
    async def _test_cluster_service_connectivity(self, endpoint: str, result: Dict[str, Any]) -> Dict[str, Any]:
        """Test connectivity to cluster service."""
        try:
            # Split endpoint and port
            if ':' in endpoint:
                service_part, port_part = endpoint.rsplit(':', 1)
                port = int(port_part)
            else:
                service_part = endpoint
                port = 80  # Default port
            
            start_time = time.time()
            
            # Test TCP connectivity to cluster service
            sock = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
            sock.settimeout(self.timeout_seconds)
            
            try:
                sock.connect((service_part, port))
                
                end_time = time.time()
                result["reachable"] = True
                result["response_time_ms"] = int((end_time - start_time) * 1000)
                
            finally:
                sock.close()
                
        except socket.timeout:
            result["error"] = f"Connection timeout after {self.timeout_seconds}s"
        except socket.gaierror as e:
            result["error"] = f"DNS resolution failed: {e}"
        except Exception as e:
            result["error"] = f"Connection failed: {e}"
        
        return result
    
    def get_endpoint_info(self, endpoint: str) -> Dict[str, Any]:
        """
        Get detailed information about an endpoint using common validator.
        
        Args:
            endpoint: The endpoint to analyze
            
        Returns:
            Dictionary with endpoint information
        """
        validation_result = self.get_validation_result(endpoint)
        
        info = {
            "endpoint": endpoint,
            "valid": validation_result.valid,
            "type": None,
            "parsed": None,
            "validation_details": validation_result.details if validation_result.details else {}
        }
        
        if not validation_result.valid:
            info["error"] = validation_result.error_message
            return info
        
        if self._is_external_route(endpoint):
            info["type"] = "external_route"
            parsed = urlparse(endpoint)
            info["parsed"] = {
                "scheme": parsed.scheme,
                "hostname": parsed.hostname,
                "port": parsed.port,
                "path": parsed.path
            }
        else:
            info["type"] = "cluster_service"
            if ':' in endpoint:
                service_part, port_part = endpoint.rsplit(':', 1)
                port = int(port_part)
            else:
                service_part = endpoint
                port = None
            
            parts = service_part.split('.')
            info["parsed"] = {
                "service": parts[0],
                "namespace": parts[1],
                "domain": '.'.join(parts[2:]),
                "port": port
            }
        
        return info