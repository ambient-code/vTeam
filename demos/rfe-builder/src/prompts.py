import os
from pathlib import Path
from typing import Dict


def load_prompt(prompt_name: str) -> str:
    """Load a prompt template from markdown file"""
    # Get prompts directory relative to this file's location
    prompt_path = Path(__file__).parent / "prompts" / f"{prompt_name}.md"

    if prompt_path.exists():
        return prompt_path.read_text()
    else:
        # Fallback to basic prompts if files don't exist
        return get_fallback_prompt(prompt_name)


def render_prompt(template: str, variables: Dict[str, str]) -> str:
    """Replace {{variable}} placeholders with values"""
    rendered = template

    for key, value in variables.items():
        placeholder = f"{{{{{key}}}}}"
        rendered = rendered.replace(placeholder, value)

    return rendered


def get_prompt(prompt_name: str, variables: Dict[str, str]) -> str:
    """Load and render a prompt template"""
    template = load_prompt(prompt_name)
    return render_prompt(template, variables)


def get_fallback_prompt(prompt_name: str) -> str:
    """Fallback prompts if markdown files aren't available"""
    fallbacks = {
        "synthesis": """
You are analyzing an RFE (Request for Enhancement) from multiple expert perspectives.

RFE: {{rfe_description}}

Expert Analyses:
{{agent_analyses}}

Synthesize into a JSON response with:
- overallComplexity: LOW/MEDIUM/HIGH/VERY_HIGH
- consensusRecommendations: [array of recommendations]
- criticalRisks: [array of risks]
- requiredCapabilities: [array of technologies/skills]
- estimatedTimeline: "timeline estimate"

Respond only with valid JSON.
""",
        "agent-analysis": """
You are a {{persona}} analyzing this RFE.

RFE: {{rfe_description}}

Relevant Context:
{{context}}

Provide analysis as JSON:
- analysis: "detailed analysis"
- concerns: [array of concerns]  
- recommendations: [array of recommendations]
- estimatedComplexity: LOW/MEDIUM/HIGH/VERY_HIGH
- requiredComponents: [array of components needed]

Respond only with valid JSON.
""",
    }

    return fallbacks.get(prompt_name, "Analyze: {{rfe_description}}")


# Prompt name constants
class PROMPT_NAMES:
    SYNTHESIS = "synthesis"
    COMPONENT_TEAMS = "component-teams"
    ARCHITECTURE_DIAGRAM = "architecture-diagram"
    EPICS_STORIES = "epics-stories"
    IMPLEMENTATION_TIMELINE = "implementation-timeline"
    RFE_DOCUMENT = "rfe-document"
    FEATURE_REFINEMENT = "feature-refinement"
    AGENT_ANALYSIS = "agent-analysis"
