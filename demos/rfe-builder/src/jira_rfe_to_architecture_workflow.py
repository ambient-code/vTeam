"""
Jira RFE to Architecture Workflow

This workflow takes a Jira RFE as input and generates detailed architecture
and epics/stories documents using AI agents. This is a separate workflow from
the initial RFE creation process.
"""

import re
import time
from typing import Any, Dict, List, Literal, Optional

from dotenv import load_dotenv
from llama_index.core import Settings
from llama_index.core.chat_ui.events import ArtifactEvent, UIEvent
from llama_index.core.chat_ui.models.artifact import (
    Artifact,
    ArtifactType,
    DocumentArtifactData,
    DocumentArtifactSource,
)
from llama_index.core.llms import LLM
from llama_index.core.prompts import PromptTemplate
from llama_index.core.workflow import (
    Context,
    Event,
    StartEvent,
    StopEvent,
    Workflow,
    step,
)
from pydantic import BaseModel, Field

from src.agents import RFEAgentManager, get_agent_personas
from src.settings import init_settings


def create_jira_rfe_to_architecture_workflow() -> Workflow:
    load_dotenv()
    init_settings()
    return JiraRFEToArchitectureWorkflow(timeout=300.0)


class JiraRFEInput(BaseModel):
    """Input for processing a Jira RFE"""

    rfe_key: str = Field(description="Jira RFE key/ID")
    rfe_content: str = Field(description="RFE content from Jira")
    additional_context: Optional[str] = Field(
        default=None, description="Additional context or requirements"
    )


class ArchitectureAnalysisEvent(Event):
    rfe_input: JiraRFEInput
    agent_insights: List[Dict[str, Any]]


class ArtifactGenerationEvent(Event):
    rfe_input: JiraRFEInput
    agent_insights: List[Dict[str, Any]]
    architecture_content: str
    epics_content: str


class JiraRFEWorkflowUIEventData(BaseModel):
    """UI event data for Jira RFE workflow"""

    stage: Literal[
        "analyzing", "generating_architecture", "generating_epics", "completed"
    ] = Field(description="Current workflow stage")
    rfe_key: Optional[str] = Field(
        default=None, description="Jira RFE key being processed"
    )
    description: Optional[str] = Field(default=None, description="Stage description")
    progress: int = Field(default=0, description="Progress percentage")
    agent_streaming: Optional[Dict[str, Any]] = Field(
        default=None, description="Agent streaming data"
    )


class JiraRFEToArchitectureWorkflow(Workflow):
    """
    Workflow for processing Jira RFEs into detailed architecture and epics/stories.
    Uses AI agents to analyze the RFE and generate comprehensive implementation documents.
    """

    def __init__(self, **kwargs: Any):
        super().__init__(**kwargs)
        self.llm: LLM = Settings.llm
        self.agent_manager = RFEAgentManager()

    @step
    async def analyze_jira_rfe(
        self, ctx: Context, ev: StartEvent
    ) -> ArchitectureAnalysisEvent:
        """Analyze the Jira RFE using AI agents"""

        rfe_key = ev.get("rfe_key", "")
        rfe_content = ev.get("rfe_content", "")
        additional_context = ev.get("additional_context", "")

        rfe_input = JiraRFEInput(
            rfe_key=rfe_key,
            rfe_content=rfe_content,
            additional_context=additional_context,
        )

        ctx.write_event_to_stream(
            UIEvent(
                type="jira_rfe_workflow_progress",
                data=JiraRFEWorkflowUIEventData(
                    stage="analyzing",
                    rfe_key=rfe_key,
                    description="Analyzing Jira RFE with AI agents for detailed design...",
                    progress=10,
                ),
            )
        )

        # Get agent personas and analyze RFE for architecture/implementation
        agent_personas = await get_agent_personas()
        agent_insights = []

        if agent_personas:
            # Filter to relevant agents for architecture/implementation analysis
            architecture_agents = {
                k: v
                for k, v in agent_personas.items()
                if k
                in [
                    "backend_eng",
                    "frontend_eng",
                    "architect",
                    "uxd",
                ]  # Focus on implementation agents
            }

            analysis_prompt = f"""
            Analyze this Jira RFE for detailed architecture and implementation planning:
            
            RFE Key: {rfe_key}
            RFE Content: {rfe_content}
            Additional Context: {additional_context or 'None provided'}
            
            Focus on:
            - Technical architecture requirements
            - Implementation approach
            - Component design
            - Integration points
            - Technology stack recommendations
            - Epic/story breakdown
            """

            for persona_key, persona_config in architecture_agents.items():
                try:
                    async for stream_event in self.agent_manager.analyze_rfe_streaming(
                        persona_key, analysis_prompt, persona_config
                    ):
                        # Forward agent events to multi-agent component
                        ctx.write_event_to_stream(
                            UIEvent(
                                type="multi_agent_analysis",
                                data={
                                    "agent_key": persona_key,
                                    "agent_name": persona_config.get(
                                        "name", persona_key
                                    ),
                                    "agent_role": persona_config.get("role", "Analyst"),
                                    "stream_event": stream_event,
                                },
                            )
                        )

                        if stream_event.get("type") == "complete":
                            agent_insights.append(stream_event.get("result"))
                except Exception as e:
                    print(f"Agent {persona_key} error: {e}")

        return ArchitectureAnalysisEvent(
            rfe_input=rfe_input, agent_insights=agent_insights
        )

    @step
    async def generate_architecture_document(
        self, ctx: Context, ev: ArchitectureAnalysisEvent
    ) -> ArtifactGenerationEvent:
        """Generate detailed architecture document"""

        ctx.write_event_to_stream(
            UIEvent(
                type="jira_rfe_workflow_progress",
                data=JiraRFEWorkflowUIEventData(
                    stage="generating_architecture",
                    rfe_key=ev.rfe_input.rfe_key,
                    description="Generating architecture document...",
                    progress=40,
                ),
            )
        )

        architecture_content = await self._generate_architecture_from_rfe(
            ev.rfe_input, ev.agent_insights
        )

        ctx.write_event_to_stream(
            UIEvent(
                type="jira_rfe_workflow_progress",
                data=JiraRFEWorkflowUIEventData(
                    stage="generating_epics",
                    rfe_key=ev.rfe_input.rfe_key,
                    description="Generating epics and user stories...",
                    progress=70,
                ),
            )
        )

        epics_content = await self._generate_epics_from_rfe(
            ev.rfe_input, ev.agent_insights
        )

        return ArtifactGenerationEvent(
            rfe_input=ev.rfe_input,
            agent_insights=ev.agent_insights,
            architecture_content=architecture_content,
            epics_content=epics_content,
        )

    @step
    async def emit_artifacts(
        self, ctx: Context, ev: ArtifactGenerationEvent
    ) -> StopEvent:
        """Emit the generated architecture and epics artifacts"""

        # Emit Architecture document
        ctx.write_event_to_stream(
            ArtifactEvent(
                data=Artifact(
                    id="jira_architecture",
                    type=ArtifactType.DOCUMENT,
                    created_at=int(time.time()),
                    data=DocumentArtifactData(
                        title=f"Architecture Document - {ev.rfe_input.rfe_key}",
                        content=ev.architecture_content,
                        type="markdown",
                        sources=[DocumentArtifactSource(id=ev.rfe_input.rfe_key)],
                    ),
                ),
            )
        )

        # Emit Epics & Stories document
        ctx.write_event_to_stream(
            ArtifactEvent(
                data=Artifact(
                    id="jira_epics_stories",
                    type=ArtifactType.DOCUMENT,
                    created_at=int(time.time()),
                    data=DocumentArtifactData(
                        title=f"Epics & Stories - {ev.rfe_input.rfe_key}",
                        content=ev.epics_content,
                        type="markdown",
                        sources=[DocumentArtifactSource(id=ev.rfe_input.rfe_key)],
                    ),
                ),
            )
        )

        ctx.write_event_to_stream(
            UIEvent(
                type="jira_rfe_workflow_progress",
                data=JiraRFEWorkflowUIEventData(
                    stage="completed",
                    rfe_key=ev.rfe_input.rfe_key,
                    description="Architecture and Epics & Stories generated successfully!",
                    progress=100,
                ),
            )
        )

        return StopEvent(
            result={
                "rfe_key": ev.rfe_input.rfe_key,
                "architecture_content": ev.architecture_content,
                "epics_content": ev.epics_content,
                "agent_insights": ev.agent_insights,
            }
        )

    async def _generate_architecture_from_rfe(
        self, rfe_input: JiraRFEInput, agent_insights: List[Dict[str, Any]]
    ) -> str:
        """Generate architecture document from Jira RFE and agent analysis"""

        insights_text = "\n".join(
            [
                f"{insight.get('persona', 'Agent')}: {insight.get('analysis', 'No analysis')}"
                for insight in agent_insights
                if insight
            ]
        )

        prompt = f"""
        Create a detailed architecture document based on this Jira RFE and agent analysis:
        
        RFE Key: {rfe_input.rfe_key}
        RFE Content: {rfe_input.rfe_content}
        Additional Context: {rfe_input.additional_context or 'None provided'}
        
        Agent Analysis:
        {insights_text}
        
        Create a comprehensive architecture document that includes:
        - Executive Summary
        - System Architecture Overview
        - Component Architecture
        - Data Architecture & Flow
        - Integration Architecture
        - Technology Stack & Rationale
        - Security Architecture
        - Performance & Scalability Considerations
        - Deployment Architecture
        - Architecture Decision Records (ADRs)
        
        Use markdown formatting with clear sections and include architectural diagrams described in text.
        Focus on implementation-ready details that development teams can use.
        """

        response = await self.llm.acomplete(prompt)
        return response.text.strip()

    async def _generate_epics_from_rfe(
        self, rfe_input: JiraRFEInput, agent_insights: List[Dict[str, Any]]
    ) -> str:
        """Generate epics and user stories from Jira RFE and agent analysis"""

        insights_text = "\n".join(
            [
                f"{insight.get('persona', 'Agent')}: {insight.get('analysis', 'No analysis')}"
                for insight in agent_insights
                if insight
            ]
        )

        prompt = f"""
        Create detailed epics and user stories based on this Jira RFE and agent analysis:
        
        RFE Key: {rfe_input.rfe_key}
        RFE Content: {rfe_input.rfe_content}
        Additional Context: {rfe_input.additional_context or 'None provided'}
        
        Agent Analysis:
        {insights_text}
        
        Create a comprehensive implementation plan that includes:
        
        ## Epic Structure
        For each Epic, provide:
        - Epic Title and Theme
        - Epic Description and Goals
        - Success Criteria
        - Epic-level Acceptance Criteria
        - Dependencies and Assumptions
        
        ## User Stories
        For each User Story, provide:
        - Story Title (As a... I want... So that... format)
        - Detailed Description
        - Acceptance Criteria (Given/When/Then format)
        - Story Points Estimation (Fibonacci scale)
        - Definition of Done
        - Technical Notes
        - Dependencies
        
        ## Additional Sections
        - Epic Roadmap and Sequencing
        - Cross-Epic Dependencies
        - Risk Assessment
        - Testing Strategy per Epic
        
        Use markdown formatting with clear hierarchy.
        Focus on implementable, testable stories that align with the architecture.
        Ensure stories are appropriately sized (typically 1-13 story points).
        """

        response = await self.llm.acomplete(prompt)
        return response.text.strip()


# Export for LlamaDeploy
jira_rfe_to_architecture_workflow = create_jira_rfe_to_architecture_workflow()
