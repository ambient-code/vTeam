"""
Safe structured prediction utilities to prevent validation failures.
Provides defensive programming patterns for LLM response handling.
"""

import logging
from typing import Any, Dict, List, Optional, Type, TypeVar
from pydantic import BaseModel
from llama_index.core.settings import Settings
from llama_index.core.prompts import PromptTemplate

# Use standard logging if structlog not available
logger = logging.getLogger(__name__)

T = TypeVar("T", bound=BaseModel)


class LLMResponseError(Exception):
    """Exception for LLM response processing errors."""

    pass


def create_fallback_response(output_cls: Type[T], persona: str, error_msg: str) -> T:
    """
    Create complete fallback response for any Pydantic model.

    Args:
        output_cls: The Pydantic model class to create
        persona: The agent persona name
        error_msg: Error message to include

    Returns:
        Complete Pydantic model instance with all required fields
    """
    # Define comprehensive defaults for common fields
    field_defaults = {
        "analysis": error_msg,
        "persona": persona,
        "estimatedComplexity": "UNKNOWN",
        "concerns": [f"Processing error: {error_msg}"],
        "recommendations": ["Review agent configuration and retry"],
        "requiredComponents": ["Error recovery needed"],
        # Synthesis model fields
        "overallComplexity": "UNKNOWN",
        "consensusRecommendations": ["System error - manual review required"],
        "criticalRisks": [f"System error: {error_msg}"],
        "requiredCapabilities": ["Error recovery and system validation"],
        "estimatedTimeline": "Unknown due to processing error",
        "synthesis": f"Error during synthesis: {error_msg}",
        # ComponentTeam fields
        "teamName": "Error Recovery Team",
        "team_components": ["System validation required"],
        "responsibilities": ["Resolve processing errors"],
        "epicTitle": f"Fix System Error: {error_msg}",
        "epicDescription": "Address system processing issues",
        # ComponentTeamsList fields
        "teams": [],
        # Architecture fields
        "type": "error",
        "mermaidCode": "graph TD\n    A[Error] --> B[Recovery Needed]",
        "description": f"Architecture generation failed: {error_msg}",
        "components": ["Error recovery system"],
        "integrations": ["Manual intervention required"],
    }

    # Get model fields and create appropriate defaults
    model_fields = output_cls.model_fields
    filtered_defaults = {}

    for field_name, field_info in model_fields.items():
        if field_name in field_defaults:
            filtered_defaults[field_name] = field_defaults[field_name]
        else:
            # Handle unknown fields based on their type annotation
            if hasattr(field_info, "annotation"):
                annotation = field_info.annotation
                if annotation == str:
                    filtered_defaults[field_name] = f"Error: {error_msg}"
                elif annotation == List[str] or str(annotation).startswith(
                    "typing.List"
                ):
                    filtered_defaults[field_name] = [f"Error: {error_msg}"]
                elif annotation == int:
                    filtered_defaults[field_name] = 0
                else:
                    filtered_defaults[field_name] = f"Error: {error_msg}"

    try:
        return output_cls(**filtered_defaults)
    except Exception as e:
        # Last resort - create minimal object
        minimal_defaults = {field_name: "Error" for field_name in model_fields.keys()}
        logger.error(
            "Failed to create fallback response",
            model=output_cls.__name__,
            error=str(e),
            fields=list(model_fields.keys()),
        )
        return output_cls(**minimal_defaults)


def validate_structured_response(response: Any, expected_type: Type[T]) -> bool:
    """
    Validate that response matches expected Pydantic model structure.

    Args:
        response: The response to validate
        expected_type: Expected Pydantic model type

    Returns:
        True if response is valid, False otherwise
    """
    if not hasattr(response, "model_dump"):
        return False

    try:
        dumped = response.model_dump()
        required_fields = expected_type.model_fields.keys()
        return all(field in dumped for field in required_fields)
    except Exception:
        return False


async def safe_structured_predict(
    output_cls: Type[T], prompt_template: PromptTemplate, persona: str, **prompt_args
) -> T:
    """
    Defensive structured prediction with complete fallback handling.

    Args:
        output_cls: Pydantic model class for output
        prompt_template: LLM prompt template
        persona: Agent persona name
        **prompt_args: Additional prompt arguments

    Returns:
        Validated Pydantic model instance
    """
    try:
        # Attempt structured prediction
        response = await Settings.llm.astructured_predict(
            output_cls, prompt_template, **prompt_args
        )

        # Validate response has required structure
        if validate_structured_response(response, output_cls):
            logger.info(
                "Successful structured prediction",
                persona=persona,
                model=output_cls.__name__,
            )
            return response
        else:
            # Response structure is invalid
            error_msg = "LLM returned invalid response structure"
            logger.warning(
                "Invalid LLM response structure",
                persona=persona,
                model=output_cls.__name__,
                response_type=type(response).__name__,
            )
            return create_fallback_response(output_cls, persona, error_msg)

    except Exception as e:
        # Prediction failed entirely
        error_msg = f"Prediction error: {str(e)}"
        logger.error(
            "Structured prediction failed",
            persona=persona,
            model=output_cls.__name__,
            error=str(e),
        )
        return create_fallback_response(output_cls, persona, error_msg)


def safe_model_dump(
    response: Any, fallback_data: Optional[Dict] = None
) -> Dict[str, Any]:
    """
    Safely extract data from Pydantic model with fallback.

    Args:
        response: Pydantic model instance or other response
        fallback_data: Fallback data if model_dump fails

    Returns:
        Dictionary representation of the model
    """
    if fallback_data is None:
        fallback_data = {
            "error": "Failed to extract model data",
            "analysis": "Model serialization failed",
            "persona": "SYSTEM_ERROR",
        }

    try:
        if hasattr(response, "model_dump"):
            return response.model_dump()
        else:
            logger.warning(
                "Response lacks model_dump method",
                response_type=type(response).__name__,
            )
            return fallback_data
    except Exception as e:
        logger.error("Model dump failed", error=str(e))
        return fallback_data


class MCPIntegrationValidator:
    """Validator for MCP integration responses."""

    @staticmethod
    def validate_mcp_response(response: Dict[str, Any]) -> bool:
        """Validate MCP response structure."""
        required_fields = ["status", "data"]
        return all(field in response for field in required_fields)

    @staticmethod
    def create_mcp_fallback(error_msg: str) -> Dict[str, Any]:
        """Create fallback MCP response."""
        return {
            "status": "error",
            "data": [],
            "error": error_msg,
            "timestamp": "unknown",
        }
