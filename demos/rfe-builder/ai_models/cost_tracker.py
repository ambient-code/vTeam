"""
Cost tracking and optimization for AI API usage
Enhanced with comprehensive activity logging integration
"""

import time
from dataclasses import asdict, dataclass
from datetime import datetime
from typing import Any, Dict, Optional

import tiktoken
from data.rfe_models import AgentRole


@dataclass
class APIUsage:
    """Track individual API call usage with enhanced activity context"""

    timestamp: datetime
    agent_role: str
    task: str
    prompt_tokens: int
    completion_tokens: int
    total_tokens: int
    cost_estimate: float
    response_time: float
    rfe_id: Optional[str] = None

    # Enhanced tracking fields
    session_id: Optional[str] = None
    prompt_template_used: Optional[str] = None
    model_used: str = "claude-3-haiku"
    success: bool = True
    error_details: Optional[str] = None
    decision_context: Optional[str] = None
    confidence_score: Optional[float] = None
    user_feedback: Optional[str] = None
    cache_hit: bool = False
    optimization_applied: bool = False


class CostTracker:
    """Track and optimize AI API costs with comprehensive activity logging"""

    # Anthropic Claude pricing (as of 2024 - update as needed)
    CLAUDE_PRICING = {
        "claude-3-haiku": {"input": 0.00025, "output": 0.00125},  # per 1K tokens
        "claude-3-sonnet": {"input": 0.003, "output": 0.015},
        "claude-3-opus": {"input": 0.015, "output": 0.075},
    }

    def __init__(
        self, model_name: str = "claude-3-haiku", enable_activity_logging: bool = True
    ):
        self.model_name = model_name
        self.usage_log: list[APIUsage] = []
        self.cache: Dict[str, Any] = {}  # Simple response cache
        self.tokenizer = tiktoken.get_encoding("cl100k_base")  # Approximate
        self.enable_activity_logging = enable_activity_logging

        # Activity tracking integration
        self._activity_tracker = None
        if enable_activity_logging:
            try:
                from ai_models.activity_tracker import get_global_tracker

                self._activity_tracker = get_global_tracker()
            except ImportError:
                # Activity tracker not available, continue without it
                pass

    def count_tokens(self, text: str) -> int:
        """Count tokens in text using tiktoken (approximate for Claude)"""
        return len(self.tokenizer.encode(text))

    def estimate_cost(self, prompt_tokens: int, completion_tokens: int) -> float:
        """Estimate cost based on token usage"""
        if self.model_name not in self.CLAUDE_PRICING:
            return 0.0  # Unknown model

        pricing = self.CLAUDE_PRICING[self.model_name]
        input_cost = (prompt_tokens / 1000) * pricing["input"]
        output_cost = (completion_tokens / 1000) * pricing["output"]
        return input_cost + output_cost

    def check_cache(self, cache_key: str) -> Optional[Any]:
        """Check if response is cached to avoid API call"""
        return self.cache.get(cache_key)

    def cache_response(self, cache_key: str, response: Any, ttl_seconds: int = 3600):
        """Cache response with optional TTL"""
        self.cache[cache_key] = {
            "response": response,
            "timestamp": time.time(),
            "ttl": ttl_seconds,
        }

        # Simple cache cleanup - remove expired entries
        current_time = time.time()
        expired_keys = [
            key
            for key, value in self.cache.items()
            if current_time - value["timestamp"] > value["ttl"]
        ]
        for key in expired_keys:
            del self.cache[key]

    def generate_cache_key(
        self, agent: AgentRole, task: str, context: Dict[str, Any]
    ) -> str:
        """Generate cache key for identical requests"""
        # Create deterministic key from agent, task, and key context elements
        key_elements = [
            agent.value,
            task,
            str(sorted(context.items())),  # Ensure consistent ordering
        ]
        return "|".join(key_elements)

    def log_usage(
        self,
        agent_role: AgentRole,
        task: str,
        prompt_tokens: int,
        completion_tokens: int,
        response_time: float,
        rfe_id: Optional[str] = None,
        **kwargs,
    ) -> APIUsage:
        """Log API usage for cost tracking with enhanced context"""

        total_tokens = prompt_tokens + completion_tokens
        cost_estimate = self.estimate_cost(prompt_tokens, completion_tokens)

        # Get session ID from activity tracker if available
        session_id = None
        if self._activity_tracker:
            session_id = self._activity_tracker.session_id

        usage = APIUsage(
            timestamp=datetime.now(),
            agent_role=agent_role.value,
            task=task,
            prompt_tokens=prompt_tokens,
            completion_tokens=completion_tokens,
            total_tokens=total_tokens,
            cost_estimate=cost_estimate,
            response_time=response_time,
            rfe_id=rfe_id,
            session_id=session_id,
            model_used=self.model_name,
            **kwargs,
        )

        self.usage_log.append(usage)

        # Log to activity tracker if available
        if self._activity_tracker:
            self._activity_tracker.log_api_call(
                agent_role=agent_role,
                model_used=self.model_name,
                prompt_tokens=prompt_tokens,
                completion_tokens=completion_tokens,
                response_time_ms=response_time * 1000,  # Convert to ms
                success=kwargs.get("success", True),
                rfe_id=rfe_id,
                error_details=kwargs.get("error_details"),
            )

        return usage

    def get_usage_summary(self, hours: int = 24) -> Dict[str, Any]:
        """Get usage summary for the last N hours"""
        cutoff_time = datetime.now().timestamp() - (hours * 3600)
        recent_usage = [
            usage
            for usage in self.usage_log
            if usage.timestamp.timestamp() > cutoff_time
        ]

        if not recent_usage:
            return {"total_calls": 0, "total_cost": 0.0, "total_tokens": 0}

        summary: Dict[str, Any] = {
            "total_calls": len(recent_usage),
            "total_cost": sum(usage.cost_estimate for usage in recent_usage),
            "total_tokens": sum(usage.total_tokens for usage in recent_usage),
            "avg_response_time": sum(usage.response_time for usage in recent_usage)
            / len(recent_usage),
            "by_agent": {},
            "by_task": {},
        }

        # Group by agent
        for usage in recent_usage:
            agent = usage.agent_role
            if agent not in summary["by_agent"]:
                summary["by_agent"][agent] = {"calls": 0, "cost": 0.0, "tokens": 0}
            summary["by_agent"][agent]["calls"] += 1
            summary["by_agent"][agent]["cost"] += usage.cost_estimate
            summary["by_agent"][agent]["tokens"] += usage.total_tokens

        # Group by task
        for usage in recent_usage:
            task = usage.task
            if task not in summary["by_task"]:
                summary["by_task"][task] = {"calls": 0, "cost": 0.0, "tokens": 0}
            summary["by_task"][task]["calls"] += 1
            summary["by_task"][task]["cost"] += usage.cost_estimate
            summary["by_task"][task]["tokens"] += usage.total_tokens

        return summary

    def optimize_prompt(self, prompt: str, max_tokens: int = 4000) -> str:
        """
        Optimize prompt for cost by truncating if necessary
        Keep the most important parts (system message + recent context)
        """
        token_count = self.count_tokens(prompt)

        if token_count <= max_tokens:
            return prompt

        # Simple optimization: truncate middle, keep beginning and end
        lines = prompt.split("\n")
        if len(lines) <= 3:
            # If very few lines, just truncate
            tokens_per_char = token_count / len(prompt)
            target_chars = int(max_tokens / tokens_per_char * 0.9)  # 90% safety margin
            return prompt[:target_chars] + "...[truncated for cost optimization]"

        # Keep first few and last few lines, truncate middle
        keep_start = len(lines) // 4
        keep_end = len(lines) // 4

        optimized_lines = (
            lines[:keep_start]
            + ["...[content truncated for cost optimization]..."]
            + lines[-keep_end:]
        )

        return "\n".join(optimized_lines)

    def export_usage_data(self) -> list:
        """Export usage data for analysis"""
        return [asdict(usage) for usage in self.usage_log]

    def get_cost_optimization_suggestions(self) -> Dict[str, Any]:
        """Analyze usage patterns and suggest cost optimizations"""
        if not self.usage_log:
            return {"suggestions": [], "potential_savings": 0.0}

        suggestions = []
        potential_savings = 0.0

        # Analyze cache hit rates
        total_calls = len(self.usage_log)
        cache_hits = len([u for u in self.usage_log if u.cache_hit])
        cache_hit_rate = cache_hits / total_calls if total_calls > 0 else 0

        if cache_hit_rate < 0.3:  # Less than 30% cache hits
            suggestions.append(
                {
                    "type": "cache_optimization",
                    "title": "Improve Cache Hit Rate",
                    "description": (
                        f"Current cache hit rate: {cache_hit_rate:.1%}. "
                        "Consider increasing cache TTL or improving cache keys."
                    ),
                    "impact": "medium",
                }
            )

        # Analyze prompt optimization opportunities
        unoptimized_calls = [u for u in self.usage_log if not u.optimization_applied]
        if len(unoptimized_calls) > total_calls * 0.5:  # More than 50% unoptimized
            avg_tokens_unoptimized = sum(
                u.total_tokens for u in unoptimized_calls
            ) / len(unoptimized_calls)
            if avg_tokens_unoptimized > 3000:  # High token usage
                suggestions.append(
                    {
                        "type": "prompt_optimization",
                        "title": "Enable Prompt Optimization",
                        "description": (
                            f"Average token usage: {avg_tokens_unoptimized:.0f}. "
                            "Enable automatic prompt optimization for cost savings."
                        ),
                        "impact": "high",
                    }
                )
                potential_savings += sum(
                    u.cost_estimate * 0.2 for u in unoptimized_calls
                )  # Estimate 20% savings

        # Analyze model selection
        high_cost_calls = [
            u
            for u in self.usage_log
            if u.model_used in ["claude-3-opus", "claude-3-sonnet"]
        ]
        if high_cost_calls:
            total_high_cost = sum(u.cost_estimate for u in high_cost_calls)
            suggestions.append(
                {
                    "type": "model_optimization",
                    "title": "Consider Model Downgrade",
                    "description": (
                        f"${total_high_cost:.4f} spent on premium models. "
                        "Evaluate if Haiku can handle some tasks."
                    ),
                    "impact": "high",
                }
            )

        # Analyze error rates
        failed_calls = [u for u in self.usage_log if not u.success]
        error_rate = len(failed_calls) / total_calls if total_calls > 0 else 0
        if error_rate > 0.1:  # More than 10% error rate
            wasted_cost = sum(u.cost_estimate for u in failed_calls)
            suggestions.append(
                {
                    "type": "error_reduction",
                    "title": "Reduce API Failures",
                    "description": (
                        f"Error rate: {error_rate:.1%}. "
                        f"${wasted_cost:.4f} wasted on failed calls."
                    ),
                    "impact": "medium",
                }
            )
            potential_savings += wasted_cost

        return {
            "suggestions": suggestions,
            "potential_savings": potential_savings,
            "cache_hit_rate": cache_hit_rate,
            "error_rate": error_rate,
            "total_cost": sum(u.cost_estimate for u in self.usage_log),
        }

    def get_agent_performance_analytics(self) -> Dict[str, Any]:
        """Get performance analytics by agent"""
        if not self.usage_log:
            return {}

        agent_stats = {}
        for usage in self.usage_log:
            agent = usage.agent_role
            if agent not in agent_stats:
                agent_stats[agent] = {
                    "total_calls": 0,
                    "total_cost": 0.0,
                    "total_tokens": 0,
                    "avg_response_time": 0.0,
                    "success_rate": 0.0,
                    "cache_hit_rate": 0.0,
                    "avg_confidence": 0.0,
                    "response_times": [],
                    "costs": [],
                    "successes": 0,
                    "cache_hits": 0,
                    "confidence_scores": [],
                }

            stats = agent_stats[agent]
            stats["total_calls"] += 1
            stats["total_cost"] += usage.cost_estimate
            stats["total_tokens"] += usage.total_tokens
            stats["response_times"].append(usage.response_time)
            stats["costs"].append(usage.cost_estimate)

            if usage.success:
                stats["successes"] += 1
            if usage.cache_hit:
                stats["cache_hits"] += 1
            if usage.confidence_score is not None:
                stats["confidence_scores"].append(usage.confidence_score)

        # Calculate averages
        for agent, stats in agent_stats.items():
            stats["avg_response_time"] = sum(stats["response_times"]) / len(
                stats["response_times"]
            )
            stats["success_rate"] = stats["successes"] / stats["total_calls"]
            stats["cache_hit_rate"] = stats["cache_hits"] / stats["total_calls"]
            if stats["confidence_scores"]:
                stats["avg_confidence"] = sum(stats["confidence_scores"]) / len(
                    stats["confidence_scores"]
                )

            # Clean up temporary lists
            del stats["response_times"]
            del stats["costs"]
            del stats["successes"]
            del stats["cache_hits"]
            del stats["confidence_scores"]

        return agent_stats

    def log_cache_hit(
        self,
        cache_key: str,
        agent_role: AgentRole,
        task: str,
        rfe_id: Optional[str] = None,
    ):
        """Log when a response was served from cache"""
        if self._activity_tracker:
            from ai_models.activity_tracker import ActivityType

            self._activity_tracker.log_activity(
                activity_type=ActivityType.COST_OPTIMIZATION,
                action="cache_hit",
                agent_role=agent_role,
                rfe_id=rfe_id,
                context={"cache_key": cache_key, "task": task},
            )

    def log_optimization_applied(
        self,
        original_tokens: int,
        optimized_tokens: int,
        agent_role: AgentRole,
        task: str,
    ):
        """Log when prompt optimization was applied"""
        savings_percent = (original_tokens - optimized_tokens) / original_tokens * 100
        if self._activity_tracker:
            from ai_models.activity_tracker import ActivityType

            self._activity_tracker.log_activity(
                activity_type=ActivityType.COST_OPTIMIZATION,
                action="prompt_optimized",
                agent_role=agent_role,
                context={
                    "original_tokens": original_tokens,
                    "optimized_tokens": optimized_tokens,
                    "savings_percent": savings_percent,
                    "task": task,
                },
            )
