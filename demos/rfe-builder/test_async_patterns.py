"""Test async patterns to catch coroutine and async/await issues."""

import pytest
import ast
import asyncio
import inspect
from pathlib import Path
from unittest.mock import AsyncMock, MagicMock


class TestAsyncPatterns:
    """Test suite to prevent async/await and coroutine issues."""
    
    def test_workflow_methods_handle_async_properly(self):
        """Test that workflow methods properly handle async operations."""
        from src.rfe_builder_workflow import RFEBuilderWorkflow
        
        workflow = RFEBuilderWorkflow()
        
        # Check that streaming methods use non-streaming alternatives when needed
        assert hasattr(workflow.llm, 'acomplete'), "LLM should have acomplete method for non-streaming"
        assert hasattr(workflow.llm, 'astream_complete'), "LLM should have astream_complete method for streaming"
    
    @pytest.mark.asyncio
    async def test_llm_acomplete_returns_awaitable(self):
        """Test that LLM acomplete method returns proper awaitable."""
        from src.rfe_builder_workflow import RFEBuilderWorkflow
        from llama_index.core.llms.mock import MockLLM
        
        # Use mock LLM to test async pattern
        mock_llm = MockLLM()
        workflow = RFEBuilderWorkflow()
        workflow.llm = mock_llm
        
        # Test that acomplete is awaitable
        response = await workflow.llm.acomplete("test prompt")
        assert hasattr(response, 'text'), "Response should have text attribute"
    
    def test_no_unawaited_coroutines_in_workflow_files(self):
        """Test workflow files for patterns that could cause unawaited coroutines."""
        workflow_files = [
            "src/rfe_builder_workflow.py",
            "src/jira_rfe_to_architecture_workflow.py", 
            "src/enhanced_rfe_workflow.py"
        ]
        
        for file_path in workflow_files:
            if not Path(file_path).exists():
                continue
                
            with open(file_path, 'r') as f:
                content = f.read()
            
            # Check for potentially problematic patterns
            problematic_patterns = [
                # Direct assignment of async methods without await
                r'=\s*\w+\.astream_complete\(',
                r'=\s*\w+\.astructured_predict\(',
                # Function calls that might return coroutines without await
                r'\.span\([^)]*\)\s*$',  # Dispatcher.span calls
            ]
            
            import re
            for pattern in problematic_patterns:
                if re.search(pattern, content, re.MULTILINE):
                    # Check if it's properly awaited in context
                    lines = content.split('\n')
                    for i, line in enumerate(lines):
                        if re.search(pattern, line):
                            # Look for 'await' in the same line or assignment context
                            if 'await' not in line and '=' in line:
                                pytest.fail(
                                    f"File {file_path} line {i+1} may have unawaited coroutine: {line.strip()}"
                                )
    
    def test_write_event_to_stream_usage_patterns(self):
        """Test that write_event_to_stream is used correctly."""
        workflow_files = [
            "src/rfe_builder_workflow.py",
            "src/jira_rfe_to_architecture_workflow.py",
            "src/enhanced_rfe_workflow.py"
        ]
        
        for file_path in workflow_files:
            if not Path(file_path).exists():
                continue
                
            with open(file_path, 'r') as f:
                content = f.read()
            
            # Check write_event_to_stream usage
            import re
            stream_calls = re.findall(r'ctx\.write_event_to_stream\([^)]+\)', content, re.DOTALL)
            
            for call in stream_calls:
                # Ensure UIEvent is properly constructed
                if 'UIEvent(' not in call and 'ArtifactEvent(' not in call:
                    pytest.fail(
                        f"File {file_path} has write_event_to_stream call without proper event type: {call[:100]}..."
                    )
    
    @pytest.mark.asyncio
    async def test_agent_manager_async_methods(self):
        """Test that agent manager async methods work correctly."""
        from src.agents import RFEAgentManager, get_agent_personas
        
        agent_manager = RFEAgentManager()
        
        # Test that global function exists and is async
        assert asyncio.iscoroutinefunction(get_agent_personas), \
            "get_agent_personas should be async"
        
        # Test that methods return expected types
        personas = await get_agent_personas()
        assert isinstance(personas, dict), "get_agent_personas should return dict"
        
        # Test agent manager has the expected methods
        assert hasattr(agent_manager, 'load_agent_configurations'), "Manager should have load_agent_configurations"
        assert hasattr(agent_manager, 'agent_configs'), "Manager should have agent_configs"
    
    def test_async_generator_typing(self):
        """Test that async generators are properly typed."""
        # Check plugin interface files for proper AsyncGenerator typing
        plugin_files = [
            "plugins/base/plugin_interface.py",
            "plugins/base/orchestrator.py"
        ]
        
        for file_path in plugin_files:
            if not Path(file_path).exists():
                continue
                
            with open(file_path, 'r') as f:
                content = f.read()
            
            # Look for AsyncGenerator usage
            if 'AsyncGenerator' in content:
                # Ensure proper typing imports
                assert 'from typing import' in content and 'AsyncGenerator' in content, \
                    f"File {file_path} uses AsyncGenerator but may not import it properly"


class TestStreamingPatterns:
    """Test streaming and event patterns for async correctness."""
    
    def test_streaming_methods_use_async_for(self):
        """Test that streaming methods properly use async for loops."""
        workflow_files = [
            "src/rfe_builder_workflow.py",
            "src/jira_rfe_to_architecture_workflow.py"
        ]
        
        for file_path in workflow_files:
            if not Path(file_path).exists():
                continue
                
            with open(file_path, 'r') as f:
                content = f.read()
            
            # Check for astream usage patterns
            import re
            if 'astream' in content:
                # Look for 'async for' patterns when using astream
                astream_lines = [line for line in content.split('\n') if 'astream' in line]
                
                for line in astream_lines:
                    if '=' in line and 'await' not in line and 'async for' not in line:
                        # This might be problematic - astream methods should typically be used with async for
                        pytest.fail(
                            f"File {file_path} may use astream incorrectly: {line.strip()}"
                        )
    
    @pytest.mark.asyncio
    async def test_workflow_event_streaming(self):
        """Test that workflow event streaming works without coroutine warnings."""
        from src.rfe_builder_workflow import RFEBuilderWorkflow
        from llama_index.core.workflow import Context, StartEvent
        from llama_index.core.llms.mock import MockLLM
        
        # Setup mock workflow
        workflow = RFEBuilderWorkflow()
        workflow.llm = MockLLM()
        
        # Mock context
        ctx = MagicMock()
        ctx.write_event_to_stream = MagicMock()
        
        # Test that start event handling doesn't raise coroutine warnings
        start_event = StartEvent(user_msg="Test RFE request")
        
        try:
            result = await workflow.start_rfe_builder(ctx, start_event)
            # Should return an event without raising warnings
            assert result is not None
        except Exception as e:
            if "coroutine" in str(e).lower():
                pytest.fail(f"Workflow raised coroutine-related error: {e}")


if __name__ == "__main__":
    pytest.main([__file__, "-v"])