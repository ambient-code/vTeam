#!/usr/bin/env python3
"""
MCP Configuration Management Utilities

This module provides standardized configuration management for MCP clients,
consolidating configuration loading and validation logic from across the codebase.
"""

import json
import os
import logging
from typing import Dict, Any, Optional, List, Union
from dataclasses import dataclass

from .validation import MCPConfigurationValidator, MCPSecurityValidator, ValidationResult
from .error_handler import MCPConfigurationError, handle_mcp_errors

# Configure logging
logger = logging.getLogger(__name__)


@dataclass
class MCPServerConfig:
    """Configuration for a single MCP server."""
    capability: str
    endpoint: str
    timeout: int = 30
    connection_type: Optional[str] = None
    enabled: bool = True
    metadata: Optional[Dict[str, Any]] = None
    
    def to_dict(self) -> Dict[str, Any]:
        """Convert to dictionary format."""
        return {
            "capability": self.capability,
            "endpoint": self.endpoint,
            "timeout": self.timeout,
            "connection_type": self.connection_type,
            "enabled": self.enabled,
            "metadata": self.metadata or {}
        }


@dataclass
class MCPConfiguration:
    """Complete MCP configuration with multiple servers."""
    servers: Dict[str, MCPServerConfig]
    default_timeout: int = 30
    health_check_interval: int = 300  # 5 minutes
    max_retries: int = 3
    metadata: Optional[Dict[str, Any]] = None
    
    def get_enabled_servers(self) -> Dict[str, MCPServerConfig]:
        """Get only enabled servers."""
        return {
            capability: config 
            for capability, config in self.servers.items() 
            if config.enabled
        }
    
    def get_server_endpoints(self) -> Dict[str, str]:
        """Get mapping of capability to endpoint for enabled servers."""
        return {
            capability: config.endpoint
            for capability, config in self.get_enabled_servers().items()
        }
    
    def to_dict(self) -> Dict[str, Any]:
        """Convert to dictionary format."""
        return {
            "servers": {
                capability: config.to_dict()
                for capability, config in self.servers.items()
            },
            "default_timeout": self.default_timeout,
            "health_check_interval": self.health_check_interval,
            "max_retries": self.max_retries,
            "metadata": self.metadata or {}
        }


class MCPConfigurationManager:
    """
    Unified configuration management for MCP clients.
    
    This class consolidates configuration loading, validation, and management
    logic, providing a single interface for all MCP configuration operations.
    """
    
    def __init__(self, default_timeout: int = 30, production_mode: bool = False):
        """
        Initialize configuration manager.
        
        Args:
            default_timeout: Default timeout for connections
            production_mode: Whether to use production-grade security validation
        """
        self.default_timeout = default_timeout
        self.production_mode = production_mode
        self.validator = MCPConfigurationValidator()
        self.security_validator = MCPSecurityValidator(production_mode)
        self._cached_config: Optional[MCPConfiguration] = None
        
        logger.debug(f"MCPConfigurationManager initialized (production_mode={production_mode})")
    
    @handle_mcp_errors("load_configuration")
    def load_configuration(self, env_var: str = "MCP_SERVERS") -> MCPConfiguration:
        """
        Load and validate MCP server configuration from environment.
        
        Args:
            env_var: Environment variable name containing configuration
            
        Returns:
            Validated MCP configuration
            
        Raises:
            MCPConfigurationError: If configuration is invalid
        """
        # Get configuration from environment
        env_value = os.getenv(env_var)
        
        if not env_value:
            # Return default configuration
            logger.info(f"No {env_var} found, using default configuration")
            return self._create_default_configuration()
        
        # Validate environment variable format
        validation_result = self.validator.validate_environment_config(env_value)
        
        if not validation_result.valid:
            raise MCPConfigurationError(validation_result.error_message)
        
        # Security validation
        security_result = self.security_validator.validate_configuration_security(env_value)
        
        if not security_result.valid:
            raise MCPConfigurationError(f"Security validation failed: {security_result.error_message}")
        
        # Parse JSON configuration (async-safe)
        try:
            # Use json.loads which is CPU-bound but fast for config sizes
            # For very large configs, could use asyncio.to_thread in future
            config_data = json.loads(env_value)
        except json.JSONDecodeError as e:
            raise MCPConfigurationError(f"Invalid JSON in {env_var}: {e}")
        
        # Convert to MCPConfiguration
        config = self._parse_configuration_dict(config_data)
        
        # Cache the configuration
        self._cached_config = config
        
        logger.info(f"Loaded configuration with {len(config.servers)} servers: {list(config.servers.keys())}")
        return config
    
    def _create_default_configuration(self) -> MCPConfiguration:
        """Create default configuration with a single server."""
        default_server = MCPServerConfig(
            capability="default",
            endpoint="https://mcp-server/sse",
            timeout=self.default_timeout
        )
        
        return MCPConfiguration(
            servers={"default": default_server},
            default_timeout=self.default_timeout
        )
    
    def _parse_configuration_dict(self, config_data: Dict[str, Any]) -> MCPConfiguration:
        """
        Parse configuration dictionary into MCPConfiguration.
        
        Args:
            config_data: Raw configuration dictionary
            
        Returns:
            Parsed MCPConfiguration
        """
        servers = {}
        
        for capability, endpoint in config_data.items():
            if isinstance(endpoint, str):
                # Simple endpoint string
                servers[capability] = MCPServerConfig(
                    capability=capability,
                    endpoint=endpoint,
                    timeout=self.default_timeout
                )
            elif isinstance(endpoint, dict):
                # Complex endpoint configuration
                servers[capability] = MCPServerConfig(
                    capability=capability,
                    endpoint=endpoint["endpoint"],
                    timeout=endpoint.get("timeout", self.default_timeout),
                    connection_type=endpoint.get("connection_type"),
                    enabled=endpoint.get("enabled", True),
                    metadata=endpoint.get("metadata")
                )
            else:
                raise MCPConfigurationError(
                    f"Invalid endpoint configuration for '{capability}': {endpoint}"
                )
        
        return MCPConfiguration(
            servers=servers,
            default_timeout=self.default_timeout
        )
    
    @handle_mcp_errors("validate_configuration")
    def validate_configuration(self, config: MCPConfiguration) -> ValidationResult:
        """
        Validate an MCPConfiguration object.
        
        Args:
            config: Configuration to validate
            
        Returns:
            ValidationResult with validation outcome
        """
        # Validate server endpoints
        endpoint_map = config.get_server_endpoints()
        return self.validator.validate_configuration_dict(endpoint_map)
    
    def get_cached_configuration(self) -> Optional[MCPConfiguration]:
        """Get cached configuration if available."""
        return self._cached_config
    
    def reload_configuration(self, env_var: str = "MCP_SERVERS") -> MCPConfiguration:
        """
        Reload configuration from environment.
        
        Args:
            env_var: Environment variable name containing configuration
            
        Returns:
            Reloaded configuration
        """
        self._cached_config = None
        return self.load_configuration(env_var)
    
    @handle_mcp_errors("create_configuration_from_dict")
    def create_configuration_from_dict(self, config_dict: Dict[str, Any]) -> MCPConfiguration:
        """
        Create configuration from dictionary.
        
        Args:
            config_dict: Configuration dictionary
            
        Returns:
            Created MCPConfiguration
            
        Raises:
            MCPConfigurationError: If configuration is invalid
        """
        # Validate the dictionary
        validation_result = self.validator.validate_json_config(config_dict)
        
        if not validation_result.valid:
            raise MCPConfigurationError(validation_result.error_message)
        
        # Parse and return configuration
        return self._parse_configuration_dict(config_dict)
    
    @handle_mcp_errors("save_configuration")
    def save_configuration_to_env(self, config: MCPConfiguration, env_var: str = "MCP_SERVERS") -> None:
        """
        Save configuration to environment variable format.
        
        Args:
            config: Configuration to save
            env_var: Environment variable name to save to
        """
        # Convert to simple endpoint mapping for environment storage
        endpoint_map = config.get_server_endpoints()
        
        # Convert to JSON string
        config_json = json.dumps(endpoint_map, indent=2)
        
        # Set environment variable (for current process)
        os.environ[env_var] = config_json
        
        logger.info(f"Configuration saved to {env_var}")
    
    def create_kubernetes_configmap_data(self, config: MCPConfiguration) -> Dict[str, str]:
        """
        Create Kubernetes ConfigMap data from configuration.
        
        Args:
            config: Configuration to convert
            
        Returns:
            Dictionary suitable for Kubernetes ConfigMap data
        """
        endpoint_map = config.get_server_endpoints()
        
        return {
            "MCP_SERVERS": json.dumps(endpoint_map, indent=2),
            "MCP_DEFAULT_TIMEOUT": str(config.default_timeout),
            "MCP_HEALTH_CHECK_INTERVAL": str(config.health_check_interval),
            "MCP_MAX_RETRIES": str(config.max_retries)
        }
    
    def get_configuration_summary(self, config: Optional[MCPConfiguration] = None) -> Dict[str, Any]:
        """
        Get configuration summary for logging/debugging.
        
        Args:
            config: Optional configuration, uses cached if not provided
            
        Returns:
            Configuration summary dictionary
        """
        if config is None:
            config = self._cached_config
        
        if config is None:
            return {"status": "no_configuration_loaded"}
        
        enabled_servers = config.get_enabled_servers()
        
        summary = {
            "total_servers": len(config.servers),
            "enabled_servers": len(enabled_servers),
            "capabilities": list(enabled_servers.keys()),
            "default_timeout": config.default_timeout,
            "health_check_interval": config.health_check_interval,
            "server_details": {}
        }
        
        for capability, server_config in enabled_servers.items():
            summary["server_details"][capability] = {
                "endpoint": server_config.endpoint,
                "timeout": server_config.timeout,
                "connection_type": server_config.connection_type or "auto-detect"
            }
        
        return summary


# Convenience functions for common operations
def load_mcp_configuration(env_var: str = "MCP_SERVERS") -> MCPConfiguration:
    """
    Convenience function to load MCP configuration.
    
    Args:
        env_var: Environment variable name containing configuration
        
    Returns:
        Loaded MCPConfiguration
    """
    manager = MCPConfigurationManager()
    return manager.load_configuration(env_var)


def create_simple_configuration(servers: Dict[str, str]) -> MCPConfiguration:
    """
    Convenience function to create simple configuration.
    
    Args:
        servers: Dictionary mapping capability to endpoint
        
    Returns:
        Created MCPConfiguration
    """
    manager = MCPConfigurationManager()
    return manager.create_configuration_from_dict(servers)


def validate_mcp_configuration_dict(config_dict: Dict[str, Any]) -> ValidationResult:
    """
    Convenience function to validate configuration dictionary.
    
    Args:
        config_dict: Configuration dictionary to validate
        
    Returns:
        ValidationResult
    """
    validator = MCPConfigurationValidator()
    return validator.validate_json_config(config_dict)