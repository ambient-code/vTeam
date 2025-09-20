#!/usr/bin/env python3

import logging
import os
import sys
import json
from datetime import datetime, timezone
from pathlib import Path
from typing import Dict, Any, List

import requests

from auth_handler import AuthHandler, BackendClient
from git_integration import GitIntegration


log_level = logging.DEBUG if os.getenv("DEBUG", "").lower() in ("true", "1", "yes") else logging.INFO
logging.basicConfig(level=log_level, format="%(asctime)s - %(levelname)s - %(message)s", stream=sys.stdout, force=True)
logger = logging.getLogger(__name__)


class SimpleClaudeRunner:
    def __init__(self) -> None:
        # Required inputs
        self.session_name = os.getenv("AGENTIC_SESSION_NAME", "")
        self.session_namespace = os.getenv("AGENTIC_SESSION_NAMESPACE", "default")
        self.prompt = os.getenv("PROMPT", "")
        self.api_key = os.getenv("ANTHROPIC_API_KEY", "")

        # Optional inputs
        self.git_user_name = os.getenv("GIT_USER_NAME", "").strip()
        self.git_user_email = os.getenv("GIT_USER_EMAIL", "").strip()
        self.backend_api_url = os.getenv("BACKEND_API_URL", f"http://backend-service:8080/api").rstrip("/")
        self.pvc_proxy_api_url = os.getenv("PVC_PROXY_API_URL", f"http://ambient-content.{self.session_namespace}.svc:8080").rstrip("/")
        self.message_store_path = os.getenv("MESSAGE_STORE_PATH", f"/sessions/{self.session_name}/messages.json")
        self.workspace_store_path = os.getenv("WORKSPACE_STORE_PATH", f"/sessions/{self.session_name}/workspace")

        # Git integration (multi-repo via GIT_REPOSITORIES)
        self.git = GitIntegration()

        logger.info(f"GIT_USER_NAME: {self.git_user_name}")
        logger.info(f"GIT_USER_EMAIL: {self.git_user_email}")
        logger.info(f"BACKEND_API_URL: {self.backend_api_url}")
        logger.info(f"PVC_PROXY_API_URL: {self.pvc_proxy_api_url}")
        logger.info(f"MESSAGE_STORE_PATH: {self.message_store_path}")
        logger.info(f"WORKSPACE_STORE_PATH: {self.workspace_store_path}")
        logger.info(f"AGENTIC_SESSION_NAME: {self.session_name}")
        logger.info(f"AGENTIC_SESSION_NAMESPACE: {self.session_namespace}")
        logger.info(f"PROMPT: {self.prompt}")
        logger.info(f"ANTHROPIC_API_KEY LENGTH: {len(self.api_key)}")
        
        # Derived
        self.workdir = Path("/tmp/workdir")
        self.artifacts_dir = self.workdir / "artifacts"
        self.messages: List[Dict[str, Any]] = []

        if not self.session_name or not self.prompt or not self.api_key:
            missing = [k for k, v in {
                "AGENTIC_SESSION_NAME": self.session_name,
                "PROMPT": self.prompt,
                "ANTHROPIC_API_KEY": self.api_key,
            }.items() if not v]
            raise RuntimeError(f"Missing required environment variables: {', '.join(missing)}")

        self.auth = AuthHandler()
        self.backend = BackendClient(self.backend_api_url, self.auth)

    # ---------------- PVC content helpers ----------------
    def _auth_headers(self) -> Dict[str, str]:
        return self.auth.get_auth_headers()

    def content_write(self, path: str, content: str, encoding: str = "utf8") -> bool:
        url = f"{self.pvc_proxy_api_url}/content/write"
        body = {"path": path, "content": content, "encoding": encoding}
        try:
            resp = requests.post(url, headers={**self._auth_headers(), "Content-Type": "application/json"}, data=json.dumps(body), timeout=30)
            if resp.status_code // 100 == 2:
                return True
            logger.error(f"content_write failed for {path}: HTTP {resp.status_code}")
        except Exception as e:
            logger.error(f"content_write error for {path}: {e}")
        return False

    def content_read(self, path: str) -> bytes:
        url = f"{self.pvc_proxy_api_url}/content/file"
        try:
            resp = requests.get(url, headers=self._auth_headers(), params={"path": path}, timeout=30)
            if resp.status_code == 200:
                return resp.content
        except Exception as e:
            logger.error(f"content_read error for {path}: {e}")
        return b""

    def content_list(self, path: str) -> List[Dict[str, Any]]:
        url = f"{self.pvc_proxy_api_url}/content/list"
        try:
            resp = requests.get(url, headers=self._auth_headers(), params={"path": path}, timeout=30)
            if resp.status_code == 200:
                return resp.json().get("items", [])
        except Exception as e:
            logger.error(f"content_list error for {path}: {e}")
        return []

    # ---------------- Workspace sync ----------------
    def _sync_workspace_from_pvc(self) -> None:
        if not self.workspace_store_path:
            logger.debug("No workspace store path configured, skipping sync from PVC")
            return
        
        logger.info(f"Starting workspace sync from PVC: {self.workspace_store_path} -> {self.workdir}")
        
        def pull_dir(pvc_path: str, dst: Path) -> None:
            logger.debug(f"Pulling directory: {pvc_path} -> {dst}")
            dst.mkdir(parents=True, exist_ok=True)
            items = self.content_list(pvc_path)
            logger.debug(f"Found {len(items)} items in {pvc_path}")
            
            for it in items:
                p = it.get("path", "")
                name = Path(p).name
                target = dst / name
                if it.get("isDir"):
                    logger.debug(f"Recursively pulling directory: {p}")
                    pull_dir(p, target)
                else:
                    try:
                        logger.debug(f"Pulling file: {p} -> {target}")
                        data = self.content_read(p) or b""
                        target.parent.mkdir(parents=True, exist_ok=True)
                        target.write_bytes(data)
                        logger.debug(f"Successfully pulled file: {p} ({len(data)} bytes)")
                    except Exception as e:
                        logger.warning(f"Failed to pull file {p} -> {target}: {e}")
        
        pull_dir(self.workspace_store_path, self.workdir)
        logger.info("Completed workspace sync from PVC")

    def _push_workspace_to_pvc(self) -> None:
        if not self.workspace_store_path:
            return
        for path in self.workdir.rglob("*"):
            if path.is_dir():
                        continue
            rel = path.relative_to(self.workdir)
            pvc_path = str(Path(self.workspace_store_path) / rel)
            try:
                content = path.read_text(encoding="utf-8")
                self.content_write(pvc_path, content, "utf8")
            except Exception:
                try:
                    import base64
                    self.content_write(pvc_path, base64.b64encode(path.read_bytes()).decode("ascii"), "base64")
                except Exception as e:
                    logger.warning(f"Failed to push file {path} -> {pvc_path}: {e}")

    # ---------------- Messaging ----------------
    def _append_message(self, text: str) -> None:
        if text and text.strip():
            # Emit as a simple SYSTEM status message for UI
            self.messages.append({
                "type": "system_message",
                "subtype": "status",
                "data": {"content": text.strip()},
            })

    def _append_stream_text(self, text: str) -> None:
        """Append or extend a streaming assistant text block message for smoother UI updates."""
        if not text:
            return
        # Try to coalesce with the last assistant text message
        if self.messages and isinstance(self.messages[-1], dict) and self.messages[-1].get("type") == "assistant_message":
            last = self.messages[-1]
            blocks = last.get("content")
            if isinstance(blocks, list) and blocks and isinstance(blocks[-1], dict) and blocks[-1].get("type") == "text_block":
                try:
                    blocks[-1]["text"] = (blocks[-1].get("text", "") or "") + text
                    return
                except Exception:
                    pass
        # Fallback: append a new assistant text message
        self.messages.append({
            "type": "assistant_message",
            "content": [
                {"type": "text_block", "text": text}
            ],
        })

    def _append_tool_use(self, tool_id: str | None, tool_name: str | None, tool_input: Any) -> None:
        try:
            block = {
                "type": "tool_use_block",
                "id": tool_id or "",
                "name": tool_name or "",
                "input": tool_input,
            }
            payload = {
                "type": "assistant_message",
                "content": [block],
            }
            self.messages.append(payload)
        except Exception:
            # Ensure we don't break the run due to serialization issues
            self.messages.append({
                "type": "assistant_message",
                "content": [
                    {
                        "type": "tool_use_block",
                        "id": tool_id or "",
                        "name": tool_name or "",
                        "input": "<unserializable tool input>",
                    }
                ],
            })

    def _append_tool_result(self, tool_use_id: str | None, content: Any, is_error: bool | None) -> None:
        try:
            block = {
                "type": "tool_result_block",
                "tool_use_id": tool_use_id or "",
                "content": content,
                "is_error": bool(is_error) if is_error is not None else None,
            }
            payload = {
                "type": "assistant_message",
                "content": [block],
            }
            self.messages.append(payload)
        except Exception:
            self.messages.append({
                "type": "assistant_message",
                "content": [
                    {
                        "type": "tool_result_block",
                        "tool_use_id": tool_use_id or "",
                        "content": "<unserializable tool result>",
                        "is_error": True,
                    }
                ],
            })

    def _flush_messages(self) -> None:
        try:
            payload = json.dumps(self.messages)
            ok = self.content_write(self.message_store_path, payload, encoding="utf8")
            if not ok:
                logger.warning("Failed to write messages to PVC proxy")
        except Exception as e:
            logger.warning(f"Failed to flush messages: {e}")

    # ---------------- Status ----------------
    def _update_status(self, phase: str, message: str | None = None, final_output: str | None = None, cost: float | None = None, completed: bool = False) -> None:
        payload: Dict[str, Any] = {"phase": phase}
        if message:
            payload["message"] = message
        if final_output is not None:
            payload["finalOutput"] = final_output
        if cost is not None:
            payload["cost"] = cost
        if completed:
            payload["completionTime"] = datetime.now(timezone.utc).isoformat()
        try:
            import asyncio
            asyncio.run(self.backend.update_session_status(self.session_name, payload))
        except RuntimeError:
            # already in event loop
            pass
        except Exception as e:
            logger.warning(f"Failed to update status: {e}")

    # ---------------- LLM call (streaming) ----------------
    def _run_llm_streaming(self, prompt: str) -> tuple[str, float]:
        """Run the LLM with streaming via Claude Code SDK, emitting structured messages for the UI."""
        # Nudge the agent to write files to artifacts folder
        full_prompt = prompt + "\n\nIMPORTANT: Save any file outputs into the 'artifacts' folder of the working directory."

        # Accumulate final text for convenience
        final_text_parts: List[str] = []

        async def run_with_client() -> None:
            from claude_code_sdk import (
                query,
                ClaudeCodeOptions,
                AssistantMessage,
                ToolUseBlock,
                ToolResultBlock,
                TextBlock,
            )

            # Allow configuring tools via env; default to common ones
            allowed_tools_env = os.getenv("CLAUDE_ALLOWED_TOOLS", "Read,Write,Bash").strip()
            allowed_tools = [t.strip() for t in allowed_tools_env.split(",") if t.strip()]

            options = ClaudeCodeOptions(
                permission_mode=os.getenv("CLAUDE_PERMISSION_MODE", "acceptEdits"),
                allowed_tools=allowed_tools if allowed_tools else None,
                cwd=str(self.workdir),
            )

            async for message in query(prompt=full_prompt, options=options):
                logger.info(f"Message: {message}")
                if isinstance(message, AssistantMessage):
                    # Emit assistant text blocks as streaming messages
                    if not any(isinstance(b, (ToolUseBlock, ToolResultBlock)) for b in message.content):
                        for block in message.content:
                            if isinstance(block, TextBlock):
                                self._append_stream_text(getattr(block, "text", "") or "")
                    for block in message.content:
                        if isinstance(block, ToolUseBlock):
                            tool_id = getattr(block, "id", None)
                            tool_name = getattr(block, "name", None)
                            tool_input = getattr(block, "input", None)
                            self._append_tool_use(tool_id, tool_name, tool_input)
                            self._flush_messages()
                        elif isinstance(block, TextBlock):
                            text = getattr(block, "text", None) or ""
                            if text:
                                final_text_parts.append(text)
                                logger.info(f"[TEXT]: {text}")
                                self._append_stream_text(text)
                                if "\n" in text or len(text) > 32:
                                    self._flush_messages()
                        elif isinstance(block, ToolResultBlock):
                            try:
                                tool_use_id = getattr(block, "tool_use_id", None)
                                content = getattr(block, "content", None)
                                is_error = getattr(block, "is_error", None)
                                self._append_tool_result(tool_use_id, content, is_error)
                            except Exception:
                                pass
                            # Ensure UI sees completion of tool
                            self._flush_messages()

        try:
            import asyncio
            asyncio.run(run_with_client())
        except RuntimeError:
            # If we're already inside an event loop (unlikely here), run in a thread
            import threading

            thread_error: List[Exception] = []
            done = threading.Event()

            def runner() -> None:
                try:
                    import asyncio as _asyncio
                    _asyncio.run(run_with_client())
                except Exception as e:  # noqa: BLE001
                    thread_error.append(e)
                finally:
                    done.set()

            t = threading.Thread(target=runner, daemon=True)
            t.start()
            done.wait()
            if thread_error:
                logger.error(f"Claude Code SDK streaming failed: {thread_error[0]}")

        # Final flush to ensure UI gets all content
        self._flush_messages()
        return ("".join(final_text_parts)).strip(), 0.0

    # ---------------- Main flow ----------------
    def run(self) -> int:
        try:
            logger.info(f"Starting session {self.session_namespace}/{self.session_name}")
            self.workdir.mkdir(parents=True, exist_ok=True)
            self.artifacts_dir.mkdir(parents=True, exist_ok=True)

            self._update_status("Running", message="Initializing session")

            # 1) Sync shared workspace from PVC (if configured)
            self._sync_workspace_from_pvc()

            # 1b) Setup Git and clone configured repositories into workdir (always)
            try:
                import asyncio
                asyncio.run(self.git.setup_git_config())
                asyncio.run(self.git.clone_repositories(self.workdir))
            except RuntimeError:
                # If an event loop is already running, skip async setup to avoid crash
                pass

            # 2) Workspace now has prior state; proceed to run prompt

            # 3) Run prompt
            self._append_message("Starting model run")
            self._flush_messages()
            result_text, cost = self._run_llm_streaming(self.prompt)
            self._append_message("Model run completed")
            self._flush_messages()

            # Save final-output.txt in artifacts for convenience
            try:
                (self.artifacts_dir / "final-output.txt").write_text(result_text or "", encoding="utf-8")
            except Exception as e:
                logger.warning(f"Failed to write local final-output.txt: {e}")

            # 4) Write messages to PVC via proxy
            self._flush_messages()

            # 5) Push entire workspace back to PVC
            self._push_workspace_to_pvc()

            # Final status
            self._update_status("Completed", message="Session completed", final_output=result_text, cost=cost, completed=True)
            logger.info("Session completed successfully")
            return 0

        except Exception as e:
            logger.error(f"Session failed: {e}")
            self._update_status("Failed", message=str(e), completed=True)
            return 1


def main() -> None:
    try:
        rc = SimpleClaudeRunner().run()
        sys.exit(rc)
    except Exception as e:
        logger.error(f"Fatal error: {e}")
        sys.exit(1)


if __name__ == "__main__":
    main()

 
