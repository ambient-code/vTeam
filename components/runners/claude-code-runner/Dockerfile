# Use Python 3.11 slim image
FROM python:3.11-slim

# Set working directory
WORKDIR /app

RUN apt-get update && apt-get install -y \
    ca-certificates \
    curl \
    git \
    && curl -fsSL https://deb.nodesource.com/setup_20.x | bash - \
    && apt-get install -y nodejs \
    && npm install -g @anthropic-ai/claude-code \
    && rm -rf /var/lib/apt/lists/*

# Install uv package manager
RUN pip install --no-cache-dir uv

# Copy pyproject for better Docker layer caching
COPY pyproject.toml .

# Install dependencies with uv
RUN uv pip install --system -e .

# Copy the application files with proper permissions
COPY main.py /app/
COPY auth_handler.py /app/
COPY git_integration.py /app/
COPY CLAUDE.md /app/
RUN chmod 755 /app/main.py && chmod 644 /app/auth_handler.py && chmod 644 /app/git_integration.py && chmod 644 /app/CLAUDE.md

# Set OpenShift-compatible environment variables
ENV PYTHONUNBUFFERED=1
ENV PYTHONDONTWRITEBYTECODE=1
ENV HOME=/app
ENV SHELL=/bin/bash
ENV TERM=xterm-256color

# OpenShift runs with random UID - don't set USER directive
# Ensure /app directory is accessible to any UID, and make agent files world-readable as fallback
RUN chmod -R g=u /app && chmod -R g=u /usr/local && chmod g=u /etc/passwd

# Run the application
CMD ["python", "main.py"]
