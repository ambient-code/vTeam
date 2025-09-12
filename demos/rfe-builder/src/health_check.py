#!/usr/bin/env python3
"""
Health Check Module for RFE Builder with MCP Integration

Provides health check endpoints for monitoring MCP connections.
"""

import asyncio
import logging
from datetime import datetime
from typing import Any, Dict

from .mcp_config_validator import validate_mcp_config_for_rfe_builder
from .services.mcp_service import get_mcp_service

logger = logging.getLogger(__name__)


class HealthChecker:
    """Health check utility for RFE Builder with MCP integration."""

    async def get_system_health(self) -> Dict[str, Any]:
        """
        Get comprehensive system health including MCP status.

        Returns:
            Dictionary with system health information
        """
        health_data = {
            "timestamp": datetime.now().isoformat(),
            "overall_status": "unknown",
            "components": {},
        }

        # Check MCP configuration
        try:
            config_result = validate_mcp_config_for_rfe_builder()
            health_data["components"]["mcp_config"] = {
                "status": "healthy" if config_result["valid"] else "unhealthy",
                "details": config_result,
            }
        except Exception as e:
            health_data["components"]["mcp_config"] = {
                "status": "error",
                "error": str(e),
            }

        # Check MCP service connections
        try:
            mcp_service = get_mcp_service()
            connection_status = await mcp_service.get_connection_status()

            healthy_servers = connection_status["healthy_count"]
            total_servers = connection_status["total_count"]

            if healthy_servers == total_servers and total_servers > 0:
                mcp_health = "healthy"
            elif healthy_servers > 0:
                mcp_health = "degraded"
            else:
                mcp_health = "unhealthy"

            health_data["components"]["mcp_connections"] = {
                "status": mcp_health,
                "healthy_servers": healthy_servers,
                "total_servers": total_servers,
                "details": connection_status,
            }

        except Exception as e:
            health_data["components"]["mcp_connections"] = {
                "status": "error",
                "error": str(e),
            }

        # Determine overall status
        component_statuses = [
            comp["status"] for comp in health_data["components"].values()
        ]

        if all(status == "healthy" for status in component_statuses):
            health_data["overall_status"] = "healthy"
        elif any(status == "healthy" for status in component_statuses):
            health_data["overall_status"] = "degraded"
        else:
            health_data["overall_status"] = "unhealthy"

        return health_data

    async def test_mcp_connectivity(self) -> Dict[str, Any]:
        """
        Test MCP connectivity with sample queries.

        Returns:
            Connectivity test results
        """
        test_results = {"timestamp": datetime.now().isoformat(), "tests": {}}

        try:
            mcp_service = get_mcp_service()

            # Test Atlassian connection
            try:
                jira_result = await mcp_service.search_jira_tickets(
                    query="test connectivity", max_results=1
                )
                test_results["tests"]["atlassian"] = {
                    "status": "success" if jira_result["success"] else "failed",
                    "details": jira_result,
                }
            except Exception as e:
                test_results["tests"]["atlassian"] = {
                    "status": "error",
                    "error": str(e),
                }

            # Test GitHub connection
            try:
                # Use a well-known public repository for testing
                github_result = await mcp_service.get_github_repository_info(
                    repo_owner="octocat", repo_name="Hello-World"
                )
                test_results["tests"]["github"] = {
                    "status": "success" if github_result["success"] else "failed",
                    "details": github_result,
                }
            except Exception as e:
                test_results["tests"]["github"] = {"status": "error", "error": str(e)}

            # Test Confluence connection
            try:
                confluence_result = await mcp_service.search_confluence_docs(
                    search_query="test", max_results=1
                )
                test_results["tests"]["confluence"] = {
                    "status": "success" if confluence_result["success"] else "failed",
                    "details": confluence_result,
                }
            except Exception as e:
                test_results["tests"]["confluence"] = {
                    "status": "error",
                    "error": str(e),
                }

        except Exception as e:
            test_results["error"] = str(e)

        return test_results


# Global health checker instance
_health_checker: HealthChecker = HealthChecker()


async def get_health() -> Dict[str, Any]:
    """Get system health status."""
    return await _health_checker.get_system_health()


async def test_connectivity() -> Dict[str, Any]:
    """Test MCP connectivity."""
    return await _health_checker.test_mcp_connectivity()
