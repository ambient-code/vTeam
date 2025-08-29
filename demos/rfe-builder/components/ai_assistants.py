"""
AI-powered assistants for each RFE workflow agent
Provides role-specific guidance and decision support with enhanced activity tracking
"""

import time
from typing import Any, Dict, List, Optional

import streamlit as st
from ai_models.cost_tracker import CostTracker
from ai_models.prompt_manager import PromptManager
from anthropic import Anthropic
from data.rfe_models import RFE, AgentRole

# Activity tracking integration
try:
    from ai_models.activity_tracker import get_global_tracker

    ACTIVITY_TRACKING_ENABLED = True
except ImportError:
    ACTIVITY_TRACKING_ENABLED = False


class AgentAIAssistant:
    """Base class for agent-specific AI assistants with enhanced activity tracking"""

    def __init__(self, agent_role: AgentRole):
        self.agent_role = agent_role
        self.prompt_manager = PromptManager()
        self.cost_tracker = CostTracker(enable_activity_logging=True)

        # Initialize Anthropic client
        self.anthropic_client = self._get_anthropic_client()

        # Activity tracking
        self.activity_tracker = None
        if ACTIVITY_TRACKING_ENABLED:
            self.activity_tracker = get_global_tracker()

    def _get_anthropic_client(self) -> Optional[Anthropic]:
        """Get Anthropic client with error handling"""
        try:
            if hasattr(st, "secrets") and "ANTHROPIC_API_KEY" in st.secrets:
                return Anthropic(api_key=st.secrets["ANTHROPIC_API_KEY"])
            else:
                import os

                api_key = os.getenv("ANTHROPIC_API_KEY")
                if api_key:
                    return Anthropic(api_key=api_key)
        except Exception:
            pass
        return None

    def render_assistance_panel(
        self, rfe: RFE, context: Optional[Dict[str, Any]] = None
    ):
        """Render the AI assistance panel for this agent"""
        if not self.anthropic_client:
            st.warning("🤖 AI assistant requires Anthropic API key configuration")
            return

        with st.expander("🤖 AI Assistant", expanded=False):
            self._render_assistant_interface(rfe, context)

    def _render_assistant_interface(
        self, rfe: RFE, context: Optional[Dict[str, Any]] = None
    ):
        """Render the specific assistant interface (to be overridden)"""
        st.info("AI assistant available for this agent")

        if st.button(f"Get {self.agent_role.value.replace('_', ' ').title()} Guidance"):
            with st.spinner("Getting AI guidance..."):
                guidance = self.get_agent_guidance(rfe, context)
                st.markdown("### 💡 AI Guidance")
                st.write(guidance)

    def get_agent_guidance(
        self, rfe: RFE, context: Optional[Dict[str, Any]] = None
    ) -> str:
        """Get AI guidance for this agent and RFE with comprehensive tracking"""
        if not self.anthropic_client:
            return "AI guidance unavailable - missing API configuration"

        start_time = time.time()
        task_context = self._get_task_context(rfe)

        # Log user interaction
        if self.activity_tracker:
            self.activity_tracker.log_user_interaction(
                action="request_guidance",
                user_input=f"Requested guidance for {task_context}",
                agent_role=self.agent_role,
                rfe_id=rfe.id,
                context=context or {},
            )

        try:
            # Get appropriate prompt template
            prompt_template = self.prompt_manager.get_agent_prompt(
                self.agent_role, task_context, rfe
            )

            # Log prompt selection
            template_name = prompt_template.get("metadata", {}).get(
                "template_name", "unknown"
            )
            if self.activity_tracker:
                self.activity_tracker.log_prompt_selection(
                    agent_role=self.agent_role,
                    prompt_template=template_name,
                    selection_reason=f"Selected for task: {task_context}",
                    context={
                        "workflow_aware": prompt_template.get("workflow_aware", False)
                    },
                    rfe_id=rfe.id,
                )

            # Format prompt with RFE and context data
            prompt_context = self._build_prompt_context(rfe, context)
            formatted_prompt = self.prompt_manager.format_prompt(
                prompt_template, **prompt_context
            )

            # Count tokens for cost tracking
            prompt_text = (
                f"{formatted_prompt['system']}\n\nUser: {formatted_prompt['user']}"
            )
            prompt_tokens = self.cost_tracker.count_tokens(prompt_text)

            # Make API call
            api_start = time.time()
            response = self.anthropic_client.messages.create(
                model="claude-3-haiku-20240307",
                max_tokens=800,
                system=formatted_prompt["system"],
                messages=[{"role": "user", "content": formatted_prompt["user"]}],
            )
            api_time = time.time() - api_start

            response_text = response.content[0].text
            completion_tokens = self.cost_tracker.count_tokens(response_text)

            # Generate confidence score based on response characteristics
            confidence_score = self._calculate_confidence_score(
                response_text, prompt_template
            )

            # Log comprehensive API usage
            self.cost_tracker.log_usage(
                agent_role=self.agent_role,
                task=task_context,
                prompt_tokens=prompt_tokens,
                completion_tokens=completion_tokens,
                response_time=api_time,
                rfe_id=rfe.id,
                prompt_template_used=template_name,
                success=True,
                confidence_score=confidence_score,
                decision_context=f"Guidance for {task_context}",
            )

            # Log agent decision
            if self.activity_tracker:
                decision_rationale = self._extract_decision_rationale(response_text)
                self.activity_tracker.log_agent_decision(
                    agent_role=self.agent_role,
                    action="provide_guidance",
                    decision_rationale=decision_rationale,
                    confidence_score=confidence_score,
                    prompt_template_used=template_name,
                    context={
                        "task_context": task_context,
                        "response_length": len(response_text),
                    },
                    rfe_id=rfe.id,
                    system_response=response_text,
                )

            return response_text

        except Exception as e:
            error_time = time.time() - start_time
            error_message = str(e)

            # Log error
            if self.activity_tracker:
                self.activity_tracker.log_error(
                    error_message="AI guidance request failed",
                    error_details=error_message,
                    agent_role=self.agent_role,
                    context={"task_context": task_context, "error_time": error_time},
                    rfe_id=rfe.id,
                )

            return f"Error getting AI guidance: {error_message}"

    def _get_task_context(self, rfe: RFE) -> str:
        """Get the current task context for this agent (to be overridden)"""
        return "general_assistance"

    def _build_prompt_context(
        self, rfe: RFE, additional_context: Optional[Dict[str, Any]] = None
    ) -> Dict[str, Any]:
        """Build context dictionary for prompt formatting"""
        context = {
            "title": rfe.title,
            "description": rfe.description,
            "business_justification": rfe.business_justification or "Not provided",
            "technical_requirements": rfe.technical_requirements or "Not provided",
            "success_criteria": rfe.success_criteria or "Not provided",
            "current_step": rfe.current_step,
            "rfe_id": rfe.id,
            "priority": getattr(rfe, "priority", "Not set"),
            "current_status": rfe.current_status.value,
        }

        # Add RFE history context
        if rfe.history:
            recent_history = []
            for entry in rfe.history[-3:]:  # Last 3 history entries
                timestamp = entry["timestamp"].strftime("%Y-%m-%d %H:%M")
                action = entry.get("action", "Unknown action")
                notes = entry.get("notes", "")
                recent_history.append(f"{timestamp}: {action} - {notes}")
            context["decision_history"] = "\n".join(recent_history)
        else:
            context["decision_history"] = "No previous decisions recorded"

        # Add additional context if provided
        if additional_context:
            context.update(additional_context)

        return context

    def _calculate_confidence_score(
        self, response_text: str, prompt_template: Dict[str, Any]
    ) -> float:
        """Calculate confidence score based on response characteristics"""
        score = 0.5  # Base score

        # Length-based confidence (longer responses often more comprehensive)
        if len(response_text) > 500:
            score += 0.1
        elif len(response_text) < 100:
            score -= 0.1

        # Structured response indicators
        if any(
            marker in response_text.lower() for marker in ["1.", "2.", "-", "##", "**"]
        ):
            score += 0.1  # Structured responses tend to be more thought-out

        # Uncertainty indicators
        uncertainty_phrases = [
            "might",
            "could",
            "possibly",
            "perhaps",
            "maybe",
            "unsure",
        ]
        uncertainty_count = sum(
            1 for phrase in uncertainty_phrases if phrase in response_text.lower()
        )
        score -= min(uncertainty_count * 0.05, 0.2)  # Cap penalty at 0.2

        # Confidence indicators
        confidence_phrases = [
            "will",
            "should",
            "recommend",
            "suggest",
            "clear",
            "definitely",
        ]
        confidence_count = sum(
            1 for phrase in confidence_phrases if phrase in response_text.lower()
        )
        score += min(confidence_count * 0.05, 0.2)  # Cap bonus at 0.2

        # Workflow awareness bonus
        if prompt_template.get("workflow_aware", False):
            score += 0.1

        return max(0.0, min(1.0, score))  # Clamp between 0 and 1

    def _extract_decision_rationale(self, response_text: str) -> str:
        """Extract key decision rationale from response"""
        # Simple heuristic: take the first few sentences or key points
        sentences = response_text.split(". ")

        # Look for rationale indicators
        rationale_indicators = [
            "because",
            "due to",
            "since",
            "as",
            "reason",
            "therefore",
        ]
        rationale_sentences = []

        for sentence in sentences[:5]:  # Check first 5 sentences
            if any(indicator in sentence.lower() for indicator in rationale_indicators):
                rationale_sentences.append(sentence.strip())

        if rationale_sentences:
            return ". ".join(rationale_sentences)
        else:
            # Fallback: return first 150 characters
            return (
                response_text[:150] + "..."
                if len(response_text) > 150
                else response_text
            )


class ParkerAIAssistant(AgentAIAssistant):
    """AI Assistant for Parker (Product Manager)"""

    def __init__(self):
        super().__init__(AgentRole.PARKER_PM)

    def _get_task_context(self, rfe: RFE) -> str:
        """Get Parker's current task context"""
        if rfe.current_step == 1:
            return "step1_prioritization"
        elif rfe.current_step == 6:
            return "step6_communication"
        else:
            return "general_pm_guidance"

    def _render_assistant_interface(
        self, rfe: RFE, context: Optional[Dict[str, Any]] = None
    ):
        """Parker-specific assistant interface"""
        if rfe.current_step == 1:
            st.markdown("**🎯 Prioritization Assistant**")
            st.markdown("I can help you assess business priority and impact.")

            col1, col2 = st.columns(2)
            with col1:
                if st.button("📊 Analyze Business Impact"):
                    guidance = self.get_business_impact_analysis(rfe)
                    st.markdown("### Business Impact Analysis")
                    st.write(guidance)

            with col2:
                if st.button("🎯 Suggest Priority Level"):
                    guidance = self.get_priority_recommendation(rfe)
                    st.markdown("### Priority Recommendation")
                    st.write(guidance)

        elif rfe.current_step == 6:
            st.markdown("**📢 Communication Assistant**")
            st.markdown("I can help draft stakeholder communications.")

            stakeholder_type = st.selectbox(
                "Stakeholder Type",
                ["RFE Submitter", "Engineering Team", "Management", "All Stakeholders"],
            )

            if st.button("✉️ Draft Communication"):
                context_update = {"stakeholder_type": stakeholder_type}
                guidance = self.get_agent_guidance(rfe, context_update)
                st.markdown("### Draft Communication")
                st.write(guidance)
        else:
            super()._render_assistant_interface(rfe, context)

    def get_business_impact_analysis(self, rfe: RFE) -> str:
        """Get specific business impact analysis"""
        context = {"analysis_type": "business_impact"}
        return self.get_agent_guidance(rfe, context)

    def get_priority_recommendation(self, rfe: RFE) -> str:
        """Get priority level recommendation"""
        context = {"analysis_type": "priority_recommendation"}
        return self.get_agent_guidance(rfe, context)


class ArchieAIAssistant(AgentAIAssistant):
    """AI Assistant for Archie (Architect)"""

    def __init__(self):
        super().__init__(AgentRole.ARCHIE_ARCHITECT)

    def _get_task_context(self, rfe: RFE) -> str:
        """Get Archie's current task context"""
        if rfe.current_step == 2:
            return "step2_technical_review"
        elif rfe.current_step == 4:
            return "step4_acceptance_criteria"
        else:
            return "general_architecture_guidance"

    def _render_assistant_interface(
        self, rfe: RFE, context: Optional[Dict[str, Any]] = None
    ):
        """Archie-specific assistant interface"""
        if rfe.current_step == 2:
            st.markdown("**🏛️ Technical Review Assistant**")

            col1, col2 = st.columns(2)
            with col1:
                if st.button("🔍 Assess Feasibility"):
                    guidance = self.get_feasibility_assessment(rfe)
                    st.markdown("### Technical Feasibility")
                    st.write(guidance)

            with col2:
                if st.button("🏗️ Architecture Impact"):
                    guidance = self.get_architecture_impact(rfe)
                    st.markdown("### Architecture Impact")
                    st.write(guidance)

        elif rfe.current_step == 4:
            st.markdown("**✅ Acceptance Criteria Assistant**")

            if st.button("📋 Evaluate Acceptance Criteria"):
                guidance = self.get_agent_guidance(rfe)
                st.markdown("### Acceptance Criteria Evaluation")
                st.write(guidance)
        else:
            super()._render_assistant_interface(rfe, context)

    def get_feasibility_assessment(self, rfe: RFE) -> str:
        """Get technical feasibility assessment"""
        context = {"analysis_type": "feasibility"}
        return self.get_agent_guidance(rfe, context)

    def get_architecture_impact(self, rfe: RFE) -> str:
        """Get architecture impact analysis"""
        context = {"analysis_type": "architecture_impact"}
        return self.get_agent_guidance(rfe, context)


class StellaAIAssistant(AgentAIAssistant):
    """AI Assistant for Stella (Staff Engineer)"""

    def __init__(self):
        super().__init__(AgentRole.STELLA_STAFF_ENGINEER)

    def _get_task_context(self, rfe: RFE) -> str:
        """Get Stella's current task context"""
        if rfe.current_step == 3:
            return "step3_completeness_check"
        elif rfe.current_step == 5:
            return "step5_final_decision"
        else:
            return "general_engineering_guidance"

    def _render_assistant_interface(
        self, rfe: RFE, context: Optional[Dict[str, Any]] = None
    ):
        """Stella-specific assistant interface"""
        if rfe.current_step == 3:
            st.markdown("**📋 Completeness Check Assistant**")

            if st.button("🔍 Check RFE Completeness"):
                guidance = self.get_completeness_analysis(rfe)
                st.markdown("### Completeness Analysis")
                st.write(guidance)

        elif rfe.current_step == 5:
            st.markdown("**⚖️ Final Decision Assistant**")

            if st.button("🎯 Final Decision Analysis"):
                guidance = self.get_agent_guidance(rfe)
                st.markdown("### Final Decision Analysis")
                st.write(guidance)
        else:
            super()._render_assistant_interface(rfe, context)

    def get_completeness_analysis(self, rfe: RFE) -> str:
        """Get RFE completeness analysis"""
        context = {"analysis_type": "completeness"}
        return self.get_agent_guidance(rfe, context)


class DerekAIAssistant(AgentAIAssistant):
    """AI Assistant for Derek (Delivery Owner)"""

    def __init__(self):
        super().__init__(AgentRole.DEREK_DELIVERY_OWNER)

    def _get_task_context(self, rfe: RFE) -> str:
        """Get Derek's current task context"""
        if rfe.current_step == 7:
            return "step7_ticket_creation"
        else:
            return "general_delivery_guidance"

    def _render_assistant_interface(
        self, rfe: RFE, context: Optional[Dict[str, Any]] = None
    ):
        """Derek-specific assistant interface"""
        if rfe.current_step == 7:
            st.markdown("**🎫 Ticket Creation Assistant**")

            col1, col2 = st.columns(2)
            with col1:
                if st.button("📝 Generate Epic Template"):
                    guidance = self.get_epic_template(rfe)
                    st.markdown("### JIRA Epic Template")
                    st.code(guidance)

            with col2:
                if st.button("📋 Break Down Tasks"):
                    guidance = self.get_task_breakdown(rfe)
                    st.markdown("### Development Task Breakdown")
                    st.write(guidance)
        else:
            super()._render_assistant_interface(rfe, context)

    def get_epic_template(self, rfe: RFE) -> str:
        """Get JIRA epic template"""
        context = {"output_type": "epic_template"}
        return self.get_agent_guidance(rfe, context)

    def get_task_breakdown(self, rfe: RFE) -> str:
        """Get development task breakdown"""
        context = {"output_type": "task_breakdown"}
        return self.get_agent_guidance(rfe, context)


class AIAssistantFactory:
    """Factory class to create appropriate AI assistants"""

    _assistants = {
        AgentRole.PARKER_PM: ParkerAIAssistant,
        AgentRole.ARCHIE_ARCHITECT: ArchieAIAssistant,
        AgentRole.STELLA_STAFF_ENGINEER: StellaAIAssistant,
        AgentRole.DEREK_DELIVERY_OWNER: DerekAIAssistant,
    }

    @classmethod
    def create_assistant(cls, agent_role: AgentRole) -> AgentAIAssistant:
        """Create AI assistant for the specified agent role"""
        assistant_class = cls._assistants.get(agent_role, AgentAIAssistant)
        if assistant_class == AgentAIAssistant:
            return AgentAIAssistant(agent_role)  # Fallback for other agents
        else:
            return assistant_class()  # Specialized classes call super().__init__()

    @classmethod
    def get_available_agents(cls) -> List[AgentRole]:
        """Get list of agents with AI assistants"""
        return list(cls._assistants.keys())
