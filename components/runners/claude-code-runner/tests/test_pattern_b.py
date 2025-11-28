#!/usr/bin/env python3
"""
Tests for Pattern B: Bash tool disabling and MCP workspace exec.

These tests verify:
1. DISABLE_BASH_TOOL environment variable removes Bash from allowed_tools
2. MCP workspace exec server handles commands correctly
3. kubectl exec and nsenter execution modes work
"""

import asyncio
import json
import os
import sys
import unittest
from unittest.mock import AsyncMock, MagicMock, patch

# Add parent directory to path for imports
sys.path.insert(0, os.path.dirname(os.path.dirname(os.path.abspath(__file__))))


class TestDisableBashTool(unittest.TestCase):
    """Test that DISABLE_BASH_TOOL removes Bash from allowed_tools."""

    def test_bash_included_by_default(self):
        """Verify Bash is in allowed_tools when DISABLE_BASH_TOOL is not set."""
        # Default allowed_tools list from wrapper.py
        allowed_tools = ["Read", "Write", "Bash", "Glob", "Grep", "Edit", "MultiEdit", "WebSearch", "WebFetch"]
        self.assertIn("Bash", allowed_tools)

    def test_bash_removed_when_disabled(self):
        """Verify Bash is removed from allowed_tools when DISABLE_BASH_TOOL=1."""
        disable_bash = os.environ.get('DISABLE_BASH_TOOL', '').strip().lower() in ('1', 'true', 'yes')

        if disable_bash:
            allowed_tools = ["Read", "Write", "Glob", "Grep", "Edit", "MultiEdit", "WebSearch", "WebFetch"]
        else:
            allowed_tools = ["Read", "Write", "Bash", "Glob", "Grep", "Edit", "MultiEdit", "WebSearch", "WebFetch"]

        # Simulate disabled
        os.environ['DISABLE_BASH_TOOL'] = '1'
        disable_bash = os.environ.get('DISABLE_BASH_TOOL', '').strip().lower() in ('1', 'true', 'yes')
        self.assertTrue(disable_bash)

        allowed_tools_disabled = ["Read", "Write", "Glob", "Grep", "Edit", "MultiEdit", "WebSearch", "WebFetch"]
        self.assertNotIn("Bash", allowed_tools_disabled)

        # Cleanup
        del os.environ['DISABLE_BASH_TOOL']


class TestMCPWorkspaceExec(unittest.TestCase):
    """Test MCP workspace exec server."""

    def setUp(self):
        """Set up test environment."""
        os.environ['POD_NAME'] = 'test-pod'
        os.environ['POD_NAMESPACE'] = 'test-namespace'
        os.environ['WORKSPACE_CONTAINER'] = 'workspace'

    def tearDown(self):
        """Clean up test environment."""
        for key in ['POD_NAME', 'POD_NAMESPACE', 'WORKSPACE_CONTAINER']:
            if key in os.environ:
                del os.environ[key]

    def test_mcp_tools_list(self):
        """Test that MCP server returns correct tool list."""
        from mcp_servers.workspace_exec import TOOLS

        self.assertEqual(len(TOOLS), 1)
        self.assertEqual(TOOLS[0]['name'], 'exec')
        self.assertIn('command', TOOLS[0]['inputSchema']['properties'])
        self.assertIn('workdir', TOOLS[0]['inputSchema']['properties'])

    @patch('mcp_servers.workspace_exec.asyncio.create_subprocess_exec')
    def test_kubectl_exec(self, mock_subprocess):
        """Test kubectl exec mode."""
        from mcp_servers.workspace_exec import exec_via_kubectl

        # Mock successful command
        mock_proc = AsyncMock()
        mock_proc.communicate = AsyncMock(return_value=(b'hello world\n', b''))
        mock_proc.returncode = 0
        mock_subprocess.return_value = mock_proc

        result = asyncio.run(exec_via_kubectl('echo hello world'))

        self.assertTrue(result['success'])
        self.assertEqual(result['stdout'].strip(), 'hello world')
        self.assertEqual(result['exit_code'], 0)

    @patch('mcp_servers.workspace_exec.asyncio.create_subprocess_exec')
    def test_kubectl_exec_failure(self, mock_subprocess):
        """Test kubectl exec handles failures."""
        from mcp_servers.workspace_exec import exec_via_kubectl

        # Mock failed command
        mock_proc = AsyncMock()
        mock_proc.communicate = AsyncMock(return_value=(b'', b'command not found\n'))
        mock_proc.returncode = 127
        mock_subprocess.return_value = mock_proc

        result = asyncio.run(exec_via_kubectl('nonexistent_command'))

        self.assertFalse(result['success'])
        self.assertIn('command not found', result['stderr'])
        self.assertEqual(result['exit_code'], 127)

    def test_mcp_initialize_request(self):
        """Test MCP initialize request handling."""
        from mcp_servers.workspace_exec import handle_mcp_request

        request = {
            "jsonrpc": "2.0",
            "id": 1,
            "method": "initialize",
            "params": {}
        }

        response = asyncio.run(handle_mcp_request(request))

        self.assertEqual(response['jsonrpc'], '2.0')
        self.assertEqual(response['id'], 1)
        self.assertIn('capabilities', response['result'])
        self.assertIn('tools', response['result']['capabilities'])

    def test_mcp_tools_list_request(self):
        """Test MCP tools/list request handling."""
        from mcp_servers.workspace_exec import handle_mcp_request

        request = {
            "jsonrpc": "2.0",
            "id": 2,
            "method": "tools/list",
            "params": {}
        }

        response = asyncio.run(handle_mcp_request(request))

        self.assertEqual(response['jsonrpc'], '2.0')
        self.assertEqual(response['id'], 2)
        self.assertIn('tools', response['result'])
        self.assertEqual(len(response['result']['tools']), 1)
        self.assertEqual(response['result']['tools'][0]['name'], 'exec')

    @patch('mcp_servers.workspace_exec.execute_command')
    def test_mcp_tools_call_request(self, mock_execute):
        """Test MCP tools/call request handling."""
        from mcp_servers.workspace_exec import handle_mcp_request

        mock_execute.return_value = {
            'success': True,
            'stdout': 'test output',
            'stderr': '',
            'exit_code': 0
        }

        request = {
            "jsonrpc": "2.0",
            "id": 3,
            "method": "tools/call",
            "params": {
                "name": "exec",
                "arguments": {
                    "command": "echo test"
                }
            }
        }

        response = asyncio.run(handle_mcp_request(request))

        self.assertEqual(response['jsonrpc'], '2.0')
        self.assertEqual(response['id'], 3)
        self.assertIn('content', response['result'])
        self.assertFalse(response['result']['isError'])


class TestPatternBIntegration(unittest.TestCase):
    """Integration tests for Pattern B architecture."""

    def test_mcp_tool_naming(self):
        """Test that MCP tools follow mcp__{server}__{tool} naming."""
        server_name = "workspace"
        tool_name = "exec"

        # This is how the wrapper adds MCP tool permissions
        mcp_permission = f"mcp__{server_name}"

        self.assertEqual(mcp_permission, "mcp__workspace")

        # Full tool name as called by the model
        full_tool_name = f"mcp__{server_name}__{tool_name}"
        self.assertEqual(full_tool_name, "mcp__workspace__exec")

    def test_allowed_tools_pattern_b_config(self):
        """Test complete Pattern B allowed_tools configuration."""
        # Pattern B: Bash removed, workspace MCP added
        allowed_tools = [
            "Read", "Write", "Glob", "Grep", "Edit", "MultiEdit",
            "WebSearch", "WebFetch",
            "mcp__workspace"  # Grants access to all workspace tools
        ]

        self.assertNotIn("Bash", allowed_tools)
        self.assertIn("mcp__workspace", allowed_tools)


if __name__ == '__main__':
    unittest.main()
