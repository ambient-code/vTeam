#!/usr/bin/env python3
"""
MCP CLI Commands for RFE Builder

Provides command-line interface for MCP operations.
"""

import asyncio
import json
from typing import Optional

import click

from .health_check import get_health, test_connectivity
from .mcp_config_validator import validate_mcp_config_for_rfe_builder
from .services.mcp_service import initialize_mcp_service


@click.group()
def mcp():
    """MCP (Model Context Protocol) management commands."""
    pass


@mcp.command()
def validate():
    """Validate MCP configuration."""
    click.echo("Validating MCP configuration...")

    result = validate_mcp_config_for_rfe_builder()

    if result["valid"]:
        click.echo("✅ MCP configuration is valid")
        click.echo(f"Production mode: {result['production_mode']}")
        click.echo(f"Configured servers: {list(result['servers'].keys())}")
    else:
        click.echo("❌ MCP configuration validation failed")
        click.echo(f"Error: {result['error']}")


@mcp.command()
def health():
    """Check MCP system health."""
    click.echo("Checking MCP system health...")

    async def _check_health():
        return await get_health()

    result = asyncio.run(_check_health())

    click.echo(f"Overall status: {result['overall_status']}")

    for component, details in result["components"].items():
        status_icon = {
            "healthy": "✅",
            "degraded": "⚠️",
            "unhealthy": "❌",
            "error": "💥",
        }
        click.echo(
            f"{status_icon.get(details['status'], '❓')} {component}: {details['status']}"
        )


@mcp.command()
def test():
    """Test MCP connectivity with sample queries."""
    click.echo("Testing MCP connectivity...")

    async def _test_connectivity():
        return await test_connectivity()

    result = asyncio.run(_test_connectivity())

    if "error" in result:
        click.echo(f"❌ Test failed: {result['error']}")
        return

    for server, test_result in result["tests"].items():
        status_icon = {"success": "✅", "failed": "❌", "error": "💥"}
        click.echo(
            f"{status_icon.get(test_result['status'], '❓')} {server}: {test_result['status']}"
        )

        if test_result["status"] != "success":
            click.echo(f"   Error: {test_result.get('error', 'Unknown error')}")


@mcp.command()
@click.option("--query", required=True, help="Search query for JIRA tickets")
@click.option("--project", help="JIRA project key")
@click.option("--max-results", default=5, help="Maximum number of results")
def search_jira(query: str, project: Optional[str], max_results: int):
    """Search JIRA tickets via MCP."""
    click.echo(f"Searching JIRA tickets: {query}")

    async def _search():
        service = await initialize_mcp_service()
        return await service.search_jira_tickets(
            query=query, project_key=project, max_results=max_results
        )

    result = asyncio.run(_search())

    if result["success"]:
        click.echo(f"✅ Found {len(result['tickets'])} tickets")
        for i, ticket in enumerate(result["tickets"], 1):
            click.echo(f"{i}. {ticket}")
    else:
        click.echo(f"❌ Search failed: {result['error']}")


@mcp.command()
@click.option("--owner", required=True, help="GitHub repository owner")
@click.option("--repo", required=True, help="GitHub repository name")
def github_info(owner: str, repo: str):
    """Get GitHub repository information via MCP."""
    click.echo(f"Getting GitHub repository info: {owner}/{repo}")

    async def _get_info():
        service = await initialize_mcp_service()
        return await service.get_github_repository_info(owner, repo)

    result = asyncio.run(_get_info())

    if result["success"]:
        click.echo("✅ Repository information retrieved")
        repo_info = result["repository"]
        click.echo(f"Repository: {repo_info}")
    else:
        click.echo(f"❌ Failed to get repository info: {result['error']}")


@mcp.command()
@click.option("--query", required=True, help="Search query for Confluence")
@click.option("--space", help="Confluence space key")
@click.option("--max-results", default=3, help="Maximum number of results")
def search_confluence(query: str, space: Optional[str], max_results: int):
    """Search Confluence documentation via MCP."""
    click.echo(f"Searching Confluence: {query}")

    async def _search():
        service = await initialize_mcp_service()
        return await service.search_confluence_docs(
            search_query=query, space_key=space, max_results=max_results
        )

    result = asyncio.run(_search())

    if result["success"]:
        click.echo(f"✅ Found {len(result['documents'])} documents")
        for i, doc in enumerate(result["documents"], 1):
            click.echo(f"{i}. {doc}")
    else:
        click.echo(f"❌ Search failed: {result['error']}")


if __name__ == "__main__":
    mcp()
