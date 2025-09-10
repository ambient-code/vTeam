#!/usr/bin/env python3

import asyncio
import logging
import os
import requests
import sys
from typing import Dict, Any
from datetime import datetime, timezone

# Import Claude Code Python SDK
from claude_code_sdk import ClaudeSDKClient, ClaudeCodeOptions

# Configure logging with immediate flush for container visibility
log_level = (
    logging.DEBUG
    if os.getenv("DEBUG", "").lower() in ("true", "1", "yes")
    else logging.INFO
)
logging.basicConfig(
    level=log_level,
    format="%(asctime)s - %(levelname)s - %(message)s",
    stream=sys.stdout,
    force=True,
)
logger = logging.getLogger(__name__)


class ClaudeRunner:
    def __init__(self):
        self.session_name = os.getenv("RESEARCH_SESSION_NAME", "")
        self.session_namespace = os.getenv("RESEARCH_SESSION_NAMESPACE", "default")
        self.prompt = os.getenv("PROMPT", "")
        self.website_url = os.getenv("WEBSITE_URL", "")
        self.timeout = int(os.getenv("TIMEOUT", "300"))
        self.backend_api_url = os.getenv(
            "BACKEND_API_URL", "http://backend-service:8080/api"
        )

        # Validate Anthropic API key for Claude Code
        api_key = os.getenv("ANTHROPIC_API_KEY")
        if not api_key:
            raise ValueError("ANTHROPIC_API_KEY environment variable is required")

        logger.info(f"Initialized ClaudeRunner for session: {self.session_name}")
        logger.info(f"Website URL: {self.website_url}")
        logger.info("Using Claude Code CLI with Playwright MCP")

    async def run_research_session(self):
        """Main method to run the research session"""
        try:
            logger.info(
                "Starting research session with Claude Code + Playwright MCP..."
            )

            # Verify browser setup before starting
            await self._verify_browser_setup()

            # Generate and set display name
            await self._generate_and_set_display_name()

            # Update status to indicate we're starting
            await self.update_session_status(
                {
                    "phase": "Running",
                    "message": "Initializing Claude Code with Playwright MCP browser capabilities",
                    "startTime": datetime.now(timezone.utc).isoformat(),
                }
            )

            # Create comprehensive research prompt for Claude Code with MCP tools
            research_prompt = self._create_research_prompt()

            # Update status
            await self.update_session_status(
                {
                    "phase": "Running",
                    "message": f"Claude Code analyzing {self.website_url} with agentic browser automation",
                }
            )

            # Run Claude Code with our research prompt
            logger.info("Running Claude Code with MCP browser automation...")

            result, cost, all_messages = await self._run_claude_code(research_prompt)

            logger.info("Received comprehensive research analysis from Claude Code")

            # Log the complete research results to console
            print("\n" + "=" * 80)
            print("🔬 RESEARCH RESULTS")
            print("=" * 80)
            print(result)
            print("=" * 80 + "\n")

            # Also log to structured logging
            logger.info(f"FINAL RESEARCH RESULTS:\n{result}")

            # Update the session with the final result
            await self.update_session_status(
                {
                    "phase": "Completed",
                    "message": "Research completed successfully using Claude Code + Playwright MCP",
                    "completionTime": datetime.now(timezone.utc).isoformat(),
                    "finalOutput": result,
                    "cost": cost,
                    "messages": all_messages,
                }
            )

            logger.info("Research session completed successfully")

        except Exception as e:
            logger.error(f"Research session failed: {str(e)}")

            # Update status to indicate failure
            await self.update_session_status(
                {
                    "phase": "Failed",
                    "message": f"Research failed: {str(e)}",
                    "completionTime": datetime.now(timezone.utc).isoformat(),
                }
            )

            sys.exit(1)

    async def _verify_browser_setup(self):
        """Verify browser installation and permissions for OpenShift compatibility"""
        try:
            import subprocess
            import os

            logger.info("Verifying browser setup for OpenShift environment...")

            # Check if browser directory exists and is accessible
            browser_path = "/tmp/.cache/ms-playwright"
            if not os.path.exists(browser_path):
                logger.warning(f"Browser cache directory not found at {browser_path}")
                return

            # Check directory permissions
            if not os.access(browser_path, os.R_OK | os.X_OK):
                logger.error(f"Browser directory {browser_path} not accessible")
                return

            # List browser contents for debugging
            try:
                contents = os.listdir(browser_path)
                logger.info(f"Browser cache contents: {contents}")
            except Exception as e:
                logger.warning(f"Could not list browser cache: {e}")

            # Check if chromium binary exists and is executable
            for root, dirs, files in os.walk(browser_path):
                for file in files:
                    if "chromium" in file.lower() and os.access(
                        os.path.join(root, file), os.X_OK
                    ):
                        logger.info(
                            f"Found executable browser binary: {os.path.join(root, file)}"
                        )
                        break
            else:
                logger.warning("No executable chromium binary found")

            # Check environment variables
            env_vars = ["PLAYWRIGHT_BROWSERS_PATH", "HOME", "DISPLAY"]
            for var in env_vars:
                value = os.getenv(var, "Not set")
                logger.info(f"{var}: {value}")

            logger.info("Browser setup verification completed")

        except Exception as e:
            logger.error(f"Error during browser setup verification: {e}")
            # Don't fail the process, just log the warning

    async def _generate_and_set_display_name(self):
        """Generate a display name using LLM and update it via backend API"""
        try:
            logger.info("Generating display name for research session...")

            display_name = await self._generate_display_name()
            logger.info(f"Generated display name: {display_name}")

            # Update the display name via backend API
            await self._update_display_name(display_name)
            logger.info("Display name updated successfully")

        except Exception as e:
            logger.error(f"Error generating or setting display name: {e}")
            # Don't fail the process, just log the warning

    async def _generate_display_name(self) -> str:
        """Generate a concise display name using Anthropic Claude API directly"""
        try:
            import anthropic

            client = anthropic.Anthropic(api_key=os.getenv("ANTHROPIC_API_KEY"))

            prompt = f"""Create a concise, descriptive display name (max 50 characters) for a research session with these details:

Research Question: {self.prompt}
Target Website: {self.website_url}

The display name should capture the essence of what's being researched and where. Use format like:
- "Pricing Analysis - acme.com"  
- "Feature Review - product-site.com"
- "Company Info - startup.io"

Return only the display name, nothing else."""

            message = client.messages.create(
                model="claude-3-5-haiku-20241022",  # Use faster, cheaper model for this simple task
                max_tokens=100,
                messages=[{"role": "user", "content": prompt}],
            )

            display_name = message.content[0].text.strip()

            # Ensure it's not too long
            if len(display_name) > 50:
                display_name = display_name[:47] + "..."

            return display_name

        except Exception as e:
            logger.error(f"Error generating display name with Claude: {e}")
            # Fallback to a simple format
            domain = (
                self.website_url.replace("http://", "")
                .replace("https://", "")
                .split("/")[0]
            )
            return f"Research - {domain}"

    async def _update_display_name(self, display_name: str):
        """Update the display name via backend API"""
        try:
            url = f"{self.backend_api_url}/research-sessions/{self.session_name}/displayname"

            payload = {"displayName": display_name}

            response = await asyncio.get_event_loop().run_in_executor(
                None, lambda: requests.put(url, json=payload, timeout=30)
            )

            if response.status_code != 200:
                logger.error(
                    f"Failed to update display name: {response.status_code} - {response.text}"
                )
            else:
                logger.info("Display name updated via backend API")

        except Exception as e:
            logger.error(f"Error updating display name via API: {e}")

    async def _run_claude_code(self, prompt: str) -> tuple[str, float, list[str]]:
        """Run Claude Code using Python SDK with MCP browser automation"""
        try:
            logger.info("Initializing Claude Code Python SDK with MCP server...")

            # Configure MCP servers for OpenShift compatibility
            mcp_servers = {
                "playwright": {
                    "command": "npx",
                    "args": [
                        "@playwright/mcp",
                        "--headless",
                        "--browser",
                        "chromium",
                        "--no-sandbox",
                    ],
                }
            }

            # Configure SDK with direct MCP server configuration
            options = ClaudeCodeOptions(
                system_prompt="You are a research assistant with browser automation capabilities via Playwright MCP tools.",
                max_turns=25,
                permission_mode="acceptEdits",
                allowed_tools=["mcp__playwright"],
                mcp_servers=mcp_servers,
                cwd="/app",
            )

            logger.info("Creating Claude SDK client with MCP browser automation...")

            async with ClaudeSDKClient(options=options) as client:
                logger.info("SDK Client initialized successfully with MCP tools")

                # Send the research prompt
                logger.info("Sending research query to Claude Code SDK...")
                await client.query(prompt)

                # Collect streaming response
                response_text = []
                all_messages = []  # Track all individual model outputs for CRD
                cost = 0.0
                duration = 0

                logger.info("Processing streaming response from Claude...")
                async for message in client.receive_response():
                    try:
                        # Log the message type for debugging
                        message_type = type(message).__name__
                        logger.debug(f"Received message type: {message_type}")

                        # Stream content as it arrives
                        print(f"[DEBUG] message object: {message}")
                        if hasattr(message, "content"):
                            import json

                            for block in message.content:
                                message_obj = None

                                # Check for TextBlock (has 'text' attribute)
                                if hasattr(block, "text"):
                                    text = block.text
                                    response_text.append(text)

                                    if (
                                        text.strip()
                                    ):  # Only log and track non-empty text
                                        logger.info(f"[MODEL OUTPUT] {text}")
                                        message_obj = {"content": text.strip()}

                                # Check for ToolUseBlock (has 'id', 'name', 'input' attributes)
                                elif (
                                    hasattr(block, "id")
                                    and hasattr(block, "name")
                                    and hasattr(block, "input")
                                ):
                                    tool_input = (
                                        json.dumps(block.input) if block.input else "{}"
                                    )
                                    logger.info(f"[TOOL USE] {block.name} ({block.id})")
                                    message_obj = {
                                        "tool_use_id": block.id,
                                        "tool_use_name": block.name,
                                        "tool_use_input": tool_input,
                                    }

                                # Check for ToolResultBlock (has 'tool_use_id', 'content', 'is_error' attributes)
                                elif hasattr(block, "tool_use_id") and hasattr(
                                    block, "content"
                                ):
                                    content = ""
                                    if isinstance(block.content, list):
                                        # Handle list of content items
                                        content_parts = []
                                        for item in block.content:
                                            if (
                                                isinstance(item, dict)
                                                and "text" in item
                                            ):
                                                content_parts.append(item["text"])
                                            elif isinstance(item, str):
                                                content_parts.append(item)
                                        content = "\n".join(content_parts)
                                    elif isinstance(block.content, str):
                                        content = block.content
                                    else:
                                        content = str(block.content)

                                    # Truncate very long content
                                    if len(content) > 5000:
                                        content = (
                                            content[:5000]
                                            + "\n\n[Content truncated - full content available in logs]"
                                        )

                                    is_error = getattr(block, "is_error", False)
                                    logger.info(
                                        f"[TOOL RESULT] {block.tool_use_id} (error: {is_error})"
                                    )

                                    # Find and update the corresponding tool use message
                                    for i, existing_msg in enumerate(
                                        reversed(all_messages)
                                    ):
                                        if (
                                            existing_msg.get("tool_use_id")
                                            == block.tool_use_id
                                            and "content" not in existing_msg
                                        ):
                                            # Update the existing tool use message with result
                                            idx = len(all_messages) - 1 - i
                                            all_messages[idx]["content"] = content
                                            all_messages[idx][
                                                "tool_use_is_error"
                                            ] = is_error
                                            message_obj = None  # Don't create new message, we updated existing
                                            break
                                    else:
                                        # No matching tool use found, create standalone result
                                        message_obj = {
                                            "content": content,
                                            "tool_use_id": block.tool_use_id,
                                            "tool_use_is_error": is_error,
                                        }

                                # Add message object to tracking if we created one
                                if message_obj:
                                    all_messages.append(message_obj)

                            # Update CRD with all messages after processing this message's blocks
                            if hasattr(message, "content") and message.content:
                                await self.update_session_status(
                                    {
                                        "phase": "Running",
                                        "message": f"Processing... ({len(all_messages)} messages received)",
                                        "messages": all_messages,
                                    }
                                )

                        # Get final result with metadata
                        if message_type == "ResultMessage":
                            cost = getattr(message, "total_cost_usd", 0.0)
                            duration = getattr(message, "duration_ms", 0)
                            logger.info(
                                f"[RESULT] Cost: ${cost:.4f}, Duration: {duration}ms"
                            )

                    except Exception as e:
                        logger.error(f"Error processing message: {e}")
                        logger.debug(f"Message content: {message}")
                        continue

                # Get final result - use the last message content
                result = ""
                if response_text:
                    # Find the last non-empty text response
                    for text in reversed(response_text):
                        if text.strip():
                            result = text.strip()
                            break

                if not result:
                    # Fallback to joining all if no single final message found
                    result = "".join(response_text).strip()

                if not result:
                    raise RuntimeError("Claude Code SDK returned empty result")

                logger.info(f"Research completed successfully ({len(result)} chars)")
                logger.info(f"Cost: ${cost:.4f}, Duration: {duration}ms")

                return result, cost, all_messages

        except Exception as e:
            logger.error(f"Error running Claude Code SDK: {str(e)}")
            raise

    def _create_research_prompt(self) -> str:
        """Create a focused research prompt for Claude Code with MCP browser instructions"""
        return f"""You are a research assistant with browser automation capabilities. 

RESEARCH QUESTION: {self.prompt}

TARGET WEBSITE: {self.website_url}

Please use your browser tools to visit {self.website_url} and answer this question: "{self.prompt}"

Use your browser automation tools to:
1. Navigate to and explore the website
2. Take snapshots and screenshots as needed
3. Extract relevant information from the page
4. Navigate to additional pages if necessary to find the answer

Provide a clear, direct answer to the research question based on what you find on the website. Focus on answering the specific question rather than providing a comprehensive website analysis."""

    async def update_session_status(self, status_update: Dict[str, Any]):
        """Update the ResearchSession status via the backend API"""
        try:
            url = f"{self.backend_api_url}/research-sessions/{self.session_name}/status"

            logger.info(
                f"Updating session status: {status_update.get('phase', 'unknown')}"
            )

            response = await asyncio.get_event_loop().run_in_executor(
                None, lambda: requests.put(url, json=status_update, timeout=30)
            )

            if response.status_code != 200:
                logger.error(
                    f"Failed to update session status: {response.status_code} - {response.text}"
                )
            else:
                logger.info("Session status updated successfully")

        except Exception as e:
            logger.error(f"Error updating session status: {str(e)}")
            # Don't raise here as this shouldn't stop the main process


async def main():
    """Main entry point"""
    logger.info("Claude Research Runner with Claude Code + Playwright MCP starting...")

    # Validate required environment variables
    required_vars = [
        "RESEARCH_SESSION_NAME",
        "PROMPT",
        "WEBSITE_URL",
        "ANTHROPIC_API_KEY",
    ]
    missing_vars = [var for var in required_vars if not os.getenv(var)]

    if missing_vars:
        logger.error(
            f"Missing required environment variables: {', '.join(missing_vars)}"
        )
        sys.exit(1)

    try:
        runner = ClaudeRunner()
        await runner.run_research_session()

    except KeyboardInterrupt:
        logger.info("Research session interrupted by user")
        sys.exit(0)

    except Exception as e:
        logger.error(f"Unexpected error: {str(e)}")
        sys.exit(1)


if __name__ == "__main__":
    asyncio.run(main())
