#!/usr/bin/env python3

import logging
import os
import sys
import json
import subprocess
from datetime import datetime, timezone
from pathlib import Path
from typing import Dict, Any, List

import requests

from auth_handler import AuthHandler, BackendClient


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
        self.github_repo = os.getenv("GITHUB_REPO", "").strip()
        self.git_user_name = os.getenv("GIT_USER_NAME", "").strip()
        self.git_user_email = os.getenv("GIT_USER_EMAIL", "").strip()
        self.backend_api_url = os.getenv("BACKEND_API_URL", f"http://backend-service:8080/api").rstrip("/")
        self.pvc_proxy_api_url = os.getenv("PVC_PROXY_API_URL", f"http://ambient-content.{self.session_namespace}.svc:8080").rstrip("/")
        self.message_store_path = os.getenv("MESSAGE_STORE_PATH", f"/sessions/{self.session_name}/messages.json")
        self.workspace_store_path = os.getenv("WORKSPACE_STORE_PATH", f"/sessions/{self.session_name}/workspace")

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

    # ---------------- Git helpers ----------------
    def _maybe_clone_repo(self) -> None:
        if not self.github_repo:
            return
        repo_dir = self.workdir / "repo"
        repo_dir.parent.mkdir(parents=True, exist_ok=True)
        try:
            if self.git_user_name:
                subprocess.run(["git", "config", "--global", "user.name", self.git_user_name], check=False)
            if self.git_user_email:
                subprocess.run(["git", "config", "--global", "user.email", self.git_user_email], check=False)
            subprocess.run(["git", "clone", "--depth", "1", self.github_repo, str(repo_dir)], check=True)
            logger.info(f"Cloned repository to {repo_dir}")
        except subprocess.CalledProcessError as e:
            logger.warning(f"Git clone failed: {e}")

    # ---------------- Workspace sync ----------------
    def _sync_workspace_from_pvc(self) -> None:
        if not self.workspace_store_path:
            return
        def pull_dir(pvc_path: str, dst: Path) -> None:
            dst.mkdir(parents=True, exist_ok=True)
            items = self.content_list(pvc_path)
            for it in items:
                p = it.get("path", "")
                name = Path(p).name
                target = dst / name
                if it.get("isDir"):
                    pull_dir(p, target)
                else:
                    try:
                        data = self.content_read(p) or b""
                        target.parent.mkdir(parents=True, exist_ok=True)
                        target.write_bytes(data)
                    except Exception as e:
                        logger.warning(f"Failed to pull file {p} -> {target}: {e}")
        pull_dir(self.workspace_store_path, self.workdir)

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
            self.messages.append({"content": text.strip()})

    def _append_stream_text(self, text: str) -> None:
        """Append streaming text to the last text message or create a new one.

        This keeps UI updates smooth by coalescing small deltas.
        """
        if not text:
            return
        if self.messages and isinstance(self.messages[-1], dict) and "content" in self.messages[-1] and not any(
            k in self.messages[-1] for k in ("tool_use_id", "tool_use_name", "tool_use_input", "tool_use_is_error")
        ):
            # Extend the last text message
            try:
                self.messages[-1]["content"] = (self.messages[-1].get("content", "") or "") + text
            except Exception:
                # Fallback: append as a new message
                self.messages.append({"content": text})
        else:
            self.messages.append({"content": text})

    def _append_tool_use(self, tool_id: str | None, tool_name: str | None, tool_input: Any) -> None:
        try:
            payload = {
                "tool_use_id": tool_id or "",
                "tool_use_name": tool_name or "",
                "tool_use_input": json.dumps(tool_input) if tool_input is not None else "",
            }
            self.messages.append(payload)
        except Exception:
            # Ensure we don't break the run due to serialization issues
            self.messages.append({
                "tool_use_id": tool_id or "",
                "tool_use_name": tool_name or "",
                "tool_use_input": "<unserializable tool input>",
                "tool_use_is_error": True,
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
        """Run the LLM with streaming, emitting structured messages for the UI."""
        import anthropic
        client = anthropic.Anthropic(api_key=self.api_key)
        # Nudge the agent to write files to artifacts folder
        full_prompt = prompt + "\n\nIMPORTANT: Save any file outputs into the 'artifacts' folder of the working directory."

        model = os.getenv("LLM_MODEL", "claude-3-7-sonnet-latest")
        max_tokens = int(os.getenv("LLM_MAX_TOKENS", "4000"))
        temperature = float(os.getenv("LLM_TEMPERATURE", "0.7"))

        # Accumulate final text for convenience
        final_text_parts: List[str] = []

        try:
            # Use streaming API to surface incremental progress
            with client.messages.stream(
                model=model,
                max_tokens=max_tokens,
                temperature=temperature,
                messages=[{"role": "user", "content": full_prompt}],
            ) as stream:
                for event in stream:
                    etype = getattr(event, "type", "")
                    # Stream textual deltas
                    if etype in ("content_block_delta", "message_delta", "text_delta"):
                        # Attempt to grab text safely across SDK versions
                        text = getattr(getattr(event, "delta", event), "text", None)
                        if text:
                            final_text_parts.append(text)
                            self._append_stream_text(text)
                            # Flush opportunistically on newlines for better UX
                            if "\n" in text or len(text) > 32:
                                self._flush_messages()
                    # Tool use start
                    elif etype in ("content_block_start", "tool_use"):
                        block = getattr(event, "content_block", None) or getattr(event, "block", None) or getattr(event, "data", None)
                        btype = getattr(block, "type", None) or getattr(event, "type", None)
                        if btype == "tool_use":
                            tool_id = getattr(block, "id", None) or getattr(event, "id", None)
                            tool_name = getattr(block, "name", None) or getattr(event, "name", None)
                            tool_input = getattr(block, "input", None) or getattr(event, "input", None)
                            self._append_tool_use(tool_id, tool_name, tool_input)
                            self._flush_messages()
                # Ensure we process the final message for any remaining text
                try:
                    final_msg = stream.get_final_message()
                    if final_msg and getattr(final_msg, "content", None):
                        for part in final_msg.content:
                            if hasattr(part, "text") and part.text:
                                final_text_parts.append(part.text)
                except Exception:
                    pass
        except Exception as e:
            logger.warning(f"Streaming failed or unsupported; falling back to non-streaming call: {e}")
            # Fall back to non-streaming call to avoid hard failure
            msg = client.messages.create(
                model=model,
                max_tokens=max_tokens,
                temperature=temperature,
                messages=[{"role": "user", "content": full_prompt}],
            )
            if msg and getattr(msg, "content", None):
                for part in msg.content:
                    if hasattr(part, "text") and part.text:
                        final_text_parts.append(part.text)

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

            # 1) Sync shared workspace from PVC (if configured), then init repo (optional)
            self._sync_workspace_from_pvc()
            self._maybe_clone_repo()

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

 
