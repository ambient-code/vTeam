#!/usr/bin/env python3
"""
MCP Service Layer for RFE Builder

Provides high-level MCP operations for RFE Builder workflows.
"""

import asyncio
import logging
from datetime import datetime
from typing import Any, Dict, Optional

from mcp_client_integration import SimpleMCPClient
from mcp_client_integration.common import MCPConnectionError, MCPError

logger = logging.getLogger(__name__)


class RFEBuilderMCPService:
    """
    High-level MCP service for RFE Builder operations.

    This service provides RFE-specific operations using MCP servers.
    """

    def __init__(self, auto_connect: bool = True):
        """
        Initialize MCP service for RFE Builder.

        Args:
            auto_connect: Whether to automatically connect to MCP servers
        """
        self.client = SimpleMCPClient()
        self._connected = False
        self._connection_status: Dict[str, Any] = {}
        # Performance configurations
        self._connection_pool_size = 3
        self._request_timeout = 30.0

        if auto_connect:
            asyncio.create_task(self._initialize_connections())

    async def _initialize_connections(self) -> None:
        """Initialize connections to all configured MCP servers."""
        try:
            await self.client.connect_all()
            self._connected = True
            self._connection_status = await self.client.health_check()

            logger.info(
                f"MCP Service initialized. Healthy servers: {sum(self._connection_status.values())}"
            )

        except Exception as e:
            logger.error(f"Failed to initialize MCP connections: {e}")
            self._connected = False

    async def get_connection_status(self) -> Dict[str, Any]:
        """
        Get current MCP connection status.

        Returns:
            Dictionary with connection status information
        """
        if not self._connected:
            await self._initialize_connections()

        status = self.client.get_server_status()
        health = await self.client.health_check()

        return {
            "connected": self._connected,
            "servers": status,
            "health": health,
            "healthy_count": sum(health.values()),
            "total_count": len(health),
            "last_check": datetime.now().isoformat(),
        }

    async def search_jira_tickets(
        self,
        project_key: Optional[str] = None,
        query: Optional[str] = None,
        max_results: int = 10,
    ) -> Dict[str, Any]:
        """
        Search for JIRA tickets related to RFE context.

        Args:
            project_key: JIRA project key to search in
            query: Free-text search query
            max_results: Maximum number of results to return

        Returns:
            Dictionary with search results and metadata
        """
        # Input validation and sanitization
        validated_params = self._validate_input_params(
            query=query, max_results=max_results, project_key=project_key
        )

        try:
            # Build search query for Atlassian MCP server
            search_params = {
                "action": "search_tickets",
                "project": validated_params.get("project_key"),
                "query": validated_params.get("query"),
                "max_results": validated_params.get("max_results", 10),
            }

            # Remove None values
            search_params = {k: v for k, v in search_params.items() if v is not None}

            # Query Atlassian MCP server
            response = await self.client.query(
                f"Search JIRA tickets: {query or 'all tickets'}", capability="atlassian"
            )

            return {
                "success": True,
                "tickets": response.get("data", []),
                "query": search_params,
                "server": "atlassian",
                "timestamp": datetime.now().isoformat(),
            }

        except MCPConnectionError as e:
            logger.error(f"Failed to search JIRA tickets: {e}")
            return self._create_error_response(
                "jira_tickets", "Connection error", str(e), []
            )

        except MCPError as e:
            logger.error(f"MCP error searching JIRA tickets: {e}")
            return self._create_error_response("jira_tickets", "MCP error", str(e), [])

        except Exception as e:
            logger.error(f"Unexpected error searching JIRA tickets: {e}")
            return self._create_error_response(
                "jira_tickets", "Unexpected error", str(e), []
            )

    async def get_github_repository_info(
        self, repo_owner: str, repo_name: str
    ) -> Dict[str, Any]:
        """
        Get GitHub repository information for RFE context.

        Args:
            repo_owner: GitHub repository owner
            repo_name: GitHub repository name

        Returns:
            Dictionary with repository information
        """
        try:
            query = f"Get repository information for {repo_owner}/{repo_name}"

            response = await self.client.query(query, capability="github")

            return {
                "success": True,
                "repository": response.get("data", {}),
                "owner": repo_owner,
                "name": repo_name,
                "server": "github",
                "timestamp": datetime.now().isoformat(),
            }

        except MCPError as e:
            logger.error(f"Failed to get GitHub repository info: {e}")
            return self._create_error_response("github_repo", "MCP error", str(e), {})

        except Exception as e:
            logger.error(f"Unexpected error getting GitHub repository info: {e}")
            return self._create_error_response(
                "github_repo", "Unexpected error", str(e), {}
            )

    async def search_confluence_docs(
        self, search_query: str, space_key: Optional[str] = None, max_results: int = 5
    ) -> Dict[str, Any]:
        """
        Search Confluence documentation for RFE context.

        Args:
            search_query: Search query for Confluence
            space_key: Optional Confluence space to search in
            max_results: Maximum number of results

        Returns:
            Dictionary with search results
        """
        try:
            query_text = f"Search Confluence for: {search_query}"
            if space_key:
                query_text += f" in space {space_key}"

            response = await self.client.query(query_text, capability="confluence")

            return {
                "success": True,
                "documents": response.get("data", []),
                "query": search_query,
                "space": space_key,
                "server": "confluence",
                "timestamp": datetime.now().isoformat(),
            }

        except MCPError as e:
            logger.error(f"Failed to search Confluence: {e}")
            return self._create_error_response(
                "confluence_docs", "MCP error", str(e), []
            )

        except Exception as e:
            logger.error(f"Unexpected error searching Confluence: {e}")
            return self._create_error_response(
                "confluence_docs", "Unexpected error", str(e), []
            )

    async def get_contextual_data_for_rfe(
        self,
        rfe_title: str,
        rfe_description: str,
        project_context: Optional[Dict] = None,
    ) -> Dict[str, Any]:
        """
        Get contextual data from all MCP servers for an RFE.

        Args:
            rfe_title: Title of the RFE
            rfe_description: Description of the RFE
            project_context: Optional project context (repo, JIRA project, etc.)

        Returns:
            Aggregated contextual data from all available MCP servers
        """
        contextual_data = {
            "rfe_title": rfe_title,
            "rfe_description": rfe_description,
            "project_context": project_context or {},
            "data_sources": {},
            "summary": {},
            "timestamp": datetime.now().isoformat(),
        }

        # Get connection status
        status = await self.get_connection_status()
        healthy_servers = [
            server for server, healthy in status["health"].items() if healthy
        ]

        if not healthy_servers:
            logger.warning("No healthy MCP servers available for contextual data")
            contextual_data["summary"]["error"] = "No healthy MCP servers available"
            return contextual_data

        # Parallel data collection from available servers
        tasks = []

        # JIRA tickets if Atlassian is available
        if "atlassian" in healthy_servers:
            tasks.append(
                (
                    "jira_tickets",
                    self.search_jira_tickets(
                        query=f"{rfe_title} {rfe_description}"[
                            :100
                        ],  # Limit query length
                        max_results=5,
                    ),
                )
            )

        # GitHub repository info if GitHub is available and project context provided
        if "github" in healthy_servers and project_context:
            repo_owner = project_context.get("github_owner")
            repo_name = project_context.get("github_repo")
            if repo_owner and repo_name:
                tasks.append(
                    (
                        "github_repo",
                        self.get_github_repository_info(repo_owner, repo_name),
                    )
                )

        # Confluence documentation if Confluence is available
        if "confluence" in healthy_servers:
            tasks.append(
                (
                    "confluence_docs",
                    self.search_confluence_docs(
                        search_query=f"{rfe_title} {rfe_description}"[:100],
                        max_results=3,
                    ),
                )
            )

        # Execute all queries in parallel with timeout protection
        if tasks:
            try:
                # Use asyncio.wait_for to prevent hanging on slow MCP servers
                results = await asyncio.wait_for(
                    asyncio.gather(
                        *[task[1] for task in tasks], return_exceptions=True
                    ),
                    timeout=self._request_timeout,
                )

                for i, (data_type, result) in enumerate(
                    zip([task[0] for task in tasks], results)
                ):
                    if isinstance(result, Exception):
                        logger.error(f"Error fetching {data_type}: {result}")
                        contextual_data["data_sources"][data_type] = {
                            "success": False,
                            "error": str(result),
                            "error_type": type(result).__name__,
                        }
                    else:
                        contextual_data["data_sources"][data_type] = result

            except asyncio.TimeoutError:
                logger.warning(
                    f"MCP data collection timed out after {self._request_timeout}s"
                )
                contextual_data["data_sources"]["timeout_error"] = {
                    "success": False,
                    "error": f"Data collection timed out after {self._request_timeout} seconds",
                    "error_type": "TimeoutError",
                }

        # Generate summary
        successful_sources = [
            k for k, v in contextual_data["data_sources"].items() if v.get("success")
        ]
        contextual_data["summary"] = {
            "healthy_servers": healthy_servers,
            "successful_sources": successful_sources,
            "total_data_points": sum(
                len(
                    v.get(
                        "tickets",
                        v.get(
                            "documents",
                            v.get("repository", {}) and [v.get("repository")] or [],
                        ),
                    )
                )
                for v in contextual_data["data_sources"].values()
                if v.get("success")
            ),
        }

        return contextual_data

    async def close(self) -> None:
        """Close MCP connections and cleanup resources."""
        if self._connected:
            try:
                await asyncio.wait_for(
                    self.client.disconnect_all(), timeout=10.0  # Don't hang on cleanup
                )
            except asyncio.TimeoutError:
                logger.warning("MCP disconnect timed out - forcing cleanup")
            except Exception as e:
                logger.error(f"Error during MCP cleanup: {e}")
            finally:
                self._connected = False
                self._connection_status.clear()
                logger.info("MCP Service connections closed")

    async def __aenter__(self):
        """Async context manager entry."""
        if not self._connected:
            await self._initialize_connections()
        return self

    async def __aexit__(self, exc_type, exc_val, exc_tb):
        """Async context manager exit with cleanup."""
        await self.close()

    def _create_error_response(
        self, data_type: str, error_type: str, error_message: str, default_data: Any
    ) -> Dict[str, Any]:
        """Create standardized error response."""
        return {
            "success": False,
            "error": f"{error_type}: {error_message}",
            "error_type": error_type,
            "data_type": data_type,
            "timestamp": datetime.now().isoformat(),
            data_type.split("_")[
                -1
            ]: default_data,  # tickets, documents, repository, etc.
        }

    def _validate_input_params(self, **params) -> Dict[str, Any]:
        """Validate and sanitize input parameters."""
        validated = {}

        for key, value in params.items():
            if value is None:
                continue

            if key in ["query", "search_query"] and isinstance(value, str):
                # Limit query length for security and performance
                validated[key] = value[:1000] if len(value) > 1000 else value
            elif key in ["max_results"] and isinstance(value, int):
                # Limit result size for security and performance
                validated[key] = min(value, 100)
            else:
                validated[key] = value

        return validated


# Global MCP service instance
_mcp_service: Optional[RFEBuilderMCPService] = None


def get_mcp_service() -> RFEBuilderMCPService:
    """
    Get or create the global MCP service instance.

    Returns:
        RFEBuilderMCPService instance
    """
    global _mcp_service
    if _mcp_service is None:
        _mcp_service = RFEBuilderMCPService()
    return _mcp_service


async def initialize_mcp_service() -> RFEBuilderMCPService:
    """
    Initialize MCP service for the application.

    Returns:
        Initialized RFEBuilderMCPService
    """
    service = get_mcp_service()
    if not service._connected:
        await service._initialize_connections()
    return service
