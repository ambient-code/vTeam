#!/usr/bin/env python3

import asyncio
import logging
import os
import re
import shutil
import subprocess
import tempfile
from pathlib import Path
from typing import Dict, Any, Optional, Tuple, List
import httpx
import sys
import json

logger = logging.getLogger(__name__)

class SpekKitIntegration:
    """Integration layer for spek-kit with claude-runner"""

    def __init__(self, workspace_dir: str = "/tmp/spek-workspace"):
        self.workspace_dir = Path(workspace_dir)
        self.spek_kit_path = None
        self.project_initialized = False

    async def setup_workspace(self) -> bool:
        """Set up the spek-kit workspace and install spek-kit"""
        try:
            logger.info("Setting up spek-kit workspace...")

            # Create workspace directory with proper error handling for OpenShift
            try:
                self.workspace_dir.mkdir(parents=True, exist_ok=True)
                # Test write access
                test_file = self.workspace_dir / ".test_write"
                test_file.write_text("test")
                test_file.unlink()
                logger.info(f"Workspace directory ready with write access: {self.workspace_dir}")
            except (PermissionError, OSError) as e:
                logger.warning(f"Cannot write to {self.workspace_dir}: {e}")
                # Fall back to a user-writable directory
                fallback_dir = Path.home() / "spek-workspace"
                logger.info(f"Using fallback workspace: {fallback_dir}")
                self.workspace_dir = fallback_dir
                self.workspace_dir.mkdir(parents=True, exist_ok=True)

            # Install spek-kit using uvx if available, otherwise use pip
            await self._install_spek_kit()

            logger.info(f"Spek-kit workspace ready at {self.workspace_dir}")
            return True

        except Exception as e:
            logger.error(f"Failed to setup spek-kit workspace: {e}")
            return False

    async def _install_spek_kit(self):
        """Spek-kit is pre-installed in the container via requirements.txt"""
        try:
            # Check if specify command is available (use --help since --version isn't supported)
            result = await asyncio.create_subprocess_exec(
                "specify", "--help",
                stdout=asyncio.subprocess.PIPE,
                stderr=asyncio.subprocess.PIPE
            )
            stdout, stderr = await result.communicate()

            if result.returncode == 0:
                output = stdout.decode().strip()
                self.spek_kit_path = "specify"
                logger.info(f"Using pre-installed spek-kit CLI (help output received)")
                return
            else:
                logger.error(f"spek-kit CLI check failed: {stderr.decode()}")
                raise RuntimeError("spek-kit CLI not responding correctly")

        except FileNotFoundError:
            logger.error("spek-kit CLI not found in PATH")
            raise RuntimeError("spek-kit CLI not installed")
        except Exception as e:
            logger.error(f"Error checking spek-kit installation: {e}")
            raise

    def detect_spek_kit_command(self, prompt: str) -> Optional[Tuple[str, str]]:
        """
        Detect if the prompt contains a spek-kit command

        Returns:
            Tuple of (command, arguments) if found, None otherwise
        """
        # Look for spek-kit commands at the start of the prompt
        spek_commands = ["specify", "plan", "tasks"]

        for command in spek_commands:
            # Match /command followed by space and arguments
            pattern = rf'^/{command}\s+(.+?)(?:\n|$)'
            match = re.search(pattern, prompt.strip(), re.MULTILINE | re.DOTALL)
            if match:
                args = match.group(1).strip()
                logger.info(f"Detected spek-kit command: /{command} with args: {args[:100]}...")
                return command, args

        return None

    async def initialize_project(self, project_name: str = "agent-session") -> bool:
        """Initialize a spek-kit project in the workspace"""
        try:
            if self.project_initialized:
                return True

            logger.info(f"Initializing spek-kit project: {project_name}")

            project_dir = self.workspace_dir / project_name

            # Initialize project using spek-kit
            result = await asyncio.create_subprocess_shell(
                f"{self.spek_kit_path} init --here --ai claude --ignore-agent-tools --no-git",
                cwd=str(project_dir) if project_dir.exists() else str(self.workspace_dir),
                stdout=asyncio.subprocess.PIPE,
                stderr=asyncio.subprocess.PIPE
            )

            stdout, stderr = await result.communicate()

            if result.returncode != 0:
                logger.error(f"Failed to initialize spek-kit project: {stderr.decode()}")
                return False

            logger.info("Spek-kit project initialized successfully")
            self.project_initialized = True

            # Set working directory to project
            if not project_dir.exists():
                # If using --here, project was created in workspace_dir
                os.chdir(self.workspace_dir)
            else:
                os.chdir(project_dir)

            return True

        except Exception as e:
            logger.error(f"Failed to initialize spek-kit project: {e}")
            return False

    async def execute_spek_command(self, command: str, args: str) -> Dict[str, Any]:
        """Execute a spek-kit command and return results"""
        try:
            logger.info(f"Executing spek-kit command: /{command}")

            # Ensure project is initialized
            if not await self.initialize_project():
                raise RuntimeError("Failed to initialize spek-kit project")

            # Build command
            if command == "specify":
                return await self._execute_specify(args)
            elif command == "plan":
                return await self._execute_plan(args)
            elif command == "tasks":
                return await self._execute_tasks(args)
            else:
                raise ValueError(f"Unknown spek-kit command: {command}")

        except Exception as e:
            logger.error(f"Failed to execute spek-kit command /{command}: {e}")
            return {
                "success": False,
                "error": str(e),
                "command": command,
                "args": args
            }

    async def _execute_specify(self, args: str) -> Dict[str, Any]:
        """Execute the /specify command"""
        try:
            # The specify command in spek-kit creates specifications
            # We'll use Claude Code to execute this within the spek-kit project

            # Create a prompt for Claude Code that includes the spek-kit command
            claude_prompt = f"""You are working in a spek-kit project. Please execute the /specify command with these requirements:

{args}

Follow the spek-kit workflow:
1. Run the specify command script to create the branch and spec file
2. Create a comprehensive specification using the spec template
3. Fill in all required sections based on the requirements provided
4. Report the created files and branch information
"""

            # For now, we'll simulate the spec creation
            # In a full implementation, this would use Claude Code SDK
            spec_content = self._generate_spec_content(args)

            # Create the spec file structure
            specs_dir = Path.cwd() / "specs" / "001-feature"
            specs_dir.mkdir(parents=True, exist_ok=True)

            spec_file = specs_dir / "spec.md"
            spec_file.write_text(spec_content)

            return {
                "success": True,
                "command": "specify",
                "files_created": [str(spec_file)],
                "branch": "001-feature",
                "spec_content": spec_content,
                "message": f"Specification created at {spec_file}"
            }

        except Exception as e:
            logger.error(f"Failed to execute /specify: {e}")
            raise

    async def _execute_plan(self, args: str) -> Dict[str, Any]:
        """Execute the /plan command"""
        try:
            plan_content = self._generate_plan_content(args)

            # Find the latest spec directory
            specs_base = Path.cwd() / "specs"
            if specs_base.exists():
                spec_dirs = [d for d in specs_base.iterdir() if d.is_dir()]
                if spec_dirs:
                    latest_spec = sorted(spec_dirs)[-1]
                    plan_file = latest_spec / "plan.md"
                    plan_file.write_text(plan_content)

                    return {
                        "success": True,
                        "command": "plan",
                        "files_created": [str(plan_file)],
                        "plan_content": plan_content,
                        "message": f"Implementation plan created at {plan_file}"
                    }

            raise RuntimeError("No specification found. Run /specify first.")

        except Exception as e:
            logger.error(f"Failed to execute /plan: {e}")
            raise

    async def _execute_tasks(self, args: str) -> Dict[str, Any]:
        """Execute the /tasks command"""
        try:
            tasks_content = self._generate_tasks_content(args)

            # Find the latest spec directory
            specs_base = Path.cwd() / "specs"
            if specs_base.exists():
                spec_dirs = [d for d in specs_base.iterdir() if d.is_dir()]
                if spec_dirs:
                    latest_spec = sorted(spec_dirs)[-1]
                    tasks_file = latest_spec / "tasks.md"
                    tasks_file.write_text(tasks_content)

                    return {
                        "success": True,
                        "command": "tasks",
                        "files_created": [str(tasks_file)],
                        "tasks_content": tasks_content,
                        "message": f"Task breakdown created at {tasks_file}"
                    }

            raise RuntimeError("No specification found. Run /specify first.")

        except Exception as e:
            logger.error(f"Failed to execute /tasks: {e}")
            raise

    def _generate_spec_content(self, requirements: str) -> str:
        """Generate specification content based on requirements"""
        return f"""# Feature Specification

## Overview
{requirements}

## User Stories
- As a user, I want to be able to [feature] so that [benefit]

## Functional Requirements
1. The system shall [requirement 1]
2. The system shall [requirement 2]
3. The system shall [requirement 3]

## Non-Functional Requirements
- Performance: [criteria]
- Usability: [criteria]
- Security: [criteria]

## Acceptance Criteria
- [ ] Feature implementation complete
- [ ] Unit tests pass
- [ ] Integration tests pass
- [ ] Documentation updated

## Technical Notes
- Implementation approach: [notes]
- Dependencies: [list]
- Risks: [identified risks]

## Generated by
Spek-kit integration in claude-runner
Timestamp: {os.popen('date').read().strip()}
"""

    def _generate_plan_content(self, tech_requirements: str) -> str:
        """Generate implementation plan content"""
        return f"""# Implementation Plan

## Technical Requirements
{tech_requirements}

## Architecture Overview
- Frontend: [technology]
- Backend: [technology]
- Database: [technology]
- Infrastructure: [requirements]

## Implementation Phases

### Phase 1: Foundation
- Set up project structure
- Configure build tools
- Implement core models

### Phase 2: Core Features
- Implement main functionality
- Add business logic
- Create API endpoints

### Phase 3: Integration
- Frontend integration
- Testing implementation
- Documentation

## Development Workflow
1. Create feature branch
2. Implement functionality
3. Write tests
4. Code review
5. Merge to main

## Dependencies
- [List external dependencies]
- [List internal dependencies]

## Risks and Mitigations
- [Risk 1]: [Mitigation strategy]
- [Risk 2]: [Mitigation strategy]

## Generated by
Spek-kit integration in claude-runner
Timestamp: {os.popen('date').read().strip()}
"""

    def _generate_tasks_content(self, task_details: str) -> str:
        """Generate task breakdown content"""
        return f"""# Task Breakdown

## Task Details
{task_details}

## Epic: Feature Implementation

### Story 1: Foundation Setup
**Tasks:**
- [ ] Set up project structure
- [ ] Configure development environment
- [ ] Create base models
- [ ] Set up testing framework

**Estimated Effort:** 2-3 days

### Story 2: Core Implementation
**Tasks:**
- [ ] Implement main feature logic
- [ ] Create API endpoints
- [ ] Add input validation
- [ ] Implement error handling

**Estimated Effort:** 3-5 days

### Story 3: Testing & Documentation
**Tasks:**
- [ ] Write unit tests
- [ ] Write integration tests
- [ ] Create user documentation
- [ ] Update technical documentation

**Estimated Effort:** 2-3 days

## Definition of Done
- [ ] All acceptance criteria met
- [ ] Code review completed
- [ ] Tests passing (>90% coverage)
- [ ] Documentation updated
- [ ] Performance requirements met

## Dependencies
- [List any blocking dependencies]

## Generated by
Spek-kit integration in claude-runner
Timestamp: {os.popen('date').read().strip()}
"""

    def get_project_artifacts(self) -> Dict[str, Any]:
        """Collect all generated project artifacts"""
        artifacts = {
            "workspace_path": str(self.workspace_dir),
            "files": [],
            "structure": {}
        }

        try:
            # Collect all generated files
            if self.workspace_dir.exists():
                for file_path in self.workspace_dir.rglob("*"):
                    if file_path.is_file() and not file_path.name.startswith('.'):
                        rel_path = str(file_path.relative_to(self.workspace_dir))
                        artifacts["files"].append({
                            "path": rel_path,
                            "size": file_path.stat().st_size,
                            "type": "text" if file_path.suffix in [".md", ".txt", ".json", ".yaml", ".yml"] else "binary"
                        })

                        # Read content for text files
                        if file_path.suffix in [".md", ".txt", ".json"]:
                            try:
                                content = file_path.read_text()
                                artifacts["structure"][rel_path] = content
                            except Exception as e:
                                logger.warning(f"Could not read {rel_path}: {e}")

            return artifacts

        except Exception as e:
            logger.error(f"Failed to collect project artifacts: {e}")
            return artifacts

    async def cleanup(self):
        """Clean up workspace resources"""
        try:
            if self.workspace_dir.exists():
                shutil.rmtree(self.workspace_dir)
                logger.info("Spek-kit workspace cleaned up")
        except Exception as e:
            logger.warning(f"Failed to clean up workspace: {e}")