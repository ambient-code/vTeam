#!/usr/bin/env python3

import asyncio
import json
import logging
import os
import requests
import sys
from typing import Dict, Any
from datetime import datetime, timezone
from pathlib import Path

# Import Claude Code Python SDK
from claude_code_sdk import ClaudeSDKClient, ClaudeCodeOptions

# Import spek-kit integration
from spek_kit_integration import SpekKitIntegration

# Import Git integration
from git_integration import GitIntegration

# Import agent support
from agent_loader import AgentLoader, get_agent_loader

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
        self.session_name = os.getenv("AGENTIC_SESSION_NAME", "")
        self.session_namespace = os.getenv("AGENTIC_SESSION_NAMESPACE", "default")
        self.prompt = os.getenv("PROMPT", "")
        self.website_url = os.getenv("WEBSITE_URL", "")
        self.timeout = int(os.getenv("TIMEOUT", "300"))
        self.backend_api_url = os.getenv(
            "BACKEND_API_URL", "http://backend-service:8080/api"
        )

        # New: Agent-specific configuration
        self.agent_persona = os.getenv("AGENT_PERSONA", "")  # e.g., "ENGINEERING_MANAGER"
        self.workflow_phase = os.getenv("WORKFLOW_PHASE", "")  # e.g., "specify", "plan", "tasks"
        self.parent_rfe = os.getenv("PARENT_RFE", "")  # e.g., "001-user-auth"
        self.shared_workspace = os.getenv("SHARED_WORKSPACE", "/workspace")  # PVC mount

        # Validate Anthropic API key for Claude Code
        api_key = os.getenv("ANTHROPIC_API_KEY")
        if not api_key:
            raise ValueError("ANTHROPIC_API_KEY environment variable is required")

        # Use persistent workspace for shared storage across agent sessions
        workspace_dir = "/workspace"

        # Initialize spek-kit integration with persistent workspace
        self.spek_kit = SpekKitIntegration(workspace_dir=self.shared_workspace)

        # Initialize Git integration
        self.git = GitIntegration()

        # Initialize agent loader
        self.agent_loader = get_agent_loader()

        logger.info(f"Initialized ClaudeRunner for session: {self.session_name}")
        logger.info(f"Agent persona: {self.agent_persona}")
        logger.info(f"Workflow phase: {self.workflow_phase}")
        logger.info(f"Parent RFE: {self.parent_rfe}")
        logger.info(f"Website URL: {self.website_url}")
        logger.info("Using Claude Code CLI with MCP and spek-kit integration")

    @staticmethod
    def _load_mcp_servers_config() -> Dict[str, Any]:
        """Load MCP servers configuration from /app/.mcp.json with fallback to hardcoded default"""
        
        # Default hardcoded configuration (existing setup)
        default_config = {
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
        
        config_file_path = "/app/.mcp.json"
        
        try:
            # Check if config file exists
            if not os.path.exists(config_file_path):
                logger.info(f"MCP config file {config_file_path} not found, using default configuration")
                return default_config
            
            # Try to load and parse the JSON file
            with open(config_file_path, 'r') as f:
                config_data = json.load(f)
            
            # Check if the loaded data has the required structure
            if not isinstance(config_data, dict) or "mcpServers" not in config_data:
                logger.warning(f"Invalid MCP config structure in {config_file_path}, missing 'mcpServers' key. Using default configuration")
                return default_config
            
            mcp_servers = config_data["mcpServers"]
            if not isinstance(mcp_servers, dict):
                logger.warning(f"Invalid 'mcpServers' value in {config_file_path}, expected dict. Using default configuration")
                return default_config
            
            logger.info(f"Successfully loaded MCP servers configuration from {config_file_path}")
            logger.info(f"Loaded MCP servers: {list(mcp_servers.keys())}")
            return mcp_servers
            
        except json.JSONDecodeError as e:
            logger.error(f"Failed to parse JSON from {config_file_path}: {e}. Using default configuration")
            return default_config
        except Exception as e:
            logger.error(f"Error loading MCP config from {config_file_path}: {e}. Using default configuration")
            return default_config

    async def run_agentic_session(self):
        """Main method to run the agentic session"""
        try:
            logger.info(
                "Starting agentic session with Claude Code + MCP + spek-kit..."
            )

            # Verify browser setup before starting
            await self._verify_browser_setup()

            # Set up Git configuration
            await self._setup_git_integration()

            # Generate and set display name
            await self._generate_and_set_display_name()

            # Determine session type based on configuration
            if self.agent_persona and self.workflow_phase:
                # Agent-specific RFE workflow session
                await self._handle_agent_rfe_session()
            else:
                # Check if this is a spek-kit command
                spek_command = self.spek_kit.detect_spek_kit_command(self.prompt)
                if spek_command:
                    await self._handle_spek_kit_session(spek_command)
                    return

                # Standard agentic session with website analysis
                await self._handle_standard_session()

        except Exception as e:
            logger.error(f"Agentic session failed: {str(e)}")

            # Update status to indicate failure
            await self.update_session_status(
                {
                    "phase": "Failed",
                    "message": f"Agentic analysis failed: {str(e)}",
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
            logger.info("Generating display name for agentic session...")

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

            prompt = f"""Create a concise, descriptive display name (max 50 characters) for an agentic session with these details:

Agentic Query: {self.prompt}
Target Website: {self.website_url}

The display name should capture the essence of what's being analyzed and where. Use format like:
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
            return f"Analysis - {domain}"

    async def _update_display_name(self, display_name: str):
        """Update the display name via backend API"""
        try:
            url = f"{self.backend_api_url}/agentic-sessions/{self.session_name}/displayname"

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

            # Load MCP servers configuration from file or use default
            mcp_servers = ClaudeRunner._load_mcp_servers_config()

            # Configure SDK with direct MCP server configuration
            options = ClaudeCodeOptions(
                system_prompt="You are an agentic assistant with browser automation capabilities via MCP tools.",
                max_turns=25,
                permission_mode="acceptEdits",
                allowed_tools=["mcp__playwright", "mcp__atlassian-mcp"],
                mcp_servers=mcp_servers,
                cwd="/app",
            )

            logger.info("Creating Claude SDK client with MCP browser automation...")

            async with ClaudeSDKClient(options=options) as client:
                logger.info("SDK Client initialized successfully with MCP tools")

                # Send the agentic prompt
                logger.info("Sending agentic query to Claude Code SDK...")
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

                logger.info(f"Agentic analysis completed successfully ({len(result)} chars)")
                logger.info(f"Cost: ${cost:.4f}, Duration: {duration}ms")

                return result, cost, all_messages

        except Exception as e:
            logger.error(f"Error running Claude Code SDK: {str(e)}")
            raise

    def _create_agentic_prompt(self) -> str:
        """Create a focused agentic prompt for Claude Code with MCP browser instructions"""
        return f"""You are an agentic assistant with browser automation capabilities. 

AGENTIC QUERY: {self.prompt}

TARGET WEBSITE: {self.website_url}

Please use your browser tools to visit {self.website_url} and answer this question: "{self.prompt}"

Use your browser automation tools to:
1. Navigate to and explore the website
2. Take snapshots and screenshots as needed
3. Extract relevant information from the page
4. Navigate to additional pages if necessary to find the answer

Provide a clear, direct answer to the agentic query based on what you find on the website. Focus on answering the specific question rather than providing a comprehensive website analysis."""

    async def _handle_spek_kit_session(self, spek_command):
        """Handle a spek-kit specific session"""
        command, args = spek_command

        logger.info(f"Processing spek-kit command: /{command}")

        # Update status to indicate we're starting spek-kit workflow
        await self.update_session_status(
            {
                "phase": "Running",
                "message": f"Initializing spek-kit workflow for /{command} command",
                "startTime": datetime.now(timezone.utc).isoformat(),
            }
        )

        # Set up spek-kit workspace
        if not await self.spek_kit.setup_workspace():
            raise RuntimeError("Failed to setup spek-kit workspace")

        # Update status
        await self.update_session_status(
            {
                "phase": "Running",
                "message": f"Executing spek-kit /{command} command with spec-driven development",
            }
        )

        # Execute the spek-kit command
        spek_result = await self.spek_kit.execute_spek_command(command, args)

        if not spek_result.get("success", False):
            raise RuntimeError(f"Spek-kit command failed: {spek_result.get('error', 'Unknown error')}")

        # Now run Claude Code to enhance the generated specs
        enhanced_prompt = self._create_spek_enhanced_prompt(command, args, spek_result)

        logger.info("Running Claude Code to enhance spek-kit specifications...")
        result, cost, all_messages = await self._run_claude_code(enhanced_prompt)

        # Collect project artifacts
        artifacts = self.spek_kit.get_project_artifacts()

        # Log the results
        print("\n" + "=" * 80)
        print("📋 SPEK-KIT SPECIFICATION RESULTS")
        print("=" * 80)
        print(f"Command: /{command}")
        print(f"Generated Files: {len(artifacts['files'])}")
        print("\nGenerated Specifications:")
        print(result)
        print("=" * 80 + "\n")

        logger.info(f"SPEK-KIT RESULTS:\n{result}")

        # Update the session with the final result including artifacts
        await self.update_session_status(
            {
                "phase": "Completed",
                "message": f"Spek-kit /{command} completed successfully with spec-driven development artifacts",
                "completionTime": datetime.now(timezone.utc).isoformat(),
                "finalOutput": result,
                "cost": cost,
                "messages": all_messages,
                "spekKitCommand": command,
                "spekKitArtifacts": artifacts,
                "spekKitResult": spek_result,
            }
        )

        logger.info("Spek-kit session completed successfully")

    async def _handle_standard_session(self):
        """Handle a standard agentic session with website analysis"""
        # Update status to indicate we're starting
        await self.update_session_status(
            {
                "phase": "Running",
                "message": "Initializing Claude Code with Playwright MCP browser capabilities",
                "startTime": datetime.now(timezone.utc).isoformat(),
            }
        )

        # Create agentic prompt for Claude Code with MCP tools
        agentic_prompt = self._create_agentic_prompt()

        # Update status
        await self.update_session_status(
            {
                "phase": "Running",
                "message": f"Claude Code analyzing {self.website_url} with agentic browser automation",
            }
        )

        # Run Claude Code with our agentic prompt
        logger.info("Running Claude Code with MCP browser automation...")

        result, cost, all_messages = await self._run_claude_code(agentic_prompt)

        logger.info("Received agentic analysis from Claude Code")

        # Log the complete agentic results to console
        print("\n" + "=" * 80)
        print("🔬 AGENTIC ANALYSIS RESULTS")
        print("=" * 80)
        print(result)
        print("=" * 80 + "\n")

        # Also log to structured logging
        logger.info(f"FINAL AGENTIC RESULTS:\n{result}")

        # Update the session with the final result
        await self.update_session_status(
            {
                "phase": "Completed",
                "message": "Agentic analysis completed successfully using Claude Code + Playwright MCP",
                "completionTime": datetime.now(timezone.utc).isoformat(),
                "finalOutput": result,
                "cost": cost,
                "messages": all_messages,
            }
        )

        logger.info("Agentic session completed successfully")

    def _create_spek_enhanced_prompt(self, command: str, args: str, spek_result: Dict[str, Any]) -> str:
        """Create an enhanced prompt for Claude Code to work with spek-kit generated content"""

        base_prompt = f"""You are working in a spek-kit project where a /{command} command has been executed.

SPEK-KIT COMMAND: /{command} {args}

GENERATED ARTIFACTS:
"""

        # Add information about generated files
        if spek_result.get("files_created"):
            base_prompt += f"Files created: {', '.join(spek_result['files_created'])}\n"

        # Add the generated content
        if command == "specify" and "spec_content" in spek_result:
            base_prompt += f"\nGenerated Specification:\n{spek_result['spec_content']}\n"
        elif command == "plan" and "plan_content" in spek_result:
            base_prompt += f"\nGenerated Plan:\n{spek_result['plan_content']}\n"
        elif command == "tasks" and "tasks_content" in spek_result:
            base_prompt += f"\nGenerated Tasks:\n{spek_result['tasks_content']}\n"

        base_prompt += f"""

ENHANCEMENT INSTRUCTIONS:
Please review and enhance the generated {command} content above. Your goal is to:

1. **Analyze and improve** the generated content for completeness and quality
2. **Add specific technical details** that may be missing
3. **Provide actionable recommendations** for implementation
4. **Ensure best practices** are reflected in the specifications
5. **Make the content more comprehensive** while maintaining clarity

"""

        if command == "specify":
            base_prompt += """
For specifications, focus on:
- More detailed user stories with clear acceptance criteria
- Comprehensive functional and non-functional requirements
- Technical constraints and dependencies
- Risk assessment and mitigation strategies
- Clear success metrics
"""
        elif command == "plan":
            base_prompt += """
For implementation plans, focus on:
- Detailed technical architecture decisions
- Clear development phases with timelines
- Specific technology choices and justifications
- Integration patterns and data flow
- Testing and deployment strategies
"""
        elif command == "tasks":
            base_prompt += """
For task breakdowns, focus on:
- More granular and actionable tasks
- Clear effort estimations and dependencies
- Specific deliverables for each task
- Quality gates and definition of done
- Resource allocation recommendations
"""

        base_prompt += """
Provide your enhanced version as a complete, production-ready document that a development team could immediately use to start implementation.
"""

        return base_prompt

    async def _setup_git_integration(self):
        """Set up Git configuration and authentication"""
        try:
            logger.info("Setting up Git integration...")

            # Set up Git configuration
            git_setup_success = await self.git.setup_git_config()
            if git_setup_success:
                logger.info("Git configuration completed successfully")

                # Log authentication status
                auth_status = self.git.get_auth_status()
                logger.info(f"Git auth status: {auth_status}")

                # Clone repositories if configured
                if self.git.repositories:
                    logger.info(f"Cloning {len(self.git.repositories)} configured repositories...")
                    workspace_path = Path("/workspace/git-repos")
                    try:
                        workspace_path.mkdir(parents=True, exist_ok=True)
                        logger.info(f"Created Git workspace: {workspace_path}")
                    except (PermissionError, OSError) as e:
                        logger.warning(f"Cannot create Git workspace at {workspace_path}: {e}")
                        # Fall back to user home directory
                        workspace_path = Path.home() / "git-repos"
                        workspace_path.mkdir(parents=True, exist_ok=True)
                        logger.info(f"Using fallback Git workspace: {workspace_path}")

                    cloned_repos = await self.git.clone_repositories(workspace_path)
                    logger.info(f"Successfully cloned {len(cloned_repos)} repositories")

                    # Store cloned repository paths for later use
                    self.cloned_repositories = cloned_repos
                else:
                    logger.info("No repositories configured for cloning")
                    self.cloned_repositories = {}
            else:
                logger.warning("Git configuration failed, continuing without Git support")
                self.cloned_repositories = {}

        except Exception as e:
            logger.error(f"Error setting up Git integration: {e}")
            self.cloned_repositories = {}

    async def update_session_status(self, status_update: Dict[str, Any]):
        """Update the AgenticSession status via the backend API"""
        try:
            url = f"{self.backend_api_url}/agentic-sessions/{self.session_name}/status"

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

    async def _handle_agent_rfe_session(self):
        """Handle an agent-specific RFE workflow session"""
        logger.info(f"Starting agent RFE session: {self.agent_persona} - {self.workflow_phase}")

        # Update status to indicate we're starting
        await self.update_session_status(
            {
                "phase": "Running",
                "message": f"Initializing {self.agent_persona} for {self.workflow_phase} phase",
                "startTime": datetime.now(timezone.utc).isoformat(),
            }
        )

        # Set up spek-kit workspace (shared across agents)
        if not await self.spek_kit.setup_workspace():
            raise RuntimeError("Failed to setup spek-kit workspace")

        # Get agent-specific prompt for this phase
        agent_prompt = self.agent_loader.get_agent_prompt(
            self.agent_persona, self.workflow_phase, self.prompt
        )

        if not agent_prompt:
            raise RuntimeError(f"No agent configuration found for: {self.agent_persona}")

        # Update status
        await self.update_session_status(
            {
                "phase": "Running",
                "message": f"{self.agent_persona} executing /{self.workflow_phase} command",
            }
        )

        # Create workspace structure for this RFE and agent
        agent_workspace = Path(self.shared_workspace) / "agents" / self.workflow_phase
        agent_workspace.mkdir(parents=True, exist_ok=True)

        # Create agent-specific prompt that combines persona with spek-kit command
        logger.info(f"Running {self.agent_persona} with spek-kit /{self.workflow_phase}...")

        # Execute with Claude Code
        result, cost, all_messages = await self._run_claude_code(agent_prompt)

        logger.info(f"Agent {self.agent_persona} completed {self.workflow_phase} phase")

        # Save agent-specific result to shared workspace
        agent_result_file = agent_workspace / f"{self.agent_persona.lower().replace('_', '-')}.md"
        agent_result_file.write_text(result)

        # Log the complete agent results to console
        print("\n" + "=" * 80)
        print(f"🤖 AGENT RESULTS: {self.agent_persona} - {self.workflow_phase.upper()}")
        print("=" * 80)
        print(result)
        print("=" * 80 + "\n")

        # Collect project artifacts from spek-kit
        artifacts = self.spek_kit.get_project_artifacts()

        # Update the session with the final result
        await self.update_session_status(
            {
                "phase": "Completed",
                "message": f"{self.agent_persona} completed {self.workflow_phase} phase successfully",
                "completionTime": datetime.now(timezone.utc).isoformat(),
                "finalOutput": result,
                "cost": cost,
                "messages": all_messages,
                "artifacts": artifacts,
                "agentResultFile": str(agent_result_file),
            }
        )

        logger.info(f"Agent RFE session completed: {self.agent_persona}")


async def main():
    """Main entry point"""
    logger.info("Claude Agentic Runner with Claude Code starting...")

    # Load and display MCP configuration early for verification (before API key check)
    logger.info("Loading MCP servers configuration for verification...")
    mcp_config = ClaudeRunner._load_mcp_servers_config()
    logger.info(f"MCP Configuration loaded with {len(mcp_config)} servers: {list(mcp_config.keys())}")
    
    # Log detailed configuration for each server
    for server_name, config in mcp_config.items():
        if 'command' in config:
            logger.info(f"  {server_name}: command={config['command']} args={config.get('args', [])}")
        elif 'url' in config:
            logger.info(f"  {server_name}: url={config['url']}")
        else:
            logger.info(f"  {server_name}: {config}")

    # Validate required environment variables
    required_vars = [
        "AGENTIC_SESSION_NAME",
        "PROMPT",
        "ANTHROPIC_API_KEY",
    ]

    # For agent RFE sessions, we don't need WEBSITE_URL
    agent_persona = os.getenv("AGENT_PERSONA", "")
    if not agent_persona:
        required_vars.append("WEBSITE_URL")  # Standard sessions need website URL

    missing_vars = [var for var in required_vars if not os.getenv(var)]

    if missing_vars:
        logger.error(
            f"Missing required environment variables: {', '.join(missing_vars)}"
        )
        sys.exit(1)

    try:
        runner = ClaudeRunner()
        await runner.run_agentic_session()

    except KeyboardInterrupt:
        logger.info("Agentic session interrupted by user")
        sys.exit(0)

    except Exception as e:
        logger.error(f"Unexpected error: {str(e)}")
        sys.exit(1)


if __name__ == "__main__":
    asyncio.run(main())