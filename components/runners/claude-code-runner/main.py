#!/usr/bin/env python3

from dataclasses import asdict
import logging
import os
import sys
import json
from datetime import datetime, timezone
from pathlib import Path
from typing import Dict, Any, List

from claude_code_sdk.types import StreamEvent, ResultMessage
import requests
from anthropic import Anthropic

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
        self.inbox_store_path = os.getenv("INBOX_STORE_PATH", f"/sessions/{self.session_name}/inbox.jsonl")

        # Git integration (multi-repo via GIT_REPOSITORIES)
        self.git = GitIntegration()
        
        # Derived
        self.workdir = Path("/tmp/workdir")
        self.artifacts_dir = self.workdir / "artifacts"
        self.messages: List[Dict[str, Any]] = []
        # Track last pushed file state to send only deltas (path -> (mtime, size))
        self._last_push_index: Dict[str, tuple[float, int]] = {}

        if not self.session_name or not self.prompt or not self.api_key:
            missing = [k for k, v in {
                "AGENTIC_SESSION_NAME": self.session_name,
                "PROMPT": self.prompt,
                "ANTHROPIC_API_KEY": self.api_key,
            }.items() if not v]
            raise RuntimeError(f"Missing required environment variables: {', '.join(missing)}")

        self.auth = AuthHandler()
        self.backend = BackendClient(self.backend_api_url, self.auth)

    # ---------------- Display name helpers ----------------
    def _fallback_display_name(self, prompt: str) -> str:
        try:
            first_line = (prompt or "").strip().splitlines()[0].strip()
            if not first_line:
                return f"Session {self.session_name}"
            title = first_line[:60]
            if len(first_line) > 60:
                title = title.rstrip() + "…"
            return title
        except Exception:
            return f"Session {self.session_name}"

    def _generate_display_name_from_prompt(self, prompt: str) -> str:
        """Use a lightweight model to summarize the prompt into a short display name."""
        try:
            api_key = self.api_key
            if not api_key:
                return self._fallback_display_name(prompt)

            model = os.getenv("CLAUDE_TITLE_MODEL", "claude-3-haiku-20240307")
            client = Anthropic(api_key=api_key)
            system_prompt = (
                "You generate concise, human-friendly session titles. "
                "Return a short title (max 8 words), no punctuation at the end, "
                "no quotes, no markdown. Title-case important words."
            )
            user_prompt = (
                "Summarize this prompt into a short session display name.\n\n" + prompt
            )
            msg = client.messages.create(
                model=model,
                max_tokens=64,
                system=system_prompt,
                messages=[{"role": "user", "content": user_prompt}],
            )
            # Extract first text block
            text = ""
            try:
                for block in getattr(msg, "content", []) or []:
                    kind = getattr(block, "type", None)
                    if kind == "text":
                        text = getattr(block, "text", "")
                        if text:
                            break
            except Exception:
                text = ""

            title = (text or "").strip()
            # Sanitize
            if title.startswith("\"") and title.endswith("\""):
                title = title[1:-1]
            title = title.replace("\n", " ").strip()
            if not title:
                return self._fallback_display_name(prompt)
            if len(title) > 60:
                title = title[:60].rstrip() + "…"
            return title
        except Exception as e:
            logger.warning(f"Title generation error: {e}")
            return self._fallback_display_name(prompt)

    def _set_display_name_early(self) -> None:
        """Generate and update the session display name as early as possible."""
        try:
            display_name = self._generate_display_name_from_prompt(self.prompt)
            if not display_name:
                return
            try:
                import asyncio as _asyncio
                _asyncio.run(self.backend.update_session_display_name(self.session_name, display_name))
            except RuntimeError:
                # Already in an event loop; skip to avoid crash
                pass
            except Exception as e:
                logger.warning(f"Failed to set display name: {e}")
        except Exception as e:
            logger.debug(f"Skipping display name set: {e}")

    def _inject_selected_agents(self) -> None:
        """Fetch selected agent persona markdown from backend and write to .claude/agents.

        Personas can be provided via AGENT_PERSONAS (comma-separated) or AGENT_PERSONA (single).
        """
        try:
            personas_env = os.getenv("AGENT_PERSONAS") or os.getenv("AGENT_PERSONA", "")
            personas = [p.strip() for p in personas_env.split(",") if p.strip()]
            if not personas:
                return
            base = f"{self.backend_api_url}/projects/{self.session_namespace}/agents"
            out_dir = self.workdir / ".claude" / "agents"
            out_dir.mkdir(parents=True, exist_ok=True)
            for p in personas:
                try:
                    url = f"{base}/{p}/markdown"
                    resp = requests.get(url, headers=self._auth_headers(), timeout=20)
                    if resp.status_code != 200:
                        logger.warning(f"Agent markdown fetch failed for {p}: HTTP {resp.status_code}")
                        continue
                    content = resp.text or ""
                    # Write to working dir for runner/Claude
                    local_path = out_dir / f"{p}.md"
                    local_path.write_text(content, encoding="utf-8")
                    # Mirror to PVC so UI can show immediately
                    pvc_path = f"{self.workspace_store_path}/.claude/agents/{p}.md"
                    self.content_write(pvc_path, content, "utf8")
                    logger.info(f"Injected agent persona: {p}")
                except Exception as e:  # noqa: BLE001
                    logger.warning(f"Failed injecting agent {p}: {e}")
        except Exception as e:  # noqa: BLE001
            logger.debug(f"Skipping agent injection: {e}")

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
    def _append_message(self, message: str) -> None:
        payload = {
            "type": "system_message",
            "data": message,
            "timestamp": datetime.now(timezone.utc).isoformat(),
        }
        self.messages.append(payload)
        self._flush_messages()

    def _flush_messages(self) -> None:
        try:
            payload = json.dumps(self.messages)
            ok = self.content_write(self.message_store_path, payload, encoding="utf8")
            if not ok:
                logger.warning("Failed to write messages to PVC proxy")
            logger.info(f"Flushed {len(self.messages)} messages to PVC proxy")
        except Exception as e:
            logger.warning(f"Failed to flush messages: {e}")

    # ---------------- Chat inbox helpers ----------------
    async def _read_inbox_lines(self, last_offset: int) -> tuple[list[dict[str, Any]], int]:
        """Read inbox.jsonl locally when present, fallback to content service. last_offset is line count processed."""
        try:
            p = Path(self.inbox_store_path)
            text = ""
            if p.exists():
                try:
                    text = p.read_text(encoding="utf-8", errors="ignore")
                except Exception:
                    text = ""
            else:
                data = self.content_read(self.inbox_store_path) or b""
                text = data.decode("utf-8", errors="ignore") if data else ""

            if not text:
                return [], last_offset
            lines = text.splitlines()
            if last_offset >= len(lines):
                return [], len(lines)
            new_lines = lines[last_offset:]
            msgs: list[dict[str, Any]] = []
            for ln in new_lines:
                ln = ln.strip()
                if not ln:
                    continue
                try:
                    obj = json.loads(ln)
                    if isinstance(obj, dict):
                        msgs.append(obj)
                except Exception:
                    continue
            return msgs, len(lines)
        except Exception as e:
            logger.debug(f"read inbox error: {e}")
            return [], last_offset

    def _push_workspace_deltas(self) -> None:
        """Mirror only changed/new files from workdir to PVC path."""
        try:
            if not self.workspace_store_path:
                return
            updated_index: Dict[str, tuple[float, int]] = {}
            files_to_push: list[Path] = []
            for path in self.workdir.rglob("*"):
                if path.is_dir():
                    continue
                try:
                    st = path.stat()
                    mtime = st.st_mtime
                    size = st.st_size
                    rel = str(path.relative_to(self.workdir))
                    updated_index[rel] = (mtime, size)
                    prev = self._last_push_index.get(rel)
                    if prev is None or prev[0] != mtime or prev[1] != size:
                        files_to_push.append(path)
                except Exception:
                    continue

            for path in files_to_push:
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

            self._last_push_index = updated_index
        except Exception as e:
            logger.debug(f"push deltas failed: {e}")

    async def _chat_mode(self) -> None:
        from claude_code_sdk import (
            ClaudeSDKClient,
            ClaudeCodeOptions,
            AssistantMessage,
            ToolUseBlock,
            ToolResultBlock,
            TextBlock,
            UserMessage,
            SystemMessage,
            ThinkingBlock,
            ResultMessage,
        )

        allowed_tools_env = "Read,Write,Bash,Glob,Grep,Edit,MultiEdit,WebSearch,WebFetch"
        allowed_tools = [t.strip() for t in allowed_tools_env.split(",") if t.strip()]

        options = ClaudeCodeOptions(
            permission_mode=os.getenv("CLAUDE_PERMISSION_MODE", "acceptEdits"),
            allowed_tools=allowed_tools if allowed_tools else None,
            cwd=str(self.workdir),
            append_system_prompt=self.prompt + "\n\nALWAYS consult sub agents to help with this task.",
        )

        # Restore cursor if present
        cursor_path = f"{self.workspace_store_path}/.inbox_cursor"
        last_offset = 0
        try:
            off_b = self.content_read(cursor_path)
            if off_b:
                last_offset = int((off_b.decode("utf-8").strip() or "0"))
        except Exception:
            pass

        async with ClaudeSDKClient(options=options) as client:
            async def _push_workspace_async() -> None:
                try:
                    loop = __import__("asyncio").get_running_loop()
                    await loop.run_in_executor(None, self._push_workspace_to_pvc)
                except Exception as e:  # noqa: BLE001
                    logger.debug(f"async push workspace failed: {e}")


            client.connect()

            while True:
                inbox, new_offset = await self._read_inbox_lines(last_offset)
                if inbox:
                    for msg in inbox:
                        logger.info(f"Inbox message: {msg}")
                        text = str(msg.get("content", ""))
                        norm = text.strip().lower()
                        if "/end" in norm:
                            # Graceful end of interactive session
                            try:
                                self._append_message("User requested session end")
                                await self.update_status_async("Completed", message="Session ended by user", completed=True)
                                await client.disconnect()
                                return
                            except Exception:
                                pass
                            return
                       
                        # Mirror user message into outbox
                        self.messages.append({
                            "type": "user_message",
                            "content": text,
                            "timestamp": datetime.now(timezone.utc).isoformat(),
                        })
                        self._flush_messages()

                        # Send to Claude and stream results
                        await client.query(text)
                        async for message in client.receive_response():
                            logger.info(f"Message: {message}")
                            message_type_map = {
                                AssistantMessage: "assistant_message",
                                UserMessage: "user_message",
                                SystemMessage: "system_message",
                                ResultMessage: "result_message",
                            }
                            message_type = message_type_map.get(type(message), "unknown_message")
                            if isinstance(message, AssistantMessage) or isinstance(message, UserMessage):
                                if isinstance(message.content, str):
                                    payload = {
                                        "type": message_type,
                                        "content": message.content,
                                        "timestamp": datetime.now(timezone.utc).isoformat(),
                                    }
                                    self.messages.append(payload)
                                    self._flush_messages()
                                else:
                                    for block in message.content:
                                        content_type_map = {
                                            TextBlock: "text_block",
                                            ThinkingBlock: "thinking_block",
                                            ToolUseBlock: "tool_use_block",
                                            ToolResultBlock: "tool_result_block",
                                        }
                                        content_type = content_type_map.get(type(block), "unknown_block")
                                        payload = {
                                            "type": message_type,
                                            "timestamp": datetime.now(timezone.utc).isoformat(),
                                            "content": {
                                                "type": content_type,
                                                **asdict(block),
                                            },
                                        }
                                        self.messages.append(payload)
                                        self._flush_messages()
                            else:
                                payload = {
                                    "type": message_type,
                                    "timestamp": datetime.now(timezone.utc).isoformat(),
                                    **asdict(message),
                                }
                                self.messages.append(payload)
                        
                        # Ensure any recent local changes are visible in UI before next run (deltas only)
                        try:
                            self._push_workspace_deltas()
                        except Exception:
                            await _push_workspace_async()
                        try:
                            self._push_workspace_deltas()
                        except Exception:
                            await _push_workspace_async()
                        self._flush_messages()

                    # Commit cursor
                    last_offset = new_offset
                    try:
                        self.content_write(cursor_path, str(last_offset), "utf8")
                    except Exception:
                        pass

                await __import__("asyncio").sleep(float(os.getenv("INBOX_POLL_INTERVAL_SEC", "0.5")))
       

    # ---------------- Status ----------------
    def _update_status(self, phase: str, message: str | None = None, completed: bool = False, result_msg: ResultMessage | None = None) -> None:
        payload: Dict[str, Any] = {"phase": phase}
        if message:
            payload["message"] = message

        if result_msg:
            payload["result"] = result_msg.result
            payload["subtype"] = result_msg.subtype
            payload["is_error"] = result_msg.is_error
            payload["num_turns"] = result_msg.num_turns
            payload["session_id"] = result_msg.session_id
            payload["total_cost_usd"] = result_msg.total_cost_usd
            payload["usage"] = result_msg.usage
      
        if completed:
            payload["completionTime"] = datetime.now(timezone.utc).isoformat()
        try:
            import asyncio
            loop = asyncio.new_event_loop()
            asyncio.set_event_loop(loop)
            try:
                loop.run_until_complete(self.backend.update_session_status(self.session_name, payload))
            finally:
                loop.close()
        except RuntimeError:
            # already in event loop
            pass
        except Exception as e:
            logger.warning(f"Failed to update status: {e}")


    async def update_status_async(self, phase: str, message: str | None = None, completed: bool = False, result_msg: ResultMessage | None = None) -> None:
        payload: Dict[str, Any] = {"phase": phase}
        if message:
            payload["message"] = message

        if result_msg:
            payload["result"] = result_msg.result
            payload["subtype"] = result_msg.subtype
            payload["is_error"] = result_msg.is_error
            payload["num_turns"] = result_msg.num_turns
            payload["session_id"] = result_msg.session_id
            payload["total_cost_usd"] = result_msg.total_cost_usd
            payload["usage"] = result_msg.usage

        if completed:
            payload["completionTime"] = datetime.now(timezone.utc).isoformat()
        try:
            await self.backend.update_session_status(self.session_name, payload)
        except Exception as e:
            logger.warning(f"Failed to update status: {e}")

    # ---------------- LLM call (streaming) ----------------
    def _run_llm_streaming(self, prompt: str) -> ResultMessage | None:
        """Run the LLM with streaming via Claude Code SDK, emitting structured messages for the UI."""
        # Nudge the agent to write files to artifacts folder

        result_message: ResultMessage | None = None


        async def run_with_client() -> None:
            from claude_code_sdk import (
                query,
                ClaudeCodeOptions,
                AssistantMessage,
                UserMessage,
                SystemMessage,
                ToolUseBlock,
                ToolResultBlock,
                TextBlock,
                ThinkingBlock,
                ResultMessage,
            )

            nonlocal result_message

            # Allow configuring tools via env; default to common ones
            allowed_tools_env = "Read,Write,Bash,Glob,Grep,Edit,MultiEdit,WebSearch,WebFetch"
            allowed_tools = [t.strip() for t in allowed_tools_env.split(",") if t.strip()]

            options = ClaudeCodeOptions(
                permission_mode=os.getenv("CLAUDE_PERMISSION_MODE", "acceptEdits"),
                allowed_tools=allowed_tools if allowed_tools else None,
                cwd=str(self.workdir),
                # include_partial_messages=True, # TODO add incremental messages
            )

            stream = query(prompt=prompt, options=options)
            try:
                async for message in stream:
                    logger.info(f"Message: {message}")
                    if isinstance(message, StreamEvent):
                        # handle stream events
                        pass
                    else:
                        message_type_map = {
                            AssistantMessage: "assistant_message",
                            UserMessage: "user_message",
                            SystemMessage: "system_message",
                            ResultMessage: "result_message",
                        }
                        message_type = message_type_map.get(type(message), "unknown_message")
                        if isinstance(message, AssistantMessage) or isinstance(message, UserMessage):
                            if isinstance(message.content, str):
                                payload = {
                                    "type": message_type,
                                    "content": message.content,
                                    "timestamp": datetime.now(timezone.utc).isoformat(),
                                }
                                self.messages.append(payload)
                            else:
                                for block in message.content:
                                    content_type_map = {
                                        TextBlock: "text_block",
                                        ThinkingBlock: "thinking_block",
                                        ToolUseBlock: "tool_use_block",
                                        ToolResultBlock: "tool_result_block",
                                    }
                                    content_type = content_type_map.get(type(block), "unknown_block")
                                    payload = {
                                        "type": message_type,
                                        "timestamp": datetime.now(timezone.utc).isoformat(),
                                        "content": {
                                            "type": content_type,
                                            **asdict(block),
                                        },
                                    }
                                    self.messages.append(payload)
                        else:
                            payload = {
                                "type": message_type,
                                "timestamp": datetime.now(timezone.utc).isoformat(),
                                **asdict(message),
                            }
                            self.messages.append(payload)
                            if isinstance(message, ResultMessage):
                                result_message = message
                    
                    # push workspace and flush messages
                    try:
                        self._push_workspace_deltas()
                    except Exception:
                        logger.warning("Failed to push workspace deltas")
                    self._flush_messages()
                    
            except GeneratorExit:
                logger.debug("Stream generator closed (GeneratorExit)")
            except Exception as e:  # noqa: BLE001
                logger.error(f"Claude Code SDK streaming error: {e}")
            finally:
                aclose = getattr(stream, "aclose", None)
                if callable(aclose):
                    try:
                        await aclose()
                    except Exception as e:  # noqa: BLE001
                        logger.debug(f"Stream aclose raised: {e}")
                        
                    


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
        return result_message

    # ---------------- Main flow ----------------
    def run(self) -> int:
        try:
            logger.info(f"Starting session {self.session_namespace}/{self.session_name}")
            self.workdir.mkdir(parents=True, exist_ok=True)
            self.artifacts_dir.mkdir(parents=True, exist_ok=True)

            self._update_status("Running", message="Initializing session")

            # Update display name immediately based on the prompt
            self._set_display_name_early()

            # 1) Sync shared workspace from PVC (if configured)
            self._update_status("Running", message="Syncing workspace from PVC")
            self._sync_workspace_from_pvc()

            try:
                self._push_workspace_deltas()
            except Exception:
                logger.warning("Failed to push workspace deltas")

            # Inject selected agents into .claude/agents as markdown
            self._inject_selected_agents()

            # 1b) Setup Git and clone configured repositories into workdir (always)
            try:
                import asyncio
                self._update_status("Running", message="Setting up Git")
                asyncio.run(self.git.setup_git_config())
                self._update_status("Running", message="Cloning repositories")
                asyncio.run(self.git.clone_repositories(self.workdir))
            except RuntimeError:
                # If an event loop is already running, skip async setup to avoid crash
                pass


            # Chat vs headless mode
            chat_enabled = os.getenv("INTERACTIVE", "").lower() in ("true", "1", "yes")
            if chat_enabled:
                logger.info("Entering chat mode")
                self._update_status("Running", message="Waiting for user input")
                import asyncio as _asyncio
                _asyncio.run(self._chat_mode())
                # Chat mode is long-running; we won't push workspace or mark completed here
                return 0

            # 3) Headless one-shot
            self._update_status("Running", message="Claude is running")
            result_msg = self._run_llm_streaming(self.prompt)
            

            # 4) Push entire workspace back to PVC
            self._update_status("Running", message="Pushing workspace to PVC")
            self._push_workspace_to_pvc()

            if result_msg is not None:
                try:
                    import asyncio as _asyncio
                    async def _send():
                        summary_payload = {
                            "message": "Session completed",
                            "phase": "Completed",
                            "subtype": getattr(result_msg, "subtype", None),
                            "is_error": getattr(result_msg, "is_error", None),
                            "num_turns": getattr(result_msg, "num_turns", None),
                            "session_id": getattr(result_msg, "session_id", None),
                            "total_cost_usd": getattr(result_msg, "total_cost_usd", None),
                            "usage": getattr(result_msg, "usage", None),
                            "result": getattr(result_msg, "result", None),
                        }
                        await self.backend.update_session_status(self.session_name, summary_payload)
                    _asyncio.run(_send())
                except RuntimeError:
                    pass
                except Exception as e:
                    logger.warning(f"Failed to send result summary: {e}")

            self._update_status("Completed", message="Session completed", completed=True, result_msg=result_msg)
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

 
