# Build context must be: components/runners/
# Example: docker build -f claude-code-runner/Dockerfile -t vteam_claude_runner .
FROM python:3.11-slim

# Install system dependencies including uv
RUN apt-get update && apt-get install -y \
    git \
    curl \
    ca-certificates \
    && curl -fsSL https://deb.nodesource.com/setup_20.x | bash - \
    && apt-get install -y nodejs \
    && npm install -g @anthropic-ai/claude-code \
    && rm -rf /var/lib/apt/lists/*

# Install uv from official container image (recommended method)
COPY --from=ghcr.io/astral-sh/uv:latest /uv /usr/local/bin/uv

# Set environment variables
ENV PYTHONUNBUFFERED=1
ENV PYTHONDONTWRITEBYTECODE=1
ENV RUNNER_TYPE=claude
ENV HOME=/app
ENV SHELL=/bin/bash
ENV TERM=xterm-256color
# UV_SYSTEM_PYTHON=1: Allow uv to use system Python instead of managing its own
ENV UV_SYSTEM_PYTHON=1

# Create working directory
WORKDIR /app

# Copy runner-shell package first (shared infrastructure layer)
COPY runner-shell /app/runner-shell

# Install runner-shell using uv pip install (installed globally in system Python)
# This provides the base runner framework used by wrapper.py
RUN cd /app/runner-shell && uv pip install --no-cache .

# Copy dependency files first for better layer caching
# Changes to these files are infrequent, so this layer will be cached
COPY claude-code-runner/pyproject.toml claude-code-runner/uv.lock /app/claude-runner/

# Install claude-runner dependencies using uv.lock
# --frozen: Fail if lock file is out of sync with pyproject.toml (ensures reproducibility)
# --no-dev: Skip development dependencies for smaller production image
# Creates a virtual environment at /app/claude-runner/.venv with exact dependency versions
RUN cd /app/claude-runner && uv sync --frozen --no-dev

# Copy application code (changes frequently, so placed after dependency installation)
COPY claude-code-runner/wrapper.py /app/claude-runner/

# OpenShift compatibility: Set group permissions equal to owner for arbitrary user IDs
RUN chmod -R g=u /app && chmod -R g=u /usr/local && chmod g=u /etc/passwd

# Use 'uv run' to execute wrapper.py in the project's virtual environment
# --frozen: Use exact dependencies from uv.lock without checking for updates
# This ensures wrapper.py has access to claude-agent-sdk and other project dependencies
CMD ["uv", "run", "--frozen", "/app/claude-runner/wrapper.py"]
