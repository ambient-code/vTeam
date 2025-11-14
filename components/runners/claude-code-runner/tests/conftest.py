"""
Pytest configuration for Claude Code Runner tests.

These tests require the container environment with runner_shell dependency.
Mark all tests to skip if running outside container.
"""
import sys
import pytest

# Check if we're in the container environment
try:
    # The container adds /app/runner-shell to the path
    sys.path.insert(0, '/app/runner-shell')
    import runner_shell  # noqa: F401
    IN_CONTAINER = True
except (ImportError, ModuleNotFoundError):
    IN_CONTAINER = False

# Skip all tests if not in container environment
if not IN_CONTAINER:
    collect_ignore_glob = ["test_*.py"]
    
    @pytest.fixture(scope="session", autouse=True)
    def skip_all_tests():
        pytest.skip("Tests require container environment with runner_shell dependency", allow_module_level=True)

