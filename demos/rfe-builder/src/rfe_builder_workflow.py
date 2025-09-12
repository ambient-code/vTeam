"""Simple RFE Builder: user input -> agents -> RFE -> artifacts -> done"""

import asyncio
import logging
import time
from enum import Enum
from typing import Any, Dict, List, Optional

from dotenv import load_dotenv
from llama_index.core import Settings
from llama_index.core.chat_ui.events import ArtifactEvent, UIEvent
from llama_index.core.chat_ui.models.artifact import (
    Artifact,
    ArtifactType,
    DocumentArtifactData,
)
from llama_index.core.llms import LLM
from llama_index.core.workflow import (
    Context,
    Event,
    StartEvent,
    StopEvent,
    Workflow,
    step,
)
from pydantic import BaseModel

from src.agents import RFEAgentManager, get_agent_personas
from src.mcp_config_validator import RFEBuilderMCPConfig
from src.services.mcp_service import get_mcp_service, initialize_mcp_service
from src.settings import init_settings

logger = logging.getLogger(__name__)


class RFEPhase(str, Enum):
    BUILDING = "building"
    GENERATING_PHASE_1 = "generating_phase_1"
    PHASE_1_READY = "phase_1_ready"
    EDITING = "editing"
    GENERATING_PHASE_2 = "generating_phase_2"
    COMPLETED = "completed"


class RFEArtifactType(str, Enum):
    RFE_DESCRIPTION = "rfe_description"
    FEATURE_REFINEMENT = "feature_refinement"
    ARCHITECTURE = "architecture"
    EPICS_STORIES = "epics_stories"


# Phase 1 artifacts (refinement phase)
PHASE_1_ARTIFACTS = [
    (RFEArtifactType.RFE_DESCRIPTION, "RFE Description"),
    (RFEArtifactType.FEATURE_REFINEMENT, "Feature Refinement"),
]

# Phase 2 artifacts (detailed design phase)
PHASE_2_ARTIFACTS = [
    (RFEArtifactType.ARCHITECTURE, "Architecture"),
    (RFEArtifactType.EPICS_STORIES, "Epics & Stories"),
]


class GenerateArtifactsEvent(Event):
    final_rfe: str
    context: Dict[str, Any]


class EditArtifactEvent(Event):
    user_input: str
    existing_artifacts: Dict[str, str]
    edit_instruction: str


class RFEBuilderUIEventData(BaseModel):
    """Simple UI event data"""

    phase: RFEPhase
    stage: str
    description: Optional[str] = None
    progress: int = 0
    agent_streaming: Optional[Dict[str, Any]] = None


def create_rfe_builder_workflow() -> Workflow:
    load_dotenv()
    init_settings()

    # Create workflow instance
    workflow = RFEBuilderWorkflow(timeout=300.0)

    # Initialize MCP integration asynchronously
    # Note: This will be handled during workflow execution
    logger.info("RFE Builder Workflow created with MCP integration capability")

    return workflow


class RFEBuilderWorkflow(Workflow):
    """RFE builder with editing support: user input -> agents -> artifacts -> editing

    Note on State Management:
    - Instance variables (session_artifacts, artifacts_generated) are reset on each new request
    - This ensures the 7-agent consultation always runs for fresh RFE creation
    - Editing functionality works within a single session but resets between sessions
    - This design prioritizes consistent agent consultation over persistent state
    """

    def __init__(self, **kwargs: Any):
        super().__init__(**kwargs)
        self.llm: LLM = Settings.llm
        self.agent_manager = RFEAgentManager()
        # Session state to track artifacts (reset per request to ensure fresh agent consultation)
        self.session_artifacts: Dict[str, str] = {}
        self.artifacts_generated = False
        # MCP integration status
        self.mcp_enabled = False
        self.mcp_status: Dict[str, Any] = {}

    @step
    async def start_rfe_builder(
        self, ctx: Context, ev: StartEvent
    ) -> GenerateArtifactsEvent | EditArtifactEvent:
        """Route to editing if artifacts exist, otherwise create new RFE"""
        user_msg = ev.get("user_msg", "")

        # Reset state for each new request to ensure fresh agent consultation
        # This prevents the workflow from getting "stuck" in editing mode
        # and ensures the 7 agents are consulted for every new RFE creation
        self.artifacts_generated = False
        self.session_artifacts.clear()

        # Initialize MCP integration if not already done
        if not self.mcp_enabled:
            try:
                mcp_status = await self.initialize_mcp_integration()
                self.mcp_enabled = mcp_status.get("mcp_enabled", False)
                self.mcp_status = mcp_status

                if self.mcp_enabled:
                    logger.info(
                        "MCP integration initialized successfully for this workflow"
                    )
                else:
                    logger.warning(
                        f"MCP integration failed: {mcp_status.get('error', 'Unknown error')}"
                    )
            except Exception as e:
                logger.error(f"Failed to initialize MCP integration: {e}")
                self.mcp_enabled = False

        # Since we reset state above, this condition will never be true now
        # Keeping the logic for potential future enhancement (e.g., explicit edit mode)
        if self.artifacts_generated and self.session_artifacts:
            return EditArtifactEvent(
                user_input=user_msg,
                existing_artifacts=self.session_artifacts.copy(),
                edit_instruction=user_msg,
            )

        # Get agent personas and build RFE
        agent_personas = await get_agent_personas()

        # Filter to only include specific agents
        filtered_agents = {
            "UX_RESEARCHER",
            "UX_FEATURE_LEAD",
            "ENGINEERING_MANAGER",
            "STAFF_ENGINEER",
            "TECHNICAL_WRITER",
            "UX_ARCHITECT",
            "PRODUCT_MANAGER",
        }

        agent_personas = {
            key: config
            for key, config in agent_personas.items()
            if key in filtered_agents
        }

        agent_insights = []

        if agent_personas:
            for persona_key, persona_config in agent_personas.items():
                try:
                    async for stream_event in self.agent_manager.analyze_rfe_streaming(
                        persona_key, user_msg, persona_config
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

        # Small delay to ensure agent completion events are processed first
        await asyncio.sleep(0.5)

        # Summarize all agent analyses
        if agent_insights:
            await self._summarize_agent_analyses(ctx, agent_insights)

        # Build final RFE from insights
        final_rfe = await self._build_final_rfe(user_msg, agent_insights)

        return GenerateArtifactsEvent(final_rfe=final_rfe, context={})

    @step
    async def generate_phase_1_artifacts(
        self, ctx: Context, ev: GenerateArtifactsEvent
    ) -> StopEvent:
        """Generate only Phase 1 artifacts (RFE + Feature Refinement)"""

        ctx.write_event_to_stream(
            UIEvent(
                type="rfe_builder_progress",
                data=RFEBuilderUIEventData(
                    phase=RFEPhase.GENERATING_PHASE_1,
                    stage="generating_phase_1",
                    description="Generating Phase 1 artifacts (RFE & Feature Refinement)...",
                    progress=50,
                ),
            )
        )

        phase_1_artifacts = {}

        # Generate only Phase 1 artifacts
        for artifact_type, display_name in PHASE_1_ARTIFACTS:
            content = await self._generate_simple_artifact(artifact_type, ev.final_rfe)
            phase_1_artifacts[artifact_type.value] = content
            # Store in session state for editing
            self.session_artifacts[artifact_type.value] = content

            # Emit artifact
            ctx.write_event_to_stream(
                ArtifactEvent(
                    data=Artifact(
                        id=artifact_type.value,
                        type=ArtifactType.DOCUMENT,
                        created_at=int(time.time()),
                        data=DocumentArtifactData(
                            title=display_name,
                            content=content,
                            type="markdown",
                            sources=[],
                        ),
                    )
                )
            )

        # Phase 1 complete - ready for iteration and then phase transition
        ctx.write_event_to_stream(
            UIEvent(
                type="rfe_builder_progress",
                data=RFEBuilderUIEventData(
                    phase=RFEPhase.PHASE_1_READY,
                    stage="phase_1_ready",
                    description="Phase 1 artifacts ready! You can now iterate on your RFE and Feature Refinement documents. When ready, continue to Phase 2 for Architecture and Epics & Stories.",
                    progress=100,
                ),
            )
        )

        # Show Create RFE button after Phase 1 completion
        ctx.write_event_to_stream(
            UIEvent(
                type="create_rfe_ready",
                data={
                    "message": "RFE documents are ready! Create the RFE in Jira when you're satisfied with the content.",
                    "artifacts": list(phase_1_artifacts.keys()),
                    "rfe_content": phase_1_artifacts.get("rfe_description", ""),
                    "refinement_content": phase_1_artifacts.get(
                        "feature_refinement", ""
                    ),
                },
            )
        )

        # Mark artifacts as generated for future editing
        self.artifacts_generated = True

        return StopEvent(
            result={
                "final_rfe": ev.final_rfe,
                "phase_1_artifacts": phase_1_artifacts,
                "ready_for_rfe_creation": True,
                "message": "Phase 1 complete! You can now iterate on your documents or create the RFE in Jira.",
                "editing_enabled": True,
            }
        )

    async def _summarize_agent_analyses(
        self, ctx: Context, agent_insights: List[Dict]
    ) -> None:
        """Summarize all agent analyses and stream as plain text to UI"""

        # Create summary prompt
        insights_text = "\n\n".join(
            [
                f"**{insight.get('persona', 'Agent')}:**\n{insight.get('analysis', 'No analysis')}"
                for insight in agent_insights
                if insight
            ]
        )

        summary_prompt = f"""
        Based on the following agent analyses, provide a concise summary that highlights:
        - Key themes and patterns across all analyses
        - Critical requirements and considerations
        - Main risks or challenges identified
        - Recommended next steps
        
        Agent Analyses:
        {insights_text}
        
        Provide a clear, structured summary in markdown format.
        """

        # Stream the summary generation
        ctx.write_event_to_stream(
            UIEvent(
                type="agent_analysis_summary",
                data={
                    "status": "generating",
                    "message": "Synthesizing insights from all agent analyses...",
                    "timestamp": int(time.time() * 1000),  # milliseconds
                },
            )
        )

        try:
            # Generate the summary (non-streaming for now to avoid async issues)
            response = await self.llm.acomplete(summary_prompt)
            summary_text = response.text.strip()

            # Send final complete event
            ctx.write_event_to_stream(
                UIEvent(
                    type="agent_analysis_summary",
                    data={
                        "status": "complete",
                        "summary": summary_text,
                        "message": "Agent analysis summary complete",
                        "timestamp": int(time.time() * 1000),
                    },
                )
            )
        except Exception as e:
            ctx.write_event_to_stream(
                UIEvent(
                    type="agent_analysis_summary",
                    data={
                        "status": "error",
                        "message": f"Failed to generate summary: {str(e)}",
                        "timestamp": int(time.time() * 1000),
                    },
                )
            )

    async def _build_final_rfe(
        self, user_input: str, agent_insights: List[Dict]
    ) -> str:
        """Simple RFE building from user input and agent insights"""

        insights_text = "\n".join(
            [
                f"{insight.get('persona', 'Agent')}: {insight.get('analysis', 'No analysis')}"
                for insight in agent_insights
                if insight
            ]
        )

        prompt = f"""
        Create a clear RFE (Request for Enhancement) document based on:
        
        User idea: {user_input}
        Agent analysis: {insights_text}
        
        Include:
        - Problem statement
        - Proposed solution  
        - Requirements
        - Success criteria
        """

        response = await self.llm.acomplete(prompt)
        return response.text.strip()

    async def _generate_simple_artifact(
        self, artifact_type: RFEArtifactType, final_rfe: str
    ) -> str:
        """Simple artifact generation"""

        artifact_prompts = {
            RFEArtifactType.RFE_DESCRIPTION: f"Create a detailed RFE document based on: {final_rfe}",
            RFEArtifactType.FEATURE_REFINEMENT: f"Create a feature breakdown document based on: {final_rfe}",
            RFEArtifactType.ARCHITECTURE: f"Create a system architecture document based on: {final_rfe}",
            RFEArtifactType.EPICS_STORIES: f"Create epics and user stories based on: {final_rfe}",
        }

        prompt = artifact_prompts[artifact_type]
        response = await self.llm.acomplete(prompt)
        return response.text.strip()

    @step
    async def edit_artifact(self, ctx: Context, ev: EditArtifactEvent) -> StopEvent:
        """Edit existing artifacts based on user input"""

        ctx.write_event_to_stream(
            UIEvent(
                type="rfe_builder_progress",
                data=RFEBuilderUIEventData(
                    phase=RFEPhase.EDITING,
                    stage="analyzing_edit",
                    description="Analyzing your edit request...",
                    progress=10,
                ),
            )
        )

        # Determine which artifact to edit (default to RFE if unclear)
        target_artifact = await self._determine_target_artifact(
            ev.user_input, ev.existing_artifacts
        )

        ctx.write_event_to_stream(
            UIEvent(
                type="rfe_builder_progress",
                data=RFEBuilderUIEventData(
                    phase=RFEPhase.EDITING,
                    stage="editing_artifact",
                    description=f"Editing {target_artifact.replace('_', ' ').title()}...",
                    progress=50,
                ),
            )
        )

        # Edit the target artifact
        updated_content = await self._edit_artifact_content(
            target_artifact, ev.edit_instruction, ev.existing_artifacts[target_artifact]
        )

        # Update session state
        self.session_artifacts[target_artifact] = updated_content

        # Emit updated artifact
        display_names = {
            "rfe_description": "RFE Description",
            "feature_refinement": "Feature Refinement",
        }

        ctx.write_event_to_stream(
            ArtifactEvent(
                data=Artifact(
                    id=target_artifact,
                    type=ArtifactType.DOCUMENT,
                    created_at=int(time.time()),
                    data=DocumentArtifactData(
                        title=display_names.get(
                            target_artifact, target_artifact.title()
                        ),
                        content=updated_content,
                        type="markdown",
                        sources=[],
                    ),
                )
            )
        )

        ctx.write_event_to_stream(
            UIEvent(
                type="rfe_builder_progress",
                data=RFEBuilderUIEventData(
                    phase=RFEPhase.EDITING,
                    stage="edit_complete",
                    description="Edit complete! You can make additional changes or create the RFE.",
                    progress=100,
                ),
            )
        )

        return StopEvent(
            result={
                "edited_artifact": target_artifact,
                "updated_content": updated_content,
                "message": f"Successfully updated {display_names.get(target_artifact, target_artifact)}!",
                "all_artifacts": self.session_artifacts,
                "editing_enabled": True,
            }
        )

    async def _determine_target_artifact(
        self, user_input: str, existing_artifacts: Dict[str, str]
    ) -> str:
        """Determine which artifact the user wants to edit"""
        user_lower = user_input.lower()

        # Simple keyword matching
        if any(word in user_lower for word in ["refinement", "feature", "technical"]):
            return "feature_refinement"
        else:
            return "rfe_description"  # Default to RFE document

    async def _edit_artifact_content(
        self, artifact_type: str, edit_instruction: str, current_content: str
    ) -> str:
        """Edit artifact content based on instruction"""

        prompt = f"""
        Edit the following document based on the user's instruction:
        
        EDIT INSTRUCTION: {edit_instruction}
        
        CURRENT DOCUMENT:
        {current_content}
        
        Please update the document according to the instruction while:
        1. Maintaining the overall structure and format
        2. Keeping existing good content that wasn't specifically targeted for change
        3. Making the requested changes clearly and comprehensively
        4. Ensuring the document remains coherent and well-formatted
        
        Return the complete updated document:
        """

        response = await self.llm.acomplete(prompt)
        return response.text.strip()

    async def initialize_mcp_integration(self) -> Dict[str, Any]:
        """
        Initialize MCP integration for enhanced RFE building.

        Returns:
            MCP initialization status
        """
        try:
            # Validate MCP configuration
            config_validator = RFEBuilderMCPConfig()
            config_result = config_validator.validate_configuration()

            if not config_result["valid"]:
                logger.warning(
                    f"MCP configuration invalid: {config_result.get('error')}"
                )
                return {
                    "mcp_enabled": False,
                    "error": config_result.get("error"),
                    "status": "configuration_invalid",
                }

            # Initialize MCP service
            mcp_service = await initialize_mcp_service()
            connection_status = await mcp_service.get_connection_status()

            self.mcp_enabled = True
            self.mcp_status = connection_status

            logger.info(
                f"MCP Integration initialized. Healthy servers: {connection_status['healthy_count']}"
            )

            return {
                "mcp_enabled": True,
                "connection_status": connection_status,
                "status": "initialized",
            }

        except Exception as e:
            logger.error(f"Failed to initialize MCP integration: {e}")
            return {
                "mcp_enabled": False,
                "error": str(e),
                "status": "initialization_failed",
            }

    async def enhance_rfe_with_mcp_data(
        self, rfe_content: Dict[str, Any], project_context: Optional[Dict] = None
    ) -> Dict[str, Any]:
        """
        Enhance RFE content with data from MCP servers.

        Args:
            rfe_content: Current RFE content
            project_context: Optional project context for targeted queries

        Returns:
            Enhanced RFE content with MCP data
        """
        try:
            mcp_service = get_mcp_service()

            # Get contextual data from MCP servers
            contextual_data = await mcp_service.get_contextual_data_for_rfe(
                rfe_title=rfe_content.get("title", ""),
                rfe_description=rfe_content.get("description", ""),
                project_context=project_context,
            )

            # Enhance RFE content with contextual data
            enhanced_content = rfe_content.copy()
            enhanced_content["mcp_context"] = contextual_data

            # Add related tickets to requirements if available
            jira_data = contextual_data["data_sources"].get("jira_tickets", {})
            if jira_data.get("success") and jira_data.get("tickets"):
                enhanced_content["related_tickets"] = jira_data["tickets"]

            # Add repository information if available
            github_data = contextual_data["data_sources"].get("github_repo", {})
            if github_data.get("success") and github_data.get("repository"):
                enhanced_content["repository_context"] = github_data["repository"]

            # Add documentation references if available
            confluence_data = contextual_data["data_sources"].get("confluence_docs", {})
            if confluence_data.get("success") and confluence_data.get("documents"):
                enhanced_content["documentation_references"] = confluence_data[
                    "documents"
                ]

            logger.info(
                f"Enhanced RFE with {contextual_data['summary']['total_data_points']} data points from MCP"
            )

            return enhanced_content

        except Exception as e:
            logger.error(f"Failed to enhance RFE with MCP data: {e}")
            # Return original content if enhancement fails
            rfe_content["mcp_enhancement_error"] = str(e)
            return rfe_content


# Export for LlamaDeploy
rfe_builder_workflow = create_rfe_builder_workflow()
