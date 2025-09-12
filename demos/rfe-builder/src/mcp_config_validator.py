#!/usr/bin/env python3
"""
MCP Configuration Validator for RFE Builder

Validates MCP server configuration before application startup.
"""

import json
import logging
import os
from typing import Any, Dict, Optional

from mcp_client_integration.common import (
    MCPConfigurationError,
    MCPConfigurationManager,
    MCPSecurityValidator,
)

logger = logging.getLogger(__name__)


class RFEBuilderMCPConfig:
    """MCP configuration validator for RFE Builder"""

    def __init__(self, production_mode: Optional[bool] = None):
        """
        Initialize MCP configuration for RFE Builder.

        Args:
            production_mode: Override production mode detection
        """
        # Auto-detect production mode from environment
        if production_mode is None:
            production_mode = (
                os.getenv("MCP_PRODUCTION_MODE", "false").lower() == "true"
            )

        self.production_mode = production_mode
        self.config_manager = MCPConfigurationManager(production_mode=production_mode)
        self.security_validator = MCPSecurityValidator(production_mode=production_mode)

        logger.info(
            f"MCP Configuration initialized (production_mode={production_mode})"
        )

    def validate_configuration(self) -> Dict[str, Any]:
        """
        Validate current MCP configuration.

        Returns:
            Dict with validation results and configuration details
        """
        try:
            # Load and validate configuration
            config = self.config_manager.load_configuration()

            # Get configuration summary
            summary = self.config_manager.get_configuration_summary(config)

            # Validate security
            servers = config.get_server_endpoints()
            security_result = self.security_validator.validate_configuration_security(
                servers
            )

            return {
                "valid": True,
                "production_mode": self.production_mode,
                "summary": summary,
                "security_validation": security_result.to_dict(),
                "servers": servers,
            }

        except MCPConfigurationError as e:
            logger.error(f"MCP configuration validation failed: {e}")
            return {
                "valid": False,
                "error": str(e),
                "production_mode": self.production_mode,
            }

    def get_mcp_client(self):
        """
        Get configured MCP client for RFE Builder.

        Returns:
            Configured SimpleMCPClient instance
        """
        from mcp_client_integration import SimpleMCPClient

        return SimpleMCPClient()


def validate_mcp_config_for_rfe_builder() -> Dict[str, Any]:
    """
    Validate MCP configuration for RFE Builder startup.

    Returns:
        Configuration validation results
    """
    validator = RFEBuilderMCPConfig()
    return validator.validate_configuration()


if __name__ == "__main__":
    # Command-line validation
    result = validate_mcp_config_for_rfe_builder()

    if result["valid"]:
        print("✅ MCP Configuration is valid")
        print(f"Production Mode: {result['production_mode']}")
        print(f"Servers: {list(result['servers'].keys())}")
    else:
        print("❌ MCP Configuration validation failed")
        print(f"Error: {result['error']}")
        exit(1)
