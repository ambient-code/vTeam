#!/usr/bin/env python3

"""
Authentication handler for ServiceAccount tokens in the Claude Code Runner.
Supports both standard Kubernetes authentication and bot token authentication.
"""

import os
import logging
import jwt
from typing import Optional, Dict, Any

logger = logging.getLogger(__name__)


class AuthHandler:
    """Handles authentication for the Claude Code Runner"""

    def __init__(self):
        self.auth_mode = os.getenv("AUTH_MODE", "kubernetes")  # kubernetes or bot_token
        self.bot_token = os.getenv("BOT_TOKEN", "")
        self.service_account_token_path = "/var/run/secrets/kubernetes.io/serviceaccount/token"

    def get_auth_headers(self) -> Dict[str, str]:
        """
        Get authentication headers for API calls.

        Returns:
            Dictionary of headers to include in API requests
        """
        headers = {}

        if self.auth_mode == "bot_token" and self.bot_token:
            # Use provided bot token (for webhook-triggered sessions)
            headers["Authorization"] = f"Bearer {self.bot_token}"

        elif os.path.exists(self.service_account_token_path):
            # Use Kubernetes ServiceAccount token
            try:
                with open(self.service_account_token_path, 'r') as f:
                    sa_token = f.read().strip()
                headers["Authorization"] = f"Bearer {sa_token}"
                logger.info("Using ServiceAccount token authentication")
            except Exception as e:
                logger.warning(f"Failed to read ServiceAccount token: {e}")

        else:
            logger.warning("No authentication method available")

        return headers

    def get_project_context(self) -> Optional[str]:
        """
        Extract project context from authentication.

        Returns:
            Project name if available, None otherwise
        """
        if self.auth_mode == "bot_token" and self.bot_token:
            # Try to decode bot token to get project claim
            try:
                # Note: In production, we wouldn't decode without verification
                # This is for extracting claims only
                decoded = jwt.decode(self.bot_token, options={"verify_signature": False})
                project = decoded.get("project")
                if project:
                    logger.info(f"Extracted project from bot token: {project}")
                    return project
            except Exception as e:
                logger.warning(f"Failed to decode bot token: {e}")

        # Fall back to namespace from environment
        namespace = os.getenv("AGENTIC_SESSION_NAMESPACE", "default")
        return namespace

    def validate_permissions(self, required_role: str = "edit") -> bool:
        """
        Validate that the current authentication has required permissions.

        Args:
            required_role: The role required (view, edit, admin)

        Returns:
            True if permissions are valid, False otherwise
        """
        if self.auth_mode == "bot_token" and self.bot_token:
            # Bot tokens have full access to their project
            return True

        # For ServiceAccount tokens, we assume proper RBAC is configured
        # in the cluster
        if os.path.exists(self.service_account_token_path):
            return True

        logger.warning(f"Cannot validate permissions for role: {required_role}")
        return False


class BackendClient:
    """Enhanced backend client with authentication support"""

    def __init__(self, backend_url: str, auth_handler: AuthHandler):
        self.backend_url = backend_url
        self.auth_handler = auth_handler

    def get_api_endpoint(self, path: str) -> str:
        """
        Get the correct API endpoint based on configuration.

        Args:
            path: The API path (e.g., "/agentic-sessions")

        Returns:
            Full API URL
        """
        # Always build project-scoped API path under /api/projects/{project}
        # The configured backend_url should already include /api
        base = self.backend_url.rstrip("/")
        project = self.auth_handler.get_project_context()
        if not project:
            project = "default"

        # Ensure path begins with a single leading slash
        norm_path = path if path.startswith("/") else f"/{path}"
        return f"{base}/projects/{project}{norm_path}"

    def get_request_headers(self) -> Dict[str, str]:
        """
        Get headers for backend API requests.

        Returns:
            Dictionary of headers including authentication and project context
        """
        headers = self.auth_handler.get_auth_headers()

        # Add project context
        project = self.auth_handler.get_project_context()
        if project:
            headers["X-OpenShift-Project"] = project

        headers["Content-Type"] = "application/json"
        return headers

    async def update_session_status(self, session_name: str, status_data: Dict[str, Any]) -> bool:
        """
        Update session status with authentication.

        Args:
            session_name: Name of the session to update
            status_data: Status data to send

        Returns:
            True if successful, False otherwise
        """
        import aiohttp
        import json
        import os

        endpoint = self.get_api_endpoint(f"/agentic-sessions/{session_name}/status")
        headers = self.get_request_headers()
        auth_headers = self.auth_handler.get_auth_headers()

        # Intercept messages: write directly to PVC proxy and strip from status
        messages = None
        if isinstance(status_data, dict) and "messages" in status_data:
            messages = status_data.get("messages")
            try:
                # Best-effort write to PVC proxy if configured
                pvc_base = os.getenv("PVC_PROXY_API_URL", "").rstrip("/")
                msg_path = os.getenv("MESSAGE_STORE_PATH", f"/sessions/{session_name}/messages.json")
                if pvc_base and messages is not None:
                    body = {"path": msg_path, "content": json.dumps(messages), "encoding": "utf8"}
                    async with aiohttp.ClientSession() as sess2:
                        async with sess2.post(
                            f"{pvc_base}/content/write",
                            headers={**auth_headers, "Content-Type": "application/json"},
                            data=json.dumps(body),
                        ) as resp2:
                            _ = await resp2.text()
            except Exception as e:
                logger.warning(f"Failed to write messages to PVC proxy: {e}")
            # Remove messages from status payload
            try:
                status_data = dict(status_data)
                status_data.pop("messages", None)
            except Exception:
                pass

        # Filter allowed fields only
        try:
            async with aiohttp.ClientSession() as session:
                async with session.put(
                    endpoint,
                    headers=headers,
                    data=json.dumps(status_data)
                ) as response:
                    if response.status == 200:
                        logger.info(f"Successfully updated session status")
                        return True
                    else:
                        logger.error(f"Failed to update status: {response.status}")
                        return False
        except Exception as e:
            logger.error(f"Error updating session status: {e}")
            return False

    async def update_session_display_name(self, session_name: str, display_name: str) -> bool:
        """
        Update only the display name for a given session.

        Args:
            session_name: Name of the session to update
            display_name: New display name

        Returns:
            True if successful, False otherwise
        """
        import aiohttp
        import json

        endpoint = self.get_api_endpoint(f"/agentic-sessions/{session_name}/displayname")
        headers = self.get_request_headers()

        try:
            async with aiohttp.ClientSession() as session:
                async with session.put(
                    endpoint,
                    headers=headers,
                    data=json.dumps({"displayName": display_name}),
                ) as response:
                    if response.status == 200:
                        logger.info("Successfully updated session display name")
                        return True
                    else:
                        logger.error(f"Failed to update display name: {response.status}")
                        return False
        except Exception as e:
            logger.error(f"Error updating session display name: {e}")
            return False