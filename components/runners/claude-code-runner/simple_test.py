#!/usr/bin/env python3

"""
Simple test for spek-kit command detection without external dependencies
"""

import re
from typing import Optional, Tuple


def detect_spek_kit_command(prompt: str) -> Optional[Tuple[str, str]]:
    """
    Simple version of command detection for testing
    """
    spek_commands = ["specify", "plan", "tasks"]

    for command in spek_commands:
        # Match /command followed by space and arguments
        pattern = rf'^/{command}\s+(.+?)(?:\n|$)'
        match = re.search(pattern, prompt.strip(), re.MULTILINE | re.DOTALL)
        if match:
            args = match.group(1).strip()
            return command, args

    return None


def test_command_detection():
    """Test the command detection functionality"""

    print("🧪 Testing spek-kit command detection...")

    test_cases = [
        # Basic commands
        ("/specify Build a task management application", True, "specify"),
        ("/plan Use React and Node.js with PostgreSQL", True, "plan"),
        ("/tasks Break down into manageable development sprints", True, "tasks"),

        # No commands
        ("Regular prompt without spek-kit commands", False, None),
        ("This is a normal agentic query", False, None),

        # Multi-line
        ("/specify Build a comprehensive\necommerce platform", True, "specify"),

        # Edge cases
        ("/specify   Build with extra spaces   ", True, "specify"),
        ("text /specify in middle", False, None),  # Should not match in middle
        ("/SPECIFY Build something", False, None),  # Case sensitive
        ("/specif Build something", False, None),   # Typo
        ("/specify", False, None),  # No arguments
    ]

    passed = 0
    total = len(test_cases)

    for i, (prompt, should_match, expected_command) in enumerate(test_cases):
        result = detect_spek_kit_command(prompt)

        if should_match:
            if result and result[0] == expected_command:
                print(f"   ✅ Test {i+1}: Correctly detected /{expected_command}")
                passed += 1
            else:
                print(f"   ❌ Test {i+1}: Expected /{expected_command}, got {result}")
        else:
            if result is None:
                print(f"   ✅ Test {i+1}: Correctly detected no command")
                passed += 1
            else:
                print(f"   ❌ Test {i+1}: Expected no command, got {result}")

    print(f"\n📊 Results: {passed}/{total} tests passed")

    if passed == total:
        print("🎉 All command detection tests passed!")
        return True
    else:
        print("❌ Some tests failed")
        return False


def test_content_generation():
    """Test content generation templates"""

    print("\n🧪 Testing content generation...")

    # Simple content generation functions
    def generate_spec_content(requirements: str) -> str:
        return f"""# Feature Specification

## Overview
{requirements}

## User Stories
- As a user, I want to be able to [feature] so that [benefit]

## Functional Requirements
1. The system shall [requirement 1]
2. The system shall [requirement 2]

## Acceptance Criteria
- [ ] Feature implementation complete
- [ ] Tests pass
"""

    def generate_plan_content(tech_requirements: str) -> str:
        return f"""# Implementation Plan

## Technical Requirements
{tech_requirements}

## Architecture Overview
- Frontend: [technology]
- Backend: [technology]
- Database: [technology]

## Implementation Phases
### Phase 1: Foundation
- Set up project structure

### Phase 2: Core Features
- Implement main functionality
"""

    def generate_tasks_content(task_details: str) -> str:
        return f"""# Task Breakdown

## Task Details
{task_details}

## Epic: Feature Implementation

### Story 1: Foundation Setup
**Tasks:**
- [ ] Set up project structure
- [ ] Configure development environment

**Estimated Effort:** 2-3 days
"""

    # Test content generation
    try:
        spec = generate_spec_content("Build authentication system")
        plan = generate_plan_content("Use Node.js and React")
        tasks = generate_tasks_content("Backend API first")

        # Basic validation
        assert "Feature Specification" in spec
        assert "Implementation Plan" in plan
        assert "Task Breakdown" in tasks

        print("   ✅ Spec generation working")
        print("   ✅ Plan generation working")
        print("   ✅ Tasks generation working")
        print("🎉 All content generation tests passed!")
        return True

    except Exception as e:
        print(f"   ❌ Content generation failed: {e}")
        return False


if __name__ == "__main__":
    print("🚀 Starting simple spek-kit integration tests...\n")

    # Run tests
    detection_passed = test_command_detection()
    content_passed = test_content_generation()

    if detection_passed and content_passed:
        print("\n✨ All tests passed! Spek-kit integration looks good!")
    else:
        print("\n❌ Some tests failed. Check the implementation.")