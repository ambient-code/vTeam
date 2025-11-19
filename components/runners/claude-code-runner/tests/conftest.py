"""
Pytest configuration for Claude Code Runner tests.

These tests require the container environment with runner_shell dependency.
Tests are skipped when runner_shell module is not available (CI environment).
"""
import sys

# Check if we're in the container environment
try:
    # The container adds /app/runner-shell to the path
    sys.path.insert(0, '/app/runner-shell')
    import runner_shell  # noqa: F401
    IN_CONTAINER = True
except (ImportError, ModuleNotFoundError):
    IN_CONTAINER = False

# Skip all tests if not in container environment
# collect_ignore_glob prevents pytest from collecting test files entirely
if not IN_CONTAINER:
    collect_ignore_glob = ["test_*.py"]

