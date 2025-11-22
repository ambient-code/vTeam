"""Unit tests for observability module."""

import pytest
import os
import logging
from unittest.mock import Mock, patch
from observability import ObservabilityManager


@pytest.fixture
def mock_langfuse_client():
    """Create mock Langfuse client for SDK v3."""
    mock_client = Mock()
    mock_trace = Mock()
    # SDK v3 uses client.trace() to create trace object
    mock_client.trace.return_value = mock_trace
    mock_client.flush = Mock()
    # trace.generation() and trace.span() return observation objects
    mock_trace.generation.return_value = Mock()
    mock_trace.span.return_value = Mock()
    return mock_client, mock_trace


@pytest.fixture
def manager():
    """Create ObservabilityManager instance."""
    return ObservabilityManager(
        session_id="test-session-123", user_id="user-456", user_name="Test User"
    )


class TestObservabilityManagerInit:
    """Tests for ObservabilityManager initialization."""

    def test_init_sets_properties(self):
        """Test that __init__ sets all properties correctly."""
        manager = ObservabilityManager(
            session_id="session-1", user_id="user-1", user_name="John Doe"
        )

        assert manager.session_id == "session-1"
        assert manager.user_id == "user-1"
        assert manager.user_name == "John Doe"
        assert manager.langfuse_client is None
        assert manager.langfuse_trace is None
        assert manager._langfuse_tool_spans == {}


class TestLangfuseInitialization:
    """Tests for Langfuse initialization."""

    @pytest.mark.asyncio
    @patch("observability.LANGFUSE_AVAILABLE", False)
    async def test_init_langfuse_unavailable(self, manager):
        """Test initialization when Langfuse SDK is not available."""
        result = await manager.initialize("test prompt", "test-namespace")

        assert result is False
        assert manager.langfuse_client is None

    @pytest.mark.asyncio
    @patch("observability.LANGFUSE_AVAILABLE", True)
    async def test_init_langfuse_disabled(self, manager):
        """Test initialization when LANGFUSE_ENABLED is false."""
        with patch.dict(os.environ, {"LANGFUSE_ENABLED": "false"}):
            result = await manager.initialize("test prompt", "test-namespace")

        assert result is False
        assert manager.langfuse_client is None

    @pytest.mark.asyncio
    @patch("observability.LANGFUSE_AVAILABLE", True)
    async def test_init_missing_public_key(self, manager, caplog):
        """Test initialization with missing LANGFUSE_PUBLIC_KEY."""
        env_vars = {
            "LANGFUSE_ENABLED": "true",
            "LANGFUSE_PUBLIC_KEY": "",
            "LANGFUSE_SECRET_KEY": "sk-lf-secret",
            "LANGFUSE_HOST": "http://localhost:3000",
        }

        with patch.dict(os.environ, env_vars, clear=True):
            with caplog.at_level(logging.WARNING):
                result = await manager.initialize("test prompt", "test-namespace")

        assert result is False
        assert manager.langfuse_client is None
        assert "LANGFUSE_PUBLIC_KEY or LANGFUSE_SECRET_KEY is missing" in caplog.text
        assert "ambient-admin-langfuse-secret" in caplog.text

    @pytest.mark.asyncio
    @patch("observability.LANGFUSE_AVAILABLE", True)
    async def test_init_missing_secret_key(self, manager, caplog):
        """Test initialization with missing LANGFUSE_SECRET_KEY."""
        env_vars = {
            "LANGFUSE_ENABLED": "true",
            "LANGFUSE_PUBLIC_KEY": "pk-lf-public",
            "LANGFUSE_SECRET_KEY": "",
            "LANGFUSE_HOST": "http://localhost:3000",
        }

        with patch.dict(os.environ, env_vars, clear=True):
            with caplog.at_level(logging.WARNING):
                result = await manager.initialize("test prompt", "test-namespace")

        assert result is False
        assert "LANGFUSE_PUBLIC_KEY or LANGFUSE_SECRET_KEY is missing" in caplog.text

    @pytest.mark.asyncio
    @patch("observability.LANGFUSE_AVAILABLE", True)
    async def test_init_missing_host(self, manager, caplog):
        """Test initialization with missing LANGFUSE_HOST."""
        env_vars = {
            "LANGFUSE_ENABLED": "true",
            "LANGFUSE_PUBLIC_KEY": "pk-lf-public",
            "LANGFUSE_SECRET_KEY": "sk-lf-secret",
            "LANGFUSE_HOST": "",
        }

        with patch.dict(os.environ, env_vars, clear=True):
            with caplog.at_level(logging.WARNING):
                result = await manager.initialize("test prompt", "test-namespace")

        assert result is False
        assert "LANGFUSE_HOST is missing" in caplog.text

    @pytest.mark.asyncio
    @patch("langfuse.Langfuse")
    async def test_init_successful(self, mock_langfuse_class, manager, caplog):
        """Test successful Langfuse initialization with SDK v3 trace() pattern."""
        mock_client = Mock()
        mock_trace = Mock()

        # Mock client.trace() method (SDK v3 correct pattern)
        mock_client.trace.return_value = mock_trace
        mock_langfuse_class.return_value = mock_client

        env_vars = {
            "LANGFUSE_ENABLED": "true",
            "LANGFUSE_PUBLIC_KEY": "pk-lf-public",
            "LANGFUSE_SECRET_KEY": "sk-lf-secret",
            "LANGFUSE_HOST": "http://localhost:3000",
        }

        with patch.dict(os.environ, env_vars, clear=True):
            with caplog.at_level(logging.INFO):
                result = await manager.initialize("test prompt", "test-namespace")

        assert result is True
        assert manager.langfuse_client is not None
        assert manager.langfuse_trace is not None

        mock_langfuse_class.assert_called_once_with(
            public_key="pk-lf-public",
            secret_key="sk-lf-secret",
            host="http://localhost:3000",
        )

        # Verify client.trace() was called with correct params
        mock_client.trace.assert_called_once()
        call_kwargs = mock_client.trace.call_args[1]
        assert call_kwargs["name"] == "claude_agent_session"
        assert call_kwargs["user_id"] == manager.user_id
        assert call_kwargs["session_id"] == manager.session_id
        assert "claude-code" in call_kwargs["tags"]
        assert call_kwargs["input"] == {"prompt": "test prompt"}

        assert "Langfuse tracing enabled" in caplog.text

    @pytest.mark.asyncio
    @patch("langfuse.Langfuse")
    async def test_init_with_user_tracking(self, mock_langfuse_class, caplog):
        """Test Langfuse initialization with user tracking."""
        mock_client = Mock()
        mock_trace = Mock()

        # Mock client.trace() method
        mock_client.trace.return_value = mock_trace
        mock_langfuse_class.return_value = mock_client

        manager = ObservabilityManager(
            session_id="session-1", user_id="user-123", user_name="Jane Doe"
        )

        env_vars = {
            "LANGFUSE_ENABLED": "true",
            "LANGFUSE_PUBLIC_KEY": "pk-lf-public",
            "LANGFUSE_SECRET_KEY": "sk-lf-secret",
            "LANGFUSE_HOST": "http://localhost:3000",
        }

        with patch.dict(os.environ, env_vars, clear=True):
            with caplog.at_level(logging.INFO):
                result = await manager.initialize("test prompt", "test-namespace")

        assert result is True
        assert "Tracking session for user Jane Doe (user-123)" in caplog.text

    @pytest.mark.asyncio
    @patch("langfuse.Langfuse")
    async def test_init_langfuse_exception(self, mock_langfuse_class, manager, caplog):
        """Test Langfuse initialization when SDK raises exception."""
        mock_langfuse_class.side_effect = Exception("Connection failed")

        env_vars = {
            "LANGFUSE_ENABLED": "true",
            "LANGFUSE_PUBLIC_KEY": "pk-lf-public",
            "LANGFUSE_SECRET_KEY": "sk-lf-secret",
            "LANGFUSE_HOST": "http://localhost:3000",
        }

        with patch.dict(os.environ, env_vars, clear=True):
            with caplog.at_level(logging.WARNING):
                result = await manager.initialize("test prompt", "test-namespace")

        assert result is False
        assert manager.langfuse_client is None
        assert manager.langfuse_trace is None
        assert "Langfuse initialization failed" in caplog.text
        assert "Observability will be disabled" in caplog.text

    @pytest.mark.asyncio
    @patch("langfuse.Langfuse")
    async def test_init_sanitizes_api_keys_in_error(
        self, mock_langfuse_class, manager, caplog
    ):
        """Test that API keys are sanitized in error messages."""
        mock_langfuse_class.side_effect = Exception("Auth failed with key pk-lf-public")

        env_vars = {
            "LANGFUSE_ENABLED": "true",
            "LANGFUSE_PUBLIC_KEY": "pk-lf-public",
            "LANGFUSE_SECRET_KEY": "sk-lf-secret",
            "LANGFUSE_HOST": "http://localhost:3000",
        }

        with patch.dict(os.environ, env_vars, clear=True):
            with caplog.at_level(logging.WARNING):
                result = await manager.initialize("test prompt", "test-namespace")

        assert result is False
        # API key should be redacted
        assert "pk-lf-public" not in caplog.text
        assert "[REDACTED_PUBLIC_KEY]" in caplog.text


class TestTrackGeneration:
    """Tests for track_generation method."""

    def test_track_generation_no_client(self, manager):
        """Test track_generation when Langfuse client is not initialized."""
        # Should not raise exception
        manager.track_generation(Mock(), "claude-3-5-sonnet", 1)

    def test_track_generation_graceful_failure(self, manager):
        """Test track_generation handles exceptions gracefully."""
        manager.langfuse_client = Mock()
        manager.langfuse_trace = Mock()
        manager.langfuse_client.start_generation.side_effect = Exception("Test error")

        # Create message that will trigger the code path
        message = Mock()
        message.content = []  # Empty content should return early

        # Should not raise exception even when start_generation fails
        manager.track_generation(message, "claude-3-5-sonnet", 1)


class TestTrackToolUse:
    """Tests for track_tool_use method."""

    def test_track_tool_use_no_client(self, manager):
        """Test track_tool_use when Langfuse client is not initialized."""
        # Should not raise exception
        manager.track_tool_use("Read", "tool-123", {"file": "test.txt"})

    def test_track_tool_use_creates_span(self, manager):
        """Test track_tool_use creates tool span with SDK v3 trace.span() pattern."""
        mock_client = Mock()
        mock_trace = Mock()
        mock_tool_span = Mock()

        # Mock trace.span() method (SDK v3 correct pattern)
        mock_trace.span.return_value = mock_tool_span

        manager.langfuse_client = mock_client
        manager.langfuse_trace = mock_trace

        tool_input = {"file_path": "/test/file.txt"}
        manager.track_tool_use("Read", "tool-456", tool_input)

        # Verify trace.span() was called with correct params
        mock_trace.span.assert_called_once_with(
            name="tool_Read",
            input=tool_input,
            metadata={
                "tool_id": "tool-456",
                "tool_name": "Read",
            },
        )

        assert "tool-456" in manager._langfuse_tool_spans
        assert manager._langfuse_tool_spans["tool-456"] == mock_tool_span


class TestTrackToolResult:
    """Tests for track_tool_result method."""

    def test_track_tool_result_no_span(self, manager):
        """Test track_tool_result when tool span doesn't exist."""
        # Should not raise exception
        manager.track_tool_result("tool-999", "result", False)

    @patch("observability.Langfuse")
    def test_track_tool_result_success(self, mock_langfuse_class):
        """Test track_tool_result for successful tool execution."""
        mock_client = Mock()
        mock_tool_span = Mock()

        manager = ObservabilityManager("session-1", "user-1", "User")
        manager.langfuse_client = mock_client
        manager._langfuse_tool_spans["tool-123"] = mock_tool_span

        manager.track_tool_result("tool-123", "File contents", is_error=False)

        # SDK v3: Uses update() then end()
        mock_tool_span.update.assert_called_once()
        call_kwargs = mock_tool_span.update.call_args[1]
        assert call_kwargs["level"] == "DEFAULT"
        assert "File contents" in call_kwargs["output"]["result"]
        assert call_kwargs["metadata"]["is_error"] is False

        # Verify span.end() was called
        mock_tool_span.end.assert_called_once()
        assert "tool-123" not in manager._langfuse_tool_spans

    @patch("observability.Langfuse")
    def test_track_tool_result_error(self, mock_langfuse_class):
        """Test track_tool_result for failed tool execution."""
        mock_client = Mock()
        mock_tool_span = Mock()

        manager = ObservabilityManager("session-1", "user-1", "User")
        manager.langfuse_client = mock_client
        manager._langfuse_tool_spans["tool-123"] = mock_tool_span

        manager.track_tool_result("tool-123", "Error: File not found", is_error=True)

        # SDK v3: Uses update() then end()
        mock_tool_span.update.assert_called_once()
        call_kwargs = mock_tool_span.update.call_args[1]
        assert call_kwargs["level"] == "ERROR"
        assert call_kwargs["metadata"]["is_error"] is True

        # Verify span.end() was called
        mock_tool_span.end.assert_called_once()


class TestFinalize:
    """Tests for finalize method."""

    @pytest.mark.asyncio
    async def test_finalize_no_client(self, manager):
        """Test finalize when Langfuse client is not initialized."""
        # Should not raise exception
        await manager.finalize(None)

    @pytest.mark.asyncio
    @patch("observability.with_sync_timeout")
    async def test_finalize_with_result_payload(self, mock_timeout):
        """Test finalize with result payload (SDK v3 trace pattern)."""
        mock_timeout.return_value = (True, None)

        mock_client = Mock()
        mock_trace = Mock()

        manager = ObservabilityManager("session-1", "user-1", "User")
        manager.langfuse_client = mock_client
        manager.langfuse_trace = mock_trace

        result_payload = {
            "num_turns": 5,
            "total_cost_usd": 0.01234,
            "duration_ms": 15000,
            "subtype": "completion",
        }

        await manager.finalize(result_payload)

        # SDK v3: Verify trace.update() was called with result payload
        mock_trace.update.assert_called_once()
        call_kwargs = mock_trace.update.call_args[1]
        assert call_kwargs["output"] == result_payload
        assert "num_turns" in call_kwargs["metadata"]
        assert "total_cost_usd" in call_kwargs["metadata"]

        # Verify flush was called
        mock_timeout.assert_called_once()

    @pytest.mark.asyncio
    @patch("observability.with_sync_timeout")
    async def test_finalize_without_result_payload(self, mock_timeout, caplog):
        """Test finalize without result payload (e.g., git push) - SDK v3."""
        mock_timeout.return_value = (True, None)

        mock_client = Mock()
        mock_trace = Mock()

        manager = ObservabilityManager("session-1", "user-1", "User")
        manager.langfuse_client = mock_client
        manager.langfuse_trace = mock_trace

        with caplog.at_level(logging.INFO):
            await manager.finalize(None)

        # Verify flush was still called (no update when no result_payload)
        mock_timeout.assert_called_once()

    @pytest.mark.asyncio
    @patch("observability.with_sync_timeout")
    async def test_finalize_flush_timeout(self, mock_timeout, caplog):
        """Test finalize when flush times out."""
        mock_timeout.return_value = (False, None)  # Timeout

        mock_client = Mock()
        mock_trace = Mock()

        manager = ObservabilityManager("session-1", "user-1", "User")
        manager.langfuse_client = mock_client
        manager.langfuse_trace = mock_trace

        with caplog.at_level(logging.ERROR):
            await manager.finalize({"num_turns": 1})

        assert "flush timed out" in caplog.text


class TestCleanupOnError:
    """Tests for cleanup_on_error method."""

    @pytest.mark.asyncio
    async def test_cleanup_no_client(self, manager):
        """Test cleanup_on_error when Langfuse client is not initialized."""
        # Should not raise exception
        await manager.cleanup_on_error(ValueError("test error"))

    @pytest.mark.asyncio
    @patch("observability.with_sync_timeout")
    async def test_cleanup_on_error(self, mock_timeout):
        """Test cleanup_on_error marks trace as error (SDK v3)."""
        mock_timeout.return_value = (True, None)

        mock_client = Mock()
        mock_trace = Mock()

        manager = ObservabilityManager("session-1", "user-1", "User")
        manager.langfuse_client = mock_client
        manager.langfuse_trace = mock_trace

        error = ValueError("Session failed")
        await manager.cleanup_on_error(error)

        # SDK v3: Verify trace.update() was called with error info
        mock_trace.update.assert_called_once()
        call_kwargs = mock_trace.update.call_args[1]
        assert call_kwargs["level"] == "ERROR"
        assert call_kwargs["status_message"] == "Session failed"

        # Verify flush was called
        mock_timeout.assert_called_once()

    @pytest.mark.asyncio
    @patch("observability.with_sync_timeout")
    async def test_cleanup_flush_timeout(self, mock_timeout, caplog):
        """Test cleanup when flush times out."""
        mock_timeout.return_value = (False, None)  # Timeout

        mock_client = Mock()
        mock_trace = Mock()

        manager = ObservabilityManager("session-1", "user-1", "User")
        manager.langfuse_client = mock_client
        manager.langfuse_trace = mock_trace

        with caplog.at_level(logging.ERROR):
            await manager.cleanup_on_error(ValueError("test"))

        assert "error cleanup flush timed out" in caplog.text


class TestEnvironmentVariableCombinations:
    """Tests for various environment variable combinations."""

    @pytest.mark.asyncio
    @patch("observability.LANGFUSE_AVAILABLE", True)
    @pytest.mark.parametrize(
        "enabled_value", ["1", "true", "True", "TRUE", "yes", "YES"]
    )
    async def test_langfuse_enabled_variations(self, enabled_value, manager):
        """Test that various truthy values for LANGFUSE_ENABLED work."""
        env_vars = {
            "LANGFUSE_ENABLED": enabled_value,
            "LANGFUSE_PUBLIC_KEY": "",  # Missing key should still return False
            "LANGFUSE_SECRET_KEY": "sk-lf-secret",
            "LANGFUSE_HOST": "http://localhost:3000",
        }

        with patch.dict(os.environ, env_vars, clear=True):
            result = await manager.initialize("test prompt", "test-namespace")

        # Should fail due to missing public key, but LANGFUSE_ENABLED was recognized as true
        assert result is False

    @pytest.mark.asyncio
    @patch("observability.LANGFUSE_AVAILABLE", True)
    @pytest.mark.parametrize("enabled_value", ["0", "false", "False", "no", "NO", ""])
    async def test_langfuse_disabled_variations(self, enabled_value, manager):
        """Test that various falsy values for LANGFUSE_ENABLED work."""
        env_vars = {
            "LANGFUSE_ENABLED": enabled_value,
            "LANGFUSE_PUBLIC_KEY": "pk-lf-public",
            "LANGFUSE_SECRET_KEY": "sk-lf-secret",
            "LANGFUSE_HOST": "http://localhost:3000",
        }

        with patch.dict(os.environ, env_vars, clear=True):
            result = await manager.initialize("test prompt", "test-namespace")

        assert result is False
        assert manager.langfuse_client is None
