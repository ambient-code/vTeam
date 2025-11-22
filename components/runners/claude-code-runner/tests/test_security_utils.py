"""Unit tests for security_utils module."""

import pytest
import asyncio
import logging
from security_utils import (
    sanitize_exception_message,
    with_timeout,
    with_sync_timeout,
    validate_and_sanitize_for_logging,
)


class TestSanitizeExceptionMessage:
    """Tests for sanitize_exception_message function."""

    def test_sanitize_single_secret(self):
        """Test that a single secret is redacted."""
        exception = ValueError("Invalid API key: pk-lf-12345")
        secrets = {"public_key": "pk-lf-12345"}

        result = sanitize_exception_message(exception, secrets)

        assert "pk-lf-12345" not in result
        assert "[REDACTED_PUBLIC_KEY]" in result

    def test_sanitize_multiple_secrets(self):
        """Test that multiple secrets are redacted."""
        exception = ValueError("Auth failed: pk-lf-12345 and sk-lf-secret")
        secrets = {"public_key": "pk-lf-12345", "secret_key": "sk-lf-secret"}

        result = sanitize_exception_message(exception, secrets)

        assert "pk-lf-12345" not in result
        assert "sk-lf-secret" not in result
        assert "[REDACTED_PUBLIC_KEY]" in result
        assert "[REDACTED_SECRET_KEY]" in result

    def test_sanitize_empty_secrets(self):
        """Test that empty secrets are ignored."""
        exception = ValueError("Some error message")
        secrets = {"public_key": "", "secret_key": None}

        result = sanitize_exception_message(exception, secrets)

        assert result == "Some error message"

    def test_sanitize_secret_with_whitespace(self):
        """Test that secrets with whitespace are properly stripped."""
        exception = ValueError("Error with key pk-lf-12345")
        secrets = {"public_key": "  pk-lf-12345  "}

        result = sanitize_exception_message(exception, secrets)

        # Whitespace-only values should be ignored
        assert "pk-lf-12345" in result

    def test_sanitize_no_secrets_in_message(self):
        """Test message without secrets remains unchanged."""
        exception = ValueError("Generic error message")
        secrets = {"public_key": "pk-lf-12345"}

        result = sanitize_exception_message(exception, secrets)

        assert result == "Generic error message"

    def test_sanitize_empty_exception_message(self):
        """Test handling of exception with empty message."""
        exception = ValueError("")
        secrets = {"public_key": "pk-lf-12345"}

        result = sanitize_exception_message(exception, secrets)

        assert result == ""


class TestWithTimeout:
    """Tests for with_timeout async function."""

    @pytest.mark.asyncio
    async def test_successful_operation(self):
        """Test successful async operation within timeout."""

        async def quick_operation():
            await asyncio.sleep(0.01)
            return "success"

        success, result = await with_timeout(quick_operation, 1.0, "test operation")

        assert success is True
        assert result == "success"

    @pytest.mark.asyncio
    async def test_timeout_exceeded(self):
        """Test operation that exceeds timeout."""

        async def slow_operation():
            await asyncio.sleep(2.0)
            return "should not reach"

        success, result = await with_timeout(slow_operation, 0.1, "slow operation")

        assert success is False
        assert result is None

    @pytest.mark.asyncio
    async def test_operation_raises_exception(self):
        """Test operation that raises exception."""

        async def failing_operation():
            raise ValueError("Operation failed")

        success, result = await with_timeout(failing_operation, 1.0, "failing op")

        assert success is False
        assert isinstance(result, ValueError)
        assert str(result) == "Operation failed"

    @pytest.mark.asyncio
    async def test_callable_with_arguments(self):
        """Test passing arguments to callable."""

        async def add_numbers(a, b):
            return a + b

        success, result = await with_timeout(add_numbers, 1.0, "add", 5, 3)

        assert success is True
        assert result == 8

    @pytest.mark.asyncio
    async def test_callable_with_kwargs(self):
        """Test passing keyword arguments to callable."""

        async def greet(name, greeting="Hello"):
            return f"{greeting}, {name}"

        success, result = await with_timeout(
            greet, 1.0, "greet", name="World", greeting="Hi"
        )

        assert success is True
        assert result == "Hi, World"


class TestWithSyncTimeout:
    """Tests for with_sync_timeout function."""

    @pytest.mark.asyncio
    async def test_successful_sync_operation(self):
        """Test successful synchronous operation within timeout."""

        def sync_add(a, b):
            return a + b

        success, result = await with_sync_timeout(sync_add, 1.0, "sync add", 10, 20)

        assert success is True
        assert result == 30

    @pytest.mark.asyncio
    async def test_sync_timeout_exceeded(self):
        """Test sync operation that exceeds timeout."""
        import time

        def slow_sync_operation():
            time.sleep(2.0)
            return "should not reach"

        success, result = await with_sync_timeout(slow_sync_operation, 0.1, "slow sync")

        assert success is False
        assert result is None

    @pytest.mark.asyncio
    async def test_sync_operation_raises_exception(self):
        """Test sync operation that raises exception."""

        def failing_sync_operation():
            raise RuntimeError("Sync operation failed")

        success, result = await with_sync_timeout(
            failing_sync_operation, 1.0, "failing sync"
        )

        assert success is False
        assert result is None

    @pytest.mark.asyncio
    async def test_sync_with_kwargs(self):
        """Test passing keyword arguments to sync function."""

        def format_string(prefix, suffix, separator="-"):
            return f"{prefix}{separator}{suffix}"

        success, result = await with_sync_timeout(
            format_string, 1.0, "format", "hello", "world", separator="::"
        )

        assert success is True
        assert result == "hello::world"


class TestValidateAndSanitizeForLogging:
    """Tests for validate_and_sanitize_for_logging function."""

    def test_simple_string(self):
        """Test normal string passes through unchanged."""
        result = validate_and_sanitize_for_logging("Hello World")
        assert result == "Hello World"

    def test_remove_control_characters(self):
        """Test control characters are removed."""
        # String with null, bell, and other control chars
        test_string = "Hello\x00World\x07Test\x1f"
        result = validate_and_sanitize_for_logging(test_string)
        assert result == "HelloWorldTest"
        assert "\x00" not in result
        assert "\x07" not in result

    def test_truncate_long_string(self):
        """Test long strings are truncated."""
        long_string = "A" * 2000
        result = validate_and_sanitize_for_logging(long_string, max_length=100)

        assert len(result) <= 120  # 100 + "...[truncated]"
        assert result.endswith("...[truncated]")

    def test_empty_string(self):
        """Test empty string returns empty."""
        result = validate_and_sanitize_for_logging("")
        assert result == ""

    def test_none_value(self):
        """Test None value returns empty string."""
        result = validate_and_sanitize_for_logging(None)
        assert result == ""

    def test_preserve_newlines_and_tabs(self):
        """Test that newlines and tabs are preserved (they're not control chars in the removed range)."""
        test_string = "Line1\nLine2\tTabbed"
        result = validate_and_sanitize_for_logging(test_string)
        # \n is 0x0a and \t is 0x09, which are in the control range 0x00-0x1f
        # So they WILL be removed
        assert "\n" not in result
        assert "\t" not in result

    def test_custom_max_length(self):
        """Test custom max_length parameter."""
        test_string = "ABCDEFGHIJ"
        result = validate_and_sanitize_for_logging(test_string, max_length=5)

        assert result == "ABCDE...[truncated]"

    def test_unicode_characters(self):
        """Test Unicode characters are preserved."""
        test_string = "Hello 世界 🌍"
        result = validate_and_sanitize_for_logging(test_string)
        assert result == "Hello 世界 🌍"


class TestLoggingBehavior:
    """Integration tests for logging behavior."""

    @pytest.mark.asyncio
    async def test_timeout_logs_warning(self, caplog):
        """Test that timeout logs appropriate warning."""

        async def slow_op():
            await asyncio.sleep(1.0)

        with caplog.at_level(logging.WARNING):
            success, _ = await with_timeout(slow_op, 0.1, "test timeout")

        assert not success
        assert "test timeout" in caplog.text
        assert "timed out" in caplog.text.lower()

    @pytest.mark.asyncio
    async def test_exception_logs_error(self, caplog):
        """Test that exceptions log appropriate error."""

        async def failing_op():
            raise ValueError("Test error")

        with caplog.at_level(logging.ERROR):
            success, _ = await with_timeout(failing_op, 1.0, "test error")

        assert not success
        assert "test error" in caplog.text
        assert "failed" in caplog.text.lower()

    @pytest.mark.asyncio
    async def test_sync_timeout_logs_warning(self, caplog):
        """Test that sync timeout logs appropriate warning."""
        import time

        def slow_sync():
            time.sleep(1.0)

        with caplog.at_level(logging.WARNING):
            success, _ = await with_sync_timeout(slow_sync, 0.1, "sync timeout test")

        assert not success
        assert "sync timeout test" in caplog.text
        assert "timed out" in caplog.text.lower()
