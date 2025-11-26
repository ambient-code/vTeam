#!/usr/bin/env python3
"""
MCP Server for executing commands in a workspace container.

This server provides a replacement for the built-in Bash tool when running
in Pattern B (separate agent container) mode. Commands are executed in the
workspace container via either:
1. kubectl exec (requires K8s API access)
2. nsenter via shared process namespace (requires SYS_PTRACE)

Usage:
    # As HTTP server (for MCP over HTTP)
    python -m mcp_servers.workspace_exec --mode http --port 9999

    # As stdio server (for MCP over stdio)
    python -m mcp_servers.workspace_exec --mode stdio

Environment Variables:
    POD_NAME: Name of the current pod
    POD_NAMESPACE: Kubernetes namespace
    WORKSPACE_CONTAINER: Name of workspace container (default: "workspace")
    EXEC_MODE: "kubectl" or "nsenter" (default: "kubectl")
    WORKSPACE_PID: PID of workspace process (required for nsenter mode)
"""

import asyncio
import json
import os
import sys
import logging
from typing import Optional

logging.basicConfig(level=logging.INFO, format='%(asctime)s - %(name)s - %(levelname)s - %(message)s')
logger = logging.getLogger(__name__)

# Configuration from environment
POD_NAME = os.environ.get("POD_NAME", "")
POD_NAMESPACE = os.environ.get("POD_NAMESPACE", "default")
WORKSPACE_CONTAINER = os.environ.get("WORKSPACE_CONTAINER", "workspace")
EXEC_MODE = os.environ.get("EXEC_MODE", "kubectl")  # "kubectl" or "nsenter"
WORKSPACE_PID: Optional[int] = None

# Try to get workspace PID from environment or discover it
_workspace_pid_env = os.environ.get("WORKSPACE_PID", "")
if _workspace_pid_env:
    try:
        WORKSPACE_PID = int(_workspace_pid_env)
    except ValueError:
        pass


async def discover_workspace_pid() -> Optional[int]:
    """Discover workspace container's PID via /proc when using shared process namespace."""
    global WORKSPACE_PID
    if WORKSPACE_PID:
        return WORKSPACE_PID

    try:
        # Look for the sleep infinity process that workspace runs
        proc = await asyncio.create_subprocess_exec(
            "sh", "-c", "ps aux | grep 'sleep infinity' | grep -v grep | awk '{print $2}' | head -1",
            stdout=asyncio.subprocess.PIPE,
            stderr=asyncio.subprocess.PIPE,
        )
        stdout, _ = await proc.communicate()
        pid_str = stdout.decode().strip()
        if pid_str:
            WORKSPACE_PID = int(pid_str)
            logger.info(f"Discovered workspace PID: {WORKSPACE_PID}")
            return WORKSPACE_PID
    except Exception as e:
        logger.warning(f"Failed to discover workspace PID: {e}")

    return None


async def exec_via_kubectl(command: str, workdir: str = "/workspace", timeout: int = 300) -> dict:
    """Execute command in workspace container via kubectl exec."""
    if not POD_NAME:
        return {
            "success": False,
            "error": "POD_NAME environment variable not set",
            "stdout": "",
            "stderr": "",
            "exit_code": 1,
        }

    # Build kubectl exec command
    kubectl_cmd = [
        "kubectl", "exec",
        "-n", POD_NAMESPACE,
        POD_NAME,
        "-c", WORKSPACE_CONTAINER,
        "--",
        "sh", "-c", f"cd {workdir} && {command}"
    ]

    logger.info(f"Executing via kubectl: {command[:100]}...")

    try:
        proc = await asyncio.create_subprocess_exec(
            *kubectl_cmd,
            stdout=asyncio.subprocess.PIPE,
            stderr=asyncio.subprocess.PIPE,
        )

        try:
            stdout, stderr = await asyncio.wait_for(proc.communicate(), timeout=timeout)
        except asyncio.TimeoutError:
            proc.kill()
            await proc.wait()
            return {
                "success": False,
                "error": f"Command timed out after {timeout} seconds",
                "stdout": "",
                "stderr": "",
                "exit_code": -1,
            }

        return {
            "success": proc.returncode == 0,
            "stdout": stdout.decode("utf-8", errors="replace"),
            "stderr": stderr.decode("utf-8", errors="replace"),
            "exit_code": proc.returncode,
        }

    except Exception as e:
        logger.error(f"kubectl exec failed: {e}")
        return {
            "success": False,
            "error": str(e),
            "stdout": "",
            "stderr": "",
            "exit_code": 1,
        }


async def exec_via_nsenter(command: str, workdir: str = "/workspace", timeout: int = 300) -> dict:
    """Execute command in workspace container via nsenter (shared process namespace)."""
    pid = await discover_workspace_pid()
    if not pid:
        return {
            "success": False,
            "error": "Could not discover workspace PID. Ensure shareProcessNamespace is enabled.",
            "stdout": "",
            "stderr": "",
            "exit_code": 1,
        }

    # Use nsenter to enter the workspace's mount and PID namespace
    nsenter_cmd = [
        "nsenter",
        "-t", str(pid),
        "-m", "-p",  # Mount and PID namespaces
        "--",
        "sh", "-c", f"cd {workdir} && {command}"
    ]

    logger.info(f"Executing via nsenter (PID {pid}): {command[:100]}...")

    try:
        proc = await asyncio.create_subprocess_exec(
            *nsenter_cmd,
            stdout=asyncio.subprocess.PIPE,
            stderr=asyncio.subprocess.PIPE,
        )

        try:
            stdout, stderr = await asyncio.wait_for(proc.communicate(), timeout=timeout)
        except asyncio.TimeoutError:
            proc.kill()
            await proc.wait()
            return {
                "success": False,
                "error": f"Command timed out after {timeout} seconds",
                "stdout": "",
                "stderr": "",
                "exit_code": -1,
            }

        return {
            "success": proc.returncode == 0,
            "stdout": stdout.decode("utf-8", errors="replace"),
            "stderr": stderr.decode("utf-8", errors="replace"),
            "exit_code": proc.returncode,
        }

    except Exception as e:
        logger.error(f"nsenter exec failed: {e}")
        return {
            "success": False,
            "error": str(e),
            "stdout": "",
            "stderr": "",
            "exit_code": 1,
        }


async def execute_command(command: str, workdir: str = "/workspace", timeout: int = 300) -> dict:
    """Execute command using configured execution mode."""
    if EXEC_MODE == "nsenter":
        return await exec_via_nsenter(command, workdir, timeout)
    else:
        return await exec_via_kubectl(command, workdir, timeout)


# MCP Tool definitions
TOOLS = [
    {
        "name": "exec",
        "description": """Execute a shell command in the workspace container.

This tool runs commands in the user's workspace environment, which may have
different tools installed than the agent container. Use this for:
- Running build commands (npm, cargo, make, etc.)
- Executing tests
- Git operations
- Any shell command that needs the workspace's installed tools

The command runs in a shell (sh -c) so you can use pipes, redirects, etc.""",
        "inputSchema": {
            "type": "object",
            "properties": {
                "command": {
                    "type": "string",
                    "description": "The shell command to execute"
                },
                "workdir": {
                    "type": "string",
                    "description": "Working directory (default: /workspace)",
                    "default": "/workspace"
                },
                "timeout": {
                    "type": "integer",
                    "description": "Timeout in seconds (default: 300)",
                    "default": 300
                }
            },
            "required": ["command"]
        }
    }
]


async def handle_mcp_request(request: dict) -> dict:
    """Handle an MCP JSON-RPC request."""
    method = request.get("method", "")
    params = request.get("params", {})
    req_id = request.get("id")

    if method == "initialize":
        return {
            "jsonrpc": "2.0",
            "id": req_id,
            "result": {
                "protocolVersion": "2024-11-05",
                "capabilities": {
                    "tools": {}
                },
                "serverInfo": {
                    "name": "workspace-exec",
                    "version": "0.1.0"
                }
            }
        }

    elif method == "tools/list":
        return {
            "jsonrpc": "2.0",
            "id": req_id,
            "result": {
                "tools": TOOLS
            }
        }

    elif method == "tools/call":
        tool_name = params.get("name", "")
        arguments = params.get("arguments", {})

        if tool_name == "exec":
            command = arguments.get("command", "")
            workdir = arguments.get("workdir", "/workspace")
            timeout = arguments.get("timeout", 300)

            result = await execute_command(command, workdir, timeout)

            # Format output for MCP
            output = result["stdout"]
            if result["stderr"]:
                output += f"\n[STDERR]\n{result['stderr']}"
            if not result["success"]:
                output += f"\n[Exit code: {result['exit_code']}]"
                if result.get("error"):
                    output += f"\n[Error: {result['error']}]"

            return {
                "jsonrpc": "2.0",
                "id": req_id,
                "result": {
                    "content": [
                        {
                            "type": "text",
                            "text": output
                        }
                    ],
                    "isError": not result["success"]
                }
            }
        else:
            return {
                "jsonrpc": "2.0",
                "id": req_id,
                "error": {
                    "code": -32601,
                    "message": f"Unknown tool: {tool_name}"
                }
            }

    elif method == "notifications/initialized":
        # Notification, no response needed
        return None

    else:
        return {
            "jsonrpc": "2.0",
            "id": req_id,
            "error": {
                "code": -32601,
                "message": f"Method not found: {method}"
            }
        }


async def run_stdio_server():
    """Run MCP server over stdio."""
    logger.info("Starting workspace-exec MCP server (stdio mode)")

    reader = asyncio.StreamReader()
    protocol = asyncio.StreamReaderProtocol(reader)
    await asyncio.get_event_loop().connect_read_pipe(lambda: protocol, sys.stdin)

    writer_transport, writer_protocol = await asyncio.get_event_loop().connect_write_pipe(
        asyncio.streams.FlowControlMixin, sys.stdout
    )
    writer = asyncio.StreamWriter(writer_transport, writer_protocol, reader, asyncio.get_event_loop())

    while True:
        try:
            line = await reader.readline()
            if not line:
                break

            request = json.loads(line.decode())
            response = await handle_mcp_request(request)

            if response:
                writer.write((json.dumps(response) + "\n").encode())
                await writer.drain()

        except json.JSONDecodeError as e:
            logger.error(f"Invalid JSON: {e}")
        except Exception as e:
            logger.error(f"Error handling request: {e}")


async def run_http_server(port: int = 9999):
    """Run MCP server over HTTP."""
    from aiohttp import web

    async def handle_mcp(request: web.Request) -> web.Response:
        try:
            data = await request.json()
            response = await handle_mcp_request(data)
            if response:
                return web.json_response(response)
            return web.Response(status=204)
        except Exception as e:
            logger.error(f"HTTP handler error: {e}")
            return web.json_response({
                "jsonrpc": "2.0",
                "id": None,
                "error": {"code": -32603, "message": str(e)}
            }, status=500)

    app = web.Application()
    app.router.add_post("/mcp", handle_mcp)
    app.router.add_post("/", handle_mcp)

    runner = web.AppRunner(app)
    await runner.setup()
    site = web.TCPSite(runner, "0.0.0.0", port)
    await site.start()

    logger.info(f"Starting workspace-exec MCP server (HTTP mode) on port {port}")

    # Keep running
    while True:
        await asyncio.sleep(3600)


def main():
    import argparse

    parser = argparse.ArgumentParser(description="Workspace exec MCP server")
    parser.add_argument("--mode", choices=["stdio", "http"], default="stdio",
                        help="Server mode (default: stdio)")
    parser.add_argument("--port", type=int, default=9999,
                        help="HTTP port (default: 9999)")

    args = parser.parse_args()

    if args.mode == "http":
        asyncio.run(run_http_server(args.port))
    else:
        asyncio.run(run_stdio_server())


if __name__ == "__main__":
    main()
