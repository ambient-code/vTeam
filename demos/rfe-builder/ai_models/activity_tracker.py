"""
Enhanced activity tracking and logging for AI agent interactions
Provides comprehensive visibility into agent decisions and system behavior
"""

import uuid
from dataclasses import asdict, dataclass, field
from datetime import datetime
from enum import Enum
from typing import Any, Dict, List, Optional

from data.rfe_models import AgentRole


class ActivityType(str, Enum):
    """Types of activities that can be tracked"""

    AGENT_DECISION = "agent_decision"
    PROMPT_SELECTION = "prompt_selection"
    API_CALL = "api_call"
    USER_INTERACTION = "user_interaction"
    AGENT_COMMUNICATION = "agent_communication"
    ERROR_OCCURRED = "error_occurred"
    WORKFLOW_STEP = "workflow_step"
    COST_OPTIMIZATION = "cost_optimization"


class ConfidenceLevel(str, Enum):
    """Confidence levels for agent decisions"""

    VERY_HIGH = "very_high"
    HIGH = "high"
    MEDIUM = "medium"
    LOW = "low"
    VERY_LOW = "very_low"


@dataclass
class ActivityLog:
    """Comprehensive activity log entry"""

    id: str = field(default_factory=lambda: f"act_{uuid.uuid4().hex[:12]}")
    timestamp: datetime = field(default_factory=datetime.now)
    agent_role: Optional[str] = None
    activity_type: ActivityType = ActivityType.AGENT_DECISION
    rfe_id: Optional[str] = None
    session_id: Optional[str] = None

    # Core activity data
    action: str = ""
    context: Dict[str, Any] = field(default_factory=dict)
    decision_rationale: Optional[str] = None
    confidence_score: Optional[float] = None
    confidence_level: Optional[ConfidenceLevel] = None

    # Prompt and API tracking
    prompt_template_used: Optional[str] = None
    prompt_template_step: Optional[str] = None
    api_model_used: Optional[str] = None
    prompt_tokens: Optional[int] = None
    completion_tokens: Optional[int] = None
    response_time_ms: Optional[float] = None

    # User interaction data
    user_input: Optional[str] = None
    system_response: Optional[str] = None
    user_feedback: Optional[str] = None

    # Outcome and follow-up
    outcome: Optional[str] = None
    success: Optional[bool] = None
    error_details: Optional[str] = None
    follow_up_actions: List[str] = field(default_factory=list)

    # Metadata
    metadata: Dict[str, Any] = field(default_factory=dict)


@dataclass
class AgentInteraction:
    """Tracks communication between agents"""

    from_agent: str
    to_agent: str
    interaction_type: str
    message: str
    id: str = field(default_factory=lambda: f"int_{uuid.uuid4().hex[:12]}")
    timestamp: datetime = field(default_factory=datetime.now)
    context: Dict[str, Any] = field(default_factory=dict)
    rfe_id: Optional[str] = None
    session_id: Optional[str] = None
    response_received: bool = False
    response_time_ms: Optional[float] = None


@dataclass
class DecisionPoint:
    """Represents a key decision made by an agent"""

    agent_role: str
    decision_context: str
    chosen_option: str = ""
    rationale: str = ""
    confidence_score: float = 0.0
    id: str = field(default_factory=lambda: f"dec_{uuid.uuid4().hex[:12]}")
    timestamp: datetime = field(default_factory=datetime.now)
    options_considered: List[str] = field(default_factory=list)
    factors_considered: Dict[str, Any] = field(default_factory=dict)
    risk_assessment: Optional[str] = None
    expected_outcome: Optional[str] = None
    actual_outcome: Optional[str] = None
    rfe_id: Optional[str] = None


class ActivityTracker:
    """Enhanced activity tracking system for AI agents"""

    def __init__(self, session_id: Optional[str] = None):
        self.session_id = session_id or f"session_{uuid.uuid4().hex[:8]}"
        self.activities: List[ActivityLog] = []
        self.interactions: List[AgentInteraction] = []
        self.decisions: List[DecisionPoint] = []
        self.start_time = datetime.now()

    def log_activity(
        self,
        activity_type: ActivityType,
        action: str,
        agent_role: Optional[AgentRole] = None,
        rfe_id: Optional[str] = None,
        **kwargs,
    ) -> ActivityLog:
        """Log a new activity with comprehensive details"""

        activity = ActivityLog(
            timestamp=datetime.now(),
            agent_role=agent_role.value if agent_role else None,
            activity_type=activity_type,
            action=action,
            rfe_id=rfe_id,
            session_id=self.session_id,
            **kwargs,
        )

        self.activities.append(activity)
        return activity

    def log_agent_decision(
        self,
        agent_role: AgentRole,
        action: str,
        decision_rationale: str,
        confidence_score: float,
        prompt_template_used: Optional[str] = None,
        context: Optional[Dict[str, Any]] = None,
        rfe_id: Optional[str] = None,
        **kwargs,
    ) -> ActivityLog:
        """Log an agent decision with full context"""

        # Determine confidence level from score
        if confidence_score >= 0.9:
            confidence_level = ConfidenceLevel.VERY_HIGH
        elif confidence_score >= 0.7:
            confidence_level = ConfidenceLevel.HIGH
        elif confidence_score >= 0.5:
            confidence_level = ConfidenceLevel.MEDIUM
        elif confidence_score >= 0.3:
            confidence_level = ConfidenceLevel.LOW
        else:
            confidence_level = ConfidenceLevel.VERY_LOW

        return self.log_activity(
            activity_type=ActivityType.AGENT_DECISION,
            action=action,
            agent_role=agent_role,
            rfe_id=rfe_id,
            decision_rationale=decision_rationale,
            confidence_score=confidence_score,
            confidence_level=confidence_level,
            prompt_template_used=prompt_template_used,
            context=context or {},
            **kwargs,
        )

    def log_prompt_selection(
        self,
        agent_role: AgentRole,
        prompt_template: str,
        selection_reason: str,
        context: Dict[str, Any],
        rfe_id: Optional[str] = None,
    ) -> ActivityLog:
        """Log prompt template selection and reasoning"""

        return self.log_activity(
            activity_type=ActivityType.PROMPT_SELECTION,
            action="prompt_selected",
            agent_role=agent_role,
            rfe_id=rfe_id,
            prompt_template_used=prompt_template,
            decision_rationale=selection_reason,
            context=context,
        )

    def log_api_call(
        self,
        agent_role: AgentRole,
        model_used: str,
        prompt_tokens: int,
        completion_tokens: int,
        response_time_ms: float,
        success: bool,
        rfe_id: Optional[str] = None,
        error_details: Optional[str] = None,
    ) -> ActivityLog:
        """Log API call details for cost and performance tracking"""

        return self.log_activity(
            activity_type=ActivityType.API_CALL,
            action="api_request",
            agent_role=agent_role,
            rfe_id=rfe_id,
            api_model_used=model_used,
            prompt_tokens=prompt_tokens,
            completion_tokens=completion_tokens,
            response_time_ms=response_time_ms,
            success=success,
            error_details=error_details,
        )

    def log_user_interaction(
        self,
        action: str,
        user_input: Optional[str] = None,
        system_response: Optional[str] = None,
        agent_role: Optional[AgentRole] = None,
        rfe_id: Optional[str] = None,
        context: Optional[Dict[str, Any]] = None,
    ) -> ActivityLog:
        """Log user interactions with the system"""

        return self.log_activity(
            activity_type=ActivityType.USER_INTERACTION,
            action=action,
            agent_role=agent_role,
            rfe_id=rfe_id,
            user_input=user_input,
            system_response=system_response,
            context=context or {},
        )

    def log_agent_interaction(
        self,
        from_agent: AgentRole,
        to_agent: AgentRole,
        interaction_type: str,
        message: str,
        context: Optional[Dict[str, Any]] = None,
        rfe_id: Optional[str] = None,
    ) -> AgentInteraction:
        """Log communication between agents"""

        interaction = AgentInteraction(
            timestamp=datetime.now(),
            from_agent=from_agent.value,
            to_agent=to_agent.value,
            interaction_type=interaction_type,
            message=message,
            context=context or {},
            rfe_id=rfe_id,
            session_id=self.session_id,
        )

        self.interactions.append(interaction)

        # Also log as activity
        self.log_activity(
            activity_type=ActivityType.AGENT_COMMUNICATION,
            action=f"{interaction_type}_to_{to_agent.value}",
            agent_role=from_agent,
            rfe_id=rfe_id,
            context=dict(
                {
                    "to_agent": to_agent.value,
                    "interaction_type": interaction_type,
                    "message": message,
                },
                **(context or {}),
            ),
        )

        return interaction

    def log_decision_point(
        self,
        agent_role: AgentRole,
        decision_context: str,
        options_considered: List[str],
        chosen_option: str,
        rationale: str,
        confidence_score: float,
        factors_considered: Optional[Dict[str, Any]] = None,
        rfe_id: Optional[str] = None,
    ) -> DecisionPoint:
        """Log a key decision point with full analysis"""

        decision = DecisionPoint(
            timestamp=datetime.now(),
            agent_role=agent_role.value,
            decision_context=decision_context,
            options_considered=options_considered,
            chosen_option=chosen_option,
            rationale=rationale,
            confidence_score=confidence_score,
            factors_considered=factors_considered or {},
            rfe_id=rfe_id,
        )

        self.decisions.append(decision)
        return decision

    def log_error(
        self,
        error_message: str,
        error_details: str,
        agent_role: Optional[AgentRole] = None,
        context: Optional[Dict[str, Any]] = None,
        rfe_id: Optional[str] = None,
    ) -> ActivityLog:
        """Log system errors and exceptions"""

        return self.log_activity(
            activity_type=ActivityType.ERROR_OCCURRED,
            action="error",
            agent_role=agent_role,
            rfe_id=rfe_id,
            decision_rationale=error_message,
            error_details=error_details,
            success=False,
            context=context or {},
        )

    def get_session_summary(self) -> Dict[str, Any]:
        """Get comprehensive session summary"""

        session_duration = (datetime.now() - self.start_time).total_seconds()

        # Activity type breakdown
        activity_breakdown = {}
        for activity_type in ActivityType:
            count = len(
                [a for a in self.activities if a.activity_type == activity_type]
            )
            activity_breakdown[activity_type.value] = count

        # Agent activity breakdown
        agent_breakdown = {}
        for activity in self.activities:
            if activity.agent_role:
                if activity.agent_role not in agent_breakdown:
                    agent_breakdown[activity.agent_role] = {"count": 0, "decisions": 0}
                agent_breakdown[activity.agent_role]["count"] += 1
                if activity.activity_type == ActivityType.AGENT_DECISION:
                    agent_breakdown[activity.agent_role]["decisions"] += 1

        # Performance metrics
        api_calls = [
            a for a in self.activities if a.activity_type == ActivityType.API_CALL
        ]
        avg_response_time = 0
        total_tokens = 0
        if api_calls:
            response_times = [
                a.response_time_ms for a in api_calls if a.response_time_ms
            ]
            if response_times:
                avg_response_time = sum(response_times) / len(response_times)

            total_tokens = sum(
                (a.prompt_tokens or 0) + (a.completion_tokens or 0) for a in api_calls
            )

        return {
            "session_id": self.session_id,
            "start_time": self.start_time.isoformat(),
            "duration_seconds": session_duration,
            "total_activities": len(self.activities),
            "total_interactions": len(self.interactions),
            "total_decisions": len(self.decisions),
            "activity_breakdown": activity_breakdown,
            "agent_breakdown": agent_breakdown,
            "performance_metrics": {
                "total_api_calls": len(api_calls),
                "avg_response_time_ms": avg_response_time,
                "total_tokens": total_tokens,
            },
        }

    def get_activities_by_type(self, activity_type: ActivityType) -> List[ActivityLog]:
        """Get all activities of a specific type"""
        return [a for a in self.activities if a.activity_type == activity_type]

    def get_activities_by_agent(self, agent_role: AgentRole) -> List[ActivityLog]:
        """Get all activities for a specific agent"""
        return [a for a in self.activities if a.agent_role == agent_role.value]

    def get_activities_by_rfe(self, rfe_id: str) -> List[ActivityLog]:
        """Get all activities for a specific RFE"""
        return [a for a in self.activities if a.rfe_id == rfe_id]

    def search_activities(
        self,
        query: str,
        agent_role: Optional[AgentRole] = None,
        activity_type: Optional[ActivityType] = None,
        start_time: Optional[datetime] = None,
        end_time: Optional[datetime] = None,
    ) -> List[ActivityLog]:
        """Search activities with various filters"""

        results = self.activities[:]

        # Filter by agent role
        if agent_role:
            results = [a for a in results if a.agent_role == agent_role.value]

        # Filter by activity type
        if activity_type:
            results = [a for a in results if a.activity_type == activity_type]

        # Filter by time range
        if start_time:
            results = [a for a in results if a.timestamp >= start_time]
        if end_time:
            results = [a for a in results if a.timestamp <= end_time]

        # Text search in action, rationale, and error details
        if query:
            query_lower = query.lower()
            results = [
                a
                for a in results
                if (
                    query_lower in (a.action or "").lower()
                    or query_lower in (a.decision_rationale or "").lower()
                    or query_lower in (a.error_details or "").lower()
                )
            ]

        return results

    def export_session_data(self) -> Dict[str, Any]:
        """Export all session data for analysis"""

        return {
            "session_info": self.get_session_summary(),
            "activities": [asdict(activity) for activity in self.activities],
            "interactions": [asdict(interaction) for interaction in self.interactions],
            "decisions": [asdict(decision) for decision in self.decisions],
        }

    def clear_session_data(self):
        """Clear all session data (useful for testing)"""
        self.activities.clear()
        self.interactions.clear()
        self.decisions.clear()
        self.start_time = datetime.now()


# Global activity tracker instance
_global_tracker: Optional[ActivityTracker] = None


def get_global_tracker() -> ActivityTracker:
    """Get or create the global activity tracker"""
    global _global_tracker
    if _global_tracker is None:
        _global_tracker = ActivityTracker()
    return _global_tracker


def set_global_tracker(tracker: ActivityTracker):
    """Set a custom global tracker (useful for testing)"""
    global _global_tracker
    _global_tracker = tracker


def clear_global_tracker():
    """Clear the global tracker"""
    global _global_tracker
    if _global_tracker:
        _global_tracker.clear_session_data()
    else:
        _global_tracker = ActivityTracker()
