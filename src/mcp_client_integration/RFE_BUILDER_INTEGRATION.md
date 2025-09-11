# MCP Client Integration with RFE Builder - Step-by-Step Guide

This document provides detailed, granular steps to integrate the MCP Client Integration library with the `demos/rfe-builder` application.

## Overview

The integration will enable the RFE Builder to:
- Connect to MCP servers (Atlassian, GitHub, Confluence, etc.)
- Fetch real-time data from external systems
- Enhance RFE content with actual project data
- Provide contextual information for better RFE quality

## Prerequisites

Before starting the integration:

1. **RFE Builder Setup**: The `demos/rfe-builder` application should be running
2. **MCP Client Library**: Installed and tested (from `src/mcp_client_integration`)
3. **MCP Servers**: At least one MCP server (Atlassian, GitHub, etc.) deployed and accessible
4. **Python Environment**: Python 3.11+ with virtual environment

## Step 1: Install MCP Client Integration in RFE Builder

### 1.1 Update RFE Builder Dependencies

First, add the MCP client integration to the RFE Builder's `pyproject.toml`:

```bash
cd demos/rfe-builder
```

Edit `pyproject.toml`:

```toml
[project]
name = "backend"
version = "0.1.0"
description = "RHOAI AI Feature Sizing Backend with Multi-Agent RAG"
readme = "README.md"
requires-python = ">=3.11,<3.14"
dependencies = [
    # ... existing dependencies ...
    
    # MCP Client Integration
    "mcp-client-integration",
    # OR for local development:
    # {path = "../../src/mcp_client_integration", develop = true},
]
```

### 1.2 Install the Local Package

For development, install the local MCP client integration:

```bash
# From demos/rfe-builder directory
uv add --editable ../../src/mcp_client_integration
```

Or manually add to pyproject.toml:

```toml
[tool.uv.sources]
mcp-client-integration = { path = "../../src/mcp_client_integration", editable = true }
```

### 1.3 Verify Installation

Test the installation:

```bash
cd demos/rfe-builder
uv run python -c "from mcp_client_integration import SimpleMCPClient; print('✅ MCP Integration installed successfully')"
```

## Step 2: Configure MCP Servers

### 2.1 Create MCP Configuration

Create or update the `.env` file in `demos/rfe-builder/src/`:

```bash
# MCP Server Configuration
MCP_SERVERS='{
  "atlassian": "https://mcp-atlassian-route.apps.cluster.com/sse",
  "github": "https://mcp-github-route.apps.cluster.com/sse",
  "confluence": "mcp-confluence.vteam-mcp.svc.cluster.local:8000"
}'

# MCP Security Settings
MCP_PRODUCTION_MODE=false
MCP_VERIFY_SSL=true
MCP_MAX_CONNECTIONS=5
MCP_TIMEOUT=30
```

### 2.2 Environment Variable Validation

Add validation script `demos/rfe-builder/src/mcp_config_validator.py`:

```python
#!/usr/bin/env python3
"""
MCP Configuration Validator for RFE Builder

Validates MCP server configuration before application startup.
"""

import os
import json
import logging
from typing import Dict, Any, Optional

from mcp_client_integration.common import (
    MCPConfigurationManager,
    MCPSecurityValidator,
    MCPConfigurationError
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
            production_mode = os.getenv("MCP_PRODUCTION_MODE", "false").lower() == "true"
        
        self.production_mode = production_mode
        self.config_manager = MCPConfigurationManager(production_mode=production_mode)
        self.security_validator = MCPSecurityValidator(production_mode=production_mode)
        
        logger.info(f"MCP Configuration initialized (production_mode={production_mode})")
    
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
            security_result = self.security_validator.validate_configuration_security(servers)
            
            return {
                "valid": True,
                "production_mode": self.production_mode,
                "summary": summary,
                "security_validation": security_result.to_dict(),
                "servers": servers
            }
            
        except MCPConfigurationError as e:
            logger.error(f"MCP configuration validation failed: {e}")
            return {
                "valid": False,
                "error": str(e),
                "production_mode": self.production_mode
            }
    
    def get_mcp_client(self) -> 'SimpleMCPClient':
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
```

### 2.3 Test Configuration

```bash
cd demos/rfe-builder
uv run python src/mcp_config_validator.py
```

## Step 3: Create MCP Service Layer

### 3.1 Create MCP Service Module

Create `demos/rfe-builder/src/services/mcp_service.py`:

```python
#!/usr/bin/env python3
"""
MCP Service Layer for RFE Builder

Provides high-level MCP operations for RFE Builder workflows.
"""

import logging
import asyncio
from typing import Dict, Any, List, Optional, Union
from datetime import datetime

from mcp_client_integration import SimpleMCPClient
from mcp_client_integration.common import MCPError, MCPConnectionError

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
        self._connection_status = {}
        
        if auto_connect:
            asyncio.create_task(self._initialize_connections())
    
    async def _initialize_connections(self) -> None:
        """Initialize connections to all configured MCP servers."""
        try:
            await self.client.connect_all()
            self._connected = True
            self._connection_status = await self.client.health_check()
            
            logger.info(f"MCP Service initialized. Healthy servers: {sum(self._connection_status.values())}")
            
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
            "last_check": datetime.now().isoformat()
        }
    
    async def search_jira_tickets(self, 
                                 project_key: Optional[str] = None,
                                 query: Optional[str] = None,
                                 max_results: int = 10) -> Dict[str, Any]:
        """
        Search for JIRA tickets related to RFE context.
        
        Args:
            project_key: JIRA project key to search in
            query: Free-text search query
            max_results: Maximum number of results to return
            
        Returns:
            Dictionary with search results and metadata
        """
        try:
            # Build search query for Atlassian MCP server
            search_params = {
                "action": "search_tickets",
                "project": project_key,
                "query": query,
                "max_results": max_results
            }
            
            # Remove None values
            search_params = {k: v for k, v in search_params.items() if v is not None}
            
            # Query Atlassian MCP server
            response = await self.client.query(
                f"Search JIRA tickets: {query or 'all tickets'}", 
                capability="atlassian"
            )
            
            return {
                "success": True,
                "tickets": response.get("data", []),
                "query": search_params,
                "server": "atlassian",
                "timestamp": datetime.now().isoformat()
            }
            
        except MCPConnectionError as e:
            logger.error(f"Failed to search JIRA tickets: {e}")
            return {
                "success": False,
                "error": f"Connection error: {e}",
                "tickets": [],
                "fallback_used": False
            }
        
        except MCPError as e:
            logger.error(f"MCP error searching JIRA tickets: {e}")
            return {
                "success": False,
                "error": f"MCP error: {e}",
                "tickets": []
            }
    
    async def get_github_repository_info(self, 
                                       repo_owner: str,
                                       repo_name: str) -> Dict[str, Any]:
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
                "timestamp": datetime.now().isoformat()
            }
            
        except MCPError as e:
            logger.error(f"Failed to get GitHub repository info: {e}")
            return {
                "success": False,
                "error": str(e),
                "repository": {}
            }
    
    async def search_confluence_docs(self, 
                                   search_query: str,
                                   space_key: Optional[str] = None,
                                   max_results: int = 5) -> Dict[str, Any]:
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
                "timestamp": datetime.now().isoformat()
            }
            
        except MCPError as e:
            logger.error(f"Failed to search Confluence: {e}")
            return {
                "success": False,
                "error": str(e),
                "documents": []
            }
    
    async def get_contextual_data_for_rfe(self, 
                                        rfe_title: str,
                                        rfe_description: str,
                                        project_context: Optional[Dict] = None) -> Dict[str, Any]:
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
            "timestamp": datetime.now().isoformat()
        }
        
        # Get connection status
        status = await self.get_connection_status()
        healthy_servers = [server for server, healthy in status["health"].items() if healthy]
        
        if not healthy_servers:
            logger.warning("No healthy MCP servers available for contextual data")
            contextual_data["summary"]["error"] = "No healthy MCP servers available"
            return contextual_data
        
        # Parallel data collection from available servers
        tasks = []
        
        # JIRA tickets if Atlassian is available
        if "atlassian" in healthy_servers:
            tasks.append(("jira_tickets", self.search_jira_tickets(
                query=f"{rfe_title} {rfe_description}"[:100],  # Limit query length
                max_results=5
            )))
        
        # GitHub repository info if GitHub is available and project context provided
        if "github" in healthy_servers and project_context:
            repo_owner = project_context.get("github_owner")
            repo_name = project_context.get("github_repo")
            if repo_owner and repo_name:
                tasks.append(("github_repo", self.get_github_repository_info(
                    repo_owner, repo_name
                )))
        
        # Confluence documentation if Confluence is available
        if "confluence" in healthy_servers:
            tasks.append(("confluence_docs", self.search_confluence_docs(
                search_query=f"{rfe_title} {rfe_description}"[:100],
                max_results=3
            )))
        
        # Execute all queries in parallel
        if tasks:
            results = await asyncio.gather(*[task[1] for task in tasks], return_exceptions=True)
            
            for i, (data_type, result) in enumerate(zip([task[0] for task in tasks], results)):
                if isinstance(result, Exception):
                    logger.error(f"Error fetching {data_type}: {result}")
                    contextual_data["data_sources"][data_type] = {
                        "success": False,
                        "error": str(result)
                    }
                else:
                    contextual_data["data_sources"][data_type] = result
        
        # Generate summary
        successful_sources = [k for k, v in contextual_data["data_sources"].items() if v.get("success")]
        contextual_data["summary"] = {
            "healthy_servers": healthy_servers,
            "successful_sources": successful_sources,
            "total_data_points": sum(
                len(v.get("tickets", v.get("documents", v.get("repository", {}) and [v.get("repository")] or [])))
                for v in contextual_data["data_sources"].values()
                if v.get("success")
            )
        }
        
        return contextual_data
    
    async def close(self) -> None:
        """Close MCP connections."""
        if self._connected:
            await self.client.disconnect_all()
            self._connected = False
            logger.info("MCP Service connections closed")

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
```

### 3.2 Create Services Directory

```bash
mkdir -p demos/rfe-builder/src/services
touch demos/rfe-builder/src/services/__init__.py
```

Add to `demos/rfe-builder/src/services/__init__.py`:

```python
"""
Services package for RFE Builder

Provides high-level service interfaces for external system integration.
"""

from .mcp_service import (
    RFEBuilderMCPService,
    get_mcp_service,
    initialize_mcp_service
)

__all__ = [
    "RFEBuilderMCPService",
    "get_mcp_service", 
    "initialize_mcp_service"
]
```

## Step 4: Integrate with RFE Builder Workflow

### 4.1 Enhance the RFE Builder Workflow

Edit `demos/rfe-builder/src/rfe_builder_workflow.py` to integrate MCP:

```python
# Add these imports at the top
from .services.mcp_service import get_mcp_service, initialize_mcp_service
from .mcp_config_validator import RFEBuilderMCPConfig

# Add this method to the RFEBuilderWorkflow class
class RFEBuilderWorkflow:
    # ... existing code ...
    
    async def initialize_mcp_integration(self) -> Dict[str, Any]:
        """
        Initialize MCP integration for enhanced RFE building.
        
        Returns:
            MCP initialization status
        """
        try:
            # Validate MCP configuration
            config_validator = RFEBuilderMCPConfig()
            config_result = config_validator.validate_configuration()
            
            if not config_result["valid"]:
                logger.warning(f"MCP configuration invalid: {config_result.get('error')}")
                return {
                    "mcp_enabled": False,
                    "error": config_result.get('error'),
                    "status": "configuration_invalid"
                }
            
            # Initialize MCP service
            mcp_service = await initialize_mcp_service()
            connection_status = await mcp_service.get_connection_status()
            
            logger.info(f"MCP Integration initialized. Healthy servers: {connection_status['healthy_count']}")
            
            return {
                "mcp_enabled": True,
                "connection_status": connection_status,
                "status": "initialized"
            }
            
        except Exception as e:
            logger.error(f"Failed to initialize MCP integration: {e}")
            return {
                "mcp_enabled": False,
                "error": str(e),
                "status": "initialization_failed"
            }
    
    async def enhance_rfe_with_mcp_data(self, 
                                      rfe_content: Dict[str, Any],
                                      project_context: Optional[Dict] = None) -> Dict[str, Any]:
        """
        Enhance RFE content with data from MCP servers.
        
        Args:
            rfe_content: Current RFE content
            project_context: Optional project context for targeted queries
            
        Returns:
            Enhanced RFE content with MCP data
        """
        try:
            mcp_service = get_mcp_service()
            
            # Get contextual data from MCP servers
            contextual_data = await mcp_service.get_contextual_data_for_rfe(
                rfe_title=rfe_content.get("title", ""),
                rfe_description=rfe_content.get("description", ""),
                project_context=project_context
            )
            
            # Enhance RFE content with contextual data
            enhanced_content = rfe_content.copy()
            enhanced_content["mcp_context"] = contextual_data
            
            # Add related tickets to requirements if available
            jira_data = contextual_data["data_sources"].get("jira_tickets", {})
            if jira_data.get("success") and jira_data.get("tickets"):
                enhanced_content["related_tickets"] = jira_data["tickets"]
            
            # Add repository information if available
            github_data = contextual_data["data_sources"].get("github_repo", {})
            if github_data.get("success") and github_data.get("repository"):
                enhanced_content["repository_context"] = github_data["repository"]
            
            # Add documentation references if available
            confluence_data = contextual_data["data_sources"].get("confluence_docs", {})
            if confluence_data.get("success") and confluence_data.get("documents"):
                enhanced_content["documentation_references"] = confluence_data["documents"]
            
            logger.info(f"Enhanced RFE with {contextual_data['summary']['total_data_points']} data points from MCP")
            
            return enhanced_content
            
        except Exception as e:
            logger.error(f"Failed to enhance RFE with MCP data: {e}")
            # Return original content if enhancement fails
            rfe_content["mcp_enhancement_error"] = str(e)
            return rfe_content
```

### 4.2 Update Settings Integration

Edit `demos/rfe-builder/src/settings.py` to include MCP configuration:

```python
# Add this import at the top
import os
from typing import Optional, Dict, Any, Type

# Add this function at the end
def configure_mcp_integration(config_override: Optional[Dict[str, Any]] = None) -> Dict[str, Any]:
    """
    Configure MCP integration for RFE Builder.
    
    Args:
        config_override: Optional configuration overrides
        
    Returns:
        MCP configuration dictionary
    """
    from .mcp_config_validator import RFEBuilderMCPConfig
    
    # Create MCP configuration
    mcp_config = RFEBuilderMCPConfig()
    
    # Validate configuration
    validation_result = mcp_config.validate_configuration()
    
    # Apply any overrides
    if config_override:
        # Apply configuration overrides here if needed
        pass
    
    return {
        "mcp_validation": validation_result,
        "mcp_enabled": validation_result.get("valid", False),
        "production_mode": mcp_config.production_mode
    }

def init_settings_with_mcp(
    llm_provider: Optional[str] = None,
    embedding_provider: Optional[str] = None,
    llm_config: Optional[Dict[str, Any]] = None,
    embedding_config: Optional[Dict[str, Any]] = None,
    mcp_config: Optional[Dict[str, Any]] = None,
    **global_settings,
) -> Dict[str, Any]:
    """
    Initialize LlamaIndex settings with MCP integration.
    
    Returns:
        Initialization results including MCP status
    """
    # Initialize LlamaIndex settings
    init_settings(llm_provider, embedding_provider, llm_config, embedding_config, **global_settings)
    
    # Configure MCP integration
    mcp_status = configure_mcp_integration(mcp_config)
    
    return {
        "llama_index_initialized": True,
        "mcp_status": mcp_status
    }
```

## Step 5: Add MCP Health Check and Monitoring

### 5.1 Create Health Check Endpoint

Create `demos/rfe-builder/src/health_check.py`:

```python
#!/usr/bin/env python3
"""
Health Check Module for RFE Builder with MCP Integration

Provides health check endpoints for monitoring MCP connections.
"""

import asyncio
import logging
from typing import Dict, Any
from datetime import datetime

from .services.mcp_service import get_mcp_service
from .mcp_config_validator import validate_mcp_config_for_rfe_builder

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
            "components": {}
        }
        
        # Check MCP configuration
        try:
            config_result = validate_mcp_config_for_rfe_builder()
            health_data["components"]["mcp_config"] = {
                "status": "healthy" if config_result["valid"] else "unhealthy",
                "details": config_result
            }
        except Exception as e:
            health_data["components"]["mcp_config"] = {
                "status": "error",
                "error": str(e)
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
                "details": connection_status
            }
            
        except Exception as e:
            health_data["components"]["mcp_connections"] = {
                "status": "error",
                "error": str(e)
            }
        
        # Determine overall status
        component_statuses = [comp["status"] for comp in health_data["components"].values()]
        
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
        test_results = {
            "timestamp": datetime.now().isoformat(),
            "tests": {}
        }
        
        try:
            mcp_service = get_mcp_service()
            
            # Test Atlassian connection
            try:
                jira_result = await mcp_service.search_jira_tickets(
                    query="test connectivity",
                    max_results=1
                )
                test_results["tests"]["atlassian"] = {
                    "status": "success" if jira_result["success"] else "failed",
                    "details": jira_result
                }
            except Exception as e:
                test_results["tests"]["atlassian"] = {
                    "status": "error",
                    "error": str(e)
                }
            
            # Test GitHub connection
            try:
                # Use a well-known public repository for testing
                github_result = await mcp_service.get_github_repository_info(
                    repo_owner="octocat",
                    repo_name="Hello-World"
                )
                test_results["tests"]["github"] = {
                    "status": "success" if github_result["success"] else "failed",
                    "details": github_result
                }
            except Exception as e:
                test_results["tests"]["github"] = {
                    "status": "error",
                    "error": str(e)
                }
            
            # Test Confluence connection
            try:
                confluence_result = await mcp_service.search_confluence_docs(
                    search_query="test",
                    max_results=1
                )
                test_results["tests"]["confluence"] = {
                    "status": "success" if confluence_result["success"] else "failed",
                    "details": confluence_result
                }
            except Exception as e:
                test_results["tests"]["confluence"] = {
                    "status": "error",
                    "error": str(e)
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
```

## Step 6: Create CLI Integration Commands

### 6.1 Add MCP Commands to RFE Builder CLI

Create `demos/rfe-builder/src/mcp_cli.py`:

```python
#!/usr/bin/env python3
"""
MCP CLI Commands for RFE Builder

Provides command-line interface for MCP operations.
"""

import asyncio
import click
import json
from typing import Optional

from .health_check import get_health, test_connectivity
from .services.mcp_service import initialize_mcp_service
from .mcp_config_validator import validate_mcp_config_for_rfe_builder

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
        status_icon = {"healthy": "✅", "degraded": "⚠️", "unhealthy": "❌", "error": "💥"}
        click.echo(f"{status_icon.get(details['status'], '❓')} {component}: {details['status']}")

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
        click.echo(f"{status_icon.get(test_result['status'], '❓')} {server}: {test_result['status']}")
        
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
            query=query,
            project_key=project,
            max_results=max_results
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
            search_query=query,
            space_key=space,
            max_results=max_results
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
```

### 6.2 Add MCP Commands to Main CLI

Edit `demos/rfe-builder/src/ingestion.py` to include MCP commands:

```python
# Add this import at the top
from .mcp_cli import mcp

# Add to the main CLI group
@cli.add_command(mcp)
```

## Step 7: Update Application Initialization

### 7.1 Modify Application Startup

Edit the main application file to initialize MCP on startup. This depends on how RFE Builder is structured, but typically in a main.py or app.py file:

```python
# Add these imports
import asyncio
from src.services.mcp_service import initialize_mcp_service
from src.settings import init_settings_with_mcp

async def initialize_application():
    """Initialize RFE Builder application with MCP integration."""
    
    # Initialize LlamaIndex settings with MCP
    settings_result = init_settings_with_mcp()
    
    if settings_result["mcp_status"]["mcp_enabled"]:
        # Initialize MCP service
        try:
            mcp_service = await initialize_mcp_service()
            print("✅ MCP Integration initialized successfully")
            
            # Test connectivity
            status = await mcp_service.get_connection_status()
            healthy = status["healthy_count"]
            total = status["total_count"]
            print(f"MCP Status: {healthy}/{total} servers healthy")
            
        except Exception as e:
            print(f"⚠️ MCP Integration failed: {e}")
    else:
        print("ℹ️ MCP Integration disabled or not configured")
    
    return settings_result

# Call during application startup
if __name__ == "__main__":
    # Initialize application
    init_result = asyncio.run(initialize_application())
    
    # Continue with normal application startup
    # ... rest of application code ...
```

## Step 8: Testing the Integration

### 8.1 Configuration Testing

```bash
cd demos/rfe-builder

# Test MCP configuration
uv run python src/mcp_config_validator.py

# Test MCP CLI commands
uv run python -m src.mcp_cli validate
uv run python -m src.mcp_cli health
uv run python -m src.mcp_cli test
```

### 8.2 Integration Testing

Create `demos/rfe-builder/test_mcp_integration.py`:

```python
#!/usr/bin/env python3
"""
Integration tests for MCP integration with RFE Builder.
"""

import asyncio
import pytest
from src.services.mcp_service import RFEBuilderMCPService, initialize_mcp_service
from src.mcp_config_validator import validate_mcp_config_for_rfe_builder

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

if __name__ == "__main__":
    pytest.main([__file__])
```

### 8.3 End-to-End Testing

```bash
# Run integration tests
cd demos/rfe-builder
uv run python test_mcp_integration.py

# Test with actual RFE workflow
uv run python -c "
import asyncio
from src.rfe_builder_workflow import RFEBuilderWorkflow

async def test_enhanced_rfe():
    workflow = RFEBuilderWorkflow()
    
    # Initialize MCP
    mcp_status = await workflow.initialize_mcp_integration()
    print(f'MCP Status: {mcp_status}')
    
    # Test RFE enhancement
    sample_rfe = {
        'title': 'Add authentication to API',
        'description': 'Implement OAuth2 authentication for REST API endpoints'
    }
    
    enhanced_rfe = await workflow.enhance_rfe_with_mcp_data(
        sample_rfe,
        {'github_owner': 'your-org', 'github_repo': 'your-repo'}
    )
    
    print('Enhanced RFE:', enhanced_rfe)

asyncio.run(test_enhanced_rfe())
"
```

## Step 9: Production Deployment Considerations

### 9.1 Environment Configuration

For production deployment, create `demos/rfe-builder/.env.production`:

```bash
# Production MCP Configuration
MCP_SERVERS='{
  "atlassian": "https://mcp-atlassian-route.apps.production.com/sse",
  "github": "https://mcp-github-route.apps.production.com/sse",
  "confluence": "mcp-confluence.production.svc.cluster.local:8000"
}'

# Security Settings
MCP_PRODUCTION_MODE=true
MCP_VERIFY_SSL=true
MCP_MAX_CONNECTIONS=10
MCP_TIMEOUT=30

# Logging
LOG_LEVEL=INFO
MCP_LOG_LEVEL=INFO
```

### 9.2 Docker Integration

Update `demos/rfe-builder/Dockerfile` to include MCP integration:

```dockerfile
# Add MCP client integration installation
COPY ../../src/mcp_client_integration ./mcp_client_integration
RUN pip install -e ./mcp_client_integration

# Copy MCP configuration
COPY .env.production .env

# Health check that includes MCP
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
  CMD python -c "
import asyncio
from src.health_check import get_health
result = asyncio.run(get_health())
exit(0 if result['overall_status'] in ['healthy', 'degraded'] else 1)
"
```

### 9.3 Kubernetes Deployment

Update Kubernetes manifests to include MCP configuration:

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: rfe-builder-mcp-config
data:
  MCP_SERVERS: |
    {
      "atlassian": "https://mcp-atlassian-route.apps.cluster.com/sse",
      "github": "https://mcp-github-route.apps.cluster.com/sse", 
      "confluence": "mcp-confluence.vteam-mcp.svc.cluster.local:8000"
    }
  MCP_PRODUCTION_MODE: "true"
  MCP_VERIFY_SSL: "true"
  MCP_MAX_CONNECTIONS: "10"
  MCP_TIMEOUT: "30"
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: rfe-builder
spec:
  template:
    spec:
      containers:
      - name: rfe-builder
        envFrom:
        - configMapRef:
            name: rfe-builder-mcp-config
        # Health check
        livenessProbe:
          exec:
            command:
            - python
            - -c
            - |
              import asyncio
              from src.health_check import get_health
              result = asyncio.run(get_health())
              exit(0 if result['overall_status'] in ['healthy', 'degraded'] else 1)
          initialDelaySeconds: 30
          periodSeconds: 30
        readinessProbe:
          exec:
            command:
            - python
            - -m
            - src.mcp_cli
            - validate
          initialDelaySeconds: 10
          periodSeconds: 10
```

## Step 10: Usage Examples

### 10.1 Basic RFE Enhancement

```python
from src.rfe_builder_workflow import RFEBuilderWorkflow

async def create_enhanced_rfe():
    workflow = RFEBuilderWorkflow()
    
    # Initialize MCP integration
    await workflow.initialize_mcp_integration()
    
    # Create RFE content
    rfe = {
        "title": "Implement user authentication",
        "description": "Add OAuth2 authentication to the web application",
        "priority": "high"
    }
    
    # Enhance with MCP data
    enhanced_rfe = await workflow.enhance_rfe_with_mcp_data(
        rfe,
        project_context={
            "github_owner": "my-org",
            "github_repo": "web-app",
            "jira_project": "WEBAPP"
        }
    )
    
    # The enhanced RFE now includes:
    # - Related JIRA tickets
    # - GitHub repository context
    # - Confluence documentation
    # - Additional contextual information
    
    return enhanced_rfe
```

### 10.2 Manual MCP Queries

```python
from src.services.mcp_service import get_mcp_service

async def get_project_context():
    mcp_service = get_mcp_service()
    
    # Search for related tickets
    tickets = await mcp_service.search_jira_tickets(
        query="authentication OAuth",
        project_key="WEBAPP",
        max_results=10
    )
    
    # Get repository information
    repo_info = await mcp_service.get_github_repository_info(
        repo_owner="my-org",
        repo_name="web-app"
    )
    
    # Search documentation
    docs = await mcp_service.search_confluence_docs(
        search_query="authentication implementation guide",
        space_key="DEV"
    )
    
    return {
        "tickets": tickets,
        "repository": repo_info,
        "documentation": docs
    }
```

## Troubleshooting

### Common Issues

1. **MCP Server Connection Failed**
   ```bash
   uv run python -m src.mcp_cli test
   ```
   Check network connectivity and server URLs.

2. **Configuration Validation Failed**
   ```bash
   uv run python -m src.mcp_cli validate
   ```
   Verify MCP_SERVERS environment variable format.

3. **Import Errors**
   ```bash
   uv run python -c "from mcp_client_integration import SimpleMCPClient; print('OK')"
   ```
   Ensure MCP client integration is properly installed.

4. **SSL Certificate Issues**
   Set `MCP_VERIFY_SSL=false` for development environments.

### Logging and Debugging

Enable debug logging:
```python
import logging
logging.basicConfig(level=logging.DEBUG)
logging.getLogger('mcp_client_integration').setLevel(logging.DEBUG)
```

Check MCP service logs:
```bash
uv run python -c "
import asyncio
from src.health_check import get_health
result = asyncio.run(get_health())
print(result)
"
```

## Summary

This integration guide provides a complete step-by-step process to integrate the MCP Client Integration library with the RFE Builder application. The integration enables:

- **Real-time data fetching** from JIRA, GitHub, and Confluence
- **Enhanced RFE content** with contextual information
- **Health monitoring** and connectivity testing
- **Production-ready deployment** with proper security
- **CLI tools** for management and debugging

The integration is designed to be robust, with fallback mechanisms and comprehensive error handling to ensure the RFE Builder continues to function even when MCP servers are unavailable.