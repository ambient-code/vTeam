"""
Prompt Management System for RFE Builder
Hybrid approach: Enum-based mapping with workflow-aware templates
Enhanced with usage analytics and optimization tracking
"""

from dataclasses import dataclass, field
from datetime import datetime
from pathlib import Path
from typing import Any, Dict, List, Optional

import yaml
from data.rfe_models import RFE, AgentRole


@dataclass
class PromptUsageStats:
    """Track usage statistics for prompt templates"""

    template_name: str
    agent_role: str
    usage_count: int = 0
    total_response_time: float = 0.0
    total_tokens_used: int = 0
    success_count: int = 0
    error_count: int = 0
    confidence_scores: List[float] = field(default_factory=list)
    last_used: Optional[datetime] = None
    first_used: Optional[datetime] = None

    @property
    def avg_response_time(self) -> float:
        return (
            self.total_response_time / self.usage_count if self.usage_count > 0 else 0.0
        )

    @property
    def avg_tokens(self) -> float:
        return (
            self.total_tokens_used / self.usage_count if self.usage_count > 0 else 0.0
        )

    @property
    def success_rate(self) -> float:
        total_attempts = self.success_count + self.error_count
        return self.success_count / total_attempts if total_attempts > 0 else 0.0

    @property
    def avg_confidence(self) -> float:
        return (
            sum(self.confidence_scores) / len(self.confidence_scores)
            if self.confidence_scores
            else 0.0
        )


@dataclass
class TemplatePerformance:
    """Performance metrics for template comparison"""

    template_name: str
    effectiveness_score: (
        float  # Composite score based on success rate, confidence, etc.
    )
    usage_frequency: int
    user_satisfaction: float = 0.0  # Could be populated from user feedback
    optimization_potential: float = 0.0  # How much this template could be improved


class PromptManager:
    """Centralized prompt template management with analytics and optimization"""

    def __init__(self, prompts_dir: Optional[Path] = None):
        if prompts_dir is None:
            # Default to prompts/ directory relative to this file
            self.prompts_dir = Path(__file__).parent.parent / "prompts"
        else:
            self.prompts_dir = Path(prompts_dir)

        self.templates: Dict[AgentRole, Dict[str, Any]] = {}
        self.usage_stats: Dict[str, PromptUsageStats] = {}  # template_key -> stats
        self._load_all_templates()

        # Workflow step to agent/task mapping
        self.workflow_step_mapping = {
            1: (AgentRole.PARKER_PM, "prioritization"),
            2: (AgentRole.ARCHIE_ARCHITECT, "technical_review"),
            3: (AgentRole.STELLA_STAFF_ENGINEER, "completeness_check"),
            4: (AgentRole.ARCHIE_ARCHITECT, "acceptance_criteria"),
            5: (AgentRole.STELLA_STAFF_ENGINEER, "final_decision"),
            6: (AgentRole.PARKER_PM, "communication"),
            7: (AgentRole.DEREK_DELIVERY_OWNER, "ticket_creation"),
        }

        # Activity tracker integration for logging
        self._activity_tracker = None
        try:
            from ai_models.activity_tracker import get_global_tracker

            self._activity_tracker = get_global_tracker()
        except ImportError:
            pass

    def _load_all_templates(self):
        """Load all prompt templates from the prompts directory"""
        if not self.prompts_dir.exists():
            print(f"Warning: Prompts directory {self.prompts_dir} does not exist")
            return

        agents_dir = self.prompts_dir / "agents"
        if not agents_dir.exists():
            print(f"Warning: Agents directory {agents_dir} does not exist")
            return

        for agent_role in AgentRole:
            agent_name = self._get_agent_dir_name(agent_role)
            agent_dir = agents_dir / agent_name

            if agent_dir.exists():
                self.templates[agent_role] = {}
                # Load all YAML files in the agent directory
                for yaml_file in agent_dir.glob("*.yaml"):
                    task_name = yaml_file.stem
                    try:
                        with open(yaml_file, "r") as f:
                            template_data = yaml.safe_load(f)
                            self.templates[agent_role][task_name] = template_data
                    except Exception as e:
                        print(f"Error loading template {yaml_file}: {e}")

    def _get_agent_dir_name(self, agent_role: AgentRole) -> str:
        """Convert AgentRole enum to directory name"""
        return agent_role.value.lower()

    def get_agent_prompt(
        self, agent: AgentRole, context: str, rfe: Optional[RFE] = None
    ) -> Dict[str, Any]:
        """
        Get appropriate prompt template for an agent

        Args:
            agent: The agent role requesting the prompt
            context: The task context (e.g., 'prioritization', 'review')
            rfe: Optional RFE object for workflow-aware prompting

        Returns:
            Dictionary containing prompt template with system, user, and metadata
        """
        # First try workflow-aware prompting if RFE is provided
        if rfe and rfe.current_step and rfe.current_step in self.workflow_step_mapping:
            workflow_agent, workflow_task = self.workflow_step_mapping[rfe.current_step]
            if workflow_agent == agent:
                # Use workflow-specific template if available
                step_template_name = f"step{rfe.current_step}_{workflow_task}"
                if (
                    agent in self.templates
                    and step_template_name in self.templates[agent]
                ):
                    template = self.templates[agent][step_template_name].copy()
                    template["workflow_aware"] = True
                    return template

        # Fallback to general context-based template
        if agent in self.templates and context in self.templates[agent]:
            template = self.templates[agent][context].copy()
            template["workflow_aware"] = False
            return template

        # Final fallback - return a basic template structure
        return {
            "system": (
                f"You are {agent.value.replace('_', ' ').title()}, "
                "assisting with RFE workflow."
            ),
            "user": "Please help with the following RFE task: {context}",
            "context": context,
            "workflow_aware": False,
            "metadata": {"agent": agent.value, "task": context, "fallback": True},
        }

    def get_workflow_prompt(self, rfe: RFE) -> Optional[Dict[str, Any]]:
        """Get the appropriate prompt for the current workflow step"""
        if rfe.current_step not in self.workflow_step_mapping:
            return None

        agent, task = self.workflow_step_mapping[rfe.current_step]
        return self.get_agent_prompt(agent, task, rfe)

    def format_prompt(self, template: Dict[str, Any], **kwargs) -> Dict[str, str]:
        """
        Format prompt template with provided context variables

        Args:
            template: Template dictionary from get_agent_prompt
            **kwargs: Context variables for template formatting

        Returns:
            Formatted prompt with system and user messages
        """
        formatted = {}

        for key in ["system", "user"]:
            if key in template:
                try:
                    formatted[key] = template[key].format(**kwargs)
                except KeyError as e:
                    print(f"Warning: Missing template variable {e} in {key} prompt")
                    formatted[key] = template[
                        key
                    ]  # Return unformatted if variables missing

        # Include metadata
        formatted["metadata"] = template.get("metadata", {})
        formatted["workflow_aware"] = template.get("workflow_aware", False)

        return formatted

    def list_available_templates(self) -> Dict[str, list]:
        """List all available templates by agent"""
        available = {}
        for agent_role, templates in self.templates.items():
            available[agent_role.value] = list(templates.keys())
        return available

    def validate_templates(self) -> Dict[str, list]:
        """Validate all templates and return any issues"""
        issues = {}

        for agent_role, templates in self.templates.items():
            agent_issues = []
            for template_name, template_data in templates.items():
                # Check required fields
                if "system" not in template_data:
                    agent_issues.append(f"{template_name}: Missing 'system' field")
                if "user" not in template_data:
                    agent_issues.append(f"{template_name}: Missing 'user' field")

            if agent_issues:
                issues[agent_role.value] = agent_issues

        return issues

    def _get_template_key(self, agent_role: AgentRole, template_name: str) -> str:
        """Generate unique key for template usage tracking"""
        return f"{agent_role.value}:{template_name}"

    def log_template_usage(
        self,
        agent_role: AgentRole,
        template_name: str,
        response_time: float,
        tokens_used: int,
        success: bool,
        confidence_score: Optional[float] = None,
    ):
        """Log template usage for analytics"""
        template_key = self._get_template_key(agent_role, template_name)

        # Initialize stats if first use
        if template_key not in self.usage_stats:
            self.usage_stats[template_key] = PromptUsageStats(
                template_name=template_name,
                agent_role=agent_role.value,
                first_used=datetime.now(),
            )

        stats = self.usage_stats[template_key]
        stats.usage_count += 1
        stats.total_response_time += response_time
        stats.total_tokens_used += tokens_used
        stats.last_used = datetime.now()

        if success:
            stats.success_count += 1
        else:
            stats.error_count += 1

        if confidence_score is not None:
            stats.confidence_scores.append(confidence_score)

    def get_template_analytics(self) -> Dict[str, Dict[str, Any]]:
        """Get comprehensive analytics for all templates"""
        analytics = {}

        for template_key, stats in self.usage_stats.items():
            analytics[template_key] = {
                "template_name": stats.template_name,
                "agent_role": stats.agent_role,
                "usage_count": stats.usage_count,
                "avg_response_time": stats.avg_response_time,
                "avg_tokens": stats.avg_tokens,
                "success_rate": stats.success_rate,
                "avg_confidence": stats.avg_confidence,
                "first_used": (
                    stats.first_used.isoformat() if stats.first_used else None
                ),
                "last_used": stats.last_used.isoformat() if stats.last_used else None,
            }

        return analytics

    def get_agent_template_performance(
        self, agent_role: AgentRole
    ) -> List[TemplatePerformance]:
        """Get template performance metrics for a specific agent"""
        agent_templates = []

        for template_key, stats in self.usage_stats.items():
            if stats.agent_role == agent_role.value:
                # Calculate effectiveness score
                effectiveness_score = self._calculate_effectiveness_score(stats)

                performance = TemplatePerformance(
                    template_name=stats.template_name,
                    effectiveness_score=effectiveness_score,
                    usage_frequency=stats.usage_count,
                    optimization_potential=1.0
                    - effectiveness_score,  # Room for improvement
                )
                agent_templates.append(performance)

        # Sort by effectiveness score descending
        return sorted(
            agent_templates, key=lambda x: x.effectiveness_score, reverse=True
        )

    def _calculate_effectiveness_score(self, stats: PromptUsageStats) -> float:
        """Calculate composite effectiveness score for a template"""
        if stats.usage_count == 0:
            return 0.0

        # Weighted combination of metrics
        success_weight = 0.4
        confidence_weight = 0.3
        usage_weight = 0.2
        efficiency_weight = 0.1

        # Success rate component
        success_component = stats.success_rate

        # Confidence component
        confidence_component = stats.avg_confidence

        # Usage frequency component (normalized by max usage)
        max_usage = max((s.usage_count for s in self.usage_stats.values()), default=1)
        usage_component = min(stats.usage_count / max_usage, 1.0)

        # Efficiency component (inverse of response time, normalized)
        if stats.avg_response_time > 0:
            max_response_time = max(
                (s.avg_response_time for s in self.usage_stats.values()), default=1.0
            )
            efficiency_component = 1.0 - min(
                stats.avg_response_time / max_response_time, 1.0
            )
        else:
            efficiency_component = 1.0

        effectiveness_score = (
            success_weight * success_component
            + confidence_weight * confidence_component
            + usage_weight * usage_component
            + efficiency_weight * efficiency_component
        )

        return min(effectiveness_score, 1.0)

    def get_optimization_recommendations(self) -> List[Dict[str, Any]]:
        """Get recommendations for template optimization"""
        recommendations = []

        for template_key, stats in self.usage_stats.items():
            # Low success rate templates
            if stats.success_rate < 0.8 and stats.usage_count >= 5:
                recommendations.append(
                    {
                        "template": stats.template_name,
                        "agent": stats.agent_role,
                        "type": "success_rate",
                        "priority": "high",
                        "issue": f"Low success rate: {stats.success_rate:.1%}",
                        "suggestion": "Review template for clarity and error patterns",
                    }
                )

            # Low confidence templates
            if stats.avg_confidence < 0.6 and stats.usage_count >= 3:
                recommendations.append(
                    {
                        "template": stats.template_name,
                        "agent": stats.agent_role,
                        "type": "confidence",
                        "priority": "medium",
                        "issue": f"Low confidence: {stats.avg_confidence:.2f}",
                        "suggestion": "Consider more specific prompts or examples",
                    }
                )

            # Slow response templates
            avg_response_time = stats.avg_response_time
            if avg_response_time > 5.0 and stats.usage_count >= 3:  # > 5 seconds
                recommendations.append(
                    {
                        "template": stats.template_name,
                        "agent": stats.agent_role,
                        "type": "performance",
                        "priority": "medium",
                        "issue": f"Slow response: {avg_response_time:.1f}s average",
                        "suggestion": (
                            "Consider prompt optimization for faster responses"
                        ),
                    }
                )

            # High token usage templates
            if stats.avg_tokens > 4000 and stats.usage_count >= 3:
                recommendations.append(
                    {
                        "template": stats.template_name,
                        "agent": stats.agent_role,
                        "type": "cost",
                        "priority": "low",
                        "issue": f"High token usage: {stats.avg_tokens:.0f} average",
                        "suggestion": "Consider prompt compression or optimization",
                    }
                )

        # Sort by priority
        priority_order = {"high": 3, "medium": 2, "low": 1}
        return sorted(
            recommendations, key=lambda x: priority_order[x["priority"]], reverse=True
        )

    def get_usage_trends(self, days: int = 7) -> Dict[str, Any]:
        """Get usage trends over time"""
        from datetime import timedelta

        cutoff_date = datetime.now() - timedelta(days=days)
        recent_stats = {}

        for template_key, stats in self.usage_stats.items():
            if stats.last_used and stats.last_used >= cutoff_date:
                recent_stats[template_key] = {
                    "template_name": stats.template_name,
                    "agent_role": stats.agent_role,
                    "usage_count": stats.usage_count,
                    "last_used": stats.last_used,
                    "effectiveness": self._calculate_effectiveness_score(stats),
                }

        return {
            "period_days": days,
            "active_templates": len(recent_stats),
            "total_templates": len(self.usage_stats),
            "recent_activity": recent_stats,
        }

    def export_analytics(self) -> Dict[str, Any]:
        """Export comprehensive analytics for external analysis"""
        return {
            "export_timestamp": datetime.now().isoformat(),
            "template_analytics": self.get_template_analytics(),
            "optimization_recommendations": self.get_optimization_recommendations(),
            "usage_trends": self.get_usage_trends(30),  # 30 day trends
            "summary": {
                "total_templates": len(self.usage_stats),
                "total_usage": sum(
                    stats.usage_count for stats in self.usage_stats.values()
                ),
                "avg_success_rate": (
                    sum(stats.success_rate for stats in self.usage_stats.values())
                    / len(self.usage_stats)
                    if self.usage_stats
                    else 0
                ),
                "avg_confidence": (
                    sum(stats.avg_confidence for stats in self.usage_stats.values())
                    / len(self.usage_stats)
                    if self.usage_stats
                    else 0
                ),
            },
        }
