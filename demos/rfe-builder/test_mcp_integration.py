#!/usr/bin/env python3
"""
Integration tests for MCP integration with RFE Builder.
"""

import asyncio
import pytest
from src.services.mcp_service import RFEBuilderMCPService, initialize_mcp_service
from src.mcp_config_validator import validate_mcp_config_for_rfe_builder
from src.health_check import get_health
from src.health_check import test_connectivity as health_test_connectivity


@pytest.mark.asyncio
async def test_mcp_configuration():
    """Test MCP configuration validation."""
    result = validate_mcp_config_for_rfe_builder()
    assert isinstance(result, dict)
    assert "valid" in result


@pytest.mark.asyncio
async def test_mcp_service_initialization():
    """Test MCP service initialization."""
    service = await initialize_mcp_service()
    assert isinstance(service, RFEBuilderMCPService)
    
    status = await service.get_connection_status()
    assert "connected" in status
    assert "servers" in status


@pytest.mark.asyncio
async def test_contextual_data_retrieval():
    """Test retrieval of contextual data for RFE."""
    service = await initialize_mcp_service()
    
    result = await service.get_contextual_data_for_rfe(
        rfe_title="Test RFE",
        rfe_description="This is a test RFE for integration testing",
        project_context={
            "github_owner": "octocat",
            "github_repo": "Hello-World"
        }
    )
    
    assert "rfe_title" in result
    assert "data_sources" in result
    assert "summary" in result


@pytest.mark.asyncio
async def test_health_check():
    """Test health check functionality."""
    health = await get_health()
    assert "overall_status" in health
    assert "components" in health
    assert "timestamp" in health


@pytest.mark.asyncio
async def test_mcp_connectivity():
    """Test MCP connectivity."""
    connectivity = await health_test_connectivity()
    assert "timestamp" in connectivity
    assert "tests" in connectivity


@pytest.mark.asyncio
async def test_mcp_service_jira_search():
    """Test JIRA search functionality."""
    service = await initialize_mcp_service()
    result = await service.search_jira_tickets(
        query="test",
        max_results=1
    )
    
    assert "success" in result
    assert "tickets" in result
    assert "timestamp" in result


@pytest.mark.asyncio
async def test_mcp_service_github_info():
    """Test GitHub repository info retrieval."""
    service = await initialize_mcp_service()
    result = await service.get_github_repository_info(
        repo_owner="octocat",
        repo_name="Hello-World"
    )
    
    assert "success" in result
    assert "repository" in result
    assert "owner" in result
    assert "name" in result


@pytest.mark.asyncio
async def test_mcp_service_confluence_search():
    """Test Confluence documentation search."""
    service = await initialize_mcp_service()
    result = await service.search_confluence_docs(
        search_query="test",
        max_results=1
    )
    
    assert "success" in result
    assert "documents" in result
    assert "query" in result


def test_mcp_config_validator_import():
    """Test that MCP config validator can be imported."""
    from src.mcp_config_validator import RFEBuilderMCPConfig
    validator = RFEBuilderMCPConfig()
    assert validator is not None


def test_mcp_service_import():
    """Test that MCP service can be imported."""
    from src.services.mcp_service import RFEBuilderMCPService
    service = RFEBuilderMCPService(auto_connect=False)
    assert service is not None


def test_health_check_import():
    """Test that health check module can be imported."""
    from src.health_check import HealthChecker
    checker = HealthChecker()
    assert checker is not None


if __name__ == "__main__":
    pytest.main([__file__])