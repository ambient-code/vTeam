#!/usr/bin/env python3
"""
Proper pytest tests for validation fixes.
Tests that the RFEAnalysis validation fixes work correctly with pytest framework.
"""

import pytest
from src.agents import RFEAnalysis
from src.safe_prediction import (
    safe_structured_predict,
    create_fallback_response,
    validate_structured_response,
)


class TestValidationFixes:
    """Test suite for validation error fixes."""

    def test_create_fallback_response_rfe_analysis(self):
        """Test that fallback responses have all required RFEAnalysis fields."""
        fallback = create_fallback_response(
            RFEAnalysis, "PRODUCT_MANAGER", "Test error"
        )

        # Verify it's a proper RFEAnalysis instance
        assert isinstance(fallback, RFEAnalysis)

        # Verify all required fields are present
        assert fallback.persona == "PRODUCT_MANAGER"
        assert fallback.estimatedComplexity == "UNKNOWN"
        assert isinstance(fallback.concerns, list)
        assert len(fallback.concerns) > 0
        assert isinstance(fallback.recommendations, list)
        assert len(fallback.recommendations) > 0
        assert isinstance(fallback.requiredComponents, list)
        assert len(fallback.requiredComponents) > 0

        # Verify model_dump works without ValidationError
        data = fallback.model_dump()
        assert isinstance(data, dict)

        # Check all required fields from original error are present
        required_fields = [
            "analysis",
            "persona",
            "estimatedComplexity",
            "concerns",
            "recommendations",
            "requiredComponents",
        ]
        for field in required_fields:
            assert field in data, f"Missing field: {field}"

    def test_validate_structured_response_valid(self):
        """Test validation of valid structured responses."""
        valid_response = RFEAnalysis(
            analysis="Valid analysis",
            persona="TEST_AGENT",
            estimatedComplexity="MEDIUM",
            concerns=["Valid concern"],
            recommendations=["Valid recommendation"],
            requiredComponents=["Valid component"],
        )

        assert validate_structured_response(valid_response, RFEAnalysis) is True

    def test_validate_structured_response_invalid(self):
        """Test validation of invalid responses."""
        # Test string response (the original problematic case)
        invalid_response = "This is just a string"
        assert validate_structured_response(invalid_response, RFEAnalysis) is False

        # Test None response
        assert validate_structured_response(None, RFEAnalysis) is False

        # Test object without model_dump
        class MockObject:
            pass

        mock_obj = MockObject()
        assert validate_structured_response(mock_obj, RFEAnalysis) is False

    def test_original_validation_error_scenario(self):
        """Test the exact scenario that caused the original ValidationError."""
        # This simulates what happened: LLM returned unexpected format
        # and error handling tried to create incomplete object

        # Original problematic response would have been like this:
        incomplete_data = {"analysis": "Some analysis", "persona": "PRODUCT_MANAGER"}

        # The original code would try: RFEAnalysis(**incomplete_data)
        # This would fail with: 4 validation errors for RFEAnalysis

        # Our fix: use create_fallback_response instead
        fallback = create_fallback_response(
            RFEAnalysis, "PRODUCT_MANAGER", "LLM returned incomplete data"
        )

        # Verify it works and has all fields
        assert isinstance(fallback, RFEAnalysis)
        data = fallback.model_dump()

        # These are the fields that were missing in the original error:
        missing_fields = ["estimatedComplexity", "concerns", "recommendations", "requiredComponents"]
        for field in missing_fields:
            assert field in data, f"Fixed: {field} now present"
            assert data[field] is not None, f"Fixed: {field} has value"

    def test_fallback_maintains_persona(self):
        """Test that fallback responses preserve the agent persona."""
        test_personas = ["PRODUCT_MANAGER", "ARCHITECT", "ENGINEER", "DESIGNER"]

        for persona in test_personas:
            fallback = create_fallback_response(RFEAnalysis, persona, "Test error")
            assert fallback.persona == persona
            # Persona is properly stored in the persona field, not necessarily in analysis

    def test_model_dump_never_fails(self):
        """Test that model_dump never fails after our fixes."""
        # Test multiple error scenarios
        error_scenarios = [
            "LLM returned string instead of object",
            "API connection failed",
            "Malformed JSON response",
            "Missing required fields",
            "Timeout during prediction",
        ]

        for error_msg in error_scenarios:
            fallback = create_fallback_response(RFEAnalysis, "TEST_AGENT", error_msg)

            # This should never raise ValidationError anymore
            try:
                data = fallback.model_dump()
                assert isinstance(data, dict)
                assert len(data) == 6  # RFEAnalysis has 6 required fields
            except Exception as e:
                pytest.fail(f"model_dump() failed for scenario '{error_msg}': {e}")

    def test_error_messages_are_informative(self):
        """Test that error fallbacks provide useful information."""
        error_msg = "Test specific error condition"
        fallback = create_fallback_response(RFEAnalysis, "TEST_AGENT", error_msg)

        # Error should be mentioned in analysis
        assert error_msg in fallback.analysis

        # Concerns should mention the error
        assert any(error_msg in concern for concern in fallback.concerns)

        # Should have actionable recommendations
        assert any("configuration" in rec.lower() for rec in fallback.recommendations)


class TestPydanticModelCompatibility:
    """Test compatibility with all Pydantic models in the system."""

    def test_all_models_can_create_fallbacks(self):
        """Test that all Pydantic models can create fallback responses."""
        from src.agents import Synthesis, ComponentTeamsList, Architecture

        models_to_test = [
            ("RFEAnalysis", RFEAnalysis),
            ("Synthesis", Synthesis),
            ("ComponentTeamsList", ComponentTeamsList),
            ("Architecture", Architecture),
        ]

        for model_name, model_cls in models_to_test:
            fallback = create_fallback_response(model_cls, "TEST_AGENT", f"Test {model_name}")

            # Should be instance of the correct type
            assert isinstance(fallback, model_cls), f"{model_name} fallback wrong type"

            # Should be able to serialize without errors
            try:
                data = fallback.model_dump()
                assert isinstance(data, dict), f"{model_name} model_dump failed"
                assert len(data) > 0, f"{model_name} model_dump empty"
            except Exception as e:
                pytest.fail(f"{model_name} model_dump failed: {e}")


if __name__ == "__main__":
    # Run tests if called directly
    pytest.main([__file__, "-v"])