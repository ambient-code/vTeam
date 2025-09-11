#!/usr/bin/env python3
"""
MCP Endpoint Validation Utilities

This module provides standardized validation utilities for MCP endpoints,
consolidating validation logic from across the codebase into a single,
reusable interface.
"""

import re
import logging
import json
from dataclasses import dataclass
from typing import Dict, Any, Optional, List
from urllib.parse import urlparse

# Configure logging
logger = logging.getLogger(__name__)


@dataclass
class ValidationResult:
    """
    Standardized validation result structure.
    
    This class provides a consistent way to return validation results
    across all validation operations.
    """
    valid: bool
    error_message: Optional[str] = None
    endpoint_type: Optional[str] = None
    parsed_info: Optional[Dict[str, Any]] = None
    details: Optional[Dict[str, Any]] = None
    
    def to_dict(self) -> Dict[str, Any]:
        """Convert validation result to dictionary."""
        return {
            "valid": self.valid,
            "error_message": self.error_message,
            "endpoint_type": self.endpoint_type,
            "parsed_info": self.parsed_info
        }


class MCPEndpointValidator:
    """
    Unified endpoint validation utility.
    
    This class consolidates all endpoint validation logic from the original
    implementation, providing a single interface for validating MCP endpoints
    in various formats.
    """
    
    def __init__(self):
        """Initialize validator with compiled regex patterns."""
        # Compiled regex patterns for performance
        self._external_route_pattern = re.compile(
            r'^https?://[a-zA-Z0-9\-\.]+\.[a-zA-Z]{2,}(?::\d+)?(?:/[a-zA-Z0-9\-\.\/]*)?(?:\?.*)?$'
        )
        
        self._cluster_service_pattern = re.compile(
            r'^[a-zA-Z0-9\-]+\.[a-zA-Z0-9\-]+\.svc\.cluster\.local(?::\d+)?$'
        )
        
        self._hostname_pattern = re.compile(
            r'^[a-zA-Z0-9]([a-zA-Z0-9\-]{0,61}[a-zA-Z0-9])?(\.[a-zA-Z0-9]([a-zA-Z0-9\-]{0,61}[a-zA-Z0-9])?)*$'
        )
        
        self._k8s_name_pattern = re.compile(
            r'^[a-z0-9]([a-z0-9\-]{0,61}[a-z0-9])?$'
        )
        
        logger.debug("MCPEndpointValidator initialized with compiled patterns")
    
    def validate_endpoint(self, endpoint: str) -> ValidationResult:
        """
        Validate any MCP endpoint format.
        
        Args:
            endpoint: The endpoint to validate
            
        Returns:
            ValidationResult with validation outcome and details
        """
        if not endpoint or not isinstance(endpoint, str):
            return ValidationResult(
                valid=False,
                error_message="Endpoint must be a non-empty string"
            )
        
        endpoint = endpoint.strip()
        
        # Check for external route format
        if self._is_external_route(endpoint):
            return self._validate_external_route(endpoint)
        
        # Check for cluster service format
        if self._is_cluster_service(endpoint):
            return self._validate_cluster_service(endpoint)
        
        return ValidationResult(
            valid=False,
            error_message=f"Endpoint format not recognized: {endpoint}"
        )
    
    def validate_configuration(self, config: Dict[str, str]) -> Dict[str, ValidationResult]:
        """
        Validate entire MCP server configuration.
        
        Args:
            config: Dictionary mapping capability names to endpoints
            
        Returns:
            Dictionary mapping capability names to validation results
        """
        results = {}
        
        for capability, endpoint in config.items():
            results[capability] = self.validate_endpoint(endpoint)
        
        return results
    
    def validate_configuration_dict(self, config: Dict[str, str]) -> ValidationResult:
        """
        Validate configuration dictionary as a whole.
        
        Args:
            config: Configuration dictionary to validate
            
        Returns:
            Overall validation result
        """
        if not isinstance(config, dict):
            return ValidationResult(
                valid=False,
                error_message="Configuration must be a dictionary"
            )
        
        if not config:
            return ValidationResult(
                valid=False,
                error_message="Configuration must contain at least one server definition"
            )
        
        # Validate each endpoint
        invalid_endpoints = []
        results = self.validate_configuration(config)
        
        for capability, result in results.items():
            if not result.valid:
                invalid_endpoints.append(f"{capability}: {result.error_message}")
        
        if invalid_endpoints:
            return ValidationResult(
                valid=False,
                error_message=f"Invalid endpoints found: {'; '.join(invalid_endpoints)}"
            )
        
        return ValidationResult(
            valid=True,
            parsed_info={"endpoint_count": len(config), "capabilities": list(config.keys())}
        )
    
    def _is_external_route(self, endpoint: str) -> bool:
        """Check if endpoint is an external route format."""
        return endpoint.startswith('http://') or endpoint.startswith('https://')
    
    def _is_cluster_service(self, endpoint: str) -> bool:
        """Check if endpoint is a cluster service format."""
        return '.svc.cluster.local' in endpoint
    
    def _validate_external_route(self, endpoint: str) -> ValidationResult:
        """
        Validate external route endpoint format.
        
        Args:
            endpoint: External route endpoint to validate
            
        Returns:
            ValidationResult with validation outcome
        """
        try:
            parsed = urlparse(endpoint)
            
            # Must have scheme
            if not parsed.scheme or parsed.scheme not in ['http', 'https']:
                return ValidationResult(
                    valid=False,
                    error_message=f"Invalid scheme: {parsed.scheme}. Must be http or https"
                )
            
            # Must have hostname
            if not parsed.hostname:
                return ValidationResult(
                    valid=False,
                    error_message="Missing hostname in URL"
                )
            
            # Validate hostname format
            if not self._is_valid_hostname(parsed.hostname):
                return ValidationResult(
                    valid=False,
                    error_message=f"Invalid hostname format: {parsed.hostname}"
                )
            
            # Validate port if specified
            if parsed.port is not None:
                if not (1 <= parsed.port <= 65535):
                    return ValidationResult(
                        valid=False,
                        error_message=f"Invalid port number: {parsed.port}"
                    )
            
            # Path validation (optional)
            if parsed.path and not self._is_valid_path(parsed.path):
                return ValidationResult(
                    valid=False,
                    error_message=f"Invalid path format: {parsed.path}"
                )
            
            return ValidationResult(
                valid=True,
                endpoint_type="external_route",
                parsed_info={
                    "scheme": parsed.scheme,
                    "hostname": parsed.hostname,
                    "port": parsed.port,
                    "path": parsed.path
                }
            )
            
        except Exception as e:
            return ValidationResult(
                valid=False,
                error_message=f"URL parsing error: {e}"
            )
    
    def _validate_cluster_service(self, endpoint: str) -> ValidationResult:
        """
        Validate cluster service endpoint format.
        
        Args:
            endpoint: Cluster service endpoint to validate
            
        Returns:
            ValidationResult with validation outcome
        """
        try:
            # Split endpoint and port if present
            port = None
            if ':' in endpoint:
                service_part, port_part = endpoint.rsplit(':', 1)
                try:
                    port = int(port_part)
                    if not (1 <= port <= 65535):
                        return ValidationResult(
                            valid=False,
                            error_message=f"Invalid port number: {port}"
                        )
                except ValueError:
                    return ValidationResult(
                        valid=False,
                        error_message=f"Invalid port format: {port_part}"
                    )
            else:
                service_part = endpoint
            
            # Validate service name format: service.namespace.svc.cluster.local
            parts = service_part.split('.')
            if len(parts) < 5:  # minimum: service.namespace.svc.cluster.local
                return ValidationResult(
                    valid=False,
                    error_message="Incomplete cluster service format. Expected: service.namespace.svc.cluster.local"
                )
            
            if parts[-3:] != ['svc', 'cluster', 'local']:
                return ValidationResult(
                    valid=False,
                    error_message="Invalid cluster service domain. Must end with .svc.cluster.local"
                )
            
            # Validate service and namespace names
            service_name = parts[0]
            namespace = parts[1]
            
            if not self._is_valid_k8s_name(service_name):
                return ValidationResult(
                    valid=False,
                    error_message=f"Invalid service name format: {service_name}"
                )
            
            if not self._is_valid_k8s_name(namespace):
                return ValidationResult(
                    valid=False,
                    error_message=f"Invalid namespace format: {namespace}"
                )
            
            return ValidationResult(
                valid=True,
                endpoint_type="cluster_service",
                parsed_info={
                    "service": service_name,
                    "namespace": namespace,
                    "domain": '.'.join(parts[2:]),
                    "port": port
                }
            )
            
        except Exception as e:
            return ValidationResult(
                valid=False,
                error_message=f"Cluster service parsing error: {e}"
            )
    
    def _is_valid_hostname(self, hostname: str) -> bool:
        """
        Validate hostname format.
        
        Args:
            hostname: Hostname to validate
            
        Returns:
            True if valid, False otherwise
        """
        if not hostname or len(hostname) > 253:
            return False
        
        return bool(self._hostname_pattern.match(hostname))
    
    def _is_valid_path(self, path: str) -> bool:
        """
        Validate URL path format.
        
        Args:
            path: URL path to validate
            
        Returns:
            True if valid, False otherwise
        """
        if not path:
            return True
        
        # Basic path validation - allow alphanumeric, hyphens, slashes, dots
        path_pattern = re.compile(r'^[a-zA-Z0-9\-\./]*$')
        return bool(path_pattern.match(path))
    
    def _is_valid_k8s_name(self, name: str) -> bool:
        """
        Validate Kubernetes resource name format.
        
        Args:
            name: Kubernetes name to validate
            
        Returns:
            True if valid, False otherwise
        """
        if not name or len(name) > 63:
            return False
        
        return bool(self._k8s_name_pattern.match(name))
    
    def get_endpoint_info(self, endpoint: str) -> Dict[str, Any]:
        """
        Get detailed information about an endpoint.
        
        Args:
            endpoint: The endpoint to analyze
            
        Returns:
            Dictionary with endpoint information
        """
        result = self.validate_endpoint(endpoint)
        
        info = {
            "endpoint": endpoint,
            "valid": result.valid,
            "type": result.endpoint_type,
            "parsed": result.parsed_info
        }
        
        if not result.valid:
            info["error"] = result.error_message
        
        return info


class MCPConfigurationValidator:
    """
    Configuration validation utility for MCP servers.
    
    This class provides high-level configuration validation,
    combining endpoint validation with configuration format validation.
    """
    
    def __init__(self):
        """Initialize configuration validator."""
        self.endpoint_validator = MCPEndpointValidator()
        logger.debug("MCPConfigurationValidator initialized")
    
    def validate_json_config(self, json_data: Any) -> ValidationResult:
        """
        Validate JSON configuration data.
        
        Args:
            json_data: Parsed JSON data to validate
            
        Returns:
            ValidationResult with validation outcome
        """
        # Check if it's a dictionary
        if not isinstance(json_data, dict):
            return ValidationResult(
                valid=False,
                error_message="MCP_SERVERS must be a JSON object with server definitions"
            )
        
        # Check if it's empty
        if not json_data:
            return ValidationResult(
                valid=False,
                error_message="MCP_SERVERS must contain at least one server definition"
            )
        
        # Validate each server configuration
        return self.endpoint_validator.validate_configuration_dict(json_data)
    
    def validate_environment_config(self, env_value: str) -> ValidationResult:
        """
        Validate environment variable configuration.
        
        Args:
            env_value: Environment variable value to validate
            
        Returns:
            ValidationResult with validation outcome
        """
        if not env_value or not isinstance(env_value, str):
            return ValidationResult(
                valid=False,
                error_message="Environment variable must be a non-empty string"
            )
        
        try:
            import json
            json_data = json.loads(env_value)
            return self.validate_json_config(json_data)
        except json.JSONDecodeError as e:
            return ValidationResult(
                valid=False,
                error_message=f"Invalid JSON in MCP_SERVERS environment variable: {e}"
            )


class MCPSecurityValidator:
    """
    Security-focused validation for MCP configurations.
    
    This validator implements additional security checks to prevent
    common security vulnerabilities in MCP configurations.
    """
    
    # Security constraints
    MAX_CONFIG_SIZE = 50 * 1024  # 50KB max config size
    MAX_ENDPOINTS = 50  # Maximum number of endpoints
    MAX_TIMEOUT = 300  # Maximum timeout in seconds
    MIN_TIMEOUT = 1   # Minimum timeout in seconds
    ALLOWED_SCHEMES = {'https', 'http'}  # Allow http for localhost/testing
    BLOCKED_HOSTS = {'localhost', '127.0.0.1', '0.0.0.0', '::1'}  # Block local hosts in production
    
    def __init__(self, production_mode: bool = False):
        """
        Initialize security validator.
        
        Args:
            production_mode: If True, applies stricter security checks
        """
        self.production_mode = production_mode
        self.base_validator = MCPEndpointValidator()
        
        # In production mode, only allow HTTPS
        if production_mode:
            self.ALLOWED_SCHEMES = {'https'}
        
        logger.debug(f"MCPSecurityValidator initialized (production_mode={production_mode})")
    
    def validate_configuration_security(self, config_data: Any) -> ValidationResult:
        """
        Comprehensive security validation of MCP configuration.
        
        Args:
            config_data: Configuration data to validate
            
        Returns:
            ValidationResult with security validation outcome
        """
        # Check configuration size
        try:
            config_str = json.dumps(config_data) if not isinstance(config_data, str) else config_data
            if len(config_str.encode('utf-8')) > self.MAX_CONFIG_SIZE:
                return ValidationResult(
                    valid=False,
                    error_message=f"Configuration exceeds maximum size of {self.MAX_CONFIG_SIZE} bytes"
                )
        except Exception as e:
            return ValidationResult(
                valid=False,
                error_message=f"Failed to serialize configuration for size check: {e}"
            )
        
        # Validate JSON structure for injection attempts
        if isinstance(config_data, str):
            try:
                config_data = json.loads(config_data)
            except json.JSONDecodeError as e:
                return ValidationResult(
                    valid=False,
                    error_message=f"Invalid JSON configuration: {e}"
                )
        
        # Check if it's a valid dictionary
        if not isinstance(config_data, dict):
            return ValidationResult(
                valid=False,
                error_message="Configuration must be a JSON object"
            )
        
        # Check number of endpoints
        if len(config_data) > self.MAX_ENDPOINTS:
            return ValidationResult(
                valid=False,
                error_message=f"Configuration exceeds maximum of {self.MAX_ENDPOINTS} endpoints"
            )
        
        # Validate each endpoint for security issues
        for capability, endpoint_config in config_data.items():
            # Validate capability name
            security_result = self._validate_capability_name(capability)
            if not security_result.valid:
                return security_result
            
            # Handle both string and dict endpoint configurations
            if isinstance(endpoint_config, str):
                endpoint = endpoint_config
                timeout = 30  # default
            elif isinstance(endpoint_config, dict):
                endpoint = endpoint_config.get("endpoint")
                timeout = endpoint_config.get("timeout", 30)
                
                # Validate timeout bounds
                if not isinstance(timeout, (int, float)) or timeout < self.MIN_TIMEOUT or timeout > self.MAX_TIMEOUT:
                    return ValidationResult(
                        valid=False,
                        error_message=f"Timeout for '{capability}' must be between {self.MIN_TIMEOUT} and {self.MAX_TIMEOUT} seconds"
                    )
            else:
                return ValidationResult(
                    valid=False,
                    error_message=f"Invalid endpoint configuration for '{capability}': must be string or object"
                )
            
            # Validate endpoint security
            security_result = self._validate_endpoint_security(capability, endpoint)
            if not security_result.valid:
                return security_result
        
        return ValidationResult(
            valid=True,
            parsed_info={
                "endpoints_validated": len(config_data),
                "security_checks_passed": True,
                "production_mode": self.production_mode
            }
        )
    
    def _validate_capability_name(self, capability: str) -> ValidationResult:
        """
        Validate capability name for security issues.
        
        Args:
            capability: Capability name to validate
            
        Returns:
            ValidationResult
        """
        # Check for dangerous characters
        if not isinstance(capability, str):
            return ValidationResult(
                valid=False,
                error_message="Capability name must be a string"
            )
        
        # Check length
        if len(capability) > 100:
            return ValidationResult(
                valid=False,
                error_message="Capability name too long (max 100 characters)"
            )
        
        # Check for injection patterns
        dangerous_patterns = [
            r'[<>"\'\`]',  # HTML/script injection
            r'[;|&]',      # Command injection
            r'\$\{',       # Variable expansion
            r'\.\./',      # Path traversal
            r'__.*__',     # Python internals
        ]
        
        for pattern in dangerous_patterns:
            if re.search(pattern, capability):
                return ValidationResult(
                    valid=False,
                    error_message=f"Capability name contains potentially dangerous characters: {capability}"
                )
        
        return ValidationResult(valid=True)
    
    def _validate_endpoint_security(self, capability: str, endpoint: str) -> ValidationResult:
        """
        Validate endpoint URL for security issues.
        
        Args:
            capability: Capability name
            endpoint: Endpoint URL to validate
            
        Returns:
            ValidationResult
        """
        if not isinstance(endpoint, str):
            return ValidationResult(
                valid=False,
                error_message=f"Endpoint for '{capability}' must be a string"
            )
        
        # Parse URL - handle cluster service format which doesn't have scheme
        try:
            # Check if this is a cluster service format first
            if '.svc.cluster.local' in endpoint and not endpoint.startswith(('http://', 'https://')):
                # This is a cluster service, skip URL parsing for scheme validation
                parsed = None
            else:
                parsed = urlparse(endpoint)
        except Exception as e:
            return ValidationResult(
                valid=False,
                error_message=f"Invalid URL format for '{capability}': {e}"
            )
        
        # Check scheme only for URLs that have schemes
        if parsed and parsed.scheme and parsed.scheme.lower() not in self.ALLOWED_SCHEMES:
            return ValidationResult(
                valid=False,
                error_message=f"Unsupported URL scheme '{parsed.scheme}' for '{capability}'. Allowed: {', '.join(self.ALLOWED_SCHEMES)}"
            )
        
        # In production mode, check for blocked hosts (only for URLs with schemes)
        if self.production_mode and parsed and parsed.hostname:
            if parsed.hostname.lower() in self.BLOCKED_HOSTS:
                return ValidationResult(
                    valid=False,
                    error_message=f"Localhost/loopback addresses not allowed in production mode: {parsed.hostname}"
                )
            
            # Check for private IP ranges in production
            if self._is_private_ip(parsed.hostname):
                return ValidationResult(
                    valid=False,
                    error_message=f"Private IP addresses not recommended in production mode: {parsed.hostname}"
                )
        
        # Check for suspicious ports (only for URLs with schemes)
        if parsed and parsed.port and parsed.port in [22, 23, 25, 110, 143, 993, 995]:  # Common non-HTTP ports
            return ValidationResult(
                valid=False,
                error_message=f"Suspicious port {parsed.port} for HTTP endpoint '{capability}'"
            )
        
        # Basic endpoint validation
        return self.base_validator.validate_endpoint(endpoint)
    
    def _is_private_ip(self, hostname: str) -> bool:
        """
        Check if hostname is a private IP address.
        
        Args:
            hostname: Hostname to check
            
        Returns:
            True if hostname appears to be a private IP
        """
        # Basic private IP pattern matching
        private_patterns = [
            r'^10\.',          # 10.0.0.0/8
            r'^172\.(1[6-9]|2[0-9]|3[01])\.',  # 172.16.0.0/12
            r'^192\.168\.',    # 192.168.0.0/16
        ]
        
        for pattern in private_patterns:
            if re.match(pattern, hostname):
                return True
        
        return False