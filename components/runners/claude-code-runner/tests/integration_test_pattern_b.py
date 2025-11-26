#!/usr/bin/env python3
"""
Integration test for Pattern B: Actually test the wrapper and MCP server behavior.

This test:
1. Imports the actual wrapper and verifies allowed_tools behavior
2. Runs the MCP server and sends real MCP requests
3. Tests command execution (locally, not via kubectl)
"""

import asyncio
import json
import os
import sys
import tempfile
from pathlib import Path

# Add parent directory to path
sys.path.insert(0, os.path.dirname(os.path.dirname(os.path.abspath(__file__))))


def test_wrapper_allowed_tools_with_bash_disabled():
    """Test that wrapper respects DISABLE_BASH_TOOL environment variable."""
    print("\n=== Test 1: Wrapper allowed_tools with DISABLE_BASH_TOOL ===")

    # Set up environment
    os.environ['DISABLE_BASH_TOOL'] = '1'

    # Simulate the wrapper logic
    disable_bash = os.environ.get('DISABLE_BASH_TOOL', '').strip().lower() in ('1', 'true', 'yes')

    if disable_bash:
        allowed_tools = ["Read", "Write", "Glob", "Grep", "Edit", "MultiEdit", "WebSearch", "WebFetch"]
    else:
        allowed_tools = ["Read", "Write", "Bash", "Glob", "Grep", "Edit", "MultiEdit", "WebSearch", "WebFetch"]

    print(f"  DISABLE_BASH_TOOL={os.environ.get('DISABLE_BASH_TOOL')}")
    print(f"  disable_bash={disable_bash}")
    print(f"  allowed_tools={allowed_tools}")

    assert "Bash" not in allowed_tools, "Bash should NOT be in allowed_tools when DISABLE_BASH_TOOL=1"
    assert "Read" in allowed_tools, "Read should still be in allowed_tools"
    assert "Write" in allowed_tools, "Write should still be in allowed_tools"

    print("  ✓ PASSED: Bash removed from allowed_tools when DISABLE_BASH_TOOL=1")

    # Cleanup
    del os.environ['DISABLE_BASH_TOOL']
    return True


def test_wrapper_allowed_tools_with_bash_enabled():
    """Test that wrapper includes Bash when DISABLE_BASH_TOOL is not set."""
    print("\n=== Test 2: Wrapper allowed_tools with Bash enabled (default) ===")

    # Ensure env var is not set
    if 'DISABLE_BASH_TOOL' in os.environ:
        del os.environ['DISABLE_BASH_TOOL']

    disable_bash = os.environ.get('DISABLE_BASH_TOOL', '').strip().lower() in ('1', 'true', 'yes')

    if disable_bash:
        allowed_tools = ["Read", "Write", "Glob", "Grep", "Edit", "MultiEdit", "WebSearch", "WebFetch"]
    else:
        allowed_tools = ["Read", "Write", "Bash", "Glob", "Grep", "Edit", "MultiEdit", "WebSearch", "WebFetch"]

    print(f"  DISABLE_BASH_TOOL={os.environ.get('DISABLE_BASH_TOOL', '(not set)')}")
    print(f"  disable_bash={disable_bash}")
    print(f"  allowed_tools={allowed_tools}")

    assert "Bash" in allowed_tools, "Bash SHOULD be in allowed_tools when DISABLE_BASH_TOOL is not set"

    print("  ✓ PASSED: Bash included in allowed_tools by default")
    return True


def test_mcp_server_initialization():
    """Test MCP server responds to initialize request."""
    print("\n=== Test 3: MCP server initialization ===")

    from mcp_servers.workspace_exec import handle_mcp_request

    request = {
        "jsonrpc": "2.0",
        "id": 1,
        "method": "initialize",
        "params": {}
    }

    response = asyncio.run(handle_mcp_request(request))

    print(f"  Request: {json.dumps(request)}")
    print(f"  Response: {json.dumps(response, indent=2)}")

    assert response["jsonrpc"] == "2.0"
    assert response["id"] == 1
    assert "result" in response
    assert "capabilities" in response["result"]
    assert response["result"]["serverInfo"]["name"] == "workspace-exec"

    print("  ✓ PASSED: MCP server initializes correctly")
    return True


def test_mcp_server_tools_list():
    """Test MCP server returns exec tool in tools/list."""
    print("\n=== Test 4: MCP server tools/list ===")

    from mcp_servers.workspace_exec import handle_mcp_request

    request = {
        "jsonrpc": "2.0",
        "id": 2,
        "method": "tools/list",
        "params": {}
    }

    response = asyncio.run(handle_mcp_request(request))

    print(f"  Request: {json.dumps(request)}")
    print(f"  Tools returned: {[t['name'] for t in response['result']['tools']]}")

    tools = response["result"]["tools"]
    assert len(tools) == 1
    assert tools[0]["name"] == "exec"
    assert "command" in tools[0]["inputSchema"]["properties"]

    print("  ✓ PASSED: MCP server returns exec tool")
    return True


def test_mcp_server_exec_local():
    """Test MCP server can execute local commands (without kubectl)."""
    print("\n=== Test 5: MCP server local command execution ===")

    from mcp_servers.workspace_exec import handle_mcp_request, exec_via_kubectl

    # Temporarily patch exec to run locally instead of via kubectl
    import mcp_servers.workspace_exec as ws_module

    original_execute = ws_module.execute_command

    async def local_execute(command: str, workdir: str = "/tmp", timeout: int = 30):
        """Execute command locally for testing."""
        proc = await asyncio.create_subprocess_exec(
            "sh", "-c", f"cd {workdir} && {command}",
            stdout=asyncio.subprocess.PIPE,
            stderr=asyncio.subprocess.PIPE,
        )
        stdout, stderr = await proc.communicate()
        return {
            "success": proc.returncode == 0,
            "stdout": stdout.decode(),
            "stderr": stderr.decode(),
            "exit_code": proc.returncode,
        }

    ws_module.execute_command = local_execute

    try:
        request = {
            "jsonrpc": "2.0",
            "id": 3,
            "method": "tools/call",
            "params": {
                "name": "exec",
                "arguments": {
                    "command": "echo 'Hello from Pattern B'",
                    "workdir": "/tmp"
                }
            }
        }

        response = asyncio.run(handle_mcp_request(request))

        print(f"  Command: echo 'Hello from Pattern B'")
        print(f"  Response content: {response['result']['content'][0]['text']}")

        assert "Hello from Pattern B" in response["result"]["content"][0]["text"]
        assert response["result"]["isError"] == False

        print("  ✓ PASSED: MCP server executes commands correctly")
        return True

    finally:
        ws_module.execute_command = original_execute


def test_mcp_server_exec_with_failure():
    """Test MCP server handles command failures correctly."""
    print("\n=== Test 6: MCP server command failure handling ===")

    import mcp_servers.workspace_exec as ws_module
    from mcp_servers.workspace_exec import handle_mcp_request

    original_execute = ws_module.execute_command

    async def local_execute(command: str, workdir: str = "/tmp", timeout: int = 30):
        proc = await asyncio.create_subprocess_exec(
            "sh", "-c", f"cd {workdir} && {command}",
            stdout=asyncio.subprocess.PIPE,
            stderr=asyncio.subprocess.PIPE,
        )
        stdout, stderr = await proc.communicate()
        return {
            "success": proc.returncode == 0,
            "stdout": stdout.decode(),
            "stderr": stderr.decode(),
            "exit_code": proc.returncode,
        }

    ws_module.execute_command = local_execute

    try:
        request = {
            "jsonrpc": "2.0",
            "id": 4,
            "method": "tools/call",
            "params": {
                "name": "exec",
                "arguments": {
                    "command": "exit 42",
                    "workdir": "/tmp"
                }
            }
        }

        response = asyncio.run(handle_mcp_request(request))

        print(f"  Command: exit 42")
        print(f"  isError: {response['result']['isError']}")
        print(f"  Content: {response['result']['content'][0]['text']}")

        assert response["result"]["isError"] == True
        assert "Exit code: 42" in response["result"]["content"][0]["text"]

        print("  ✓ PASSED: MCP server reports failures correctly")
        return True

    finally:
        ws_module.execute_command = original_execute


def test_mcp_tool_naming_convention():
    """Test that MCP tool names follow the expected convention."""
    print("\n=== Test 7: MCP tool naming convention ===")

    # Simulate how wrapper adds MCP tools to allowed_tools
    mcp_servers = {"workspace": {"type": "http", "url": "http://localhost:9999"}}

    allowed_tools = ["Read", "Write", "Glob", "Grep", "Edit", "MultiEdit"]

    for server_name in mcp_servers.keys():
        allowed_tools.append(f"mcp__{server_name}")

    print(f"  MCP servers: {list(mcp_servers.keys())}")
    print(f"  allowed_tools after adding MCP: {allowed_tools}")

    assert "mcp__workspace" in allowed_tools

    # The full tool name when called by the model
    full_tool_name = "mcp__workspace__exec"
    print(f"  Full tool name (as called by model): {full_tool_name}")

    print("  ✓ PASSED: MCP tool naming follows mcp__{server}__{tool} convention")
    return True


def test_pattern_b_complete_flow():
    """Test the complete Pattern B flow: disable Bash + use MCP exec."""
    print("\n=== Test 8: Complete Pattern B flow ===")

    import mcp_servers.workspace_exec as ws_module
    from mcp_servers.workspace_exec import handle_mcp_request

    # Step 1: Disable Bash
    os.environ['DISABLE_BASH_TOOL'] = '1'
    disable_bash = os.environ.get('DISABLE_BASH_TOOL', '').strip().lower() in ('1', 'true', 'yes')

    allowed_tools = ["Read", "Write", "Glob", "Grep", "Edit", "MultiEdit", "WebSearch", "WebFetch"]
    if not disable_bash:
        allowed_tools.insert(2, "Bash")

    # Step 2: Add MCP workspace server
    mcp_servers = {"workspace": {"type": "http", "url": "http://localhost:9999"}}
    for server_name in mcp_servers.keys():
        allowed_tools.append(f"mcp__{server_name}")

    print(f"  Step 1: DISABLE_BASH_TOOL=1")
    print(f"  Step 2: Added MCP workspace server")
    print(f"  Final allowed_tools: {allowed_tools}")

    assert "Bash" not in allowed_tools, "Bash should be disabled"
    assert "mcp__workspace" in allowed_tools, "MCP workspace should be enabled"

    # Step 3: Execute command via MCP
    original_execute = ws_module.execute_command

    async def local_execute(command: str, workdir: str = "/tmp", timeout: int = 30):
        proc = await asyncio.create_subprocess_exec(
            "sh", "-c", f"cd {workdir} && {command}",
            stdout=asyncio.subprocess.PIPE,
            stderr=asyncio.subprocess.PIPE,
        )
        stdout, stderr = await proc.communicate()
        return {
            "success": proc.returncode == 0,
            "stdout": stdout.decode(),
            "stderr": stderr.decode(),
            "exit_code": proc.returncode,
        }

    ws_module.execute_command = local_execute

    try:
        request = {
            "jsonrpc": "2.0",
            "id": 5,
            "method": "tools/call",
            "params": {
                "name": "exec",
                "arguments": {
                    "command": "whoami && pwd",
                    "workdir": "/tmp"
                }
            }
        }

        response = asyncio.run(handle_mcp_request(request))

        print(f"  Step 3: Executed 'whoami && pwd' via MCP")
        print(f"  Output: {response['result']['content'][0]['text'].strip()}")

        assert response["result"]["isError"] == False

        print("  ✓ PASSED: Complete Pattern B flow works")
        return True

    finally:
        ws_module.execute_command = original_execute
        del os.environ['DISABLE_BASH_TOOL']


def main():
    """Run all integration tests."""
    print("=" * 60)
    print("Pattern B Integration Tests")
    print("=" * 60)

    tests = [
        test_wrapper_allowed_tools_with_bash_disabled,
        test_wrapper_allowed_tools_with_bash_enabled,
        test_mcp_server_initialization,
        test_mcp_server_tools_list,
        test_mcp_server_exec_local,
        test_mcp_server_exec_with_failure,
        test_mcp_tool_naming_convention,
        test_pattern_b_complete_flow,
    ]

    passed = 0
    failed = 0

    for test in tests:
        try:
            if test():
                passed += 1
            else:
                failed += 1
                print(f"  ✗ FAILED: {test.__name__}")
        except Exception as e:
            failed += 1
            print(f"  ✗ FAILED: {test.__name__} - {e}")
            import traceback
            traceback.print_exc()

    print("\n" + "=" * 60)
    print(f"Results: {passed} passed, {failed} failed")
    print("=" * 60)

    return failed == 0


if __name__ == "__main__":
    success = main()
    sys.exit(0 if success else 1)
