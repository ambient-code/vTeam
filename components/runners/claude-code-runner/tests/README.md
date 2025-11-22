# Claude Code Runner Tests

This directory contains unit tests for the Claude Code runner components.

## Test Files

- `test_observability.py` - Tests for Langfuse observability manager
- `test_security_utils.py` - Tests for security utilities (secret sanitization, timeouts)
- `test_model_mapping.py` - Tests for model mapping (existing)
- `test_wrapper_vertex.py` - Tests for Vertex AI wrapper (existing)

## Running Tests

### Prerequisites

Install development dependencies:

```bash
# Using uv (recommended)
uv pip install -e ".[dev]"

# Or using pip
pip install -e ".[dev]"
```

### Run All Tests

```bash
# Run all tests with verbose output
pytest -v

# Run all tests with coverage
pytest --cov=. --cov-report=term-missing
```

### Run Specific Test Files

```bash
# Run observability tests only
pytest tests/test_observability.py -v

# Run security utils tests only
pytest tests/test_security_utils.py -v

# Run a specific test class
pytest tests/test_observability.py::TestLangfuseInitialization -v

# Run a specific test
pytest tests/test_observability.py::TestLangfuseInitialization::test_init_missing_public_key -v
```

### Run Tests with Output

```bash
# Show print statements and log output
pytest -v -s

# Show captured logs
pytest -v --log-cli-level=DEBUG
```

## Test Coverage

The test suite covers:

### Observability (`test_observability.py`)

- ✅ Initialization with missing/invalid credentials
- ✅ Graceful fallback when Langfuse SDK unavailable
- ✅ Secret sanitization in error messages
- ✅ Generation tracking with usage data
- ✅ Tool execution tracking (start and result)
- ✅ Session finalization with and without result payload
- ✅ Error cleanup and flushing
- ✅ All environment variable combinations

### Security Utils (`test_security_utils.py`)

- ✅ Exception message sanitization (prevent API key leaks)
- ✅ Async timeout wrapper (`with_timeout`)
- ✅ Sync-to-async timeout wrapper (`with_sync_timeout`)
- ✅ Log value sanitization (control character removal, truncation)
- ✅ Timeout logging behavior
- ✅ Exception logging behavior

## Writing New Tests

### Test Structure

```python
import pytest
from unittest.mock import Mock, patch

class TestFeatureName:
    """Tests for specific feature."""

    def test_basic_case(self):
        """Test description."""
        # Arrange
        # Act
        # Assert
        pass

    @pytest.mark.asyncio
    async def test_async_case(self):
        """Test async function."""
        # Arrange
        # Act
        # Assert
        pass
```

### Mocking Langfuse

```python
@patch('observability.Langfuse')
def test_with_langfuse(mock_langfuse_class):
    mock_client = Mock()
    mock_langfuse_class.return_value = mock_client
    # Test code
```

### Testing Environment Variables

```python
with patch.dict(os.environ, {'VAR_NAME': 'value'}, clear=True):
    # Test code with specific env vars
```

## Continuous Integration

Tests run automatically on:
- Pull requests
- Commits to main branch
- Via GitHub Actions workflow

See `.github/workflows/` for CI configuration.

## Troubleshooting

### Import Errors

If you see `ModuleNotFoundError`, ensure dependencies are installed:

```bash
uv pip install -e ".[dev]"
```

### Async Test Failures

Make sure to:
1. Use `@pytest.mark.asyncio` decorator for async tests
2. Install `pytest-asyncio` (included in dev dependencies)

### Mock Issues

When mocking:
- Use `@patch` for module-level imports
- Use `Mock()` for object instances
- Use `AsyncMock()` for async methods

## Adding Test Coverage

To check which lines need test coverage:

```bash
pytest --cov=. --cov-report=html
open htmlcov/index.html
```
