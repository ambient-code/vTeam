#!/usr/bin/env python3
"""
Claude Code CLI wrapper for runner-shell integration.
Bridges the existing Claude Code CLI with the standardized runner-shell framework.
"""

import asyncio
import os
import sys
import logging
import json as _json
import re
from pathlib import Path
from urllib.parse import urlparse, urlunparse
from urllib import request as _urllib_request, error as _urllib_error

# Add runner-shell to Python path
sys.path.insert(0, '/app/runner-shell')

from runner_shell.core.shell import RunnerShell
from runner_shell.core.protocol import MessageType, SessionStatus, PartialInfo
from runner_shell.core.context import RunnerContext


class ClaudeCodeAdapter:
    """Adapter that wraps the existing Claude Code CLI for runner-shell."""

    def __init__(self):
        self.context = None
        self.shell = None
        self.claude_process = None
        self._incoming_queue: "asyncio.Queue[dict]" = asyncio.Queue()

    async def initialize(self, context: RunnerContext):
        """Initialize the adapter with context."""
        self.context = context
        logging.info(f"Initialized Claude Code adapter for session {context.session_id}")
        # Prepare workspace from input repo if provided
        await self._prepare_workspace()
        # Validate prerequisite files exist for phase-based commands
        await self._validate_prerequisites()

    async def run(self):
        """Run the Claude Code CLI session."""
        try:
            # Wait for WebSocket connection to be established before sending messages
            # The shell.start() call happens before this method, but the WS connection is async
            # and may not be ready yet. Retry first message send to ensure connection is up.
            await self._wait_for_ws_connection()

            # Get prompt from environment
            prompt = self.context.get_env("PROMPT", "")
            if not prompt:
                prompt = self.context.get_metadata("prompt", "Hello! How can I help you today?")

            # Send progress update
            await self._send_log("Starting Claude Code session...")

            # Mark CR Running (best-effort)
            try:
                await self._update_cr_status({
                    "phase": "Running",
                    "message": "Runner started",
                })
            except Exception as _:
                logging.debug("CR status update (Running) skipped")


            # Append token to websocket URL if available (to pass SA token to backend)
            try:
                if self.shell and getattr(self.shell, 'transport', None):
                    ws = getattr(self.shell.transport, 'url', '') or ''
                    bot = (os.getenv('BOT_TOKEN') or '').strip()
                    if bot and ws and '?' not in ws:
                        # Safe to append token as query for backend to map into Authorization
                        setattr(self.shell.transport, 'url', ws + f"?token={bot}")
            except Exception:
                pass

            # Execute Claude Code CLI (interactive or one-shot based on env)
            result = await self._run_claude_agent_sdk(prompt)

            # Send completion
            await self._send_log("Claude Code session completed")
            

            # Optional auto-push on completion (default: disabled)
            try:
                auto_push = str(self.context.get_env('AUTO_PUSH_ON_COMPLETE', 'false')).strip().lower() in ('1','true','yes')
            except Exception:
                auto_push = False
            if auto_push:
                await self._push_results_if_any()

            # CR status update based on result - MUST complete before pod exits
            try:
                if isinstance(result, dict) and result.get("success"):
                    logging.info(f"Updating CR status to Completed (result.success={result.get('success')})")
                    result_summary = ""
                    if isinstance(result.get("result"), dict):
                        # Prefer subtype and output if present
                        subtype = result["result"].get("subtype")
                        if subtype:
                            result_summary = f"Completed with subtype: {subtype}"
                    stdout_text = result.get("stdout") or ""
                    # Use BLOCKING call to ensure completion before container exits
                    await self._update_cr_status({
                        "phase": "Completed",
                        "completionTime": self._utc_iso(),
                        "message": "Runner completed",
                        "subtype": (result.get("result") or {}).get("subtype", "success"),
                        "is_error": False,
                        "num_turns": getattr(self, "_turn_count", 0),
                        "session_id": self.context.session_id,
                        "result": stdout_text[:10000],
                    }, blocking=True)
                    logging.info("CR status update to Completed completed")
                elif isinstance(result, dict) and not result.get("success"):
                    # Handle failure case (e.g., SDK crashed without ResultMessage)
                    error_msg = result.get("error", "Unknown error")
                    # Use BLOCKING call to ensure completion before container exits
                    await self._update_cr_status({
                        "phase": "Failed",
                        "completionTime": self._utc_iso(),
                        "message": error_msg,
                        "is_error": True,
                        "num_turns": getattr(self, "_turn_count", 0),
                        "session_id": self.context.session_id,
                    }, blocking=True)
            except Exception as e:
                logging.error(f"CR status update exception: {e}")

            return result

        except Exception as e:
            logging.error(f"Claude Code adapter failed: {e}")
            # Best-effort CR failure update
            try:
                await self._update_cr_status({
                    "phase": "Failed",
                    "completionTime": self._utc_iso(),
                    "message": f"Runner failed: {e}",
                    "is_error": True,
                    "session_id": self.context.session_id,
                })
            except Exception:
                logging.debug("CR status update (Failed) skipped")
            return {
                "success": False,
                "error": str(e)
            }

    async def _run_claude_agent_sdk(self, prompt: str):
        """Execute the Claude Code SDK with the given prompt."""
        try:
            from claude_code_sdk import ClaudeSDKClient, ClaudeCodeOptions
            api_key = self.context.get_env('ANTHROPIC_API_KEY', '')
            if not api_key:
                raise RuntimeError("ANTHROPIC_API_KEY is required for Claude Code SDK")

            # Determine cwd and additional dirs from multi-repo config
            repos_cfg = self._get_repos_config()
            cwd_path = self.context.workspace_path
            add_dirs = []
            if repos_cfg:
                # Prefer explicit MAIN_REPO_NAME, else use MAIN_REPO_INDEX, else default to 0
                main_name = (os.getenv('MAIN_REPO_NAME') or '').strip()
                if not main_name:
                    idx_raw = (os.getenv('MAIN_REPO_INDEX') or '').strip()
                    try:
                        idx_val = int(idx_raw) if idx_raw else 0
                    except Exception:
                        idx_val = 0
                    if idx_val < 0 or idx_val >= len(repos_cfg):
                        idx_val = 0
                    main_name = (repos_cfg[idx_val].get('name') or '').strip()
                # CWD becomes main repo folder under workspace
                if main_name:
                    cwd_path = str(Path(self.context.workspace_path) / main_name)
                # Add other repos as additional directories
                for r in repos_cfg:
                    name = (r.get('name') or '').strip()
                    if not name:
                        continue
                    p = str(Path(self.context.workspace_path) / name)
                    if p != cwd_path:
                        add_dirs.append(p)

            # Log working directory and additional directories for debugging
            logging.info(f"Claude SDK CWD: {cwd_path}")
            logging.info(f"Claude SDK additional directories: {add_dirs}")
            
            options = ClaudeCodeOptions(cwd=cwd_path, permission_mode="acceptEdits", allowed_tools=["Read","Write","Bash","Glob","Grep","Edit","MultiEdit","WebSearch","WebFetch"])
            # Best-effort set add_dirs if supported by SDK version
            try:
                if add_dirs:
                    options.add_dirs = add_dirs  # type: ignore[attr-defined]
            except Exception:
                pass
            # Model settings from both legacy and LLM_* envs
            model = self.context.get_env('LLM_MODEL')
            if model:
                try:
                    options.model = model  # type: ignore[attr-defined]
                except Exception:
                    pass
            max_tokens_env = (
                self.context.get_env('LLM_MAX_TOKENS') or
                self.context.get_env('MAX_TOKENS')
            )
            if max_tokens_env:
                try:
                    options.max_tokens = int(max_tokens_env)  # type: ignore[attr-defined]
                except Exception:
                    pass
            temperature_env = (
                self.context.get_env('LLM_TEMPERATURE') or
                self.context.get_env('TEMPERATURE')
            )
            if temperature_env:
                try:
                    options.temperature = float(temperature_env)  # type: ignore[attr-defined]
                except Exception:
                    pass

            os.environ['ANTHROPIC_API_KEY'] = api_key

            result_payload = None
            self._turn_count = 0
            # Import SDK message and content types for accurate mapping
            from claude_code_sdk import (
                AssistantMessage,
                UserMessage,
                SystemMessage,
                ResultMessage,
                TextBlock,
                ThinkingBlock,
                ToolUseBlock,
                ToolResultBlock,
            )
            # Determine interactive mode once for this run
            interactive = str(self.context.get_env('INTERACTIVE', 'false')).strip().lower() in ('1', 'true', 'yes')

            async def process_response_stream(client_obj):
                nonlocal result_payload
                async for message in client_obj.receive_response():
                    logging.info(f"[ClaudeSDKClient]: {message}")

                    if isinstance(message, (AssistantMessage, UserMessage)):
                        for block in getattr(message, 'content', []) or []:
                            if isinstance(block, TextBlock):
                                text_piece = getattr(block, 'text', None)
                                if text_piece:
                                    await self.shell._send_message(
                                        MessageType.AGENT_MESSAGE,
                                        {"type": "agent_message", "content": {"type": "text_block", "text": text_piece}},
                                    )
                            elif isinstance(block, ToolUseBlock):
                                tool_name = getattr(block, 'name', '') or 'unknown'
                                tool_input = getattr(block, 'input', {}) or {}
                                tool_id = getattr(block, 'id', None)
                                await self.shell._send_message(
                                    MessageType.AGENT_MESSAGE,
                                    {"tool": tool_name, "input": tool_input, "id": tool_id},
                                )
                                self._turn_count += 1
                            elif isinstance(block, ToolResultBlock):
                                tool_use_id = getattr(block, 'tool_use_id', None)
                                content = getattr(block, 'content', None)
                                is_error = getattr(block, 'is_error', None)
                                result_text = getattr(block, 'text', None)

                                await self.shell._send_message(
                                    MessageType.AGENT_MESSAGE,
                                    {
                                        "tool_result": {
                                            "tool_use_id": tool_use_id,
                                            "content": content if content is not None else result_text,
                                            "is_error": is_error,
                                        }
                                    },
                                )
                                if interactive:
                                    await self.shell._send_message(MessageType.WAITING_FOR_INPUT, {})
                                self._turn_count += 1
                            elif isinstance(block, ThinkingBlock):
                                await self._send_log({"level": "debug", "message": "Model is reasoning..."})
                    elif isinstance(message, (SystemMessage)):
                        text = getattr(message, 'text', None)
                        if text:
                            await self._send_log({"level": "debug", "message": str(text)})
                    elif isinstance(message, (ResultMessage)):
                        # Only surface result envelope to UI in non-interactive mode
                        result_payload = {
                            "subtype": getattr(message, 'subtype', None),
                            "duration_ms": getattr(message, 'duration_ms', None),
                            "duration_api_ms": getattr(message, 'duration_api_ms', None),
                            "is_error": getattr(message, 'is_error', None),
                            "num_turns": getattr(message, 'num_turns', None),
                            "session_id": getattr(message, 'session_id', None),
                            "total_cost_usd": getattr(message, 'total_cost_usd', None),
                            "usage": getattr(message, 'usage', None),
                            "result": getattr(message, 'result', None),
                        }
                        if not interactive:
                            await self.shell._send_message(
                                MessageType.AGENT_MESSAGE,
                                {"type": "result.message", "payload": result_payload},
                            )

            async with ClaudeSDKClient(options=options) as client:
                async def process_one_prompt(text: str):
                    await self.shell._send_message(MessageType.AGENT_RUNNING, {})
                    await client.query(text)
                    await process_response_stream(client)

                # Initial prompt (if any)
                if prompt and prompt.strip():
                    await process_one_prompt(prompt)

                if interactive:
                    await self._send_log({"level": "system", "message": "Chat ready"})
                    # Consume incoming user messages until end_session
                    while True:
                        incoming = await self._incoming_queue.get()
                        # Normalize mtype: backend can send 'user_message' or 'user.message'
                        mtype_raw = str(incoming.get('type') or '').strip()
                        mtype = mtype_raw.replace('.', '_')
                        payload = incoming.get('payload') or {}
                        if mtype in ('user_message', 'user_message'):
                            text = str(payload.get('content') or payload.get('text') or '').strip()
                            if text:
                                await process_one_prompt(text)
                        elif mtype in ('end_session', 'terminate', 'stop'):
                            await self._send_log({"level": "system", "message": "interactive.ended"})
                            break
                        elif mtype == 'interrupt':
                            try:
                                await client.interrupt()  # type: ignore[attr-defined]
                                await self._send_log({"level": "info", "message": "interrupt.sent"})
                            except Exception as e:
                                await self._send_log({"level": "warn", "message": f"interrupt.failed: {e}"})
                        else:
                            await self._send_log({"level": "debug", "message": f"ignored.message: {mtype_raw}"})

            # Note: All output is streamed via WebSocket, not collected here
            await self._check_pr_intent("")

            # Return success - result_payload may be None if SDK didn't send ResultMessage
            # (which can happen legitimately for some operations like git push)
            return {
                "success": True,
                "result": result_payload,
                "returnCode": 0,
                "stdout": "",
                "stderr": ""
            }
        except Exception as e:
            logging.error(f"Failed to run Claude Code SDK: {e}")
            return {
                "success": False,
                "error": str(e)
            }

    async def _prepare_workspace(self):
        """Clone input repo/branch into workspace and configure git remotes."""
        token = os.getenv("GITHUB_TOKEN") or await self._fetch_github_token()
        workspace = Path(self.context.workspace_path)
        workspace.mkdir(parents=True, exist_ok=True)

        repos_cfg = self._get_repos_config()
        if repos_cfg:
            # Multi-repo: clone each into workspace/<name>
            try:
                for r in repos_cfg:
                    name = (r.get('name') or '').strip()
                    inp = r.get('input') or {}
                    url = (inp.get('url') or '').strip()
                    branch = (inp.get('branch') or '').strip() or 'main'
                    if not name or not url:
                        continue
                    repo_dir = workspace / name
                    if not repo_dir.exists():
                        await self._send_log(f"Cloning {name}...")
                        clone_url = self._url_with_token(url, token) if token else url
                        await self._run_cmd(["git", "clone", "--branch", branch, "--single-branch", clone_url, str(repo_dir)], cwd=str(workspace))
                    else:
                        # Fetch/reset existing repo
                        await self._run_cmd(["git", "remote", "set-url", "origin", self._url_with_token(url, token) if token else url], cwd=str(repo_dir), ignore_errors=True)
                        await self._run_cmd(["git", "fetch", "origin", branch], cwd=str(repo_dir))
                        await self._run_cmd(["git", "checkout", branch], cwd=str(repo_dir))
                        await self._run_cmd(["git", "reset", "--hard", f"origin/{branch}"], cwd=str(repo_dir))

                    # Git identity with fallbacks
                    user_name = os.getenv("GIT_USER_NAME", "").strip() or "Ambient Code Bot"
                    user_email = os.getenv("GIT_USER_EMAIL", "").strip() or "bot@ambient-code.local"
                    await self._run_cmd(["git", "config", "user.name", user_name], cwd=str(repo_dir))
                    await self._run_cmd(["git", "config", "user.email", user_email], cwd=str(repo_dir))
                    logging.info(f"Git identity configured: {user_name} <{user_email}>")

                    # Configure output remote if present
                    out = r.get('output') or {}
                    out_url_raw = (out.get('url') or '').strip()
                    if out_url_raw:
                        out_url = self._url_with_token(out_url_raw, token) if token else out_url_raw
                        await self._run_cmd(["git", "remote", "remove", "output"], cwd=str(repo_dir), ignore_errors=True)
                        await self._run_cmd(["git", "remote", "add", "output", out_url], cwd=str(repo_dir))
            except Exception as e:
                logging.error(f"Failed to prepare multi-repo workspace: {e}")
                await self._send_log(f"Workspace preparation failed: {e}")
            return

        # Single-repo legacy flow
        input_repo = os.getenv("INPUT_REPO_URL", "").strip()
        if not input_repo:
            return
        input_branch = os.getenv("INPUT_BRANCH", "").strip() or "main"
        output_repo = os.getenv("OUTPUT_REPO_URL", "").strip()
        try:
            if not (workspace / ".git").exists():
                await self._send_log("Cloning input repository...")
                clone_url = self._url_with_token(input_repo, token) if token else input_repo
                await self._run_cmd(["git", "clone", "--branch", input_branch, "--single-branch", clone_url, str(workspace)], cwd=str(workspace.parent))
            else:
                await self._run_cmd(["git", "remote", "set-url", "origin", self._url_with_token(input_repo, token) if token else input_repo], cwd=str(workspace))
                await self._run_cmd(["git", "fetch", "origin", input_branch], cwd=str(workspace))
                await self._run_cmd(["git", "checkout", input_branch], cwd=str(workspace))
                await self._run_cmd(["git", "reset", "--hard", f"origin/{input_branch}"], cwd=str(workspace))

            # Git identity with fallbacks
            user_name = os.getenv("GIT_USER_NAME", "").strip() or "Ambient Code Bot"
            user_email = os.getenv("GIT_USER_EMAIL", "").strip() or "bot@ambient-code.local"
            await self._run_cmd(["git", "config", "user.name", user_name], cwd=str(workspace))
            await self._run_cmd(["git", "config", "user.email", user_email], cwd=str(workspace))
            logging.info(f"Git identity configured: {user_name} <{user_email}>")

            if output_repo:
                await self._send_log("Configuring output remote...")
                out_url = self._url_with_token(output_repo, token) if token else output_repo
                await self._run_cmd(["git", "remote", "remove", "output"], cwd=str(workspace), ignore_errors=True)
                await self._run_cmd(["git", "remote", "add", "output", out_url], cwd=str(workspace))

        except Exception as e:
            logging.error(f"Failed to prepare workspace: {e}")
            await self._send_log(f"Workspace preparation failed: {e}")

    async def _validate_prerequisites(self):
        """Validate prerequisite files exist for phase-based slash commands."""
        prompt = self.context.get_env("PROMPT", "")
        if not prompt:
            return

        # Extract slash command from prompt (e.g., "/plan", "/tasks", "/implement")
        prompt_lower = prompt.strip().lower()

        # Define prerequisite requirements
        prerequisites = {
            "/plan": ("spec.md", "Specification file (spec.md) not found. Please run /specify first to generate the specification."),
            "/tasks": ("plan.md", "Planning file (plan.md) not found. Please run /plan first to generate the implementation plan."),
            "/implement": ("tasks.md", "Tasks file (tasks.md) not found. Please run /tasks first to generate the task breakdown.")
        }

        # Check if prompt starts with a slash command that requires prerequisites
        for cmd, (required_file, error_msg) in prerequisites.items():
            if prompt_lower.startswith(cmd):
                # Search for the required file in workspace
                workspace = Path(self.context.workspace_path)
                found = False

                # Check in main workspace
                if (workspace / required_file).exists():
                    found = True
                    break

                # Check in multi-repo subdirectories (specs/XXX-feature-name/)
                for subdir in workspace.rglob("specs/*/"):
                    if (subdir / required_file).exists():
                        found = True
                        break

                if not found:
                    error_message = f"❌ {error_msg}"
                    await self._send_log(error_message)
                    # Mark session as failed
                    try:
                        await self._update_cr_status({
                            "phase": "Failed",
                            "completionTime": self._utc_iso(),
                            "message": error_msg,
                            "is_error": True,
                        })
                    except Exception:
                        logging.debug("CR status update (Failed) skipped")
                    raise RuntimeError(error_msg)

                break  # Only check the first matching command


    async def _push_results_if_any(self):
        """Commit and push changes to output repo/branch if configured."""
        # Get GitHub token once for all repos
        token = os.getenv("GITHUB_TOKEN") or await self._fetch_github_token()
        if token:
            logging.info("GitHub token obtained for push operations")
        else:
            logging.warning("No GitHub token available - push may fail for private repos")
        
        repos_cfg = self._get_repos_config()
        if repos_cfg:
            # Multi-repo flow
            try:
                for r in repos_cfg:
                    name = (r.get('name') or '').strip()
                    if not name:
                        continue
                    repo_dir = Path(self.context.workspace_path) / name
                    status = await self._run_cmd(["git", "status", "--porcelain"], cwd=str(repo_dir), capture_stdout=True)
                    if not status.strip():
                        logging.info(f"No changes detected for {name}, skipping push")
                        continue
                    
                    out = r.get('output') or {}
                    out_url_raw = (out.get('url') or '').strip()
                    if not out_url_raw:
                        logging.warning(f"No output URL configured for {name}, skipping push")
                        continue
                    
                    # Add token to output URL
                    out_url = self._url_with_token(out_url_raw, token) if token else out_url_raw
                    
                    in_ = r.get('input') or {}
                    in_branch = (in_.get('branch') or '').strip()
                    out_branch = (out.get('branch') or '').strip() or f"sessions/{self.context.session_id}"

                    await self._send_log(f"Pushing changes for {name}...")
                    logging.info(f"Configuring output remote with authentication for {name}")
                    
                    # Reconfigure output remote with token before push
                    await self._run_cmd(["git", "remote", "remove", "output"], cwd=str(repo_dir), ignore_errors=True)
                    await self._run_cmd(["git", "remote", "add", "output", out_url], cwd=str(repo_dir))
                    
                    logging.info(f"Checking out branch {out_branch} for {name}")
                    await self._run_cmd(["git", "checkout", "-B", out_branch], cwd=str(repo_dir))
                    
                    logging.info(f"Staging all changes for {name}")
                    await self._run_cmd(["git", "add", "-A"], cwd=str(repo_dir))
                    
                    logging.info(f"Committing changes for {name}")
                    try:
                        await self._run_cmd(["git", "commit", "-m", f"Session {self.context.session_id}: update"], cwd=str(repo_dir))
                    except RuntimeError as e:
                        if "nothing to commit" in str(e).lower():
                            logging.info(f"No changes to commit for {name}")
                            continue
                        else:
                            logging.error(f"Commit failed for {name}: {e}")
                            raise
                    
                    # Verify we have a valid output remote
                    logging.info(f"Verifying output remote for {name}")
                    remotes_output = await self._run_cmd(["git", "remote", "-v"], cwd=str(repo_dir), capture_stdout=True)
                    logging.info(f"Git remotes for {name}:\n{self._redact_secrets(remotes_output)}")
                    
                    if "output" not in remotes_output:
                        raise RuntimeError(f"Output remote not configured for {name}")
                    
                    logging.info(f"Pushing to output remote: {out_branch} for {name}")
                    await self._send_log(f"Pushing {name} to {out_branch}...")
                    await self._run_cmd(["git", "push", "-u", "output", f"HEAD:{out_branch}"], cwd=str(repo_dir))
                    
                    logging.info(f"Push completed for {name}")
                    await self._send_log(f"✓ Push completed for {name}")

                    create_pr_flag = (os.getenv("CREATE_PR", "").strip().lower() == "true")
                    if create_pr_flag and in_branch and out_branch and out_branch != in_branch and out_url:
                        upstream_url = (in_.get('url') or '').strip() or out_url
                        target_branch = os.getenv("PR_TARGET_BRANCH", "").strip() or in_branch
                        try:
                            pr_url = await self._create_pull_request(upstream_repo=upstream_url, fork_repo=out_url, head_branch=out_branch, base_branch=target_branch)
                            if pr_url:
                                await self._send_log({"level": "info", "message": f"Pull request created for {name}: {pr_url}"})
                        except Exception as e:
                            await self._send_log({"level": "error", "message": f"PR creation failed for {name}: {e}"})
            except Exception as e:
                logging.error(f"Failed to push results: {e}")
                await self._send_log(f"Push failed: {e}")
            return

        # Single-repo legacy flow
        output_repo_raw = os.getenv("OUTPUT_REPO_URL", "").strip()
        if not output_repo_raw:
            logging.info("No OUTPUT_REPO_URL configured, skipping legacy single-repo push")
            return
        
        # Add token to output URL
        output_repo = self._url_with_token(output_repo_raw, token) if token else output_repo_raw
        
        output_branch = os.getenv("OUTPUT_BRANCH", "").strip() or f"sessions/{self.context.session_id}"
        input_repo = os.getenv("INPUT_REPO_URL", "").strip()
        input_branch = os.getenv("INPUT_BRANCH", "").strip()
        workspace = Path(self.context.workspace_path)
        try:
            status = await self._run_cmd(["git", "status", "--porcelain"], cwd=str(workspace), capture_stdout=True)
            if not status.strip():
                await self._send_log({"level": "system", "message": "No changes to push."})
                return
            
            await self._send_log("Committing and pushing changes...")
            logging.info("Configuring output remote with authentication")
            
            # Reconfigure output remote with token before push
            await self._run_cmd(["git", "remote", "remove", "output"], cwd=str(workspace), ignore_errors=True)
            await self._run_cmd(["git", "remote", "add", "output", output_repo], cwd=str(workspace))
            
            logging.info(f"Checking out branch {output_branch}")
            await self._run_cmd(["git", "checkout", "-B", output_branch], cwd=str(workspace))
            
            logging.info("Staging all changes")
            await self._run_cmd(["git", "add", "-A"], cwd=str(workspace))
            
            logging.info("Committing changes")
            try:
                await self._run_cmd(["git", "commit", "-m", f"Session {self.context.session_id}: update"], cwd=str(workspace))
            except RuntimeError as e:
                if "nothing to commit" in str(e).lower():
                    logging.info("No changes to commit")
                    await self._send_log({"level": "system", "message": "No new changes to commit."})
                    return
                else:
                    logging.error(f"Commit failed: {e}")
                    raise
            
            # Verify we have a valid output remote
            logging.info("Verifying output remote")
            remotes_output = await self._run_cmd(["git", "remote", "-v"], cwd=str(workspace), capture_stdout=True)
            logging.info(f"Git remotes:\n{self._redact_secrets(remotes_output)}")
            
            if "output" not in remotes_output:
                raise RuntimeError("Output remote not configured")
            
            logging.info(f"Pushing to output remote: {output_branch}")
            await self._send_log(f"Pushing to {output_branch}...")
            await self._run_cmd(["git", "push", "-u", "output", f"HEAD:{output_branch}"], cwd=str(workspace))
            
            logging.info("Push completed")
            await self._send_log("✓ Push completed")

            create_pr_flag = (os.getenv("CREATE_PR", "").strip().lower() == "true")
            if create_pr_flag and input_branch and output_branch and output_branch != input_branch:
                target_branch = os.getenv("PR_TARGET_BRANCH", "").strip() or input_branch
                try:
                    pr_url = await self._create_pull_request(upstream_repo=input_repo or output_repo, fork_repo=output_repo, head_branch=output_branch, base_branch=target_branch)
                    if pr_url:
                        await self._send_log({"level": "info", "message": f"Pull request created: {pr_url}"})
                except Exception as e:
                    await self._send_log({"level": "error", "message": f"PR creation failed: {e}"})
        except Exception as e:
            logging.error(f"Failed to push results: {e}")
            await self._send_log(f"Push failed: {e}")

    async def _create_pull_request(self, upstream_repo: str, fork_repo: str, head_branch: str, base_branch: str) -> str | None:
        """Create a GitHub Pull Request from fork_repo:head_branch into upstream_repo:base_branch.

        Returns the PR HTML URL on success, or None.
        """

        token = (os.getenv("GITHUB_TOKEN") or await self._fetch_github_token() or "").strip()
        if not token:
            raise RuntimeError("Missing token for PR creation")

        up_owner, up_name, up_host = self._parse_owner_repo(upstream_repo)
        fk_owner, fk_name, fk_host = self._parse_owner_repo(fork_repo)
        if not up_owner or not up_name or not fk_owner or not fk_name:
            raise RuntimeError("Invalid repository URLs for PR creation")

        # API base from upstream host
        api = self._github_api_base(up_host)
        # For cross-fork PRs, head must be in the form "owner:branch"
        is_same_repo = (up_owner == fk_owner and up_name == fk_name)
        head = head_branch if is_same_repo else f"{fk_owner}:{head_branch}"

        url = f"{api}/repos/{up_owner}/{up_name}/pulls"
        title = f"Changes from session {self.context.session_id[:8]}"
        body = {
            "title": title,
            "body": f"Automated changes from runner session {self.context.session_id}",
            "head": head,
            "base": base_branch,
        }

        # Use blocking urllib in a thread to avoid adding deps
        data = _json.dumps(body).encode("utf-8")
        req = _urllib_request.Request(url, data=data, headers={
            "Accept": "application/vnd.github+json",
            "Authorization": f"token {token}",
            "X-GitHub-Api-Version": "2022-11-28",
            "Content-Type": "application/json",
            "User-Agent": "vTeam-Runner",
        }, method="POST")

        loop = asyncio.get_event_loop()

        def _do_req():
            try:
                with _urllib_request.urlopen(req, timeout=15) as resp:
                    return resp.read().decode("utf-8", errors="replace")
            except _urllib_error.HTTPError as he:
                err_body = he.read().decode("utf-8", errors="replace")
                raise RuntimeError(f"GitHub PR create failed: HTTP {he.code}: {err_body}")
            except Exception as e:
                raise RuntimeError(str(e))

        resp_text = await loop.run_in_executor(None, _do_req)
        try:
            pr = _json.loads(resp_text)
            return pr.get("html_url") or None
        except Exception:
            return None

    def _parse_owner_repo(self, url: str) -> tuple[str, str, str]:
        """Return (owner, name, host) from various URL formats."""
        s = (url or "").strip()
        s = s.removesuffix(".git")
        host = "github.com"
        try:
            if s.startswith("http://") or s.startswith("https://"):
                p = urlparse(s)
                host = p.netloc
                parts = [p for p in p.path.split("/") if p]
                if len(parts) >= 2:
                    return parts[0], parts[1], host
            if s.startswith("git@") or ":" in s:
                # Normalize SSH like git@host:owner/repo
                s2 = s
                if s2.startswith("git@"):
                    s2 = s2.replace(":", "/", 1)
                    s2 = s2.replace("git@", "ssh://git@", 1)
                p = urlparse(s2)
                host = p.hostname or host
                parts = [p for p in (p.path or "").split("/") if p]
                if len(parts) >= 2:
                    return parts[-2], parts[-1], host
            # owner/repo
            parts = [p for p in s.split("/") if p]
            if len(parts) == 2:
                return parts[0], parts[1], host
        except Exception:
            return "", "", host
        return "", "", host

    def _github_api_base(self, host: str) -> str:
        if not host or host == "github.com":
            return "https://api.github.com"
        return f"https://{host}/api/v3"

    def _utc_iso(self) -> str:
        try:
            from datetime import datetime, timezone
            return datetime.now(timezone.utc).isoformat()
        except Exception:
            return ""

    def _compute_status_url(self) -> str | None:
        """Compute CR status endpoint from WS URL or env.

        Expected WS path: /api/projects/{project}/sessions/{session}/ws
        We transform to:  /api/projects/{project}/agentic-sessions/{session}/status
        """
        try:
            ws_url = getattr(self.shell.transport, 'url', None)
            session_id = self.context.session_id
            if ws_url:
                parsed = urlparse(ws_url)
                scheme = 'https' if parsed.scheme == 'wss' else 'http'
                parts = [p for p in parsed.path.split('/') if p]
                # ... api projects <project> sessions <session> ws
                if 'projects' in parts and 'sessions' in parts:
                    pi = parts.index('projects')
                    si = parts.index('sessions')
                    project = parts[pi+1] if len(parts) > pi+1 else os.getenv('PROJECT_NAME', '')
                    sess = parts[si+1] if len(parts) > si+1 else session_id
                    path = f"/api/projects/{project}/agentic-sessions/{sess}/status"
                    return urlunparse((scheme, parsed.netloc, path, '', '', ''))
            # Fallback to BACKEND_API_URL and PROJECT_NAME
            base = os.getenv('BACKEND_API_URL', '').rstrip('/')
            project = os.getenv('PROJECT_NAME', '').strip()
            if base and project and session_id:
                return f"{base}/projects/{project}/agentic-sessions/{session_id}/status"
        except Exception:
            return None
        return None

    async def _update_cr_status(self, fields: dict, blocking: bool = False):
        """Update CR status. Set blocking=True for critical final updates before container exit."""
        url = self._compute_status_url()
        if not url:
            return
        data = _json.dumps(fields).encode('utf-8')
        req = _urllib_request.Request(url, data=data, headers={'Content-Type': 'application/json'}, method='PUT')
        # Propagate runner token if present
        token = (os.getenv('BOT_TOKEN') or '').strip()
        if token:
            req.add_header('Authorization', f'Bearer {token}')

        def _do():
            try:
                with _urllib_request.urlopen(req, timeout=10) as resp:
                    _ = resp.read()
                logging.info(f"CR status update successful to {fields.get('phase', 'unknown')}")
                return True
            except _urllib_error.HTTPError as he:
                logging.error(f"CR status HTTPError: {he.code} - {he.read().decode('utf-8', errors='replace')}")
                return False
            except Exception as e:
                logging.error(f"CR status update failed: {e}")
                return False

        if blocking:
            # Synchronous blocking call - ensures completion before container exit
            logging.info(f"BLOCKING CR status update to {fields.get('phase', 'unknown')}")
            success = _do()
            logging.info(f"BLOCKING update {'succeeded' if success else 'failed'}")
        else:
            # Async call for non-critical updates
            loop = asyncio.get_event_loop()
            await loop.run_in_executor(None, _do)

    async def _run_cmd(self, cmd, cwd=None, capture_stdout=False, ignore_errors=False):
        """Run a subprocess command asynchronously."""
        # Redact secrets from command for logging
        cmd_safe = [self._redact_secrets(str(arg)) for arg in cmd]
        logging.info(f"Running command: {' '.join(cmd_safe)}")
        
        proc = await asyncio.create_subprocess_exec(
            *cmd,
            stdout=asyncio.subprocess.PIPE,
            stderr=asyncio.subprocess.PIPE,
            cwd=cwd or self.context.workspace_path,
        )
        stdout_data, stderr_data = await proc.communicate()
        stdout_text = stdout_data.decode("utf-8", errors="replace")
        stderr_text = stderr_data.decode("utf-8", errors="replace")
        
        # Log output for debugging (redacted)
        if stdout_text.strip():
            logging.info(f"Command stdout: {self._redact_secrets(stdout_text.strip())}")
        if stderr_text.strip():
            logging.info(f"Command stderr: {self._redact_secrets(stderr_text.strip())}")
            
        if proc.returncode != 0 and not ignore_errors:
            raise RuntimeError(stderr_text or f"Command failed: {' '.join(cmd_safe)}")
        
        logging.info(f"Command completed with return code: {proc.returncode}")
        
        if capture_stdout:
            return stdout_text
        return ""

    async def _wait_for_ws_connection(self, timeout_seconds: int = 10):
        """Wait for WebSocket connection to be established before proceeding.

        Retries sending a test message until it succeeds or timeout is reached.
        This prevents race condition where runner sends messages before WS is connected.
        """
        if not self.shell:
            logging.warning("No shell available - skipping WebSocket wait")
            return

        start_time = asyncio.get_event_loop().time()
        attempt = 0

        while True:
            elapsed = asyncio.get_event_loop().time() - start_time
            if elapsed > timeout_seconds:
                logging.error(f"WebSocket connection not established after {timeout_seconds}s - proceeding anyway")
                return

            try:
                # Try to send a test message
                await self.shell._send_message(
                    MessageType.SYSTEM_MESSAGE,
                    "WebSocket connection test",
                )
                logging.info(f"WebSocket connection established (attempt {attempt + 1})")
                return  # Success!
            except Exception as e:
                attempt += 1
                if attempt == 1:
                    logging.warning(f"WebSocket not ready yet, retrying... ({e})")
                # Wait 200ms before retry
                await asyncio.sleep(0.2)

    async def _send_log(self, payload):
        """Send a system-level message. Accepts either a string or a dict payload."""
        if not self.shell:
            return
        text: str
        if isinstance(payload, str):
            text = payload
        elif isinstance(payload, dict):
            text = str(payload.get("message", ""))
        else:
            text = str(payload)
        await self.shell._send_message(
            MessageType.SYSTEM_MESSAGE,
            text,
        )

    def _url_with_token(self, url: str, token: str) -> str:
        if not token or not url.lower().startswith("http"):
            return url
        try:
            parsed = urlparse(url)
            netloc = parsed.netloc
            if "@" in netloc:
                netloc = netloc.split("@", 1)[1]
            auth = f"x-access-token:{token}@"
            new_netloc = auth + netloc
            return urlunparse((parsed.scheme, new_netloc, parsed.path, parsed.params, parsed.query, parsed.fragment))
        except Exception:
            return url

    def _redact_secrets(self, text: str) -> str:
        """Redact tokens and secrets from text for safe logging."""
        if not text:
            return text
        # Redact GitHub tokens (ghs_, ghp_, gho_, ghu_ prefixes)
        text = re.sub(r'gh[pousr]_[a-zA-Z0-9]{36,255}', 'gh*_***REDACTED***', text)
        # Redact x-access-token: patterns in URLs
        text = re.sub(r'x-access-token:[^@\s]+@', 'x-access-token:***REDACTED***@', text)
        # Redact oauth tokens in URLs
        text = re.sub(r'oauth2:[^@\s]+@', 'oauth2:***REDACTED***@', text)
        # Redact basic auth credentials
        text = re.sub(r'://[^:@\s]+:[^@\s]+@', '://***REDACTED***@', text)
        return text

    async def _fetch_github_token(self) -> str:
        # Try cached value from env first
        cached = os.getenv("GITHUB_TOKEN", "").strip()
        if cached:
            logging.info("Using GITHUB_TOKEN from environment")
            return cached
        
        # Build mint URL from status URL if available
        status_url = self._compute_status_url()
        if not status_url:
            logging.warning("Cannot fetch GitHub token: status URL not available")
            return ""
        
        try:
            from urllib.parse import urlparse as _up, urlunparse as _uu
            p = _up(status_url)
            new_path = p.path.rstrip("/")
            if new_path.endswith("/status"):
                new_path = new_path[:-7] + "/github/token"
            else:
                new_path = new_path + "/github/token"
            url = _uu((p.scheme, p.netloc, new_path, '', '', ''))
            logging.info(f"Fetching GitHub token from: {url}")
        except Exception as e:
            logging.error(f"Failed to construct token URL: {e}")
            return ""
        
        req = _urllib_request.Request(url, data=b"{}", headers={'Content-Type': 'application/json'}, method='POST')
        bot = (os.getenv('BOT_TOKEN') or '').strip()
        if bot:
            req.add_header('Authorization', f'Bearer {bot}')
            logging.debug("Using BOT_TOKEN for authentication")
        else:
            logging.warning("No BOT_TOKEN available for token fetch")
        
        loop = asyncio.get_event_loop()
        def _do_req():
            try:
                with _urllib_request.urlopen(req, timeout=10) as resp:
                    return resp.read().decode('utf-8', errors='replace')
            except Exception as e:
                logging.warning(f"GitHub token fetch failed: {e}")
                return ''
        
        resp_text = await loop.run_in_executor(None, _do_req)
        if not resp_text:
            logging.warning("Empty response from token endpoint")
            return ""
        
        try:
            data = _json.loads(resp_text)
            token = str(data.get('token') or '')
            if token:
                logging.info("Successfully fetched GitHub token from backend")
            else:
                logging.warning("Token endpoint returned empty token")
            return token
        except Exception as e:
            logging.error(f"Failed to parse token response: {e}")
            return ""

    async def _send_partial_output(self, output_chunk: str, *, stream_id: str, index: int):
        """Send partial assistant output using MESSAGE_PARTIAL with PartialInfo."""
        if self.shell and output_chunk.strip():
            partial = PartialInfo(
                id=stream_id,
                index=index,
                total=0,
                data=output_chunk.strip(),
            )
            await self.shell._send_message(
                MessageType.AGENT_MESSAGE,
                "",
                partial=partial,
            )


    async def _check_pr_intent(self, output: str):
        """Check if output indicates PR creation intent."""
        pr_indicators = [
            "pull request",
            "PR created",
            "merge request",
            "git push",
            "branch created"
        ]

        if any(indicator.lower() in output.lower() for indicator in pr_indicators):
            if self.shell:
                await self.shell._send_message(
                    MessageType.SYSTEM_MESSAGE,
                    "pr.intent",
                )

    async def handle_message(self, message: dict):
        """Handle incoming messages from backend."""
        msg_type = message.get('type', '')

        # Queue interactive messages for processing loop
        if msg_type in ('user_message', 'interrupt', 'end_session', 'terminate', 'stop'):
            await self._incoming_queue.put(message)
            logging.debug(f"Queued incoming message: {msg_type}")
            return

        logging.debug(f"Claude Code adapter received message: {msg_type}")

    def _get_repos_config(self) -> list[dict]:
        """Read repos mapping from REPOS_JSON env if present."""
        try:
            raw = os.getenv('REPOS_JSON', '').strip()
            if not raw:
                return []
            data = _json.loads(raw)
            if isinstance(data, list):
                # normalize names/keys
                out = []
                for it in data:
                    if not isinstance(it, dict):
                        continue
                    name = str(it.get('name') or '').strip()
                    input_obj = it.get('input') or {}
                    output_obj = it.get('output') or None
                    url = str((input_obj or {}).get('url') or '').strip()
                    if not name and url:
                        # Derive repo folder name from URL if not provided
                        try:
                            owner, repo, _ = self._parse_owner_repo(url)
                            derived = repo or ''
                            if not derived:
                                # Fallback: last path segment without .git
                                from urllib.parse import urlparse as _urlparse
                                p = _urlparse(url)
                                parts = [p for p in (p.path or '').split('/') if p]
                                if parts:
                                    derived = parts[-1]
                            name = (derived or '').removesuffix('.git').strip()
                        except Exception:
                            name = ''
                    if name and isinstance(input_obj, dict) and url:
                        out.append({'name': name, 'input': input_obj, 'output': output_obj})
                return out
        except Exception:
            return []
        return []


async def main():
    """Main entry point for the Claude Code runner wrapper."""
    # Setup logging
    logging.basicConfig(
        level=logging.INFO,
        format='%(asctime)s - %(name)s - %(levelname)s - %(message)s'
    )

    # Get configuration from environment
    session_id = os.getenv('SESSION_ID', 'test-session')
    workspace_path = os.getenv('WORKSPACE_PATH', '/workspace')
    websocket_url = os.getenv('WEBSOCKET_URL', 'ws://backend:8080/session/ws')

    # Ensure workspace exists
    Path(workspace_path).mkdir(parents=True, exist_ok=True)

    # Create adapter instance
    adapter = ClaudeCodeAdapter()

    # Create and run shell
    shell = RunnerShell(
        session_id=session_id,
        workspace_path=workspace_path,
        websocket_url=websocket_url,
        adapter=adapter,
    )

    # Link shell to adapter
    adapter.shell = shell

    try:
        await shell.start()
        logging.info("Claude Code runner session completed successfully")
        return 0
    except KeyboardInterrupt:
        logging.info("Claude Code runner session interrupted")
        return 130
    except Exception as e:
        logging.error(f"Claude Code runner session failed: {e}")
        return 1


if __name__ == '__main__':
    exit(asyncio.run(main()))
