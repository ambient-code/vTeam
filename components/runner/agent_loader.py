"""
Agent Loader Component
C4 Architecture: Loads and orchestrates 17 specialized AI agents
Manages agent configurations and capabilities
"""

import os
import yaml
import logging
from typing import Dict, Any, List
from dataclasses import dataclass
from pathlib import Path

logger = logging.getLogger(__name__)


@dataclass
class Agent:
    """Represents an AI agent with specific capabilities"""
    name: str
    type: str
    description: str
    role: str
    competencies: List[str]
    capabilities: Dict[str, Any]
    prompts: Dict[str, str]
    tools: List[str]


class AgentLoader:
    """Loads and manages the 17 specialized AI agents"""

    # Define all 17 agents based on vTeam specification
    AGENTS = {
        "general-purpose": {
            "name": "General Purpose Agent",
            "role": "Versatile problem solver",
            "competencies": ["research", "code generation", "analysis"],
            "capabilities": {"multi_step": True, "code_execution": True}
        },
        "emma": {
            "name": "Emma (Engineering Manager)",
            "role": "Team wellbeing and strategic planning",
            "competencies": ["team management", "capacity planning", "technical excellence"],
            "capabilities": {"planning": True, "coordination": True}
        },
        "olivia": {
            "name": "Olivia (Product Owner)",
            "role": "Backlog management and sprint execution",
            "competencies": ["story refinement", "acceptance criteria", "scope negotiation"],
            "capabilities": {"requirements": True, "prioritization": True}
        },
        "diego": {
            "name": "Diego (Program Manager)",
            "role": "Documentation program management",
            "competencies": ["content roadmap", "resource allocation", "delivery coordination"],
            "capabilities": {"documentation": True, "planning": True}
        },
        "ryan": {
            "name": "Ryan (UX Researcher)",
            "role": "User insights and evidence-based design",
            "competencies": ["user research", "usability testing", "design recommendations"],
            "capabilities": {"research": True, "web_search": True}
        },
        "taylor": {
            "name": "Taylor (Team Member)",
            "role": "Pragmatic implementation and code quality",
            "competencies": ["development", "technical debt assessment", "story estimation"],
            "capabilities": {"code_execution": True, "testing": True}
        },
        "felix": {
            "name": "Felix (UX Feature Lead)",
            "role": "Component design and accessibility",
            "competencies": ["feature design", "component specification", "accessibility"],
            "capabilities": {"design": True, "web_fetch": True}
        },
        "phoenix": {
            "name": "Phoenix (PXE Specialist)",
            "role": "Product experience engineering",
            "competencies": ["customer impact", "lifecycle management", "field insights"],
            "capabilities": {"telemetry": True, "web_search": True}
        },
        "terry": {
            "name": "Terry (Technical Writer)",
            "role": "User-centered documentation",
            "competencies": ["documentation", "procedure testing", "technical communication"],
            "capabilities": {"documentation": True, "testing": True}
        },
        "uma": {
            "name": "Uma (UX Team Lead)",
            "role": "Design quality and team coordination",
            "competencies": ["design process", "critique facilitation", "design system"],
            "capabilities": {"design": True, "coordination": True}
        },
        "parker": {
            "name": "Parker (Product Manager)",
            "role": "Market strategy and business value",
            "competencies": ["product roadmap", "competitive analysis", "requirements"],
            "capabilities": {"strategy": True, "web_search": True}
        },
        "jack": {
            "name": "Jack (Delivery Owner)",
            "role": "Cross-team coordination and milestones",
            "competencies": ["release planning", "risk mitigation", "status reporting"],
            "capabilities": {"coordination": True, "tracking": True}
        },
        "sam": {
            "name": "Sam (Scrum Master)",
            "role": "Agile facilitation and process optimization",
            "competencies": ["sprint planning", "retrospectives", "process improvement"],
            "capabilities": {"facilitation": True, "process": True}
        },
        "archie": {
            "name": "Archie (Architect)",
            "role": "System design and technical vision",
            "competencies": ["architecture", "technology strategy", "technical planning"],
            "capabilities": {"design": True, "web_search": True}
        },
        "lee": {
            "name": "Lee (Team Lead)",
            "role": "Team coordination and technical decisions",
            "competencies": ["sprint leadership", "technical planning", "communication"],
            "capabilities": {"coordination": True, "code_review": True}
        },
        "casey": {
            "name": "Casey (Content Strategist)",
            "role": "Information architecture and content standards",
            "competencies": ["content taxonomy", "style guidelines", "content measurement"],
            "capabilities": {"content": True, "web_search": True}
        },
        "stella": {
            "name": "Stella (Staff Engineer)",
            "role": "Technical leadership and implementation excellence",
            "competencies": ["technical problems", "code review", "mentoring"],
            "capabilities": {"code_execution": True, "architecture": True}
        }
    }

    def __init__(self, config_dir: str = "/agents"):
        """Initialize the agent loader"""
        self.config_dir = Path(config_dir)
        self.loaded_agents: Dict[str, Agent] = {}

    def load_agent(self, agent_type: str) -> Agent:
        """
        Load and configure a specific agent

        Args:
            agent_type: Type of agent to load

        Returns:
            Configured Agent instance
        """
        # Check if agent already loaded
        if agent_type in self.loaded_agents:
            return self.loaded_agents[agent_type]

        # Get agent definition
        if agent_type not in self.AGENTS:
            logger.warning(f"Unknown agent type: {agent_type}, using general-purpose")
            agent_type = "general-purpose"

        agent_def = self.AGENTS[agent_type]

        # Load additional configuration if exists
        config = self._load_agent_config(agent_type)

        # Create agent instance
        agent = Agent(
            name=agent_def["name"],
            type=agent_type,
            description=agent_def["role"],
            role=agent_def["role"],
            competencies=agent_def["competencies"],
            capabilities=agent_def["capabilities"],
            prompts=config.get("prompts", self._get_default_prompts(agent_type)),
            tools=self._get_agent_tools(agent_def["capabilities"])
        )

        # Cache the agent
        self.loaded_agents[agent_type] = agent

        logger.info(f"Loaded agent: {agent.name}")
        return agent

    def _load_agent_config(self, agent_type: str) -> Dict[str, Any]:
        """Load agent configuration from file if exists"""
        config_file = self.config_dir / f"{agent_type}.yaml"

        if config_file.exists():
            try:
                with open(config_file, 'r') as f:
                    return yaml.safe_load(f) or {}
            except Exception as e:
                logger.error(f"Failed to load config for {agent_type}: {e}")

        return {}

    def _get_default_prompts(self, agent_type: str) -> Dict[str, str]:
        """Get default prompts for an agent"""
        agent_def = self.AGENTS.get(agent_type, {})

        system_prompt = f"""You are {agent_def.get('name', 'an AI agent')}.
Role: {agent_def.get('role', 'General assistant')}
Competencies: {', '.join(agent_def.get('competencies', []))}

Your task is to help with the request using your specialized knowledge and capabilities.
Focus on your area of expertise and provide actionable insights."""

        return {
            "system": system_prompt,
            "task_prefix": f"As {agent_def.get('name', 'an agent')}, I will help you with: ",
            "response_format": "Provide detailed, actionable response based on your expertise."
        }

    def _get_agent_tools(self, capabilities: Dict[str, Any]) -> List[str]:
        """Determine which tools an agent should have access to"""
        tools = []

        if capabilities.get("code_execution"):
            tools.extend(["bash", "python", "read", "write", "edit"])

        if capabilities.get("web_search"):
            tools.append("web_search")

        if capabilities.get("web_fetch"):
            tools.append("web_fetch")

        if capabilities.get("documentation"):
            tools.extend(["write", "edit", "read"])

        if capabilities.get("design"):
            tools.extend(["diagram", "mockup"])

        if capabilities.get("browser_automation"):
            tools.append("mcp_browser")

        return list(set(tools))  # Remove duplicates

    def get_all_agents(self) -> List[str]:
        """Get list of all available agent types"""
        return list(self.AGENTS.keys())

    def get_agent_info(self, agent_type: str) -> Dict[str, Any]:
        """Get information about a specific agent"""
        return self.AGENTS.get(agent_type, {})