#!/usr/bin/env python3
"""
Claude Code Runner for vTeam Platform
Based on C4 Architecture: Execution pod running Claude Code CLI with multi-agent capabilities
Technologies: Python, Claude Code CLI, MCP SDK, anthropic SDK
"""

import os
import sys
import json
import logging
import argparse
from typing import Dict, Any, Optional
from datetime import datetime

from agent_loader import AgentLoader
from mcp_integration import MCPIntegration
from session_executor import SessionExecutor
from claude_api import ClaudeAPIClient
from k8s_status_updater import K8sStatusUpdater

# Configure logging
logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(name)s - %(levelname)s - %(message)s'
)
logger = logging.getLogger(__name__)


class VTeamRunner:
    """Main runner class that orchestrates AI task execution"""

    def __init__(self):
        """Initialize the vTeam runner with all components"""
        # Initialize components based on C4 architecture
        self.agent_loader = AgentLoader()
        self.mcp_integration = MCPIntegration()
        self.claude_api = ClaudeAPIClient()
        self.session_executor = SessionExecutor(
            agent_loader=self.agent_loader,
            mcp_integration=self.mcp_integration,
            claude_api=self.claude_api
        )
        self.k8s_updater = K8sStatusUpdater()

        # Get environment configuration
        self.session_name = os.getenv('SESSION_NAME', 'unknown')
        self.session_namespace = os.getenv('SESSION_NAMESPACE', 'default')
        self.agent_type = os.getenv('AGENT_TYPE', 'general-purpose')
        self.task_description = os.getenv('TASK_DESCRIPTION', '')

    def run(self, task_file: Optional[str] = None, output_format: str = 'json') -> Dict[str, Any]:
        """
        Execute the AI task

        Args:
            task_file: Path to task file (overrides environment variable)
            output_format: Output format (json, text, markdown)

        Returns:
            Execution results
        """
        logger.info(f"Starting vTeam Runner for session: {self.session_name}")
        logger.info(f"Agent type: {self.agent_type}")

        # Update session status to Running
        self.k8s_updater.update_status(
            session_name=self.session_name,
            namespace=self.session_namespace,
            phase="Running",
            message="AI task execution started"
        )

        try:
            # Load task description
            if task_file:
                with open(task_file, 'r') as f:
                    task = f.read()
            else:
                task = self.task_description

            if not task:
                raise ValueError("No task description provided")

            logger.info(f"Task: {task[:200]}...")

            # Load agent configuration
            agent = self.agent_loader.load_agent(self.agent_type)
            logger.info(f"Loaded agent: {agent.name} - {agent.description}")

            # Initialize MCP if needed for browser automation
            if agent.capabilities.get('browser_automation'):
                self.mcp_integration.initialize()
                logger.info("MCP integration initialized for browser automation")

            # Execute the task
            logger.info("Executing AI task...")
            result = self.session_executor.execute(
                task=task,
                agent=agent,
                parameters=self._get_parameters()
            )

            # Format output
            output = self._format_output(result, output_format)

            # Update session status to Succeeded
            self.k8s_updater.update_status(
                session_name=self.session_name,
                namespace=self.session_namespace,
                phase="Succeeded",
                message="Task completed successfully",
                output=output
            )

            logger.info("Task execution completed successfully")
            return result

        except Exception as e:
            logger.error(f"Task execution failed: {str(e)}")

            # Update session status to Failed
            self.k8s_updater.update_status(
                session_name=self.session_name,
                namespace=self.session_namespace,
                phase="Failed",
                message="Task execution failed",
                error=str(e)
            )

            raise

    def _get_parameters(self) -> Dict[str, str]:
        """Extract parameters from environment variables"""
        params = {}
        for key, value in os.environ.items():
            if key.startswith('PARAM_'):
                param_name = key[6:].lower()
                params[param_name] = value
        return params

    def _format_output(self, result: Dict[str, Any], format_type: str) -> str:
        """Format the execution result based on requested format"""
        if format_type == 'json':
            return json.dumps(result, indent=2)
        elif format_type == 'text':
            return result.get('output', '')
        elif format_type == 'markdown':
            return self._format_markdown(result)
        else:
            return str(result)

    def _format_markdown(self, result: Dict[str, Any]) -> str:
        """Format result as markdown"""
        md = f"""# vTeam AI Execution Result

## Session Information
- **Session**: {self.session_name}
- **Agent**: {self.agent_type}
- **Timestamp**: {datetime.now().isoformat()}

## Task
{self.task_description}

## Result
{result.get('output', 'No output generated')}

## Metadata
- **Tokens Used**: {result.get('tokens_used', 0)}
- **Execution Time**: {result.get('execution_time', 0)}s
- **Model**: {result.get('model', 'Unknown')}
"""
        return md


def main():
    """Main entry point"""
    parser = argparse.ArgumentParser(description='vTeam AI Runner')
    parser.add_argument(
        '--task-file',
        help='Path to task file',
        default=None
    )
    parser.add_argument(
        '--agent',
        help='Agent type to use',
        default=os.getenv('AGENT_TYPE', 'general-purpose')
    )
    parser.add_argument(
        '--output-format',
        choices=['json', 'text', 'markdown'],
        default='json',
        help='Output format'
    )

    args = parser.parse_args()

    # Override environment variable if agent specified
    if args.agent:
        os.environ['AGENT_TYPE'] = args.agent

    # Create and run the runner
    runner = VTeamRunner()

    try:
        result = runner.run(
            task_file=args.task_file,
            output_format=args.output_format
        )

        # Output result
        if args.output_format == 'json':
            print(json.dumps(result, indent=2))
        else:
            print(runner._format_output(result, args.output_format))

        sys.exit(0)

    except Exception as e:
        logger.error(f"Runner failed: {str(e)}")
        sys.exit(1)


if __name__ == '__main__':
    main()