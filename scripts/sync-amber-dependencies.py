#!/usr/bin/env python3
"""
Amber Dependency Sync Script

Automatically updates Amber's dependency knowledge by parsing version
information from go.mod, pyproject.toml, and package.json files across
the ACP codebase.

This script is run weekly by .github/workflows/amber-dependency-sync.yml
to keep Amber's expertise current with the actual dependencies in use.

Usage:
    python scripts/sync-amber-dependencies.py

Exit codes:
    0 - Success (no changes or changes applied successfully)
    1 - Error (file not found, parsing failed, etc.)
"""

import json
import re
import sys
from datetime import datetime
from pathlib import Path
from typing import Dict


def parse_go_mod(file_path: Path) -> Dict[str, str]:
    """Parse a go.mod file and extract relevant dependency versions.

    Args:
        file_path: Path to go.mod file

    Returns:
        Dictionary mapping package names to versions

    Example:
        {'k8s.io/api': '0.34.0', 'github.com/gin-gonic/gin': '1.10.1'}
    """
    if not file_path.exists():
        print(f"Warning: {file_path} not found, skipping")
        return {}

    dependencies = {}

    with open(file_path, "r") as f:
        content = f.read()

    # Match direct dependencies (not indirect)
    # Format: module_name v.X.Y.Z
    pattern = r"^\s*([a-zA-Z0-9.\-_/]+)\s+v([0-9]+\.[0-9]+\.[0-9]+)"

    for line in content.split("\n"):
        # Skip indirect dependencies
        if "// indirect" in line:
            continue

        match = re.match(pattern, line.strip())
        if match:
            module, version = match.groups()
            dependencies[module] = version

    return dependencies


def parse_pyproject_toml(file_path: Path) -> Dict[str, str]:
    """Parse pyproject.toml and extract dependency versions.

    Args:
        file_path: Path to pyproject.toml file

    Returns:
        Dictionary mapping package names to version constraints

    Example:
        {'anthropic': '>=0.68.0', 'claude-agent-sdk': '>=0.1.4'}
    """
    if not file_path.exists():
        print(f"Warning: {file_path} not found, skipping")
        return {}

    dependencies = {}

    try:
        # Try to import toml library
        try:
            import tomllib  # Python 3.11+
        except ImportError:
            import tomli as tomllib  # Fallback for Python 3.10

        with open(file_path, "rb") as f:
            data = tomllib.load(f)

        # Extract from project.dependencies array
        if "project" in data and "dependencies" in data["project"]:
            for dep in data["project"]["dependencies"]:
                # Format: "package>=version" or "package[extras]>=version"
                match = re.match(r"([a-zA-Z0-9\-_]+)(\[[^\]]+\])?(>=|==)([0-9.]+)", dep)
                if match:
                    package = match.group(1)
                    extras = match.group(2) or ""
                    operator = match.group(3)
                    version = match.group(4)
                    dependencies[package + extras] = f"{operator}{version}"

    except Exception as e:
        print(f"Error parsing {file_path}: {e}")
        return {}

    return dependencies


def parse_package_json(file_path: Path) -> Dict[str, str]:
    """Parse package.json and extract dependency versions.

    Args:
        file_path: Path to package.json file

    Returns:
        Dictionary mapping package names to versions

    Example:
        {'next': '15.1.4', 'react': '19.0.0'}
    """
    if not file_path.exists():
        print(f"Warning: {file_path} not found, skipping")
        return {}

    dependencies = {}

    try:
        with open(file_path, "r") as f:
            data = json.load(f)

        # Combine dependencies and devDependencies
        for dep_type in ["dependencies", "devDependencies"]:
            if dep_type in data:
                for package, version in data[dep_type].items():
                    # Remove ^ or ~ prefix if present
                    clean_version = version.lstrip("^~")
                    dependencies[package] = clean_version

    except Exception as e:
        print(f"Error parsing {file_path}: {e}")
        return {}

    return dependencies


def generate_dependency_markdown(
    go_backend: Dict[str, str],
    go_operator: Dict[str, str],
    python_runner: Dict[str, str],
    js_frontend: Dict[str, str],
) -> str:
    """Generate markdown section for dependency versions.

    Args:
        go_backend: Backend Go dependencies
        go_operator: Operator Go dependencies
        python_runner: Runner Python dependencies
        js_frontend: Frontend JavaScript dependencies

    Returns:
        Formatted markdown string
    """
    # Combine Go dependencies from backend and operator
    go_deps = {**go_backend, **go_operator}

    # Extract key dependencies
    k8s_version = go_deps.get("k8s.io/api", "unknown")
    gin_version = go_deps.get("github.com/gin-gonic/gin", "unknown")
    websocket_version = go_deps.get("github.com/gorilla/websocket", "unknown")
    jwt_version = go_deps.get("github.com/golang-jwt/jwt/v5", "unknown")

    anthropic_version = python_runner.get("anthropic[vertex]", python_runner.get("anthropic", "unknown"))
    sdk_version = python_runner.get("claude-agent-sdk", "unknown")

    next_version = js_frontend.get("next", "unknown")
    react_version = js_frontend.get("react", "unknown")
    react_query_version = js_frontend.get("@tanstack/react-query", "unknown")

    langfuse_version = python_runner.get("langfuse", js_frontend.get("langfuse", "unknown"))

    markdown = f"""**Kubernetes Ecosystem:**
- `k8s.io/{{api,apimachinery,client-go}}@{k8s_version}` - Watch for breaking changes in 1.31+
- Operator patterns: reconciliation, watch reconnection, leader election
- RBAC: Understand namespace isolation, service account permissions

**Claude Code SDK:**
- `anthropic[vertex]{anthropic_version}`, `claude-agent-sdk{sdk_version}`
- Message types, tool use blocks, session resumption, MCP servers
- Cost tracking: `total_cost_usd`, token usage patterns

**OpenShift Specifics:**
- OAuth proxy authentication, Routes, SecurityContextConstraints
- Project isolation (namespace-scoped service accounts)

**Go Stack:**
- Gin v{gin_version}, gorilla/websocket v{websocket_version}, jwt/v5 v{jwt_version}
- Unstructured resources, dynamic clients

**NextJS Stack:**
- Next.js v{next_version}, React v{react_version}, React Query v{react_query_version}, Shadcn UI
- TypeScript strict mode, ESLint

**Langfuse:**
- Langfuse {langfuse_version} (observability integration)
- Tracing, cost analytics, integration points in ACP"""

    return markdown


def update_amber_agent_file(new_content: str, agent_file: Path) -> bool:
    """Update the AUTO-GENERATED section in Amber's agent file.

    Args:
        new_content: New dependency markdown content
        agent_file: Path to amber.md

    Returns:
        True if file was modified, False if no changes needed
    """
    if not agent_file.exists():
        print(f"Error: {agent_file} not found")
        return False

    with open(agent_file, "r") as f:
        content = f.read()

    # Find AUTO-GENERATED markers
    start_marker = "<!-- AUTO-GENERATED: Dependencies"
    end_marker = "<!-- END AUTO-GENERATED: Dependencies -->"

    start_idx = content.find(start_marker)
    end_idx = content.find(end_marker)

    if start_idx == -1 or end_idx == -1:
        print("Error: AUTO-GENERATED markers not found in agent file")
        print("Expected markers:")
        print(f"  {start_marker}")
        print(f"  {end_marker}")
        return False

    # Extract current content between markers
    # Skip to end of start marker comment line
    start_content_idx = content.find("\n", start_idx) + 1
    current_content = content[start_content_idx:end_idx].strip()

    # Check if content actually changed
    if current_content == new_content.strip():
        print("✓ Dependency versions are already current - no update needed")
        return False

    # Build new file content
    timestamp = datetime.now().strftime("%Y-%m-%d")
    new_marker = f"""<!-- AUTO-GENERATED: Dependencies - Last updated: {timestamp}
     This section is automatically updated weekly by .github/workflows/amber-dependency-sync.yml
     DO NOT EDIT MANUALLY - Changes will be overwritten -->

{new_content}

{end_marker}"""

    new_file_content = (
        content[:start_idx] + new_marker + content[end_idx + len(end_marker) :]
    )

    # Write updated content
    with open(agent_file, "w") as f:
        f.write(new_file_content)

    print(f"✅ Updated {agent_file} with current dependency versions")
    return True


def main() -> int:
    """Main entry point for the dependency sync script.

    Returns:
        Exit code (0 for success, 1 for error)
    """
    print("=" * 60)
    print("Amber Dependency Knowledge Sync")
    print("=" * 60)

    # Determine repository root (script is in scripts/ directory)
    script_dir = Path(__file__).parent
    repo_root = script_dir.parent

    print(f"Repository root: {repo_root}")
    print()

    # Parse dependency files
    print("Parsing dependency files...")

    go_backend = parse_go_mod(repo_root / "components" / "backend" / "go.mod")
    print(f"  Backend (Go): {len(go_backend)} dependencies")

    go_operator = parse_go_mod(repo_root / "components" / "operator" / "go.mod")
    print(f"  Operator (Go): {len(go_operator)} dependencies")

    python_runner = parse_pyproject_toml(
        repo_root / "components" / "runners" / "claude-code-runner" / "pyproject.toml"
    )
    print(f"  Runner (Python): {len(python_runner)} dependencies")

    js_frontend = parse_package_json(
        repo_root / "components" / "frontend" / "package.json"
    )
    print(f"  Frontend (JavaScript): {len(js_frontend)} dependencies")

    print()

    # Generate markdown
    print("Generating dependency markdown...")
    markdown_content = generate_dependency_markdown(
        go_backend, go_operator, python_runner, js_frontend
    )

    # Update Amber's agent file
    print("Updating Amber's agent file...")
    agent_file = repo_root / "agents" / "amber.md"

    file_modified = update_amber_agent_file(markdown_content, agent_file)

    print()
    print("=" * 60)

    if file_modified:
        print("✅ Sync completed successfully - file updated")
        print()
        print("Next steps:")
        print("  1. Review changes: git diff agents/amber.md")
        print("  2. Commit will be created automatically by GitHub Actions workflow")
        return 0
    else:
        print("✓ Sync completed - no changes needed")
        return 0


if __name__ == "__main__":
    sys.exit(main())
