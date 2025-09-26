#!/usr/bin/env python3
"""
Integration tests for US-001: MCP Client Integration

This test suite validates end-to-end integration scenarios for the MCP client
implementation with real configuration patterns.
"""

import pytest
import asyncio
import json
import os
from unittest.mock import patch

from ...simple_mcp_client import SimpleMCPClient
from ...endpoint_connector import MCPEndpointConnector
from ...llama_integration import MCPEnhancedLlamaIndex
from ...common import MCPConfigurationError


class TestIntegrationScenarios:
    """Integration test scenarios for US-001 implementation."""

    def test_single_server_configuration(self):
        """Test single MCP server configuration scenario."""
        config = '{"atlassian": "https://mcp-atlassian-route.apps.cluster.com/sse"}'
        
        with patch.dict(os.environ, {'MCP_SERVERS': config}):
            client = SimpleMCPClient()
            
            assert len(client.servers) == 1
            assert "atlassian" in client.servers
            assert client.servers["atlassian"] == "https://mcp-atlassian-route.apps.cluster.com/sse"
    
    def test_multi_server_configuration(self):
        """Test multi-MCP server configuration scenario."""
        config = json.dumps({
            "atlassian": "https://mcp-atlassian-route.apps.cluster.com/sse",
            "github": "https://mcp-github-route.apps.cluster.com/sse",
            "confluence": "mcp-confluence.vteam-mcp.svc.cluster.local:8000"
        })
        
        with patch.dict(os.environ, {'MCP_SERVERS': config}):
            client = SimpleMCPClient()
            
            assert len(client.servers) == 3
            assert all(capability in client.servers for capability in ["atlassian", "github", "confluence"])
    
    def test_endpoint_validation_scenarios(self):
        """Test various endpoint validation scenarios."""
        connector = MCPEndpointConnector()
        
        # Valid external routes
        assert connector.validate_endpoint_config("https://mcp-route.apps.cluster.com/sse")
        assert connector.validate_endpoint_config("https://mcp-atlassian-route.apps.cluster.com/sse")
        assert connector.validate_endpoint_config("http://mcp-dev.example.com:8080/sse")
        
        # Valid cluster services
        assert connector.validate_endpoint_config("mcp-atlassian.namespace.svc.cluster.local:8000")
        assert connector.validate_endpoint_config("mcp-service.vteam-mcp.svc.cluster.local:9000")
        
        # Invalid formats
        assert not connector.validate_endpoint_config("invalid-format")
        assert not connector.validate_endpoint_config("ftp://invalid.com")
        assert not connector.validate_endpoint_config("")
        assert not connector.validate_endpoint_config("mcp-service.incomplete")
    
    def test_capability_detection_scenarios(self):
        """Test capability detection with various query patterns."""
        config = json.dumps({
            "atlassian": "https://mcp-atlassian.com/sse",
            "github": "https://mcp-github.com/sse",
            "confluence": "https://mcp-confluence.com/sse"
        })
        
        with patch.dict(os.environ, {'MCP_SERVERS': config}):
            client = SimpleMCPClient()
            
            # Test Jira/Atlassian detection
            assert client._detect_capability("Get Jira tickets for project ABC") == "atlassian"
            assert client._detect_capability("List Jira issues") == "atlassian"
            assert client._detect_capability("Show project tickets") == "atlassian"
            
            # Test GitHub detection
            assert client._detect_capability("List GitHub repositories") == "github"
            assert client._detect_capability("Show commit history") == "github"
            assert client._detect_capability("Get repository info") == "github"
            
            # Test Confluence detection
            assert client._detect_capability("Search confluence docs") == "confluence"
            assert client._detect_capability("Find wiki pages") == "confluence"
            assert client._detect_capability("Get document content") == "confluence"
            
            # Test explicit capability mention
            assert client._detect_capability("Get data from atlassian") == "atlassian"
            assert client._detect_capability("Query github for info") == "github"
    
    @pytest.mark.asyncio
    async def test_mcp_enhanced_llama_index_integration(self):
        """Test MCPEnhancedLlamaIndex integration scenario."""
        config = '{"atlassian": "https://test-mcp.com/sse"}'
        
        with patch.dict(os.environ, {'MCP_SERVERS': config}):
            enhanced = MCPEnhancedLlamaIndex()
            
            # Test initialization
            assert not enhanced._initialized
            await enhanced.initialize()
            assert enhanced._initialized
            
            # Test status reporting
            status = await enhanced.get_mcp_status()
            assert status["initialized"] == True
            assert "servers" in status
            
            # Clean up
            await enhanced.close()
            assert not enhanced._initialized
    
    @pytest.mark.asyncio
    async def test_error_handling_scenarios(self):
        """Test error handling in various failure scenarios."""
        # Test invalid JSON configuration
        with patch.dict(os.environ, {'MCP_SERVERS': 'invalid-json'}):
            with pytest.raises(MCPConfigurationError, match="Invalid JSON"):
                SimpleMCPClient()
        
        # Test empty configuration
        with patch.dict(os.environ, {'MCP_SERVERS': '{}'}):
            with pytest.raises(MCPConfigurationError, match="at least one server"):
                SimpleMCPClient()
        
        # Test invalid endpoint in configuration
        invalid_config = '{"test": "invalid-endpoint"}'
        with patch.dict(os.environ, {'MCP_SERVERS': invalid_config}):
            with pytest.raises(MCPConfigurationError, match="not recognized"):
                SimpleMCPClient()
    
    def test_server_status_reporting(self):
        """Test comprehensive server status reporting."""
        config = json.dumps({
            "atlassian": "https://mcp-atlassian.com/sse",
            "confluence": "mcp-confluence.ns.svc.cluster.local:8000"
        })
        
        with patch.dict(os.environ, {'MCP_SERVERS': config}):
            client = SimpleMCPClient()
            
            status = client.get_server_status()
            
            # Validate status structure
            assert len(status) == 2
            assert "atlassian" in status
            assert "confluence" in status
            
            # Validate atlassian status (external route)
            atlassian_status = status["atlassian"]
            assert atlassian_status["endpoint"] == "https://mcp-atlassian.com/sse"
            assert "connection_type" in atlassian_status  # Type detection may be automatic or unknown
            assert "connected" in atlassian_status
            assert "healthy" in atlassian_status
            
            # Validate confluence status (cluster service)
            confluence_status = status["confluence"]
            assert confluence_status["endpoint"] == "mcp-confluence.ns.svc.cluster.local:8000"
            assert "connection_type" in confluence_status  # Type detection may be automatic or unknown
    
    def test_endpoint_info_analysis(self):
        """Test detailed endpoint information analysis."""
        connector = MCPEndpointConnector()
        
        # Test external route analysis
        external_info = connector.get_endpoint_info("https://mcp-route.apps.cluster.com:8443/sse")
        assert external_info["valid"] == True
        assert external_info["type"] == "external_route"
        assert external_info["parsed"]["scheme"] == "https"
        assert external_info["parsed"]["hostname"] == "mcp-route.apps.cluster.com"
        assert external_info["parsed"]["port"] == 8443
        assert external_info["parsed"]["path"] == "/sse"
        
        # Test cluster service analysis
        cluster_info = connector.get_endpoint_info("mcp-service.vteam.svc.cluster.local:9000")
        assert cluster_info["valid"] == True
        assert cluster_info["type"] == "cluster_service"
        assert cluster_info["parsed"]["service"] == "mcp-service"
        assert cluster_info["parsed"]["namespace"] == "vteam"
        assert cluster_info["parsed"]["domain"] == "svc.cluster.local"
        assert cluster_info["parsed"]["port"] == 9000


class TestProductionScenarios:
    """Test scenarios that would occur in production deployment."""
    
    def test_kubernetes_configmap_simulation(self):
        """Simulate Kubernetes ConfigMap configuration pattern."""
        # Simulate ConfigMap data as it would appear in environment
        configmap_data = {
            "MCP_SERVERS": json.dumps({
                "atlassian": "https://mcp-atlassian-route.apps.cluster.com/sse",
                "github": "https://mcp-github-route.apps.cluster.com/sse"
            })
        }
        
        with patch.dict(os.environ, configmap_data):
            client = SimpleMCPClient()
            
            assert len(client.servers) == 2
            assert client.servers["atlassian"].startswith("https://mcp-atlassian-route")
            assert client.servers["github"].startswith("https://mcp-github-route")
    
    def test_openshift_route_patterns(self):
        """Test OpenShift route URL patterns."""
        connector = MCPEndpointConnector()
        
        # Test typical OpenShift route patterns
        openshift_routes = [
            "https://mcp-atlassian-route.apps.cluster.example.com/sse",
            "https://mcp-service-vteam.apps.openshift.local/sse",
            "https://mcp-app-dev.apps.cluster.local:8443/api/sse"
        ]
        
        for route in openshift_routes:
            assert connector.validate_endpoint_config(route), f"Should validate OpenShift route: {route}"
    
    def test_cluster_service_dns_patterns(self):
        """Test Kubernetes cluster service DNS patterns."""
        connector = MCPEndpointConnector()
        
        # Test typical cluster service patterns
        cluster_services = [
            "mcp-atlassian.vteam-mcp.svc.cluster.local:8000",
            "mcp-service.default.svc.cluster.local:9000",
            "mcp-app.my-namespace.svc.cluster.local:80"
        ]
        
        for service in cluster_services:
            assert connector.validate_endpoint_config(service), f"Should validate cluster service: {service}"


if __name__ == "__main__":
    pytest.main([__file__, "-v", "--tb=short"])